package s3

import (
	"bytes"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"fmt"
	"hash"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"cloudWeave/internal/auth"
	"cloudWeave/internal/chunk"
	"cloudWeave/internal/metadata"
	"cloudWeave/internal/metrics"
)

const DefaultChunkSize = 1024 * 1024

// ChunkStorageEngine abstracts storing/fetching individual chunks locally or across quorum.
type ChunkStorageEngine interface {
	PutChunk(chunkID string, data []byte) ([]string, error)
	GetChunk(chunkID string, locations []string) ([]byte, error)
}

type InFlightRecorder interface {
	Register(chunkIDs ...string)
	Unregister(chunkIDs ...string)
}

// S3Handler implements the Amazon S3 REST API surface.
type S3Handler struct {
	metaStore *metadata.Store
	engine    ChunkStorageEngine
	auth      *auth.Authenticator
	chunkSize int
	mpStore   *MultipartStore
	inFlight  InFlightRecorder
}

// NewS3Handler initializes a new S3Handler.
func NewS3Handler(metaStore *metadata.Store, engine ChunkStorageEngine, chunkSize int, authOpts ...*auth.Authenticator) *S3Handler {
	if chunkSize <= 0 {
		chunkSize = DefaultChunkSize
	}
	var authenticator *auth.Authenticator
	if len(authOpts) > 0 && authOpts[0] != nil {
		authenticator = authOpts[0]
	}

	return &S3Handler{
		metaStore: metaStore,
		engine:    engine,
		auth:      authenticator,
		chunkSize: chunkSize,
		mpStore:   NewMultipartStore(),
	}
}

func (s *S3Handler) SetInFlightRegistry(inFlight InFlightRecorder) {
	s.inFlight = inFlight
}

func (s *S3Handler) GetMultipartStore() *MultipartStore {
	return s.mpStore
}

func computeETag(data []byte) string {
	h := md5.Sum(data)
	return fmt.Sprintf(`"%s"`, hex.EncodeToString(h[:]))
}

func (s *S3Handler) authenticate(r *http.Request, bucket string) (*SigV4AuthResult, bool, int, string, string) {
	if s.auth == nil {
		return nil, true, 200, "", ""
	}

	authHeader := r.Header.Get("Authorization")
	queryAlg := r.URL.Query().Get("X-Amz-Algorithm")

	if strings.HasPrefix(authHeader, "AWS4-HMAC-SHA256") || queryAlg == "AWS4-HMAC-SHA256" {
		sigRes, err := VerifySigV4(r, s.auth)
		if err != nil {
			if err.Error() == "InvalidAccessKeyId" {
				return nil, false, http.StatusForbidden, "InvalidAccessKeyId", "The AWS Access Key Id you provided does not exist in our records."
			}
			return nil, false, http.StatusForbidden, "SignatureDoesNotMatch", "The request signature we calculated does not match the signature you provided."
		}

		if bucket != "" && !sigRes.Credential.CanAccessNamespace(bucket) {
			return nil, false, http.StatusForbidden, "AccessDenied", "Access Denied"
		}
		return sigRes, true, 200, "", ""
	}

	// Native or Bearer key authentication fallback
	key := auth.ExtractKey(r)
	if key == "" {
		return nil, false, http.StatusForbidden, "AccessDenied", "Access Denied: API key missing"
	}

	cred, ok := s.auth.ValidateKey(key)
	if !ok {
		return nil, false, http.StatusForbidden, "InvalidAccessKeyId", "The API key provided is invalid."
	}

	if bucket != "" && !cred.CanAccessNamespace(bucket) {
		return nil, false, http.StatusForbidden, "AccessDenied", "Access Denied"
	}

	return &SigV4AuthResult{Credential: cred}, true, 200, "", ""
}

// ServeHTTP dispatches incoming S3 requests to bucket or object handlers.
func (s *S3Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/")

	// GET / -> ListAllMyBuckets
	if path == "" {
		if r.Method == http.MethodGet {
			s.handleListBuckets(w, r)
		} else {
			WriteError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "The specified method is not allowed.", "/")
		}
		return
	}

	parts := strings.SplitN(path, "/", 2)
	bucket := parts[0]
	var key string
	if len(parts) > 1 {
		key = parts[1]
	}

	// Bucket-level requests (path == "bucket" or "bucket/")
	if key == "" {
		switch r.Method {
		case http.MethodPut:
			s.handleCreateBucket(w, r, bucket)
		case http.MethodDelete:
			s.handleDeleteBucket(w, r, bucket)
		case http.MethodGet:
			if r.URL.Query().Get("list-type") == "2" || r.URL.Query().Has("list-type") {
				s.handleListObjectsV2(w, r, bucket)
			} else {
				s.handleListObjectsV2(w, r, bucket)
			}
		case http.MethodHead:
			s.handleHeadBucket(w, r, bucket)
		default:
			WriteError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "Method not allowed", "/"+bucket)
		}
		return
	}

	// Object-level & Multipart requests
	q := r.URL.Query()
	uploadID := q.Get("uploadId")
	partNumberStr := q.Get("partNumber")
	hasUploads := q.Has("uploads")

	switch r.Method {
	case http.MethodPut:
		if uploadID != "" && partNumberStr != "" {
			s.handleUploadPart(w, r, bucket, key, uploadID, partNumberStr)
		} else {
			s.handlePutObject(w, r, bucket, key)
		}
	case http.MethodGet:
		s.handleGetObject(w, r, bucket, key)
	case http.MethodHead:
		s.handleHeadObject(w, r, bucket, key)
	case http.MethodDelete:
		if uploadID != "" {
			s.handleAbortMultipartUpload(w, r, bucket, key, uploadID)
		} else {
			s.handleDeleteObject(w, r, bucket, key)
		}
	case http.MethodPost:
		if hasUploads {
			s.handleCreateMultipartUpload(w, r, bucket, key)
		} else if uploadID != "" {
			s.handleCompleteMultipartUpload(w, r, bucket, key, uploadID)
		} else {
			WriteError(w, http.StatusBadRequest, "InvalidRequest", "Unsupported POST request", "/"+bucket+"/"+key)
		}
	default:
		WriteError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "Method not allowed", "/"+bucket+"/"+key)
	}
}

func (s *S3Handler) handleListBuckets(w http.ResponseWriter, r *http.Request) {
	_, ok, status, code, msg := s.authenticate(r, "")
	if !ok {
		WriteError(w, status, code, msg, "/")
		return
	}

	buckets := s.metaStore.ListBuckets()
	res := ListAllMyBucketsResult{
		Xmlns: S3Namespace,
		Owner: Owner{
			ID:          "cloudweave",
			DisplayName: "cloudweave",
		},
	}
	for _, b := range buckets {
		res.Buckets.Bucket = append(res.Buckets.Bucket, Bucket{
			Name:         b.Name,
			CreationDate: FormatISO8601(b.CreatedAt),
		})
	}

	WriteXML(w, http.StatusOK, res)
}

func (s *S3Handler) handleCreateBucket(w http.ResponseWriter, r *http.Request, bucket string) {
	_, ok, status, code, msg := s.authenticate(r, bucket)
	if !ok {
		WriteError(w, status, code, msg, "/"+bucket)
		return
	}

	if err := s.metaStore.CreateBucket(bucket); err != nil {
		WriteError(w, http.StatusInternalServerError, "InternalError", err.Error(), "/"+bucket)
		return
	}

	w.Header().Set("Location", "/"+bucket)
	w.WriteHeader(http.StatusOK)
}

func (s *S3Handler) handleDeleteBucket(w http.ResponseWriter, r *http.Request, bucket string) {
	_, ok, status, code, msg := s.authenticate(r, bucket)
	if !ok {
		WriteError(w, status, code, msg, "/"+bucket)
		return
	}

	if err := s.metaStore.DeleteBucket(bucket); err != nil {
		if strings.Contains(err.Error(), "not empty") {
			WriteError(w, http.StatusConflict, "BucketNotEmpty", "The bucket you tried to delete is not empty.", "/"+bucket)
			return
		}
		WriteError(w, http.StatusNotFound, "NoSuchBucket", "The specified bucket does not exist.", "/"+bucket)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *S3Handler) handleHeadBucket(w http.ResponseWriter, r *http.Request, bucket string) {
	_, ok, status, code, msg := s.authenticate(r, bucket)
	if !ok {
		WriteError(w, status, code, msg, "/"+bucket)
		return
	}

	if !s.metaStore.BucketExists(bucket) {
		WriteError(w, http.StatusNotFound, "NoSuchBucket", "The specified bucket does not exist.", "/"+bucket)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *S3Handler) handleListObjectsV2(w http.ResponseWriter, r *http.Request, bucket string) {
	_, ok, status, code, msg := s.authenticate(r, bucket)
	if !ok {
		WriteError(w, status, code, msg, "/"+bucket)
		return
	}

	if !s.metaStore.BucketExists(bucket) {
		WriteError(w, http.StatusNotFound, "NoSuchBucket", "The specified bucket does not exist.", "/"+bucket)
		return
	}

	q := r.URL.Query()
	prefix := q.Get("prefix")
	delimiter := q.Get("delimiter")
	startAfter := q.Get("start-after")
	continuationToken := q.Get("continuation-token")
	if continuationToken != "" {
		startAfter = continuationToken
	}

	maxKeys := 1000
	if mkStr := q.Get("max-keys"); mkStr != "" {
		if mk, err := strconv.Atoi(mkStr); err == nil && mk > 0 {
			maxKeys = mk
		}
	}

	contents, commonPrefixes, isTruncated, nextToken := s.metaStore.ListObjectsV2(bucket, prefix, delimiter, startAfter, maxKeys)

	res := ListBucketResult{
		Xmlns:             S3Namespace,
		Name:              bucket,
		Prefix:            prefix,
		MaxKeys:           maxKeys,
		Delimiter:         delimiter,
		IsTruncated:       isTruncated,
		ContinuationToken: continuationToken,
		StartAfter:        startAfter,
	}

	for _, c := range contents {
		res.Contents = append(res.Contents, ObjectContent{
			Key:          c.FileID,
			LastModified: FormatISO8601(time.Now().UTC()),
			ETag:         computeETag([]byte(c.FileID)),
			Size:         c.Size,
			StorageClass: "STANDARD",
		})
	}

	for _, cp := range commonPrefixes {
		res.CommonPrefixes = append(res.CommonPrefixes, CommonPrefix{Prefix: cp})
	}

	res.KeyCount = len(res.Contents) + len(res.CommonPrefixes)
	if isTruncated {
		res.NextContinuationToken = nextToken
	}

	WriteXML(w, http.StatusOK, res)
}

func (s *S3Handler) handlePutObject(w http.ResponseWriter, r *http.Request, bucket, key string) {
	sigRes, ok, status, code, msg := s.authenticate(r, bucket)
	if !ok {
		WriteError(w, status, code, msg, "/"+bucket+"/"+key)
		return
	}

	_ = s.metaStore.CreateBucket(bucket)
	if r.Body != nil {
		defer r.Body.Close()
	}

	var reader io.Reader = r.Body
	if r.Body == nil {
		reader = bytes.NewReader(nil)
	}

	contentEncoding := r.Header.Get("Content-Encoding")
	expectedHash := r.Header.Get("X-Amz-Content-Sha256")

	if strings.Contains(contentEncoding, "aws-chunked") || strings.HasPrefix(expectedHash, "STREAMING-") {
		chunkedReader := NewAWSChunkedReader(reader)
		if sigRes != nil && len(sigRes.SigningKey) > 0 {
			chunkedReader.SetSignatureValidator(sigRes.SigningKey, sigRes.SeedSignature, sigRes.AmzDate, sigRes.Scope)
		}
		reader = chunkedReader
	}

	var shaHasher hash.Hash
	if expectedHash != "" && expectedHash != "UNSIGNED-PAYLOAD" && !strings.HasPrefix(expectedHash, "STREAMING-") {
		shaHasher = sha256.New()
		reader = io.TeeReader(reader, shaHasher)
	}
	md5Hasher := md5.New()
	reader = io.TeeReader(reader, md5Hasher)

	var chunkIDs []string
	chunkLocations := make(map[string][]string)
	var mu sync.Mutex

	var inFlightRegistered []string
	var regMu sync.Mutex
	defer func() {
		regMu.Lock()
		if s.inFlight != nil && len(inFlightRegistered) > 0 {
			s.inFlight.Unregister(inFlightRegistered...)
		}
		regMu.Unlock()
	}()

	totalBytes, chunkIDs, err := chunk.SplitStream(reader, s.chunkSize, func(c chunk.Chunk) error {
		if s.inFlight != nil {
			s.inFlight.Register(c.ID)
			regMu.Lock()
			inFlightRegistered = append(inFlightRegistered, c.ID)
			regMu.Unlock()
		}
		locs, err := s.engine.PutChunk(c.ID, c.Data)
		if err != nil {
			return err
		}
		mu.Lock()
		chunkLocations[c.ID] = locs
		mu.Unlock()
		return nil
	})

	if err != nil {
		if errors.Is(err, ErrChunkSignatureMismatch) || strings.Contains(err.Error(), "SignatureDoesNotMatch") {
			WriteError(w, http.StatusForbidden, "SignatureDoesNotMatch", "The request signature we calculated does not match the signature you provided.", "/"+bucket+"/"+key)
			return
		}
		WriteError(w, http.StatusInternalServerError, "InternalError", fmt.Sprintf("Failed to store object stream: %v", err), "/"+bucket+"/"+key)
		return
	}

	if shaHasher != nil {
		actualHash := hex.EncodeToString(shaHasher.Sum(nil))
		if !strings.EqualFold(actualHash, expectedHash) {
			WriteError(w, http.StatusBadRequest, "BadDigest", "The Sha256 signature calculated does not match the provided SHA-256 payload hash.", "/"+bucket+"/"+key)
			return
		}
	}

	etag := fmt.Sprintf(`"%s"`, hex.EncodeToString(md5Hasher.Sum(nil)))
	contentType := r.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	manifest := metadata.Manifest{
		Namespace:      bucket,
		FileID:         key,
		Size:           totalBytes,
		ChunkIDs:       chunkIDs,
		ChunkLocations: chunkLocations,
		ContentType:    contentType,
	}

	if err := s.metaStore.RecordPlacement(manifest); err != nil {
		WriteError(w, http.StatusInternalServerError, "InternalError", err.Error(), "/"+bucket+"/"+key)
		return
	}

	metrics.IncFileUploads()

	w.Header().Set("ETag", etag)
	w.WriteHeader(http.StatusOK)
}

func (s *S3Handler) handleGetObject(w http.ResponseWriter, r *http.Request, bucket, key string) {
	_, ok, status, code, msg := s.authenticate(r, bucket)
	if !ok {
		WriteError(w, status, code, msg, "/"+bucket+"/"+key)
		return
	}

	manifest, found := s.metaStore.LookupScoped(bucket, key)
	if !found {
		WriteError(w, http.StatusNotFound, "NoSuchKey", "The specified key does not exist.", "/"+bucket+"/"+key)
		return
	}

	contentType := manifest.ContentType
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	etag := computeETag([]byte(manifest.FileID))
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("ETag", etag)
	w.Header().Set("Last-Modified", time.Now().UTC().Format(http.TimeFormat))
	w.Header().Set("Accept-Ranges", "bytes")

	rangeHeader := r.Header.Get("Range")
	if rangeHeader != "" && strings.HasPrefix(rangeHeader, "bytes=") {
		rangeStr := strings.TrimPrefix(rangeHeader, "bytes=")
		parts := strings.Split(rangeStr, "-")
		if len(parts) == 2 {
			start, _ := strconv.ParseInt(parts[0], 10, 64)
			endVal := manifest.Size - 1
			if parts[1] != "" {
				if parsedEnd, err := strconv.ParseInt(parts[1], 10, 64); err == nil && parsedEnd < endVal {
					endVal = parsedEnd
				}
			}
			if start >= 0 && start <= endVal && start < manifest.Size {
				subLen := endVal - start + 1
				w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, endVal, manifest.Size))
				w.Header().Set("Content-Length", fmt.Sprintf("%d", subLen))
				w.WriteHeader(http.StatusPartialContent)

				var currentOffset int64 = 0
				for _, chunkID := range manifest.ChunkIDs {
					locs := manifest.ChunkLocations[chunkID]
					data, err := s.engine.GetChunk(chunkID, locs)
					if err != nil {
						return
					}
					chunkLen := int64(len(data))
					chunkStart := currentOffset
					chunkEnd := currentOffset + chunkLen - 1

					if chunkEnd >= start && chunkStart <= endVal {
						sliceStart := int64(0)
						if start > chunkStart {
							sliceStart = start - chunkStart
						}
						sliceEnd := chunkLen
						if endVal < chunkEnd {
							sliceEnd = endVal - chunkStart + 1
						}
						if sliceStart < sliceEnd && sliceStart < chunkLen {
							w.Write(data[sliceStart:sliceEnd])
						}
					}

					currentOffset += chunkLen
					if currentOffset > endVal {
						break
					}
				}
				metrics.IncFileDownloads()
				return
			}
		}
	}

	metrics.IncFileDownloads()

	w.Header().Set("Content-Length", fmt.Sprintf("%d", manifest.Size))
	w.WriteHeader(http.StatusOK)

	for _, chunkID := range manifest.ChunkIDs {
		locs := manifest.ChunkLocations[chunkID]
		data, err := s.engine.GetChunk(chunkID, locs)
		if err != nil {
			return
		}
		w.Write(data)
	}
}

func (s *S3Handler) handleHeadObject(w http.ResponseWriter, r *http.Request, bucket, key string) {
	_, ok, status, code, msg := s.authenticate(r, bucket)
	if !ok {
		WriteError(w, status, code, msg, "/"+bucket+"/"+key)
		return
	}

	manifest, found := s.metaStore.LookupScoped(bucket, key)
	if !found {
		WriteError(w, http.StatusNotFound, "NoSuchKey", "The specified key does not exist.", "/"+bucket+"/"+key)
		return
	}

	contentType := manifest.ContentType
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", manifest.Size))
	w.Header().Set("ETag", computeETag([]byte(manifest.FileID)))
	w.Header().Set("Last-Modified", time.Now().UTC().Format(http.TimeFormat))
	w.Header().Set("Accept-Ranges", "bytes")
	w.WriteHeader(http.StatusOK)
}

func (s *S3Handler) handleDeleteObject(w http.ResponseWriter, r *http.Request, bucket, key string) {
	_, ok, status, code, msg := s.authenticate(r, bucket)
	if !ok {
		WriteError(w, status, code, msg, "/"+bucket+"/"+key)
		return
	}

	s.metaStore.DeleteScoped(bucket, key)
	w.WriteHeader(http.StatusNoContent)
}

func (s *S3Handler) handleCreateMultipartUpload(w http.ResponseWriter, r *http.Request, bucket, key string) {
	_, ok, status, code, msg := s.authenticate(r, bucket)
	if !ok {
		WriteError(w, status, code, msg, "/"+bucket+"/"+key)
		return
	}

	_ = s.metaStore.CreateBucket(bucket)
	uploadID := s.mpStore.CreateUpload(bucket, key)

	res := InitiateMultipartUploadResult{
		Xmlns:    S3Namespace,
		Bucket:   bucket,
		Key:      key,
		UploadId: uploadID,
	}

	WriteXML(w, http.StatusOK, res)
}

func (s *S3Handler) handleUploadPart(w http.ResponseWriter, r *http.Request, bucket, key, uploadID, partNumberStr string) {
	sigRes, ok, status, code, msg := s.authenticate(r, bucket)
	if !ok {
		WriteError(w, status, code, msg, "/"+bucket+"/"+key)
		return
	}

	partNum, err := strconv.Atoi(partNumberStr)
	if err != nil || partNum <= 0 {
		WriteError(w, http.StatusBadRequest, "InvalidArgument", "Invalid partNumber parameter", "/"+bucket+"/"+key)
		return
	}

	if r.Body != nil {
		defer r.Body.Close()
	}

	var reader io.Reader = r.Body
	if r.Body == nil {
		reader = bytes.NewReader(nil)
	}

	contentEncoding := r.Header.Get("Content-Encoding")
	expectedHash := r.Header.Get("X-Amz-Content-Sha256")

	if strings.Contains(contentEncoding, "aws-chunked") || strings.HasPrefix(expectedHash, "STREAMING-") {
		chunkedReader := NewAWSChunkedReader(reader)
		if sigRes != nil && len(sigRes.SigningKey) > 0 {
			chunkedReader.SetSignatureValidator(sigRes.SigningKey, sigRes.SeedSignature, sigRes.AmzDate, sigRes.Scope)
		}
		reader = chunkedReader
	}

	var shaHasher hash.Hash
	if expectedHash != "" && expectedHash != "UNSIGNED-PAYLOAD" && !strings.HasPrefix(expectedHash, "STREAMING-") {
		shaHasher = sha256.New()
		reader = io.TeeReader(reader, shaHasher)
	}
	md5Hasher := md5.New()
	reader = io.TeeReader(reader, md5Hasher)

	var chunkIDs []string
	chunkLocations := make(map[string][]string)
	var mu sync.Mutex

	totalBytes, chunkIDs, err := chunk.SplitStream(reader, s.chunkSize, func(c chunk.Chunk) error {
		locs, err := s.engine.PutChunk(c.ID, c.Data)
		if err != nil {
			return err
		}
		mu.Lock()
		chunkLocations[c.ID] = locs
		mu.Unlock()
		return nil
	})

	if err != nil {
		if errors.Is(err, ErrChunkSignatureMismatch) || strings.Contains(err.Error(), "SignatureDoesNotMatch") {
			WriteError(w, http.StatusForbidden, "SignatureDoesNotMatch", "The request signature we calculated does not match the signature you provided.", "/"+bucket+"/"+key)
			return
		}
		WriteError(w, http.StatusInternalServerError, "InternalError", fmt.Sprintf("Failed to store part stream: %v", err), "/"+bucket+"/"+key)
		return
	}

	if shaHasher != nil {
		actualHash := hex.EncodeToString(shaHasher.Sum(nil))
		if !strings.EqualFold(actualHash, expectedHash) {
			WriteError(w, http.StatusBadRequest, "BadDigest", "The Sha256 signature calculated does not match the provided SHA-256 payload hash.", "/"+bucket+"/"+key)
			return
		}
	}

	etag := fmt.Sprintf(`"%s"`, hex.EncodeToString(md5Hasher.Sum(nil)))

	if err := s.mpStore.AddPart(uploadID, partNum, etag, totalBytes, chunkIDs, chunkLocations); err != nil {
		WriteError(w, http.StatusNotFound, "NoSuchUpload", "The specified multipart upload does not exist.", "/"+bucket+"/"+key)
		return
	}

	w.Header().Set("ETag", etag)
	w.WriteHeader(http.StatusOK)
}

func (s *S3Handler) handleCompleteMultipartUpload(w http.ResponseWriter, r *http.Request, bucket, key, uploadID string) {
	_, ok, status, code, msg := s.authenticate(r, bucket)
	if !ok {
		WriteError(w, status, code, msg, "/"+bucket+"/"+key)
		return
	}

	rec, exists := s.mpStore.GetUpload(uploadID)
	if !exists {
		WriteError(w, http.StatusNotFound, "NoSuchUpload", "The specified multipart upload does not exist.", "/"+bucket+"/"+key)
		return
	}

	var reqParts []struct {
		PartNumber int
		ETag       string
	}

	bodyBytes, _ := io.ReadAll(io.LimitReader(r.Body, 1<<20)) // 1 MiB max CompleteMultipartUpload XML body (finding #13)
	if len(bodyBytes) > 0 {
		var reqBody CompleteMultipartUpload
		if err := xml.Unmarshal(bodyBytes, &reqBody); err == nil && len(reqBody.Parts) > 0 {
			for _, p := range reqBody.Parts {
				reqParts = append(reqParts, struct {
					PartNumber int
					ETag       string
				}{PartNumber: p.PartNumber, ETag: p.ETag})
			}
		}
	}

	// Fallback if no parts specified in body: complete all uploaded parts in order
	if len(reqParts) == 0 {
		for partNum, p := range rec.Parts {
			reqParts = append(reqParts, struct {
				PartNumber int
				ETag       string
			}{PartNumber: partNum, ETag: p.ETag})
		}
	}

	combinedChunkIDs, combinedChunkLocations, totalSize, err := s.mpStore.CompleteUpload(uploadID, reqParts)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "InvalidPart", err.Error(), "/"+bucket+"/"+key)
		return
	}

	manifest := metadata.Manifest{
		Namespace:      bucket,
		FileID:         key,
		Size:           totalSize,
		ChunkIDs:       combinedChunkIDs,
		ChunkLocations: combinedChunkLocations,
		ContentType:    "application/octet-stream",
	}

	if err := s.metaStore.RecordPlacement(manifest); err != nil {
		WriteError(w, http.StatusInternalServerError, "InternalError", err.Error(), "/"+bucket+"/"+key)
		return
	}

	metrics.IncFileUploads()

	res := CompleteMultipartUploadResult{
		Xmlns:    S3Namespace,
		Location: "/" + bucket + "/" + key,
		Bucket:   bucket,
		Key:      key,
		ETag:     computeETag([]byte(key)),
	}

	WriteXML(w, http.StatusOK, res)
}

func (s *S3Handler) handleAbortMultipartUpload(w http.ResponseWriter, r *http.Request, bucket, key, uploadID string) {
	_, ok, status, code, msg := s.authenticate(r, bucket)
	if !ok {
		WriteError(w, status, code, msg, "/"+bucket+"/"+key)
		return
	}

	if _, err := s.mpStore.AbortUpload(uploadID); err != nil {
		WriteError(w, http.StatusNotFound, "NoSuchUpload", "The specified multipart upload does not exist.", "/"+bucket+"/"+key)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

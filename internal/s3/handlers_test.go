package s3

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"runtime"
	"sync"
	"testing"
	"time"

	"cloudWeave/internal/auth"
	"cloudWeave/internal/gc"
	"cloudWeave/internal/metadata"
	"cloudWeave/internal/storage"
)

type mockEngine struct {
	mu     sync.Mutex
	chunks map[string][]byte
}

func newMockEngine() *mockEngine {
	return &mockEngine{chunks: make(map[string][]byte)}
}

func (m *mockEngine) PutChunk(chunkID string, data []byte) ([]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := make([]byte, len(data))
	copy(cp, data)
	m.chunks[chunkID] = cp
	return []string{"node1"}, nil
}

func (m *mockEngine) GetChunk(chunkID string, locations []string) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	data, ok := m.chunks[chunkID]
	if !ok {
		return nil, io.ErrUnexpectedEOF
	}
	return data, nil
}

func setupTestS3Handler() (*S3Handler, *metadata.Store, *auth.Authenticator) {
	metaStore := metadata.NewStore()
	engine := newMockEngine()
	authenticator := auth.NewDefaultAuthenticator()
	handler := NewS3Handler(metaStore, engine, 1024, authenticator)
	return handler, metaStore, authenticator
}

func TestS3BucketLifecycle(t *testing.T) {
	handler, _, authenticator := setupTestS3Handler()
	rawKey := "test-admin-key-123"
	authenticator.AddRawKey(rawKey, []string{"*"}, true)

	// 1. List buckets (initially "default")
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", rawKey)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var listRes ListAllMyBucketsResult
	if err := xml.Unmarshal(w.Body.Bytes(), &listRes); err != nil {
		t.Fatalf("failed to unmarshal list buckets XML: %v", err)
	}
	if len(listRes.Buckets.Bucket) == 0 {
		t.Fatalf("expected at least default bucket")
	}

	// 2. Create bucket "my-bucket"
	req = httptest.NewRequest(http.MethodPut, "/my-bucket", nil)
	req.Header.Set("Authorization", rawKey)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 on bucket creation, got %d", w.Code)
	}

	// 3. Delete bucket "my-bucket"
	req = httptest.NewRequest(http.MethodDelete, "/my-bucket", nil)
	req.Header.Set("Authorization", rawKey)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204 on bucket deletion, got %d", w.Code)
	}
}

func TestS3ObjectLifecycle(t *testing.T) {
	handler, _, authenticator := setupTestS3Handler()
	rawKey := "test-admin-key-123"
	authenticator.AddRawKey(rawKey, []string{"*"}, true)

	// 1. Put Object
	bodyContent := []byte("Hello CloudWeave S3 API!")
	req := httptest.NewRequest(http.MethodPut, "/test-bucket/hello.txt", bytes.NewReader(bodyContent))
	req.Header.Set("Authorization", rawKey)
	req.Header.Set("Content-Type", "text/plain")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 on put object, got %d: %s", w.Code, w.Body.String())
	}
	etag := w.Header().Get("ETag")
	if etag == "" {
		t.Fatalf("expected ETag header on put object")
	}

	// 2. Head Object
	req = httptest.NewRequest(http.MethodHead, "/test-bucket/hello.txt", nil)
	req.Header.Set("Authorization", rawKey)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 on head object, got %d", w.Code)
	}
	if w.Header().Get("Content-Length") != "24" {
		t.Errorf("expected Content-Length 24, got %s", w.Header().Get("Content-Length"))
	}

	// 3. Get Object
	req = httptest.NewRequest(http.MethodGet, "/test-bucket/hello.txt", nil)
	req.Header.Set("Authorization", rawKey)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 on get object, got %d", w.Code)
	}
	if !bytes.Equal(w.Body.Bytes(), bodyContent) {
		t.Fatalf("expected body %q, got %q", string(bodyContent), w.Body.String())
	}

	// 4. ListObjectsV2
	req = httptest.NewRequest(http.MethodGet, "/test-bucket?list-type=2", nil)
	req.Header.Set("Authorization", rawKey)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 on ListObjectsV2, got %d", w.Code)
	}
	var listObjRes ListBucketResult
	if err := xml.Unmarshal(w.Body.Bytes(), &listObjRes); err != nil {
		t.Fatalf("failed to unmarshal ListObjectsV2 XML: %v", err)
	}
	if len(listObjRes.Contents) != 1 || listObjRes.Contents[0].Key != "hello.txt" {
		t.Fatalf("expected 1 object 'hello.txt', got %v", listObjRes.Contents)
	}

	// 5. Delete Object
	req = httptest.NewRequest(http.MethodDelete, "/test-bucket/hello.txt", nil)
	req.Header.Set("Authorization", rawKey)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204 on delete object, got %d", w.Code)
	}
}

func TestS3MultipartUpload(t *testing.T) {
	handler, _, authenticator := setupTestS3Handler()
	rawKey := "test-admin-key-123"
	authenticator.AddRawKey(rawKey, []string{"*"}, true)

	// 1. Create Multipart Upload
	req := httptest.NewRequest(http.MethodPost, "/mp-bucket/bigfile.bin?uploads", nil)
	req.Header.Set("Authorization", rawKey)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 on create multipart, got %d: %s", w.Code, w.Body.String())
	}

	var initRes InitiateMultipartUploadResult
	if err := xml.Unmarshal(w.Body.Bytes(), &initRes); err != nil {
		t.Fatalf("failed to unmarshal InitiateMultipartUploadResult XML: %v", err)
	}
	uploadID := initRes.UploadId
	if uploadID == "" {
		t.Fatalf("expected valid UploadId")
	}

	// 2. Upload Part 1
	part1Data := []byte("Part 1 payload data ")
	req = httptest.NewRequest(http.MethodPut, "/mp-bucket/bigfile.bin?partNumber=1&uploadId="+uploadID, bytes.NewReader(part1Data))
	req.Header.Set("Authorization", rawKey)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 on upload part 1, got %d", w.Code)
	}
	etag1 := w.Header().Get("ETag")

	// 3. Upload Part 2
	part2Data := []byte("Part 2 payload data")
	req = httptest.NewRequest(http.MethodPut, "/mp-bucket/bigfile.bin?partNumber=2&uploadId="+uploadID, bytes.NewReader(part2Data))
	req.Header.Set("Authorization", rawKey)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 on upload part 2, got %d", w.Code)
	}
	etag2 := w.Header().Get("ETag")

	// 4. Complete Multipart Upload
	completeXML := fmt.Sprintf(`<CompleteMultipartUpload><Part><PartNumber>1</PartNumber><ETag>%s</ETag></Part><Part><PartNumber>2</PartNumber><ETag>%s</ETag></Part></CompleteMultipartUpload>`, etag1, etag2)
	req = httptest.NewRequest(http.MethodPost, "/mp-bucket/bigfile.bin?uploadId="+uploadID, bytes.NewReader([]byte(completeXML)))
	req.Header.Set("Authorization", rawKey)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 on complete multipart, got %d: %s", w.Code, w.Body.String())
	}

	// 5. Get Completed Object
	req = httptest.NewRequest(http.MethodGet, "/mp-bucket/bigfile.bin", nil)
	req.Header.Set("Authorization", rawKey)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 on get completed object, got %d", w.Code)
	}
	expectedData := append(part1Data, part2Data...)
	if !bytes.Equal(w.Body.Bytes(), expectedData) {
		t.Fatalf("expected body %q, got %q", string(expectedData), w.Body.String())
	}
}

func TestS3StreamingRangeAndSha256Validation(t *testing.T) {
	handler, _, authenticator := setupTestS3Handler()
	rawKey := "test-admin-key-123"
	authenticator.AddRawKey(rawKey, []string{"*"}, true)

	// 1. Put Object with valid X-Amz-Content-Sha256 payload hash header
	content := []byte("0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ")
	sum := sha256.Sum256(content)
	hexHash := hex.EncodeToString(sum[:])

	req := httptest.NewRequest(http.MethodPut, "/range-bucket/stream.txt", bytes.NewReader(content))
	req.Header.Set("Authorization", rawKey)
	req.Header.Set("X-Amz-Content-Sha256", hexHash)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 on put object with sha256, got %d: %s", w.Code, w.Body.String())
	}

	// 2. Put Object with INVALID X-Amz-Content-Sha256 header (expect 400 BadDigest)
	req = httptest.NewRequest(http.MethodPut, "/range-bucket/bad.txt", bytes.NewReader(content))
	req.Header.Set("Authorization", rawKey)
	req.Header.Set("X-Amz-Content-Sha256", "0000000000000000000000000000000000000000000000000000000000000000")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 BadDigest on sha256 mismatch, got %d", w.Code)
	}

	// 3. Test Streaming Range GET
	req = httptest.NewRequest(http.MethodGet, "/range-bucket/stream.txt", nil)
	req.Header.Set("Authorization", rawKey)
	req.Header.Set("Range", "bytes=10-19")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusPartialContent {
		t.Fatalf("expected 206 Partial Content, got %d", w.Code)
	}
	if w.Header().Get("Content-Range") != "bytes 10-19/36" {
		t.Fatalf("expected Content-Range 'bytes 10-19/36', got %s", w.Header().Get("Content-Range"))
	}
	expectedSlice := string(content[10:20])
	if w.Body.String() != expectedSlice {
		t.Fatalf("expected range body %q, got %q", expectedSlice, w.Body.String())
	}
}

type zeroStream struct {
	remaining int64
}

func (z *zeroStream) Read(p []byte) (n int, err error) {
	if z.remaining <= 0 {
		return 0, io.EOF
	}
	toRead := int64(len(p))
	if toRead > z.remaining {
		toRead = z.remaining
	}
	for i := int64(0); i < toRead; i++ {
		p[i] = 'A'
	}
	z.remaining -= toRead
	return int(toRead), nil
}

func TestS3StreamingMemoryBounded(t *testing.T) {
	handler, _, authenticator := setupTestS3Handler()
	rawKey := "test-admin-key-123"
	authenticator.AddRawKey(rawKey, []string{"*"}, true)

	// Stream 256 MB payload via zeroStream (0 bytes allocated in RAM for payload)
	const uploadSize = int64(256 * 1024 * 1024)
	stream := &zeroStream{remaining: uploadSize}

	req := httptest.NewRequest(http.MethodPut, "/mem-bucket/large.bin", stream)
	req.Header.Set("Authorization", rawKey)
	req.Header.Set("X-Amz-Content-Sha256", "UNSIGNED-PAYLOAD")

	runtime.GC()
	var mBefore runtime.MemStats
	runtime.ReadMemStats(&mBefore)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	runtime.GC()
	var mAfter runtime.MemStats
	runtime.ReadMemStats(&mAfter)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 on 256MB streaming put object, got %d: %s", w.Code, w.Body.String())
	}

	heapAllocMB := float64(mAfter.Alloc) / (1024 * 1024)
	t.Logf("Heap Alloc after 256MB upload: %.2f MB", heapAllocMB)

	// If payload was buffered in RAM, heapAllocMB would be > 256 MB.
	// Because it is streamed in 1MB chunks, HeapAlloc stays well under 50 MB!
	if heapAllocMB > 50.0 {
		t.Fatalf("memory stream leaked! Heap alloc %.2f MB exceeds 50 MB threshold for 256MB upload", heapAllocMB)
	}
}

type errorStream struct {
	remaining int64
	readSoFar int64
}

func (e *errorStream) Read(p []byte) (n int, err error) {
	if e.readSoFar >= 3*1024*1024 {
		return 0, fmt.Errorf("simulated network disconnect mid-upload")
	}
	toRead := int64(len(p))
	if toRead > e.remaining {
		toRead = e.remaining
	}
	for i := int64(0); i < toRead; i++ {
		p[i] = 'B'
	}
	e.readSoFar += toRead
	e.remaining -= toRead
	return int(toRead), nil
}

type testDiskEngine struct {
	diskStore *storage.DiskStore
}

func (d *testDiskEngine) PutChunk(chunkID string, data []byte) ([]string, error) {
	if err := d.diskStore.Put(chunkID, data); err != nil {
		return nil, err
	}
	return []string{"node1"}, nil
}

func (d *testDiskEngine) GetChunk(chunkID string, locations []string) ([]byte, error) {
	return d.diskStore.Get(chunkID)
}

func TestAbortedUploadGCCleanup(t *testing.T) {
	metaStore := metadata.NewStore()
	diskDir := t.TempDir()
	diskStore, err := storage.NewDiskStore(diskDir)
	if err != nil {
		t.Fatalf("failed to create disk store: %v", err)
	}

	adapter := &testDiskEngine{diskStore: diskStore}

	authenticator := auth.NewDefaultAuthenticator()
	rawKey := "test-admin-key-123"
	authenticator.AddRawKey(rawKey, []string{"*"}, true)

	handler := NewS3Handler(metaStore, adapter, 1024*1024, authenticator)

	// Stream upload that fails midway after writing 3MB of chunks to disk
	stream := &errorStream{remaining: 10 * 1024 * 1024}
	req := httptest.NewRequest(http.MethodPut, "/abort-bucket/aborted.bin", stream)
	req.Header.Set("Authorization", rawKey)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code == http.StatusOK {
		t.Fatalf("expected upload failure on network abort, got 200 OK")
	}

	// Verify chunks landed on disk before abort
	chunksOnDisk, err := diskStore.ListChunks()
	if err != nil {
		t.Fatalf("failed to list chunks: %v", err)
	}
	if len(chunksOnDisk) == 0 {
		t.Fatalf("expected partial chunks on disk from streaming before abort")
	}

	// Verify NO manifest was registered for aborted.bin
	_, found := metaStore.LookupScoped("abort-bucket", "aborted.bin")
	if found {
		t.Fatalf("manifest should NOT exist for aborted upload")
	}

	// Run Mark-and-Sweep Garbage Collector
	collector := gc.NewGarbageCollector(metaStore, diskStore)
	sweptCount, err := collector.CollectGarbage()
	if err != nil {
		t.Fatalf("GC failed: %v", err)
	}

	if sweptCount == 0 {
		t.Fatalf("expected GC to sweep orphaned chunks from aborted upload")
	}

	chunksAfterGC, _ := diskStore.ListChunks()
	if len(chunksAfterGC) != 0 {
		t.Fatalf("expected 0 chunks on disk after GC sweep, found %d", len(chunksAfterGC))
	}
}

func TestS3AWSChunkedDecodingRoundTrip(t *testing.T) {
	handler, _, authenticator := setupTestS3Handler()
	rawKey := "test-admin-key-123"
	authenticator.AddRawKey(rawKey, []string{"*"}, true)

	// Clean payload data
	originalData := []byte("Hello, this is a test payload encoded with AWS aws-chunked streaming headers!")

	// Build aws-chunked framed body:
	// <hex-size>;chunk-signature=<sig>\r\n<data>\r\n0;chunk-signature=<sig>\r\n\r\n
	var framedBody bytes.Buffer
	chunk1Data := originalData[:20]
	chunk2Data := originalData[20:]

	framedBody.WriteString(fmt.Sprintf("%x;chunk-signature=1111111111111111111111111111111111111111111111111111111111111111\r\n", len(chunk1Data)))
	framedBody.Write(chunk1Data)
	framedBody.WriteString("\r\n")

	framedBody.WriteString(fmt.Sprintf("%x;chunk-signature=2222222222222222222222222222222222222222222222222222222222222222\r\n", len(chunk2Data)))
	framedBody.Write(chunk2Data)
	framedBody.WriteString("\r\n")

	framedBody.WriteString("0;chunk-signature=3333333333333333333333333333333333333333333333333333333333333333\r\n\r\n")

	// PUT with aws-chunked headers
	req := httptest.NewRequest(http.MethodPut, "/chunk-bucket/chunked.txt", &framedBody)
	req.Header.Set("Authorization", rawKey)
	req.Header.Set("Content-Encoding", "aws-chunked")
	req.Header.Set("X-Amz-Content-Sha256", "STREAMING-AWS4-HMAC-SHA256-PAYLOAD")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 on aws-chunked PUT, got %d: %s", w.Code, w.Body.String())
	}

	// GET object and verify downloaded content matches originalData byte-for-byte
	req = httptest.NewRequest(http.MethodGet, "/chunk-bucket/chunked.txt", nil)
	req.Header.Set("Authorization", rawKey)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 on GET chunked object, got %d", w.Code)
	}

	downloadedData := w.Body.Bytes()
	if !bytes.Equal(downloadedData, originalData) {
		t.Fatalf("data corruption detected!\nExpected clean content (%d bytes): %q\nGot stored content (%d bytes): %q",
			len(originalData), string(originalData), len(downloadedData), string(downloadedData))
	}
}

type fragmentedReader struct {
	r    io.Reader
	step int
}

func (f *fragmentedReader) Read(p []byte) (n int, err error) {
	if len(p) == 0 {
		return 0, nil
	}
	f.step++
	maxLen := (f.step % 3) + 1 // Cycles 1, 2, 3 bytes
	if maxLen > len(p) {
		maxLen = len(p)
	}
	return f.r.Read(p[:maxLen])
}

func TestS3AWSChunkedDecodingFragmentedRead(t *testing.T) {
	handler, _, authenticator := setupTestS3Handler()
	rawKey := "test-admin-key-123"
	authenticator.AddRawKey(rawKey, []string{"*"}, true)

	originalData := []byte("Fragmented network read payload testing AWSChunkedReader partial header and partial payload buffering capabilities across 1-3 byte reads!")

	var framedBody bytes.Buffer
	chunk1Data := originalData[:40]
	chunk2Data := originalData[40:]

	framedBody.WriteString(fmt.Sprintf("%x;chunk-signature=1111111111111111111111111111111111111111111111111111111111111111\r\n", len(chunk1Data)))
	framedBody.Write(chunk1Data)
	framedBody.WriteString("\r\n")

	framedBody.WriteString(fmt.Sprintf("%x;chunk-signature=2222222222222222222222222222222222222222222222222222222222222222\r\n", len(chunk2Data)))
	framedBody.Write(chunk2Data)
	framedBody.WriteString("\r\n")

	framedBody.WriteString("0;chunk-signature=3333333333333333333333333333333333333333333333333333333333333333\r\n\r\n")

	fragStream := &fragmentedReader{r: &framedBody}

	req := httptest.NewRequest(http.MethodPut, "/frag-bucket/frag.txt", fragStream)
	req.Header.Set("Authorization", rawKey)
	req.Header.Set("Content-Encoding", "aws-chunked")
	req.Header.Set("X-Amz-Content-Sha256", "STREAMING-AWS4-HMAC-SHA256-PAYLOAD")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on fragmented aws-chunked PUT, got %d: %s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/frag-bucket/frag.txt", nil)
	req.Header.Set("Authorization", rawKey)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on GET fragmented object, got %d", w.Code)
	}

	if !bytes.Equal(w.Body.Bytes(), originalData) {
		t.Fatalf("fragmented read data corruption!\nExpected: %q\nGot: %q", string(originalData), w.Body.String())
	}
}

func TestS3AWSChunkedSignatureValidation(t *testing.T) {
	handler, _, authenticator := setupTestS3Handler()
	accessKey := "AKIAIOSFODNN7EXAMPLE"
	secretKey := "AKIAIOSFODNN7EXAMPLE"
	authenticator.AddRawKey(accessKey, []string{"*"}, true)

	now := time.Now().UTC()
	dateStamp := now.Format("20060102")
	amzDate := now.Format("20060102T150405Z")
	region := "us-east-1"
	service := "s3"
	scope := fmt.Sprintf("%s/%s/%s/aws4_request", dateStamp, region, service)
	signingKey := deriveSigningKey(secretKey, dateStamp, region, service)

	canonicalHeaders := fmt.Sprintf("host:localhost:8080\nx-amz-content-sha256:STREAMING-AWS4-HMAC-SHA256-PAYLOAD\nx-amz-date:%s\n", amzDate)
	signedHeaders := "host;x-amz-content-sha256;x-amz-date"
	canonicalReq := fmt.Sprintf("PUT\n/sig-bucket/valid.txt\n\n%s\n%s\nSTREAMING-AWS4-HMAC-SHA256-PAYLOAD", canonicalHeaders, signedHeaders)
	reqHash := sha256.Sum256([]byte(canonicalReq))
	stringToSign := fmt.Sprintf("AWS4-HMAC-SHA256\n%s\n%s\n%s", amzDate, scope, hex.EncodeToString(reqHash[:]))
	seedSigBytes := hmacSHA256(signingKey, []byte(stringToSign))
	seedSig := hex.EncodeToString(seedSigBytes)

	chunk1Data := []byte("Hello CloudWeave SigV4 streaming chunk!")
	chunk1Hash := sha256.Sum256(chunk1Data)
	chunk1HashHex := hex.EncodeToString(chunk1Hash[:])

	emptyHeaderHash := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	chunk1StringToSign := fmt.Sprintf("AWS4-HMAC-SHA256-PAYLOAD\n%s\n%s\n%s\n%s\n%s", amzDate, scope, seedSig, emptyHeaderHash, chunk1HashHex)
	chunk1Sig := hex.EncodeToString(hmacSHA256(signingKey, []byte(chunk1StringToSign)))

	chunk2StringToSign := fmt.Sprintf("AWS4-HMAC-SHA256-PAYLOAD\n%s\n%s\n%s\n%s\n%s", amzDate, scope, chunk1Sig, emptyHeaderHash, emptyHeaderHash)
	chunk2Sig := hex.EncodeToString(hmacSHA256(signingKey, []byte(chunk2StringToSign)))

	authHeaderValid := fmt.Sprintf("AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s", accessKey, scope, signedHeaders, seedSig)

	// Test A: Valid signatures -> 200 OK
	var bodyValid bytes.Buffer
	bodyValid.WriteString(fmt.Sprintf("%x;chunk-signature=%s\r\n", len(chunk1Data), chunk1Sig))
	bodyValid.Write(chunk1Data)
	bodyValid.WriteString("\r\n")
	bodyValid.WriteString(fmt.Sprintf("0;chunk-signature=%s\r\n\r\n", chunk2Sig))

	req := httptest.NewRequest(http.MethodPut, "/sig-bucket/valid.txt", &bodyValid)
	req.Host = "localhost:8080"
	req.Header.Set("Authorization", authHeaderValid)
	req.Header.Set("X-Amz-Date", amzDate)
	req.Header.Set("X-Amz-Content-Sha256", "STREAMING-AWS4-HMAC-SHA256-PAYLOAD")
	req.Header.Set("Content-Encoding", "aws-chunked")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on valid rolling chunk signatures, got %d: %s", w.Code, w.Body.String())
	}

	// Test B: Deliberately invalid chunk signature on payload chunk -> 403 SignatureDoesNotMatch
	var bodyInvalidChunk bytes.Buffer
	badSig := "0000000000000000000000000000000000000000000000000000000000000000"
	bodyInvalidChunk.WriteString(fmt.Sprintf("%x;chunk-signature=%s\r\n", len(chunk1Data), badSig))
	bodyInvalidChunk.Write(chunk1Data)
	bodyInvalidChunk.WriteString("\r\n")
	bodyInvalidChunk.WriteString(fmt.Sprintf("0;chunk-signature=%s\r\n\r\n", chunk2Sig))

	reqBad := httptest.NewRequest(http.MethodPut, "/sig-bucket/invalid.txt", &bodyInvalidChunk)
	reqBad.Host = "localhost:8080"
	reqBad.Header.Set("Authorization", authHeaderValid)
	reqBad.Header.Set("X-Amz-Date", amzDate)
	reqBad.Header.Set("X-Amz-Content-Sha256", "STREAMING-AWS4-HMAC-SHA256-PAYLOAD")
	reqBad.Header.Set("Content-Encoding", "aws-chunked")

	wBad := httptest.NewRecorder()
	handler.ServeHTTP(wBad, reqBad)
	if wBad.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden on invalid chunk signature, got %d: %s", wBad.Code, wBad.Body.String())
	}

	// Test C: Deliberately invalid zero-byte trailer signature -> 403 SignatureDoesNotMatch
	var bodyInvalidTrailer bytes.Buffer
	bodyInvalidTrailer.WriteString(fmt.Sprintf("%x;chunk-signature=%s\r\n", len(chunk1Data), chunk1Sig))
	bodyInvalidTrailer.Write(chunk1Data)
	bodyInvalidTrailer.WriteString("\r\n")
	bodyInvalidTrailer.WriteString(fmt.Sprintf("0;chunk-signature=%s\r\n\r\n", badSig))

	reqBadTrailer := httptest.NewRequest(http.MethodPut, "/sig-bucket/invalid_trailer.txt", &bodyInvalidTrailer)
	reqBadTrailer.Host = "localhost:8080"
	reqBadTrailer.Header.Set("Authorization", authHeaderValid)
	reqBadTrailer.Header.Set("X-Amz-Date", amzDate)
	reqBadTrailer.Header.Set("X-Amz-Content-Sha256", "STREAMING-AWS4-HMAC-SHA256-PAYLOAD")
	reqBadTrailer.Header.Set("Content-Encoding", "aws-chunked")

	wBadTrailer := httptest.NewRecorder()
	handler.ServeHTTP(wBadTrailer, reqBadTrailer)
	if wBadTrailer.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden on invalid trailer signature, got %d: %s", wBadTrailer.Code, wBadTrailer.Body.String())
	}
}

type discardMockEngine struct{}

func (d *discardMockEngine) PutChunk(chunkID string, data []byte) ([]string, error) {
	return []string{"node1"}, nil
}

func (d *discardMockEngine) GetChunk(chunkID string, locations []string) ([]byte, error) {
	return nil, nil
}

func TestS3ConcurrentStreamingMemoryBounded(t *testing.T) {
	metaStore := metadata.NewStore()
	engine := &discardMockEngine{}
	authenticator := auth.NewDefaultAuthenticator()
	// Use production DefaultChunkSize (1MB) instead of 1KB test override
	handler := NewS3Handler(metaStore, engine, DefaultChunkSize, authenticator)
	rawKey := "test-admin-key-123"
	authenticator.AddRawKey(rawKey, []string{"*"}, true)


	t.Logf("[Memory Accounting - Production Config] Handler Chunk Size: %d bytes (%.2f KB / %.2f MB)",
		handler.chunkSize, float64(handler.chunkSize)/1024, float64(handler.chunkSize)/(1024*1024))

	concurrencyLevels := []int{5, 20, 50}
	const uploadSizePerStream = int64(10 * 1024 * 1024) // 10 MB per stream

	for _, numConcurrent := range concurrencyLevels {
		var wg sync.WaitGroup
		errs := make(chan error, numConcurrent)

		peakAllocCh := make(chan uint64, 1)
		doneSampling := make(chan struct{})

		go func() {
			var peak uint64
			ticker := time.NewTicker(2 * time.Millisecond)
			defer ticker.Stop()
			for {
				select {
				case <-doneSampling:
					peakAllocCh <- peak
					return
				case <-ticker.C:
					var m runtime.MemStats
					runtime.ReadMemStats(&m)
					if m.Alloc > peak {
						peak = m.Alloc
					}
				}
			}
		}()

		runtime.GC()

		for i := 0; i < numConcurrent; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				stream := &zeroStream{remaining: uploadSizePerStream}
				key := fmt.Sprintf("conc_%d_%d.bin", numConcurrent, idx)
				req := httptest.NewRequest(http.MethodPut, "/concurrent-bucket/"+key, stream)
				req.Header.Set("Authorization", rawKey)
				req.Header.Set("X-Amz-Content-Sha256", "UNSIGNED-PAYLOAD")
				w := httptest.NewRecorder()
				handler.ServeHTTP(w, req)
				if w.Code != http.StatusOK {
					errs <- fmt.Errorf("stream %d failed with code %d: %s", idx, w.Code, w.Body.String())
				}
			}(i)
		}

		wg.Wait()
		close(doneSampling)
		peakAllocBytes := <-peakAllocCh
		close(errs)

		for err := range errs {
			t.Fatalf("concurrent upload error (%d streams): %v", numConcurrent, err)
		}

		peakAllocMB := float64(peakAllocBytes) / (1024 * 1024)
		totalTransferMB := float64(int64(numConcurrent)*uploadSizePerStream) / (1024 * 1024)
		perWorkerHeapMB := peakAllocMB / float64(numConcurrent)

		t.Logf("[%d Concurrent Uploads] Total Transfer: %.0f MB | Peak Heap Alloc: %.2f MB | Per-Worker Heap: %.2f MB",
			numConcurrent, totalTransferMB, peakAllocMB, perWorkerHeapMB)

		if peakAllocMB > 256.0 {
			t.Fatalf("peak heap alloc %.2f MB for %d streams exceeds container 256 MB limit!", peakAllocMB, numConcurrent)
		}
	}
}

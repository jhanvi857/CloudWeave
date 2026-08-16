package s3

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"cloudWeave/internal/auth"
	"cloudWeave/internal/metadata"
)

type mockEngine struct {
	chunks map[string][]byte
}

func newMockEngine() *mockEngine {
	return &mockEngine{chunks: make(map[string][]byte)}
}

func (m *mockEngine) PutChunk(chunkID string, data []byte) ([]string, error) {
	cp := make([]byte, len(data))
	copy(cp, data)
	m.chunks[chunkID] = cp
	return []string{"node1"}, nil
}

func (m *mockEngine) GetChunk(chunkID string, locations []string) ([]byte, error) {
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

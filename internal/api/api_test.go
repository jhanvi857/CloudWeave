package api

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"cloudWeave/internal/metadata"
	"cloudWeave/internal/storage"
	"cloudWeave/internal/transport"
)

func TestAPIHandler_PutAndGetSingleNode(t *testing.T) {
	tempDir := t.TempDir()
	store, err := storage.NewDiskStore(tempDir)
	if err != nil {
		t.Fatalf("failed to create disk store: %v", err)
	}

	adapter := &LocalStorageAdapter{
		PutFunc: store.Put,
		GetFunc: store.Get,
		NodeID:  "localhost:8080",
	}

	metaStore := metadata.NewStore()
	apiHandler := NewAPIHandler(metaStore, adapter, 10) // 10 bytes per chunk to force multi-chunk split

	transportServer := transport.NewServer(store)
	router := NewRouter(apiHandler, transportServer.Handler(), nil)

	ts := httptest.NewServer(router)
	defer ts.Close()

	payload := []byte("The quick brown fox jumps over the lazy dog. 1234567890!")
	fileID := "test-file-1"

	// 1. PUT /files/test-file-1
	req, err := http.NewRequest(http.MethodPut, ts.URL+"/files/"+fileID, bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("failed to create PUT request: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("PUT request failed: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 201 Created, got %d: %s", resp.StatusCode, string(body))
	}
	resp.Body.Close()

	// Verify metadata recorded
	m, found := metaStore.Lookup(fileID)
	if !found {
		t.Fatalf("metadata manifest not recorded")
	}
	if m.Size != int64(len(payload)) {
		t.Errorf("manifest size mismatch: got %d, want %d", m.Size, len(payload))
	}
	if len(m.ChunkIDs) <= 1 {
		t.Errorf("expected payload to be split into multiple chunks, got %d chunk(s)", len(m.ChunkIDs))
	}

	// 2. GET /files/test-file-1
	getReq, err := http.NewRequest(http.MethodGet, ts.URL+"/files/"+fileID, nil)
	if err != nil {
		t.Fatalf("failed to create GET request: %v", err)
	}
	getResp, err := http.DefaultClient.Do(getReq)
	if err != nil {
		t.Fatalf("GET request failed: %v", err)
	}
	defer getResp.Body.Close()

	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", getResp.StatusCode)
	}

	downloaded, err := io.ReadAll(getResp.Body)
	if err != nil {
		t.Fatalf("failed to read GET response body: %v", err)
	}

	if !bytes.Equal(downloaded, payload) {
		t.Fatalf("byte mismatch!\nGot:  %s\nWant: %s", string(downloaded), string(payload))
	}
}

func TestAPIHandler_RangeRequest(t *testing.T) {
	tempDir := t.TempDir()
	store, _ := storage.NewDiskStore(tempDir)
	adapter := &LocalStorageAdapter{
		PutFunc: store.Put,
		GetFunc: store.Get,
		NodeID:  "localhost:8080",
	}

	metaStore := metadata.NewStore()
	apiHandler := NewAPIHandler(metaStore, adapter, 10) // 10 bytes per chunk
	router := NewRouter(apiHandler, nil, nil)
	ts := httptest.NewServer(router)
	defer ts.Close()

	payload := []byte("0123456789abcdefghijklmnopqrstuvwxyz") // 36 bytes
	fileID := "range-file"

	req, _ := http.NewRequest(http.MethodPut, ts.URL+"/files/"+fileID, bytes.NewReader(payload))
	resp, err := http.DefaultClient.Do(req)
	if err != nil || resp.StatusCode != http.StatusCreated {
		t.Fatalf("PUT failed: %v, status %d", err, resp.StatusCode)
	}
	resp.Body.Close()

	// Range request for bytes 5 to 15
	getReq, _ := http.NewRequest(http.MethodGet, ts.URL+"/files/"+fileID, nil)
	getReq.Header.Set("Range", "bytes=5-15")

	getResp, err := http.DefaultClient.Do(getReq)
	if err != nil {
		t.Fatalf("Range GET failed: %v", err)
	}
	defer getResp.Body.Close()

	if getResp.StatusCode != http.StatusPartialContent {
		t.Fatalf("expected 206 Partial Content, got %d", getResp.StatusCode)
	}

	if contentRange := getResp.Header.Get("Content-Range"); contentRange != "bytes 5-15/36" {
		t.Fatalf("expected Content-Range 'bytes 5-15/36', got '%s'", contentRange)
	}

	rangeBody, _ := io.ReadAll(getResp.Body)
	expectedRange := payload[5:16] // bytes 5 through 15 inclusive (11 bytes)
	if !bytes.Equal(rangeBody, expectedRange) {
		t.Fatalf("range body mismatch! Got '%s', want '%s'", string(rangeBody), string(expectedRange))
	}
}

func TestAPIHandler_DeleteFile(t *testing.T) {
	tempDir := t.TempDir()
	store, _ := storage.NewDiskStore(tempDir)
	adapter := &LocalStorageAdapter{
		PutFunc: store.Put,
		GetFunc: store.Get,
		NodeID:  "localhost:8080",
	}

	metaStore := metadata.NewStore()
	apiHandler := NewAPIHandler(metaStore, adapter, 100)
	router := NewRouter(apiHandler, nil, nil)
	ts := httptest.NewServer(router)
	defer ts.Close()

	fileID := "delete-me"
	req, _ := http.NewRequest(http.MethodPut, ts.URL+"/files/"+fileID, bytes.NewReader([]byte("temp content")))
	resp, _ := http.DefaultClient.Do(req)
	resp.Body.Close()

	// DELETE /files/delete-me
	delReq, _ := http.NewRequest(http.MethodDelete, ts.URL+"/files/"+fileID, nil)
	delResp, err := http.DefaultClient.Do(delReq)
	if err != nil {
		t.Fatalf("DELETE failed: %v", err)
	}
	delResp.Body.Close()

	if delResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK on DELETE, got %d", delResp.StatusCode)
	}

	_, exists := metaStore.Lookup(fileID)
	if exists {
		t.Fatalf("expected file manifest to be deleted from store")
	}
}


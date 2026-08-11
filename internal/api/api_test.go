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
	router := NewRouter(apiHandler, transportServer.Handler())

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

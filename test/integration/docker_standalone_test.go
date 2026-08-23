package integration

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"cloudWeave/internal/api"
	"cloudWeave/internal/auth"
	"cloudWeave/internal/coordinator"
	"cloudWeave/internal/metadata"
	"cloudWeave/internal/ring"
	"cloudWeave/internal/storage"
	"cloudWeave/internal/transport"
)

func TestDockerStandalone_HealthAndSingleNodeFlow(t *testing.T) {
	tempDir := t.TempDir()
	diskStore, err := storage.NewDiskStore(tempDir)
	if err != nil {
		t.Fatalf("failed to init storage: %v", err)
	}

	hashRing := ring.New()
	metaStore := metadata.NewStore()

	// In standalone mode, N=1, W=1, R=1
	coord := coordinator.NewCoordinator(hashRing, metaStore, "http://localhost:9000", diskStore, 1, 1, 1)
	hashRing.AddNode("http://localhost:9000")

	adminKey := "test-standalone-secret"
	authenticator := auth.NewAuthenticator([]auth.Credential{
		{
			KeyHash:    auth.HashKey(adminKey),
			Namespaces: []string{"*"},
			IsAdmin:    true,
		},
	})

	apiHandler := api.NewAPIHandler(metaStore, coord, api.DefaultChunkSize)
	transportServer := transport.NewServer(diskStore)
	router := api.NewRouterWithClusterSecret(apiHandler, transportServer.Handler(), nil, "", authenticator)

	ts := httptest.NewServer(router)
	defer ts.Close()

	// 1. Test /health and /healthz endpoints (Docker HEALTHCHECK compatibility)
	for _, healthPath := range []string{"/health", "/healthz"} {
		resp, err := http.Get(ts.URL + healthPath)
		if err != nil {
			t.Fatalf("GET %s failed: %v", healthPath, err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 OK from %s, got %d", healthPath, resp.StatusCode)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if string(body) != "OK" {
			t.Fatalf("expected OK from %s, got %s", healthPath, string(body))
		}
	}

	// 2. Test object upload in standalone mode (single-node N=1, W=1, R=1)
	filePayload := []byte("Docker standalone single container test payload for CloudWeave")
	putReq, err := http.NewRequest(http.MethodPut, ts.URL+"/files/standalone-obj.txt", bytes.NewReader(filePayload))
	if err != nil {
		t.Fatalf("failed to create PUT request: %v", err)
	}
	putReq.Header.Set("Authorization", "Bearer "+adminKey)

	putResp, err := http.DefaultClient.Do(putReq)
	if err != nil {
		t.Fatalf("PUT request failed: %v", err)
	}
	if putResp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(putResp.Body)
		t.Fatalf("expected 201 Created, got %d: %s", putResp.StatusCode, string(body))
	}
	putResp.Body.Close()

	// 3. Test object retrieval in standalone mode
	getReq, err := http.NewRequest(http.MethodGet, ts.URL+"/files/standalone-obj.txt", nil)
	if err != nil {
		t.Fatalf("failed to create GET request: %v", err)
	}
	getReq.Header.Set("Authorization", "Bearer "+adminKey)

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
		t.Fatalf("failed to read downloaded bytes: %v", err)
	}
	if !bytes.Equal(downloaded, filePayload) {
		t.Fatalf("downloaded content mismatch: got %s, want %s", string(downloaded), string(filePayload))
	}
}

package client

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"cloudWeave/internal/api"
	"cloudWeave/internal/auth"
	"cloudWeave/internal/cluster"
	"cloudWeave/internal/coordinator"
	"cloudWeave/internal/metadata"
	"cloudWeave/internal/ring"
	"cloudWeave/internal/storage"
	"cloudWeave/internal/transport"
)

func TestClient_EndToEndOperations(t *testing.T) {
	// Setup 2-node cluster
	hashRing := ring.New()
	metaStore := metadata.NewStore()

	tempDir1 := t.TempDir()
	diskStore1, _ := storage.NewDiskStore(tempDir1)
	coord1 := coordinator.NewCoordinator(hashRing, metaStore, "", diskStore1, 2, 1, 1)
	apiH1 := api.NewAPIHandler(metaStore, coord1, 16)
	router1 := api.NewRouter(apiH1, transport.NewServer(diskStore1).Handler(), nil)
	ts1 := httptest.NewServer(router1)
	defer ts1.Close()

	tempDir2 := t.TempDir()
	diskStore2, _ := storage.NewDiskStore(tempDir2)
	coord2 := coordinator.NewCoordinator(hashRing, metaStore, "", diskStore2, 2, 1, 1)
	apiH2 := api.NewAPIHandler(metaStore, coord2, 16)
	router2 := api.NewRouter(apiH2, transport.NewServer(diskStore2).Handler(), nil)
	ts2 := httptest.NewServer(router2)
	defer ts2.Close()

	membership := cluster.NewMembership(hashRing, nil)
	membership.AddNode(ts1.URL)
	membership.AddNode(ts2.URL)

	apiH1.SetPeerManager(membership, ts1.URL)
	apiH2.SetPeerManager(membership, ts2.URL)

	// Create client with multiple endpoints
	cli, err := New(Config{
		Endpoints: []string{ts1.URL, ts2.URL},
		APIKey:    "default-admin-key",
		Namespace: "tenant-sdk",
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	ctx := context.Background()
	objectKey := "sdk-test-file.txt"
	payload := []byte("CloudWeave Go Client SDK Integration Test Payload")

	// 1. Put
	err = cli.Put(ctx, objectKey, payload, PutOptions{
		ContentType: "text/plain",
		Metadata: map[string]string{
			"environment": "testing",
		},
	})
	if err != nil {
		t.Fatalf("Client.Put failed: %v", err)
	}

	// 2. Get
	body, info, err := cli.Get(ctx, objectKey)
	if err != nil {
		t.Fatalf("Client.Get failed: %v", err)
	}
	downloaded, _ := io.ReadAll(body)
	body.Close()

	if string(downloaded) != string(payload) {
		t.Errorf("Get content mismatch! Got '%s', want '%s'", string(downloaded), string(payload))
	}
	if info.ContentType != "text/plain" {
		t.Errorf("ContentType mismatch! Got '%s'", info.ContentType)
	}
	if info.Metadata["environment"] != "testing" {
		t.Errorf("Metadata environment mismatch! Got '%s'", info.Metadata["environment"])
	}

	// 3. RangeGet
	rBody, _, err := cli.RangeGet(ctx, objectKey, 0, 9)
	if err != nil {
		t.Fatalf("Client.RangeGet failed: %v", err)
	}
	rData, _ := io.ReadAll(rBody)
	rBody.Close()

	if string(rData) != string(payload[0:10]) {
		t.Errorf("RangeGet mismatch! Got '%s', want '%s'", string(rData), string(payload[0:10]))
	}

	// 4. Delete
	err = cli.Delete(ctx, objectKey)
	if err != nil {
		t.Fatalf("Client.Delete failed: %v", err)
	}

	// 5. Verify Get returns error after deletion
	_, _, err = cli.Get(ctx, objectKey)
	if err == nil {
		t.Errorf("expected Get to fail after deletion")
	}
}

func TestClient_FailoverRetry(t *testing.T) {
	// Node 1 is broken/down
	badTS := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}))
	defer badTS.Close()

	// Node 2 is healthy
	metaStore := metadata.NewStore()
	tempDir := t.TempDir()
	diskStore, _ := storage.NewDiskStore(tempDir)
	hashRing := ring.New()
	coord := coordinator.NewCoordinator(hashRing, metaStore, "", diskStore, 1, 1, 1)
	apiH := api.NewAPIHandler(metaStore, coord, 16)
	router := api.NewRouter(apiH, transport.NewServer(diskStore).Handler(), nil)
	goodTS := httptest.NewServer(router)
	defer goodTS.Close()

	hashRing.AddNode(goodTS.URL)

	// Client includes both bad and good node
	cli, err := New(Config{
		Endpoints:  []string{badTS.URL, goodTS.URL},
		APIKey:     "default-admin-key",
		Namespace:  "test-ns",
		MaxRetries: 3,
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	ctx := context.Background()
	err = cli.Put(ctx, "failover-doc", []byte("failover content"))
	if err != nil {
		t.Fatalf("Client.Put should succeed via failover retry to healthy node: %v", err)
	}
}

func TestClient_AuthRejection(t *testing.T) {
	authenticator := auth.NewAuthenticator([]auth.Credential{
		{KeyHash: auth.HashKey("valid-key"), Namespaces: []string{"tenant-x"}},
	})
	metaStore := metadata.NewStore()
	tempDir := t.TempDir()
	diskStore, _ := storage.NewDiskStore(tempDir)
	coord := coordinator.NewCoordinator(ring.New(), metaStore, "", diskStore, 1, 1, 1)
	apiH := api.NewAPIHandler(metaStore, coord, 16)
	router := api.NewRouter(apiH, nil, nil, authenticator)
	ts := httptest.NewServer(router)
	defer ts.Close()

	// Client with wrong key
	cli, _ := New(Config{
		Endpoints: []string{ts.URL},
		APIKey:    "bad-key",
		Namespace: "tenant-x",
	})

	err := cli.Put(context.Background(), "doc", []byte("data"))
	if err == nil {
		t.Errorf("expected Client.Put to fail with invalid APIKey")
	}
}

func TestClient_DynamicNodeDiscovery(t *testing.T) {
	hashRing := ring.New()
	metaStore := metadata.NewStore()
	tempDir := t.TempDir()
	diskStore, _ := storage.NewDiskStore(tempDir)
	coord := coordinator.NewCoordinator(hashRing, metaStore, "", diskStore, 1, 1, 1)
	apiH := api.NewAPIHandler(metaStore, coord, 16)
	router := api.NewRouter(apiH, nil, nil)
	ts := httptest.NewServer(router)
	defer ts.Close()

	membership := cluster.NewMembership(hashRing, nil)
	membership.AddNode(ts.URL)
	apiH.SetPeerManager(membership, ts.URL)

	// Create client with 1 seed node
	cli, err := New(Config{
		Endpoints: []string{ts.URL},
		APIKey:    "default-admin-key",
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	// Add second node to cluster topology dynamically
	newNodeURL := "http://127.0.0.1:9999"
	membership.AddNode(newNodeURL)

	// Run DiscoverNodes
	err = cli.DiscoverNodes(context.Background())
	if err != nil {
		t.Fatalf("DiscoverNodes failed: %v", err)
	}

	eps := cli.GetEndpoints()
	if len(eps) != 2 {
		t.Fatalf("expected 2 endpoints after discovery, got %d (%v)", len(eps), eps)
	}
}

func TestClient_Versioning(t *testing.T) {
	hashRing := ring.New()
	metaStore := metadata.NewStore()
	tempDir := t.TempDir()
	diskStore, _ := storage.NewDiskStore(tempDir)
	coord := coordinator.NewCoordinator(hashRing, metaStore, "", diskStore, 1, 1, 1)
	apiH := api.NewAPIHandler(metaStore, coord, 16)
	router := api.NewRouter(apiH, transport.NewServer(diskStore).Handler(), nil)
	ts := httptest.NewServer(router)
	defer ts.Close()

	hashRing.AddNode(ts.URL)

	cli, err := New(Config{
		Endpoints: []string{ts.URL},
		APIKey:    "default-admin-key",
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	ctx := context.Background()
	key := "versioned-doc.txt"

	v1Data := []byte("Version 1 content payload")
	if err := cli.Put(ctx, key, v1Data); err != nil {
		t.Fatalf("Put v1 failed: %v", err)
	}

	v2Data := []byte("Version 2 updated content payload")
	if err := cli.Put(ctx, key, v2Data); err != nil {
		t.Fatalf("Put v2 failed: %v", err)
	}

	versions, err := cli.ListVersions(ctx, key)
	if err != nil {
		t.Fatalf("ListVersions failed: %v", err)
	}
	if len(versions) != 2 {
		t.Fatalf("expected 2 versions, got %d", len(versions))
	}

	v1VersionID := versions[0].VersionID
	r1, _, err := cli.GetVersion(ctx, key, v1VersionID)
	if err != nil {
		t.Fatalf("GetVersion v1 failed: %v", err)
	}
	gotV1, _ := io.ReadAll(r1)
	r1.Close()

	if string(gotV1) != string(v1Data) {
		t.Errorf("v1 content mismatch: got '%s', want '%s'", string(gotV1), string(v1Data))
	}

	r2, _, err := cli.Get(ctx, key)
	if err != nil {
		t.Fatalf("Get latest failed: %v", err)
	}
	gotLatest, _ := io.ReadAll(r2)
	r2.Close()

	if string(gotLatest) != string(v2Data) {
		t.Errorf("latest content mismatch: got '%s', want '%s'", string(gotLatest), string(v2Data))
	}
}

func TestClient_ClientSideEncryption(t *testing.T) {
	hashRing := ring.New()
	metaStore := metadata.NewStore()
	tempDir := t.TempDir()
	diskStore, _ := storage.NewDiskStore(tempDir)
	coord := coordinator.NewCoordinator(hashRing, metaStore, "", diskStore, 1, 1, 1)
	apiH := api.NewAPIHandler(metaStore, coord, 16)
	router := api.NewRouter(apiH, transport.NewServer(diskStore).Handler(), nil)
	ts := httptest.NewServer(router)
	defer ts.Close()

	hashRing.AddNode(ts.URL)

	passphrase := "super-secret-master-key"
	cli, err := New(Config{
		Endpoints:            []string{ts.URL},
		APIKey:               "default-admin-key",
		EncryptionPassphrase: passphrase,
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	ctx := context.Background()
	key := "encrypted-document.pdf"
	rawContent := []byte("Top Secret Confidential Payload Data 2026")

	// 1. Put encrypted payload
	if err := cli.Put(ctx, key, rawContent); err != nil {
		t.Fatalf("Put encrypted file failed: %v", err)
	}

	// 2. Verify raw disk content is ciphertext (not raw text)
	chunks, _ := diskStore.ListChunks()
	if len(chunks) == 0 {
		t.Fatalf("expected chunks stored on disk")
	}
	diskBytes, _ := diskStore.Get(chunks[0])
	if string(diskBytes) == string(rawContent) {
		t.Fatalf("DISK LEAK: Raw unencrypted plaintext found on storage node disk!")
	}

	// 3. Get with authorized client (correct passphrase) -> successfully decrypts
	reader, info, err := cli.Get(ctx, key)
	if err != nil {
		t.Fatalf("Get encrypted file failed: %v", err)
	}
	decrypted, _ := io.ReadAll(reader)
	reader.Close()

	if string(decrypted) != string(rawContent) {
		t.Errorf("Decrypted content mismatch! Got '%s', want '%s'", string(decrypted), string(rawContent))
	}
	if info.Metadata["encrypted"] != "true" {
		t.Errorf("Expected encrypted metadata header")
	}

	// 4. Get with unauthorized client (wrong passphrase) -> fails decryption
	badCli, _ := New(Config{
		Endpoints:            []string{ts.URL},
		APIKey:               "default-admin-key",
		EncryptionPassphrase: "wrong-password",
	})
	_, _, err = badCli.Get(ctx, key)
	if err == nil {
		t.Errorf("Expected Get with wrong passphrase to fail decryption")
	}
}

func TestClient_ConvergentEncryptionDeduplication(t *testing.T) {
	hashRing := ring.New()
	metaStore := metadata.NewStore()
	tempDir := t.TempDir()
	diskStore, _ := storage.NewDiskStore(tempDir)
	coord := coordinator.NewCoordinator(hashRing, metaStore, "", diskStore, 1, 1, 1)
	apiH := api.NewAPIHandler(metaStore, coord, 16)
	router := api.NewRouter(apiH, transport.NewServer(diskStore).Handler(), nil)
	ts := httptest.NewServer(router)
	defer ts.Close()

	hashRing.AddNode(ts.URL)

	passphrase := "shared-secret-passphrase"
	cli, _ := New(Config{
		Endpoints:            []string{ts.URL},
		APIKey:               "default-admin-key",
		EncryptionPassphrase: passphrase,
	})

	ctx := context.Background()
	payload := []byte("Identical Secret Payload Content for Deduplication Test")

	// Upload key 1
	if err := cli.Put(ctx, "encrypted-doc-1.txt", payload); err != nil {
		t.Fatalf("Put doc 1 failed: %v", err)
	}
	chunksAfterDoc1, _ := diskStore.ListChunks()

	// Upload key 2 with identical payload and passphrase
	if err := cli.Put(ctx, "encrypted-doc-2.txt", payload); err != nil {
		t.Fatalf("Put doc 2 failed: %v", err)
	}
	chunksAfterDoc2, _ := diskStore.ListChunks()

	// Convergent Encryption guarantees that identical encrypted payloads produce identical SHA-256 chunk IDs
	if len(chunksAfterDoc1) != len(chunksAfterDoc2) {
		t.Errorf("Expected deduplication to prevent duplicate chunk storage! Got %d chunks for doc1, %d chunks for doc2", len(chunksAfterDoc1), len(chunksAfterDoc2))
	}
}

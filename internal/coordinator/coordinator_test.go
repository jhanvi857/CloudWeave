package coordinator

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"cloudWeave/internal/api"
	"cloudWeave/internal/metadata"
	"cloudWeave/internal/ring"
	"cloudWeave/internal/storage"
	"cloudWeave/internal/transport"
)

func TestCoordinator_QuorumReadWrite(t *testing.T) {
	// 1. Spin up 3 storage node servers
	const numNodes = 3
	var nodeServers []*httptest.Server
	var nodeAddrs []string
	var nodeStores []*storage.DiskStore

	r := ring.New()

	for i := 0; i < numNodes; i++ {
		tempDir := t.TempDir()
		diskStore, err := storage.NewDiskStore(tempDir)
		if err != nil {
			t.Fatalf("failed to init storage: %v", err)
		}
		nodeStores = append(nodeStores, diskStore)

		transportSvr := transport.NewServer(diskStore)
		ts := httptest.NewServer(transportSvr.Handler())
		nodeServers = append(nodeServers, ts)
		nodeAddrs = append(nodeAddrs, ts.URL)

		r.AddNode(ts.URL)
	}

	defer func() {
		for _, ts := range nodeServers {
			ts.Close()
		}
	}()

	metaStore := metadata.NewStore()
	coord := NewCoordinator(r, metaStore, "", nil, 3, 2, 2)

	apiHandler := api.NewAPIHandler(metaStore, coord, 16)
	router := api.NewRouter(apiHandler, nil, nil)

	coordinatorServer := httptest.NewServer(router)
	defer coordinatorServer.Close()

	payload := []byte("Multi-node quorum write and read test with CloudWeave distributed storage!")
	fileID := "quorum-test-doc"

	// 2. PUT file through coordinator API
	putReq, err := http.NewRequest(http.MethodPut, coordinatorServer.URL+"/files/"+fileID, bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("failed to build PUT request: %v", err)
	}
	putReq.Header.Set("Authorization", "Bearer default-admin-key")

	putResp, err := http.DefaultClient.Do(putReq)
	if err != nil {
		t.Fatalf("PUT failed: %v", err)
	}
	if putResp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(putResp.Body)
		t.Fatalf("expected 201 Created, got %d: %s", putResp.StatusCode, string(body))
	}
	putResp.Body.Close()

	// 3. Verify chunk replication in metadata and physical nodes
	manifest, found := metaStore.Lookup(fileID)
	if !found {
		t.Fatalf("manifest not found for %s", fileID)
	}

	for chunkID, locs := range manifest.ChunkLocations {
		if len(locs) < 2 {
			t.Errorf("chunk %s under-replicated: got %d ACKs, expected >= 2", chunkID, len(locs))
		}

		// Verify chunk actually exists on disk of the reported nodes
		for _, loc := range locs {
			foundLocally := false
			for idx, addr := range nodeAddrs {
				if addr == loc {
					if nodeStores[idx].Exists(chunkID) {
						foundLocally = true
						break
					}
				}
			}
			if !foundLocally {
				t.Errorf("chunk %s reported on %s but not present on disk", chunkID, loc)
			}
		}
	}

	// 4. GET file through coordinator API
	getReq, err := http.NewRequest(http.MethodGet, coordinatorServer.URL+"/files/"+fileID, nil)
	if err != nil {
		t.Fatalf("failed to build GET request: %v", err)
	}
	getReq.Header.Set("Authorization", "Bearer default-admin-key")

	getResp, err := http.DefaultClient.Do(getReq)
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	defer getResp.Body.Close()

	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", getResp.StatusCode)
	}

	downloaded, err := io.ReadAll(getResp.Body)
	if err != nil {
		t.Fatalf("failed to read response: %v", err)
	}

	if !bytes.Equal(downloaded, payload) {
		t.Fatalf("byte mismatch!\nGot:  %s\nWant: %s", string(downloaded), string(payload))
	}
}

func TestCoordinator_ReadIntegrityVerification(t *testing.T) {
	tempDir := t.TempDir()
	store, _ := storage.NewDiskStore(tempDir)
	r := ring.New()
	metaStore := metadata.NewStore()
	coord := NewCoordinator(r, metaStore, "http://localhost:9000", store, 1, 1, 1)

	validPayload := []byte("Original authentic chunk data")
	sum := sha256.Sum256(validPayload)
	chunkID := hex.EncodeToString(sum[:])

	// Store corrupted payload under valid chunk ID
	corruptedPayload := []byte("Corrupted tampered data!")
	_ = store.Put(chunkID, corruptedPayload)

	// Attempt to read via coordinator - must fail integrity check
	_, err := coord.ReadChunk(context.Background(), chunkID, []string{"http://localhost:9000"})
	if err == nil {
		t.Fatalf("expected ReadChunk to fail due to corrupted payload, but it succeeded")
	}
}

func TestCoordinator_ErasureCodingReadWrite(t *testing.T) {
	// Spin up 5 storage nodes for K=3, M=2 erasure coding
	const numNodes = 5
	var nodeServers []*httptest.Server
	var nodeStores []*storage.DiskStore

	r := ring.New()
	for i := 0; i < numNodes; i++ {
		tempDir := t.TempDir()
		diskStore, _ := storage.NewDiskStore(tempDir)
		nodeStores = append(nodeStores, diskStore)

		transportSvr := transport.NewServer(diskStore)
		ts := httptest.NewServer(transportSvr.Handler())
		nodeServers = append(nodeServers, ts)
		r.AddNode(ts.URL)
	}
	defer func() {
		for _, ts := range nodeServers {
			ts.Close()
		}
	}()

	metaStore := metadata.NewStore()
	coord := NewCoordinator(r, metaStore, "", nil, 3, 2, 2)
	coord.SetStorageMode("erasure", 3, 2)

	payload := []byte("Erasure Coding distributed storage payload test across 5 nodes!")
	sum := sha256.Sum256(payload)
	chunkID := hex.EncodeToString(sum[:])

	// 1. PutChunk in erasure mode
	locs, err := coord.PutChunk(chunkID, payload)
	if err != nil {
		t.Fatalf("PutChunk in erasure mode failed: %v", err)
	}
	if len(locs) < 3 {
		t.Errorf("expected at least 3 successful shard writes, got %d", len(locs))
	}

	// 2. ReadChunk in erasure mode
	recovered, err := coord.GetChunk(chunkID, locs)
	if err != nil {
		t.Fatalf("GetChunk in erasure mode failed: %v", err)
	}
	if !bytes.Equal(recovered, payload) {
		t.Fatalf("data mismatch!\nGot:  %s\nWant: %s", string(recovered), string(payload))
	}
}


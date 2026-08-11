package coordinator

import (
	"bytes"
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
	router := api.NewRouter(apiHandler, nil)

	coordinatorServer := httptest.NewServer(router)
	defer coordinatorServer.Close()

	payload := []byte("Multi-node quorum write and read test with CloudWeave distributed storage!")
	fileID := "quorum-test-doc"

	// 2. PUT file through coordinator API
	putReq, err := http.NewRequest(http.MethodPut, coordinatorServer.URL+"/files/"+fileID, bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("failed to build PUT request: %v", err)
	}

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

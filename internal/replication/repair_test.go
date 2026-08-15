package replication

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"cloudWeave/internal/api"
	"cloudWeave/internal/cluster"
	"cloudWeave/internal/coordinator"
	"cloudWeave/internal/metadata"
	"cloudWeave/internal/ring"
	"cloudWeave/internal/storage"
	"cloudWeave/internal/transport"
)

func TestReplication_SelfHealingOnNodeFailure(t *testing.T) {
	// 1. Spin up 4 storage node servers
	const numNodes = 4
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
	}

	metaStore := metadata.NewStore()
	repairMgr := NewRepairManager(metaStore, r, 3, "", nil)
	workerPool := NewRepairWorkerPool(repairMgr, 2)
	defer workerPool.Stop()

	membership := cluster.NewMembership(r, func(deadAddr string) {
		workerPool.SubmitDeadNodeJob(deadAddr)
	})

	for _, addr := range nodeAddrs {
		membership.AddNode(addr)
	}

	coord := coordinator.NewCoordinator(r, metaStore, "", nil, 3, 2, 2)
	apiHandler := api.NewAPIHandler(metaStore, coord, 16)
	router := api.NewRouter(apiHandler, nil, nil)

	coordinatorServer := httptest.NewServer(router)
	defer coordinatorServer.Close()

	payload := []byte("Self-healing distributed storage repair demo payload!")
	fileID := "self-healing-doc"

	// 2. PUT file through coordinator API
	putReq, err := http.NewRequest(http.MethodPut, coordinatorServer.URL+"/files/"+fileID, bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("failed to create PUT request: %v", err)
	}
	putReq.Header.Set("Authorization", "Bearer default-admin-key")

	putResp, err := http.DefaultClient.Do(putReq)
	if err != nil {
		t.Fatalf("PUT failed: %v", err)
	}
	if putResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d", putResp.StatusCode)
	}
	putResp.Body.Close()

	// 3. Pick one node that holds chunks and kill it
	manifest, _ := metaStore.Lookup(fileID)
	chunkID := manifest.ChunkIDs[0]
	initialLocs := manifest.ChunkLocations[chunkID]

	deadNodeAddr := initialLocs[0]
	var deadServerIdx int
	for idx, addr := range nodeAddrs {
		if addr == deadNodeAddr {
			deadServerIdx = idx
			break
		}
	}

	// Kill dead node
	nodeServers[deadServerIdx].Close()
	membership.MarkDead(deadNodeAddr)

	// Wait for background repair worker
	time.Sleep(500 * time.Millisecond)

	// 4. Verify repaired metadata chunk locations
	repairedManifest, _ := metaStore.Lookup(fileID)
	repairedLocs := repairedManifest.ChunkLocations[chunkID]

	activeLocations := 0
	for _, loc := range repairedLocs {
		if loc != deadNodeAddr {
			activeLocations++
		}
	}

	if activeLocations < 3 {
		t.Errorf("expected replication count to climb back to 3 active nodes, got %d (locations: %v)", activeLocations, repairedLocs)
	}

	// 5. Verify GET file still succeeds after node failure and self-healing
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
		t.Fatalf("failed to read GET body: %v", err)
	}

	if !bytes.Equal(downloaded, payload) {
		t.Fatalf("downloaded data mismatch after self-healing repair!\nGot:  %s\nWant: %s", string(downloaded), string(payload))
	}
}

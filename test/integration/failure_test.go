package integration

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
	"cloudWeave/internal/replication"
	"cloudWeave/internal/ring"
	"cloudWeave/internal/storage"
	"cloudWeave/internal/transport"
)

func TestEndToEnd_NodeFailureAndSelfHealing(t *testing.T) {
	const numNodes = 5
	var nodeServers []*httptest.Server
	var nodeAddrs []string
	var nodeStores []*storage.DiskStore

	hashRing := ring.New()
	metaStore := metadata.NewStore()

	for i := 0; i < numNodes; i++ {
		tempDir := t.TempDir()
		diskStore, err := storage.NewDiskStore(tempDir)
		if err != nil {
			t.Fatalf("failed to init diskstore: %v", err)
		}
		nodeStores = append(nodeStores, diskStore)

		transportSvr := transport.NewServer(diskStore)
		ts := httptest.NewServer(transportSvr.Handler())
		nodeServers = append(nodeServers, ts)
		nodeAddrs = append(nodeAddrs, ts.URL)
	}

	repairMgr := replication.NewRepairManager(metaStore, hashRing, 3, "", nil)
	workerPool := replication.NewRepairWorkerPool(repairMgr, 2)
	defer workerPool.Stop()

	membership := cluster.NewMembership(hashRing, func(deadAddr string) {
		t.Logf("[Test] Detected dead node %s, triggering repair worker", deadAddr)
		workerPool.SubmitDeadNodeJob(deadAddr)
	})

	for _, addr := range nodeAddrs {
		membership.AddNode(addr)
	}

	heartbeat := cluster.NewHeartbeatChecker(membership, 100*time.Millisecond, 300*time.Millisecond)
	heartbeat.Start()
	defer heartbeat.Stop()

	coord := coordinator.NewCoordinator(hashRing, metaStore, "", nil, 3, 2, 2)
	apiHandler := api.NewAPIHandler(metaStore, coord, 32)
	router := api.NewRouter(apiHandler, nil)

	coordinatorServer := httptest.NewServer(router)
	defer coordinatorServer.Close()

	// Step 1: Upload big payload
	payload := []byte("CloudWeave End-To-End Distributed Storage Integration Test. " +
		"Testing N=3 replication, W=2 write quorum, R=2 read quorum, node failure detection, " +
		"and automatic self-healing under-replication repair!")
	fileID := "e2e-failure-demo"

	putReq, err := http.NewRequest(http.MethodPut, coordinatorServer.URL+"/files/"+fileID, bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("failed to build PUT request: %v", err)
	}

	putResp, err := http.DefaultClient.Do(putReq)
	if err != nil {
		t.Fatalf("PUT request failed: %v", err)
	}
	if putResp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(putResp.Body)
		t.Fatalf("expected 201 Created, got %d: %s", putResp.StatusCode, string(body))
	}
	putResp.Body.Close()

	// Step 2: Identify node holding chunks and shut it down
	manifest, found := metaStore.Lookup(fileID)
	if !found {
		t.Fatalf("manifest for %s not recorded", fileID)
	}

	targetChunkID := manifest.ChunkIDs[0]
	chunkLocs := manifest.ChunkLocations[targetChunkID]
	deadAddr := chunkLocs[0]

	var deadNodeIdx int
	for idx, addr := range nodeAddrs {
		if addr == deadAddr {
			deadNodeIdx = idx
			break
		}
	}

	t.Logf("[Test] Simulating node failure for node %s", deadAddr)
	nodeServers[deadNodeIdx].Close()

	// Step 3: Wait for heartbeat failure detection and self-healing repair
	time.Sleep(1000 * time.Millisecond)

	// Step 4: Verify metadata locations updated to surviving active nodes
	repairedManifest, _ := metaStore.Lookup(fileID)
	repairedLocs := repairedManifest.ChunkLocations[targetChunkID]

	activeCount := 0
	for _, loc := range repairedLocs {
		if loc != deadAddr {
			activeCount++
		}
	}

	if activeCount < 3 {
		t.Errorf("expected replication factor to return to >= 3 active nodes, got %d (locations: %v)", activeCount, repairedLocs)
	}

	// Step 5: Read file back via GET /files/
	getReq, err := http.NewRequest(http.MethodGet, coordinatorServer.URL+"/files/"+fileID, nil)
	if err != nil {
		t.Fatalf("failed to build GET request: %v", err)
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
		t.Fatalf("failed to read downloaded file: %v", err)
	}

	if !bytes.Equal(downloaded, payload) {
		t.Fatalf("downloaded data mismatch after node failure recovery!\nGot:  %s\nWant: %s", string(downloaded), string(payload))
	}

	t.Logf("[Test] Integration test completed successfully! Data intact (%d bytes)", len(downloaded))
}

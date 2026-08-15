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

func TestDynamicClusterMembership_JoinAndLeave(t *testing.T) {
	hashRing := ring.New()

	// Spin up 3 initial nodes
	const numNodes = 3
	var nodeServers []*httptest.Server
	var nodeAddrs []string
	var apiHandlers []*api.APIHandler
	var memberships []*cluster.Membership

	metaStore := metadata.NewStore()
	repairMgr := replication.NewRepairManager(metaStore, hashRing, 3, "", nil)
	workerPool := replication.NewRepairWorkerPool(repairMgr, 2)
	defer workerPool.Stop()

	for i := 0; i < numNodes; i++ {
		tempDir := t.TempDir()
		diskStore, _ := storage.NewDiskStore(tempDir)

		coord := coordinator.NewCoordinator(hashRing, metaStore, "", diskStore, 3, 2, 2)
		apiH := api.NewAPIHandler(metaStore, coord, 32)
		apiHandlers = append(apiHandlers, apiH)

		transportSvr := transport.NewServer(diskStore)
		router := api.NewRouter(apiH, transportSvr.Handler(), nil)

		ts := httptest.NewServer(router)
		nodeServers = append(nodeServers, ts)
		nodeAddrs = append(nodeAddrs, ts.URL)
	}

	for i, apiH := range apiHandlers {
		localAddr := nodeAddrs[i]
		membership := cluster.NewMembership(hashRing, func(deadAddr string) {
			workerPool.SubmitDeadNodeJob(deadAddr)
		})
		memberships = append(memberships, membership)
		for _, addr := range nodeAddrs {
			membership.AddNode(addr)
		}
		apiH.SetPeerManager(membership, localAddr)
	}

	authHeader := "Bearer default-admin-key"
	payload := []byte("Testing dynamic cluster join and leave endpoints in CloudWeave Phase 2!")
	fileID := "dynamic-membership-doc"

	// 1. Upload file to 3-node cluster
	putReq, _ := http.NewRequest(http.MethodPut, nodeAddrs[0]+"/files/"+fileID, bytes.NewReader(payload))
	putReq.Header.Set("Authorization", authHeader)
	putResp, err := http.DefaultClient.Do(putReq)
	if err != nil || putResp.StatusCode != http.StatusCreated {
		t.Fatalf("Initial PUT failed: %v, status %d", err, putResp.StatusCode)
	}
	putResp.Body.Close()

	// 2. Launch Node 4 dynamically
	tempDir4 := t.TempDir()
	diskStore4, _ := storage.NewDiskStore(tempDir4)
	coord4 := coordinator.NewCoordinator(hashRing, metaStore, "", diskStore4, 3, 2, 2)
	apiH4 := api.NewAPIHandler(metaStore, coord4, 32)
	transportSvr4 := transport.NewServer(diskStore4)
	router4 := api.NewRouter(apiH4, transportSvr4.Handler(), nil)
	ts4 := httptest.NewServer(router4)
	defer ts4.Close()

	membership4 := cluster.NewMembership(hashRing, func(deadAddr string) {
		workerPool.SubmitDeadNodeJob(deadAddr)
	})
	membership4.AddNode(ts4.URL)
	apiH4.SetPeerManager(membership4, ts4.URL)

	// 3. Register Node 4 into running cluster via POST /admin/join against Node 0
	joinReq, _ := http.NewRequest(http.MethodPost, nodeAddrs[0]+"/admin/join?node_addr="+ts4.URL, nil)
	joinReq.Header.Set("Authorization", authHeader)

	joinResp, err := http.DefaultClient.Do(joinReq)
	if err != nil || joinResp.StatusCode != http.StatusOK {
		t.Fatalf("POST /admin/join failed: %v, status %d", err, joinResp.StatusCode)
	}
	joinResp.Body.Close()

	time.Sleep(100 * time.Millisecond)

	// Verify Node 4 is active in hash ring and membership
	if !membership4.IsAlive(ts4.URL) {
		t.Errorf("Node 4 should be alive in membership")
	}

	// 4. Send GET directly to newly joined Node 4
	getReq4, _ := http.NewRequest(http.MethodGet, ts4.URL+"/files/"+fileID, nil)
	getReq4.Header.Set("Authorization", authHeader)
	getResp4, err := http.DefaultClient.Do(getReq4)
	if err != nil || getResp4.StatusCode != http.StatusOK {
		t.Fatalf("GET from newly joined Node 4 failed: %v, status %d", err, getResp4.StatusCode)
	}
	downloaded4, _ := io.ReadAll(getResp4.Body)
	getResp4.Body.Close()
	if !bytes.Equal(downloaded4, payload) {
		t.Errorf("Node 4 served wrong payload! Got '%s'", string(downloaded4))
	}

	// 5. Gracefully remove Node 0 via POST /admin/leave against Node 1
	leaveReq, _ := http.NewRequest(http.MethodPost, nodeAddrs[1]+"/admin/leave?node_addr="+nodeAddrs[0], nil)
	leaveReq.Header.Set("Authorization", authHeader)

	leaveResp, err := http.DefaultClient.Do(leaveReq)
	if err != nil || leaveResp.StatusCode != http.StatusOK {
		t.Fatalf("POST /admin/leave failed: %v, status %d", err, leaveResp.StatusCode)
	}
	leaveResp.Body.Close()

	time.Sleep(200 * time.Millisecond)

	// Verify Node 0 is no longer active
	activeNodes := memberships[1].GetActiveNodes()
	for _, n := range activeNodes {
		if n == nodeAddrs[0] {
			t.Errorf("Node 0 should be removed from active nodes after leave")
		}
	}

	// Verify GET still works from remaining node
	getReq1, _ := http.NewRequest(http.MethodGet, nodeAddrs[1]+"/files/"+fileID, nil)
	getReq1.Header.Set("Authorization", authHeader)
	getResp1, err := http.DefaultClient.Do(getReq1)
	if err != nil || getResp1.StatusCode != http.StatusOK {
		t.Fatalf("GET from Node 1 after leave failed: %v, status %d", err, getResp1.StatusCode)
	}
	getResp1.Body.Close()

	for _, ts := range nodeServers {
		ts.Close()
	}
}

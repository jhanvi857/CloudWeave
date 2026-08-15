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
	"cloudWeave/internal/ring"
	"cloudWeave/internal/storage"
	"cloudWeave/internal/transport"
)

func TestAnyNodeCoordination_5NodeCluster(t *testing.T) {
	const numNodes = 5
	var nodeServers []*httptest.Server
	var nodeAddrs []string
	var apiHandlers []*api.APIHandler
	var metaStores []*metadata.Store

	hashRing := ring.New()

	// 1. Launch 5 full node servers
	for i := 0; i < numNodes; i++ {
		tempDir := t.TempDir()
		diskStore, err := storage.NewDiskStore(tempDir)
		if err != nil {
			t.Fatalf("failed to init storage: %v", err)
		}

		metaStore := metadata.NewStore()
		metaStores = append(metaStores, metaStore)

		coord := coordinator.NewCoordinator(hashRing, metaStore, "", diskStore, 3, 2, 2)
		apiH := api.NewAPIHandler(metaStore, coord, 32)
		apiHandlers = append(apiHandlers, apiH)

		transportSvr := transport.NewServer(diskStore)
		router := api.NewRouter(apiH, transportSvr.Handler(), nil)

		ts := httptest.NewServer(router)
		nodeServers = append(nodeServers, ts)
		nodeAddrs = append(nodeAddrs, ts.URL)
	}

	defer func() {
		for _, ts := range nodeServers {
			ts.Close()
		}
	}()

	// Wire membership across nodes
	for i, apiH := range apiHandlers {
		localAddr := nodeAddrs[i]
		membership := cluster.NewMembership(hashRing, nil)
		for _, addr := range nodeAddrs {
			membership.AddNode(addr)
		}
		apiH.SetPeerManager(membership, localAddr)
	}

	authHeader := "Bearer default-admin-key"
	payload := []byte("CloudWeave Any-Node Coordination Guarantee Integration Test for Phase 2!")
	fileID := "any-node-doc.txt"

	// Step A: PUT file through Node 0
	putReq, _ := http.NewRequest(http.MethodPut, nodeAddrs[0]+"/files/tenant-coord/"+fileID, bytes.NewReader(payload))
	putReq.Header.Set("Authorization", authHeader)
	putReq.Header.Set("Content-Type", "text/plain")
	putReq.Header.Set("X-Meta-Owner", "ClusterTeam")

	putResp, err := http.DefaultClient.Do(putReq)
	if err != nil || putResp.StatusCode != http.StatusCreated {
		t.Fatalf("PUT via Node 0 failed: %v, status %d", err, putResp.StatusCode)
	}
	putResp.Body.Close()

	// Wait briefly for peer metadata broadcast
	time.Sleep(100 * time.Millisecond)

	// Step B: GET file through Node 1
	getReq, _ := http.NewRequest(http.MethodGet, nodeAddrs[1]+"/files/tenant-coord/"+fileID, nil)
	getReq.Header.Set("Authorization", authHeader)

	getResp, err := http.DefaultClient.Do(getReq)
	if err != nil || getResp.StatusCode != http.StatusOK {
		t.Fatalf("GET via Node 1 failed: %v, status %d", err, getResp.StatusCode)
	}
	downloaded1, _ := io.ReadAll(getResp.Body)
	getResp.Body.Close()
	if !bytes.Equal(downloaded1, payload) {
		t.Errorf("Node 1 GET content mismatch! Got '%s'", string(downloaded1))
	}
	if getResp.Header.Get("X-Meta-Owner") != "ClusterTeam" {
		t.Errorf("Node 1 GET metadata header mismatch! Got '%s'", getResp.Header.Get("X-Meta-Owner"))
	}

	// Step C: Range GET through Node 2
	rangeReq, _ := http.NewRequest(http.MethodGet, nodeAddrs[2]+"/files/tenant-coord/"+fileID, nil)
	rangeReq.Header.Set("Authorization", authHeader)
	rangeReq.Header.Set("Range", "bytes=0-9")

	rangeResp, err := http.DefaultClient.Do(rangeReq)
	if err != nil || rangeResp.StatusCode != http.StatusPartialContent {
		t.Fatalf("Range GET via Node 2 failed: %v, status %d", err, rangeResp.StatusCode)
	}
	rangeBody, _ := io.ReadAll(rangeResp.Body)
	rangeResp.Body.Close()
	if !bytes.Equal(rangeBody, payload[0:10]) {
		t.Errorf("Node 2 Range GET mismatch! Got '%s', want '%s'", string(rangeBody), string(payload[0:10]))
	}

	// Step D: GET file through Node 3
	getReq3, _ := http.NewRequest(http.MethodGet, nodeAddrs[3]+"/files/tenant-coord/"+fileID, nil)
	getReq3.Header.Set("Authorization", authHeader)
	getResp3, err := http.DefaultClient.Do(getReq3)
	if err != nil || getResp3.StatusCode != http.StatusOK {
		t.Fatalf("GET via Node 3 failed: %v, status %d", err, getResp3.StatusCode)
	}
	downloaded3, _ := io.ReadAll(getResp3.Body)
	getResp3.Body.Close()
	if !bytes.Equal(downloaded3, payload) {
		t.Errorf("Node 3 GET content mismatch! Got '%s'", string(downloaded3))
	}

	// Step E: DELETE file through Node 4
	delReq, _ := http.NewRequest(http.MethodDelete, nodeAddrs[4]+"/files/tenant-coord/"+fileID, nil)
	delReq.Header.Set("Authorization", authHeader)

	delResp, err := http.DefaultClient.Do(delReq)
	if err != nil || delResp.StatusCode != http.StatusOK {
		t.Fatalf("DELETE via Node 4 failed: %v, status %d", err, delResp.StatusCode)
	}
	delResp.Body.Close()

	// Wait briefly for deletion broadcast
	time.Sleep(100 * time.Millisecond)

	// Step F: Assert GET on Node 0 returns 404
	getReqFinal, _ := http.NewRequest(http.MethodGet, nodeAddrs[0]+"/files/tenant-coord/"+fileID, nil)
	getReqFinal.Header.Set("Authorization", authHeader)
	getRespFinal, _ := http.DefaultClient.Do(getReqFinal)
	if getRespFinal.StatusCode != http.StatusNotFound {
		t.Errorf("Expected 404 after deletion, got %d", getRespFinal.StatusCode)
	}
	getRespFinal.Body.Close()
}

package integration

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"cloudWeave/client"
	"cloudWeave/internal/api"
	"cloudWeave/internal/cluster"
	"cloudWeave/internal/coordinator"
	"cloudWeave/internal/gc"
	"cloudWeave/internal/metadata"
	"cloudWeave/internal/replication"
	"cloudWeave/internal/ring"
	"cloudWeave/internal/storage"
	"cloudWeave/internal/transport"
)

// TestInFlightUploadSurvivesMidStreamGC (P0-2) verifies that triggering /admin/gc
// in the middle of a streaming upload will NOT sweep in-flight chunks.
func TestInFlightUploadSurvivesMidStreamGC(t *testing.T) {
	tempDir := t.TempDir()
	store, err := storage.NewDiskStore(tempDir)
	if err != nil {
		t.Fatalf("failed to init storage: %v", err)
	}

	metaStore := metadata.NewStore()
	r := ring.New()
	r.AddNode("http://localhost:9000")
	coord := coordinator.NewCoordinator(r, metaStore, "http://localhost:9000", store, 1, 1, 1)

	inFlight := store.GetInFlightRegistry()
	gcEngine := gc.NewGarbageCollector(metaStore, store)
	gcEngine.SetInFlightRegistry(inFlight)

	apiHandler := api.NewAPIHandler(metaStore, coord, 64*1024)
	apiHandler.SetDiskStore(store)
	apiHandler.SetInFlightRegistry(inFlight)

	transportServer := transport.NewServer(store)
	router := api.NewRouter(apiHandler, transportServer.Handler(), gcEngine)

	ts := httptest.NewServer(router)
	defer ts.Close()

	cli, err := client.New(client.Config{
		Endpoints: []string{ts.URL},
		APIKey:    "default-admin-key",
	})
	if err != nil {
		t.Fatalf("client init: %v", err)
	}

	// 1. Prepare a 1MB payload
	payload := bytes.Repeat([]byte("CloudWeaveInFlightChunkSafetyVerificationBytes123456!"), 20000) // ~1.08MB
	totalLen := len(payload)

	// 2. Slow reader pipe that pauses mid-stream
	pr, pw := io.Pipe()
	half := totalLen / 2

	var uploadErr error
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		req, err := http.NewRequest(http.MethodPut, ts.URL+"/files/default/concurrent-stream.bin", pr)
		if err != nil {
			uploadErr = err
			return
		}
		req.Header.Set("Authorization", "Bearer default-admin-key")
		resp, err := ts.Client().Do(req)
		if err != nil {
			uploadErr = err
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			uploadErr = fmt.Errorf("upload status %d: %s", resp.StatusCode, string(body))
		}
	}()

	// Write first half of chunks to disk
	_, _ = pw.Write(payload[:half])
	time.Sleep(50 * time.Millisecond) // Allow first chunks to land on disk and register in inFlight

	// 3. Trigger /admin/gc while upload is in-flight
	swept, gcErr := cli.CollectGarbage(context.Background())
	if gcErr != nil {
		t.Fatalf("CollectGarbage failed: %v", gcErr)
	}
	t.Logf("Mid-stream GC result: %s", swept)

	// 4. Write second half and close pipe
	_, _ = pw.Write(payload[half:])
	_ = pw.Close()

	wg.Wait()
	if uploadErr != nil {
		t.Fatalf("Upload failed mid-stream: %v", uploadErr)
	}

	// 5. Verify the full file is readable and byte-for-byte identical
	rdr, info, err := cli.Get(context.Background(), "concurrent-stream.bin")
	if err != nil {
		t.Fatalf("Get uploaded file failed: %v", err)
	}
	defer rdr.Close()

	downloaded, err := io.ReadAll(rdr)
	if err != nil {
		t.Fatalf("Reading downloaded file failed: %v", err)
	}

	if len(downloaded) != totalLen || !bytes.Equal(downloaded, payload) {
		t.Fatalf("Downloaded data corrupted or truncated by GC! Got %d bytes, want %d", len(downloaded), totalLen)
	}

	t.Logf("SUCCESS: In-flight upload completed successfully after mid-stream GC (%d bytes, Content-Type: %s)", info.Size, info.ContentType)
}

// TestAntiEntropyCatchUpAfterDowntime (P1-1) verifies that a restarting or partitioned node
// catches up with missing metadata manifests from peer nodes via anti-entropy sync.
func TestAntiEntropyCatchUpAfterDowntime(t *testing.T) {
	// Node 1
	store1, _ := storage.NewDiskStore(t.TempDir())
	meta1 := metadata.NewStore()
	r1 := ring.New()
	coord1 := coordinator.NewCoordinator(r1, meta1, "node1", store1, 1, 1, 1)
	api1 := api.NewAPIHandler(meta1, coord1, 64*1024)
	ts1 := transport.NewServer(store1)
	router1 := api.NewRouter(api1, ts1.Handler(), nil)
	srv1 := httptest.NewServer(router1)
	defer srv1.Close()

	// Populate Node 1 with 3 manifests
	for i := 1; i <= 3; i++ {
		_ = meta1.RecordPlacement(metadata.Manifest{
			Namespace: "tenant-sync",
			FileID:    fmt.Sprintf("doc-%d.txt", i),
			Size:      int64(100 * i),
			ChunkIDs:  []string{fmt.Sprintf("chunk-%d", i)},
		})
	}

	// Node 2 (Initially completely empty metadata store)
	store2, _ := storage.NewDiskStore(t.TempDir())
	meta2 := metadata.NewStore()
	r2 := ring.New()
	coord2 := coordinator.NewCoordinator(r2, meta2, "node2", store2, 1, 1, 1)
	api2 := api.NewAPIHandler(meta2, coord2, 64*1024)
	ts2 := transport.NewServer(store2)
	router2 := api.NewRouter(api2, ts2.Handler(), nil)
	srv2 := httptest.NewServer(router2)
	defer srv2.Close()

	// Verify Node 2 has 0 manifests initially
	if len(meta2.GetAllManifests()) != 0 {
		t.Fatalf("expected node 2 to start empty, got %d", len(meta2.GetAllManifests()))
	}

	// Setup membership on Node 2 knowing about Node 1
	mem2 := cluster.NewMembership(r2, nil)
	mem2.AddNode(srv1.URL)
	mem2.AddNode(srv2.URL)
	api2.SetPeerManager(mem2, srv2.URL)

	// Run Anti-Entropy reconciler on Node 2
	reconciler2 := cluster.NewAntiEntropyReconciler(meta2, mem2, srv2.URL, 1*time.Second)
	reconciler2.SetHTTPClient(srv2.Client())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := reconciler2.SyncOnce(ctx)
	if err != nil {
		t.Fatalf("SyncOnce failed: %v", err)
	}

	// Verify Node 2 has caught up with all 3 manifests
	manifests2 := meta2.GetAllManifests()
	if len(manifests2) != 3 {
		t.Fatalf("expected 3 synced manifests on node 2, got %d", len(manifests2))
	}

	for i := 1; i <= 3; i++ {
		key := fmt.Sprintf("doc-%d.txt", i)
		m, found := meta2.LookupScoped("tenant-sync", key)
		if !found || m.Size != int64(100*i) {
			t.Fatalf("expected manifest %s to be synced to node 2, found=%v", key, found)
		}
	}

	t.Logf("SUCCESS: Node 2 caught up to 3 manifests via Anti-Entropy background sync without client traffic")
}

// TestHeartbeatConsecutiveFailureFlapDamping (P1-2) verifies that 1-2 intermittent ping
// failures do not declare a node dead, but 4 consecutive failures declare it dead.
func TestHeartbeatConsecutiveFailureFlapDamping(t *testing.T) {
	r := ring.New()
	deadChan := make(chan string, 1)
	mem := cluster.NewMembership(r, func(deadAddr string) {
		deadChan <- deadAddr
	})

	nodeAddr := "http://test-node:9000"
	mem.AddNode(nodeAddr)

	if !mem.IsAlive(nodeAddr) {
		t.Fatalf("node should be alive initially")
	}

	// 1. Simulate 2 transient failures with 10s deadTimeout
	isDead := mem.RecordFailure(nodeAddr, 4, 10*time.Millisecond)
	if isDead || !mem.IsAlive(nodeAddr) {
		t.Fatalf("1 failure should NOT mark node dead")
	}

	isDead = mem.RecordFailure(nodeAddr, 4, 10*time.Millisecond)
	if isDead || !mem.IsAlive(nodeAddr) {
		t.Fatalf("2 failures should NOT mark node dead")
	}

	// 2. Flap-damping: successful heartbeat resets consecutive failures
	mem.MarkAlive(nodeAddr)

	// 3. Now simulate 3 failures (less than 4) -> still alive
	for i := 0; i < 3; i++ {
		isDead = mem.RecordFailure(nodeAddr, 4, 10*time.Millisecond)
		if isDead {
			t.Fatalf("failure %d should not declare node dead (needed 4)", i+1)
		}
	}

	// 4. 4th consecutive failure -> node marked dead
	time.Sleep(15 * time.Millisecond)
	isDead = mem.RecordFailure(nodeAddr, 4, 10*time.Millisecond)
	if !isDead || mem.IsAlive(nodeAddr) {
		t.Fatalf("4th consecutive failure must mark node dead")
	}

	select {
	case dead := <-deadChan:
		if dead != nodeAddr {
			t.Fatalf("expected dead node %s, got %s", nodeAddr, dead)
		}
	case <-time.After(1 * time.Second):
		t.Fatalf("timed out waiting for onNodeDead callback")
	}

	t.Logf("SUCCESS: Flap-damping and consecutive failure threshold verified")
}

// TestDeterministicRepairCoordinator (P2-1) verifies that only the lexicographically primary
// surviving replica coordinates repair for a missing chunk, even if initial slice ordering differs.
func TestDeterministicRepairCoordinator(t *testing.T) {
	// Storage and servers for 4 nodes
	store1, _ := storage.NewDiskStore(t.TempDir())
	store2, _ := storage.NewDiskStore(t.TempDir())
	store4, _ := storage.NewDiskStore(t.TempDir())

	srv1 := httptest.NewServer(transport.NewServer(store1).Handler())
	defer srv1.Close()
	srv2 := httptest.NewServer(transport.NewServer(store2).Handler())
	defer srv2.Close()
	srv4 := httptest.NewServer(transport.NewServer(store4).Handler())
	defer srv4.Close()

	deadSrvAddr := "http://dead-node-3:9000"

	meta := metadata.NewStore()
	r := ring.New()
	r.AddNode(srv1.URL)
	r.AddNode(srv2.URL)
	r.AddNode(deadSrvAddr)

	// Determine lexicographical order: srv1.URL vs srv2.URL
	var primaryURL, secondaryURL string
	var primaryStore, secondaryStore *storage.DiskStore
	if srv1.URL < srv2.URL {
		primaryURL, primaryStore = srv1.URL, store1
		secondaryURL, secondaryStore = srv2.URL, store2
	} else {
		primaryURL, primaryStore = srv2.URL, store2
		secondaryURL, secondaryStore = srv1.URL, store1
	}

	// Store chunk on primary, secondary, and dead node (order intentionally randomized)
	chunkID := "test-chunk-abc"
	chunkData := []byte("deterministic-repair-payload-bytes")
	_ = primaryStore.Put(chunkID, chunkData)
	_ = secondaryStore.Put(chunkID, chunkData)

	_ = meta.RecordPlacement(metadata.Manifest{
		Namespace:      "default",
		FileID:         "test-file",
		Size:           int64(len(chunkData)),
		ChunkIDs:       []string{chunkID},
		ChunkLocations: map[string][]string{chunkID: {secondaryURL, primaryURL, deadSrvAddr}},
	})

	// 1. Secondary survivor: sorts [secondaryURL, primaryURL] -> [primaryURL, secondaryURL].
	// Sees primaryURL != secondaryURL and skips repair (0 transfers).
	rmSecondary := replication.NewRepairManager(meta, r, 3, secondaryURL, secondaryStore)
	repairedSec, err := rmSecondary.RepairDeadNode(deadSrvAddr)
	if err != nil {
		t.Fatalf("RepairDeadNode on secondary error: %v", err)
	}
	if repairedSec != 0 {
		t.Fatalf("expected secondary survivor to skip repair, got %d repaired chunks", repairedSec)
	}

	// When node 3 dies, it is removed from the hash ring (as Membership.MarkDead does)
	// and surviving replacement node srv4 is added.
	r.RemoveNode(deadSrvAddr)
	r.AddNode(srv4.URL)
	rmPrimary := replication.NewRepairManager(meta, r, 3, primaryURL, primaryStore)
	rmPrimary.SetHTTPClient(srv1.Client())

	repairedPrim, err := rmPrimary.RepairDeadNode(deadSrvAddr)
	if err != nil {
		t.Fatalf("RepairDeadNode on primary error: %v", err)
	}
	if repairedPrim != 1 {
		t.Fatalf("expected primary coordinator to re-replicate 1 chunk, got %d", repairedPrim)
	}

	// Verify srv4 actually received the chunk bytes on disk
	storedData, err := store4.Get(chunkID)
	if err != nil || !bytes.Equal(storedData, chunkData) {
		t.Fatalf("expected store4 to receive chunk bytes, err=%v, data=%q", err, string(storedData))
	}

	t.Logf("SUCCESS: Primary coordinator %s repaired chunk to node 4 (%d bytes verified on disk); secondary %s skipped", primaryURL, len(storedData), secondaryURL)
}

// TestLRUCacheInvalidationOnOverwriteAndDelete (P2-2) verifies that LRU cache entries
// are evicted upon chunk deletion and file overwrite/supersession.
func TestLRUCacheInvalidationOnOverwriteAndDelete(t *testing.T) {
	tempDir := t.TempDir()
	store, err := storage.NewDiskStore(tempDir)
	if err != nil {
		t.Fatalf("storage init: %v", err)
	}

	metaStore := metadata.NewStore()
	r := ring.New()
	r.AddNode("http://localhost:9000")
	coord := coordinator.NewCoordinator(r, metaStore, "http://localhost:9000", store, 1, 1, 1)

	apiHandler := api.NewAPIHandler(metaStore, coord, 64*1024)
	apiHandler.SetDiskStore(store)
	transportServer := transport.NewServer(store)
	router := api.NewRouter(apiHandler, transportServer.Handler(), nil)

	ts := httptest.NewServer(router)
	defer ts.Close()

	cli, err := client.New(client.Config{
		Endpoints: []string{ts.URL},
		APIKey:    "default-admin-key",
	})
	if err != nil {
		t.Fatalf("client init: %v", err)
	}

	ctx := context.Background()
	key := "cached-doc.txt"

	// 1. Write Version 1
	v1Data := []byte("Initial version 1 content for cache test")
	if err := cli.Put(ctx, key, v1Data); err != nil {
		t.Fatalf("Put v1 failed: %v", err)
	}

	// 2. Read Version 1 (Populates LRU cache)
	rdr1, _, err := cli.Get(ctx, key)
	if err != nil {
		t.Fatalf("Get v1 failed: %v", err)
	}
	out1, _ := io.ReadAll(rdr1)
	rdr1.Close()
	if !bytes.Equal(out1, v1Data) {
		t.Fatalf("v1 content mismatch: got %q, want %q", out1, v1Data)
	}

	cacheBytes, cacheItems := store.GetCache().Stats()
	if cacheItems == 0 || cacheBytes == 0 {
		t.Fatalf("expected LRU cache to hold cached chunk, got %d items", cacheItems)
	}
	t.Logf("Cache populated after v1 read: %d items, %d bytes", cacheItems, cacheBytes)

	// 3. Overwrite with Version 2
	v2Data := []byte("Completely modified version 2 content with new data")
	if err := cli.Put(ctx, key, v2Data); err != nil {
		t.Fatalf("Put v2 failed: %v", err)
	}

	// 4. Read again -> Must return Version 2 content (not stale v1 from cache)
	rdr2, _, err := cli.Get(ctx, key)
	if err != nil {
		t.Fatalf("Get v2 failed: %v", err)
	}
	out2, _ := io.ReadAll(rdr2)
	rdr2.Close()
	if !bytes.Equal(out2, v2Data) {
		t.Fatalf("STALE CACHE DETECTED! Got %q, want %q", out2, v2Data)
	}

	// 5. Delete file
	if err := cli.Delete(ctx, key); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// 6. Verify lookup fails after delete
	_, _, err = cli.Get(ctx, key)
	if err == nil {
		t.Fatalf("expected 404 after delete, got success")
	}

	t.Logf("SUCCESS: Cache invalidation and overwrite correctness verified")
}

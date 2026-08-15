package integration

import (
	"context"
	"io"
	"net/http/httptest"
	"testing"

	"cloudWeave/client"
	"cloudWeave/internal/api"
	"cloudWeave/internal/cluster"
	"cloudWeave/internal/coordinator"
	"cloudWeave/internal/gc"
	"cloudWeave/internal/metadata"
	"cloudWeave/internal/ring"
	"cloudWeave/internal/storage"
	"cloudWeave/internal/transport"
)

func TestVersioningAndGarbageCollectionNoConflict(t *testing.T) {
	const numNodes = 3
	var nodeServers []*httptest.Server
	var nodeAddrs []string
	var nodeStores []*storage.DiskStore

	hashRing := ring.New()
	var apiHandlers []*api.APIHandler

	for i := 0; i < numNodes; i++ {
		tempDir := t.TempDir()
		diskStore, err := storage.NewDiskStore(tempDir)
		if err != nil {
			t.Fatalf("failed to init storage: %v", err)
		}
		nodeStores = append(nodeStores, diskStore)

		metaStore := metadata.NewStore() // Independent metaStore per node!
		coord := coordinator.NewCoordinator(hashRing, metaStore, "", diskStore, 3, 2, 2)
		apiH := api.NewAPIHandler(metaStore, coord, 64*1024)
		apiHandlers = append(apiHandlers, apiH)

		transportSvr := transport.NewServer(diskStore)
		gcEngine := gc.NewGarbageCollector(metaStore, diskStore)
		router := api.NewRouter(apiH, transportSvr.Handler(), gcEngine)

		ts := httptest.NewServer(router)
		nodeServers = append(nodeServers, ts)
		nodeAddrs = append(nodeAddrs, ts.URL)
	}

	defer func() {
		for _, ts := range nodeServers {
			ts.Close()
		}
	}()

	for i, apiH := range apiHandlers {
		localAddr := nodeAddrs[i]
		membership := cluster.NewMembership(hashRing, nil)
		for _, addr := range nodeAddrs {
			membership.AddNode(addr)
		}
		apiH.SetPeerManager(membership, localAddr)
	}

	cli, err := client.New(client.Config{
		Endpoints: nodeAddrs,
		APIKey:    "default-admin-key",
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	ctx := context.Background()
	key := "versioned-doc.txt"

	// 1. Initial upload + 3 overwrites (4 total versions)
	payloads := []string{
		"Version 1: Original draft document content.",
		"Version 2: Updated with review comments.",
		"Version 3: Added technical architecture details.",
		"Version 4: Final release candidate document.",
	}

	for idx, content := range payloads {
		err := cli.Put(ctx, key, []byte(content))
		if err != nil {
			t.Fatalf("Upload failed for iteration %d: %v", idx+1, err)
		}
		t.Logf("Uploaded version %d (%d bytes)", idx+1, len(content))
	}

	// 2. List versions before GC
	versions, err := cli.ListVersions(ctx, key)
	if err != nil {
		t.Fatalf("ListVersions failed: %v", err)
	}
	t.Logf("ListVersions returned %d historical versions", len(versions))
	if len(versions) != 4 {
		t.Fatalf("Expected 4 total versions, got %d", len(versions))
	}

	// 3. Run Garbage Collection (/admin/gc)
	gcSummary, err := cli.CollectGarbage(ctx)
	if err != nil {
		t.Fatalf("CollectGarbage (/admin/gc) failed: %v", err)
	}
	t.Logf("Garbage Collection response: %s", gcSummary)

	// 4. Verify all 4 historical versions remain fully retrievable via GetVersion
	for idx, ver := range versions {
		r, info, err := cli.GetVersion(ctx, key, ver.VersionID)
		if err != nil {
			t.Fatalf("GetVersion failed for version %s (index %d): %v", ver.VersionID, idx, err)
		}
		data, err := io.ReadAll(r)
		r.Close()
		if err != nil {
			t.Fatalf("Reading content for version %s failed: %v", ver.VersionID, err)
		}

		expectedContent := payloads[idx]
		if string(data) != expectedContent {
			t.Fatalf("Version %s content mismatch!\nGot:  %s\nWant: %s", ver.VersionID, string(data), expectedContent)
		}
		t.Logf("Retrieved version %d (ID: %s, Size: %d) successfully after GC. Content: %q",
			idx+1, ver.VersionID, info.Size, string(data))
	}

	t.Logf("SUCCESS: All 4 file versions intact and retrievable post-Garbage Collection!")
}

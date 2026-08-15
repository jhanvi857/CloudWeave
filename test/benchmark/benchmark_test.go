package benchmark

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"net/http/httptest"
	"runtime"
	"sync/atomic"
	"testing"

	"cloudWeave/client"
	"cloudWeave/internal/api"
	"cloudWeave/internal/cluster"
	"cloudWeave/internal/coordinator"
	"cloudWeave/internal/metadata"
	"cloudWeave/internal/ring"
	"cloudWeave/internal/storage"
	"cloudWeave/internal/transport"
)

func setupCluster(b *testing.B, numNodes int) ([]*httptest.Server, *client.Client) {
	b.Helper()
	var nodeServers []*httptest.Server
	var nodeAddrs []string
	var apiHandlers []*api.APIHandler

	hashRing := ring.New()

	for i := 0; i < numNodes; i++ {
		tempDir := b.TempDir()
		diskStore, err := storage.NewDiskStore(tempDir)
		if err != nil {
			b.Fatalf("failed to init storage: %v", err)
		}

		metaStore := metadata.NewStore()
		coord := coordinator.NewCoordinator(hashRing, metaStore, "", diskStore, 3, 2, 2)
		apiH := api.NewAPIHandler(metaStore, coord, 64*1024) // 64KB chunk size for benchmarking
		apiHandlers = append(apiHandlers, apiH)

		transportSvr := transport.NewServer(diskStore)
		router := api.NewRouter(apiH, transportSvr.Handler(), nil)

		ts := httptest.NewServer(router)
		nodeServers = append(nodeServers, ts)
		nodeAddrs = append(nodeAddrs, ts.URL)
	}

	b.Cleanup(func() {
		for _, ts := range nodeServers {
			ts.Close()
		}
	})

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
		b.Fatalf("failed to create client: %v", err)
	}

	return nodeServers, cli
}

func BenchmarkUploadThroughput(b *testing.B) {
	nodeServers, cli := setupCluster(b, 3)

	singleCli, err := client.New(client.Config{
		Endpoints: []string{nodeServers[0].URL},
		APIKey:    "default-admin-key",
	})
	if err != nil {
		b.Fatalf("failed to create single endpoint client: %v", err)
	}

	dataSize := 2 * 1024 * 1024 // 2 MB
	payload := make([]byte, dataSize)
	rand.Read(payload)

	ctx := context.Background()
	runtime.GC()
	b.SetBytes(int64(dataSize))
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("bench-upload-%d.dat", i)
		if err := singleCli.Put(ctx, key, payload); err != nil {
			b.Fatalf("Put failed: %v", err)
		}
	}
	_ = cli
}

func BenchmarkDownloadPostWriteRead(b *testing.B) {
	nodeServers, cli := setupCluster(b, 3)

	singleCli, err := client.New(client.Config{
		Endpoints: []string{nodeServers[0].URL},
		APIKey:    "default-admin-key",
	})
	if err != nil {
		b.Fatalf("failed to create single endpoint client: %v", err)
	}

	dataSize := 2 * 1024 * 1024 // 2 MB
	ctx := context.Background()

	// Reuse a single payload buffer during setup to prevent heap memory thrashing
	payload := make([]byte, dataSize)
	rand.Read(payload)

	keys := make([]string, b.N)
	for i := 0; i < b.N; i++ {
		keys[i] = fmt.Sprintf("postwrite-download-%d.dat", i)
		if err := singleCli.Put(ctx, keys[i], payload); err != nil {
			b.Fatalf("Put failed: %v", err)
		}
	}

	runtime.GC()
	b.SetBytes(int64(dataSize))
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		reader, _, err := singleCli.Get(ctx, keys[i])
		if err != nil {
			b.Fatalf("Get failed: %v", err)
		}
		if _, err := io.Copy(io.Discard, reader); err != nil {
			reader.Close()
			b.Fatalf("Read failed: %v", err)
		}
		reader.Close()
	}
	_ = cli
}

func BenchmarkDownloadPostWriteReadLarge(b *testing.B) {
	nodeServers, cli := setupCluster(b, 3)

	singleCli, err := client.New(client.Config{
		Endpoints: []string{nodeServers[0].URL},
		APIKey:    "default-admin-key",
	})
	if err != nil {
		b.Fatalf("failed to create single endpoint client: %v", err)
	}

	dataSize := 10 * 1024 * 1024 // 10 MB file (160 chunks of 64KB)
	ctx := context.Background()

	// Reuse a single payload buffer during setup to prevent heap memory thrashing
	payload := make([]byte, dataSize)
	rand.Read(payload)

	keys := make([]string, b.N)
	for i := 0; i < b.N; i++ {
		keys[i] = fmt.Sprintf("postwrite-download-large-%d.dat", i)
		if err := singleCli.Put(ctx, keys[i], payload); err != nil {
			b.Fatalf("Put failed: %v", err)
		}
	}

	runtime.GC()
	b.SetBytes(int64(dataSize))
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		reader, _, err := singleCli.Get(ctx, keys[i])
		if err != nil {
			b.Fatalf("Get failed: %v", err)
		}
		if _, err := io.Copy(io.Discard, reader); err != nil {
			reader.Close()
			b.Fatalf("Read failed: %v", err)
		}
		reader.Close()
	}
	_ = cli
}

func BenchmarkDownloadWarmCache(b *testing.B) {
	nodeServers, cli := setupCluster(b, 3)

	singleCli, err := client.New(client.Config{
		Endpoints: []string{nodeServers[0].URL},
		APIKey:    "default-admin-key",
	})
	if err != nil {
		b.Fatalf("failed to create single endpoint client: %v", err)
	}

	dataSize := 2 * 1024 * 1024 // 2 MB
	payload := make([]byte, dataSize)
	rand.Read(payload)

	ctx := context.Background()
	key := "warm-download-target.dat"
	if err := singleCli.Put(ctx, key, payload); err != nil {
		b.Fatalf("Put failed: %v", err)
	}

	// Prime the cache
	r, _, _ := singleCli.Get(ctx, key)
	if r != nil {
		io.Copy(io.Discard, r)
		r.Close()
	}

	runtime.GC()
	b.SetBytes(int64(dataSize))
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		reader, _, err := singleCli.Get(ctx, key)
		if err != nil {
			b.Fatalf("Get failed: %v", err)
		}
		if _, err := io.Copy(io.Discard, reader); err != nil {
			reader.Close()
			b.Fatalf("Read failed: %v", err)
		}
		reader.Close()
	}
	_ = cli
}

func BenchmarkConcurrentLoad(b *testing.B) {
	nodeServers, cli := setupCluster(b, 3)

	// Create single-endpoint client for concurrent QPS test to avoid async broadcast race
	singleCli, err := client.New(client.Config{
		Endpoints: []string{nodeServers[0].URL},
		APIKey:    "default-admin-key",
	})
	if err != nil {
		b.Fatalf("failed to create single endpoint client: %v", err)
	}

	dataSize := 64 * 1024 // 64 KB
	payload := make([]byte, dataSize)
	rand.Read(payload)

	ctx := context.Background()
	key := "bench-concurrent-target.dat"
	if err := singleCli.Put(ctx, key, payload); err != nil {
		b.Fatalf("Put failed: %v", err)
	}
	var counter uint64

	runtime.GC()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			id := atomic.AddUint64(&counter, 1)
			key := fmt.Sprintf("bench-concurrent-%d.dat", id)

			p := make([]byte, dataSize)
			copy(p, payload)
			p[0] = byte(id)

			if err := singleCli.Put(ctx, key, p); err != nil {
				b.Errorf("Put failed: %v", err)
				return
			}
			reader, _, err := singleCli.Get(ctx, key)
			if err != nil {
				b.Errorf("Get failed: %v", err)
				return
			}
			io.Copy(io.Discard, reader)
			reader.Close()
		}
	})
	_ = cli
}

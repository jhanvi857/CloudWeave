package integration

import (
	"context"
	"crypto/rand"
	"bytes"
	"io"
	"net/http/httptest"
	"os"
	"path/filepath"
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

func dirSize(dir string) (int64, error) {
	var size int64
	err := filepath.Walk(dir, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size, err
}

func TestDeduplication_DiskSpaceWithEncryption(t *testing.T) {
	const numNodes = 3
	var nodeServers []*httptest.Server
	var nodeAddrs []string
	var nodeDataDirs []string

	hashRing := ring.New()
	metaStore := metadata.NewStore()
	var apiHandlers []*api.APIHandler

	for i := 0; i < numNodes; i++ {
		tempDir := t.TempDir()
		nodeDataDirs = append(nodeDataDirs, tempDir)
		diskStore, err := storage.NewDiskStore(tempDir)
		if err != nil {
			t.Fatalf("failed to init storage: %v", err)
		}

		coord := coordinator.NewCoordinator(hashRing, metaStore, "", diskStore, 3, 2, 2)
		apiH := api.NewAPIHandler(metaStore, coord, 64*1024)
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

	for i, apiH := range apiHandlers {
		localAddr := nodeAddrs[i]
		membership := cluster.NewMembership(hashRing, nil)
		for _, addr := range nodeAddrs {
			membership.AddNode(addr)
		}
		apiH.SetPeerManager(membership, localAddr)
	}

	cli, err := client.New(client.Config{
		Endpoints:            nodeAddrs,
		APIKey:               "default-admin-key",
		EncryptionPassphrase: "test-dedup-secret-passphrase",
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	ctx := context.Background()
	payloadSize := 1024 * 1024 // 1 MB payload
	payload := make([]byte, payloadSize)
	rand.Read(payload)

	// Step 1: Measure disk usage before 1st upload
	var initialBytes int64
	for _, dir := range nodeDataDirs {
		sz, _ := dirSize(dir)
		initialBytes += sz
	}
	t.Logf("Disk usage BEFORE 1st upload: %d bytes", initialBytes)

	// Step 2: First upload with encryption enabled
	err = cli.Put(ctx, "encrypted-file-1.dat", payload)
	if err != nil {
		t.Fatalf("First upload failed: %v", err)
	}

	var bytesAfterFirstUpload int64
	for _, dir := range nodeDataDirs {
		sz, _ := dirSize(dir)
		bytesAfterFirstUpload += sz
	}
	t.Logf("Disk usage AFTER 1st upload (encrypted-file-1.dat): %d bytes", bytesAfterFirstUpload)

	if bytesAfterFirstUpload <= initialBytes {
		t.Fatalf("Expected disk usage to increase after 1st upload, got %d", bytesAfterFirstUpload)
	}

	// Step 3: Second upload with SAME content & encryption enabled under different key
	err = cli.Put(ctx, "encrypted-file-2.dat", payload)
	if err != nil {
		t.Fatalf("Second upload failed: %v", err)
	}

	var bytesAfterSecondUpload int64
	for _, dir := range nodeDataDirs {
		sz, _ := dirSize(dir)
		bytesAfterSecondUpload += sz
	}
	t.Logf("Disk usage AFTER 2nd upload (encrypted-file-2.dat): %d bytes", bytesAfterSecondUpload)

	// Step 4: Assert 2nd upload did NOT add new chunk bytes to disk
	addedBytes := bytesAfterSecondUpload - bytesAfterFirstUpload
	t.Logf("Bytes added by 2nd upload of identical encrypted file: %d bytes", addedBytes)

	if addedBytes != 0 {
		t.Fatalf("Deduplication failed! Second upload added %d bytes (expected 0 new bytes)", addedBytes)
	}

	// Step 5: Download and verify both files decrypt correctly
	r1, info1, err := cli.Get(ctx, "encrypted-file-1.dat")
	if err != nil {
		t.Fatalf("Get file 1 failed: %v", err)
	}
	data1, err := io.ReadAll(r1)
	r1.Close()
	if err != nil {
		t.Fatalf("Read file 1 failed: %v", err)
	}

	r2, info2, err := cli.Get(ctx, "encrypted-file-2.dat")
	if err != nil {
		t.Fatalf("Get file 2 failed: %v", err)
	}
	data2, err := io.ReadAll(r2)
	r2.Close()
	if err != nil {
		t.Fatalf("Read file 2 failed: %v", err)
	}

	if !bytes.Equal(data1, payload) {
		t.Fatalf("File 1 content mismatch after decryption")
	}
	if !bytes.Equal(data2, payload) {
		t.Fatalf("File 2 content mismatch after decryption")
	}

	t.Logf("SUCCESS: Deduplication verified! Both encrypted files retrieve identical %d byte payload. File 1 encrypted=%s, File 2 encrypted=%s",
		len(data1), info1.Metadata["encrypted"], info2.Metadata["encrypted"])
}

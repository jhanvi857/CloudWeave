package gc

import (
	"os"
	"testing"

	"cloudWeave/internal/metadata"
	"cloudWeave/internal/storage"
)

func TestGarbageCollector_SweepOrphanChunks(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "gc_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	diskStore, err := storage.NewDiskStore(tempDir)
	if err != nil {
		t.Fatalf("failed to create disk store: %v", err)
	}

	metaStore := metadata.NewStore()

	// Put 3 chunks on disk
	chunk1 := "chunk1_live"
	chunk2 := "chunk2_orphan"
	chunk3 := "chunk3_orphan"

	_ = diskStore.Put(chunk1, []byte("data 1"))
	_ = diskStore.Put(chunk2, []byte("data 2"))
	_ = diskStore.Put(chunk3, []byte("data 3"))

	// Manifest only references chunk1
	_ = metaStore.RecordPlacement(metadata.Manifest{
		FileID:   "fileA",
		Size:     6,
		ChunkIDs: []string{chunk1},
	})

	gcEngine := NewGarbageCollector(metaStore, diskStore)
	deleted, err := gcEngine.CollectGarbage()
	if err != nil {
		t.Fatalf("CollectGarbage failed: %v", err)
	}

	if deleted != 2 {
		t.Fatalf("expected 2 orphan chunks deleted, got %d", deleted)
	}

	if !diskStore.Exists(chunk1) {
		t.Errorf("expected live chunk1 to remain on disk")
	}

	if diskStore.Exists(chunk2) || diskStore.Exists(chunk3) {
		t.Errorf("expected orphan chunks chunk2 and chunk3 to be deleted")
	}
}

func TestGarbageCollector_NamespaceIsolationSafety(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "gc_ns_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	diskStore, _ := storage.NewDiskStore(tempDir)
	metaStore := metadata.NewStore()

	sharedChunk := "shared_chunk_1"
	nsAOnlyChunk := "nsA_chunk"
	nsBOnlyChunk := "nsB_chunk"

	_ = diskStore.Put(sharedChunk, []byte("shared content"))
	_ = diskStore.Put(nsAOnlyChunk, []byte("nsA content"))
	_ = diskStore.Put(nsBOnlyChunk, []byte("nsB content"))

	// Manifest in tenant-a uses sharedChunk & nsAOnlyChunk
	_ = metaStore.RecordPlacement(metadata.Manifest{
		Namespace: "tenant-a",
		FileID:    "fileA",
		ChunkIDs:  []string{sharedChunk, nsAOnlyChunk},
	})

	// Manifest in tenant-b uses sharedChunk & nsBOnlyChunk
	_ = metaStore.RecordPlacement(metadata.Manifest{
		Namespace: "tenant-b",
		FileID:    "fileB",
		ChunkIDs:  []string{sharedChunk, nsBOnlyChunk},
	})

	gcEngine := NewGarbageCollector(metaStore, diskStore)

	// Now delete tenant-a file
	metaStore.DeleteScoped("tenant-a", "fileA")

	// Run GC
	deleted, err := gcEngine.CollectGarbageForNamespace("tenant-a")
	if err != nil {
		t.Fatalf("GC failed: %v", err)
	}

	if deleted != 1 {
		t.Fatalf("expected exactly 1 chunk (nsAOnlyChunk) deleted, got %d", deleted)
	}

	if diskStore.Exists(nsAOnlyChunk) {
		t.Errorf("nsAOnlyChunk should be deleted after fileA deletion")
	}
	if !diskStore.Exists(sharedChunk) {
		t.Errorf("sharedChunk MUST NOT be deleted because tenant-b fileB still references it!")
	}
	if !diskStore.Exists(nsBOnlyChunk) {
		t.Errorf("nsBOnlyChunk MUST remain on disk")
	}
}


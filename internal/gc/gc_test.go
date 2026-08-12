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

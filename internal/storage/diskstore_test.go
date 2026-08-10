package storage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewDiskStore(t *testing.T) {
	t.Run("empty baseDir", func(t *testing.T) {
		_, err := NewDiskStore("")
		if err == nil {
			t.Errorf("expected error for empty baseDir, got nil")
		}
	})

	t.Run("valid baseDir creation", func(t *testing.T) {
		tempDir := t.TempDir()
		storeDir := filepath.Join(tempDir, "test-data")

		store, err := NewDiskStore(storeDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if store == nil {
			t.Fatal("expected non-nil DiskStore")
		}

		fi, err := os.Stat(storeDir)
		if err != nil {
			t.Fatalf("expected directory to exist: %v", err)
		}
		if !fi.IsDir() {
			t.Errorf("expected %s to be a directory", storeDir)
		}
	})
}

func TestDiskStore_PutGetExists(t *testing.T) {
	tempDir := t.TempDir()
	store, err := NewDiskStore(tempDir)
	if err != nil {
		t.Fatalf("failed to create DiskStore: %v", err)
	}

	chunkID := "chunk-1"
	data := []byte("hello cloudweave")

	if store.Exists(chunkID) {
		t.Errorf("expected chunk %s to not exist yet", chunkID)
	}

	if err := store.Put(chunkID, data); err != nil {
		t.Fatalf("failed to Put chunk: %v", err)
	}

	if !store.Exists(chunkID) {
		t.Errorf("expected chunk %s to exist after Put", chunkID)
	}

	got, err := store.Get(chunkID)
	if err != nil {
		t.Fatalf("failed to Get chunk: %v", err)
	}

	if string(got) != string(data) {
		t.Errorf("got %q, want %q", string(got), string(data))
	}
}

package metadata

import (
	"path/filepath"
	"testing"

	"cloudWeave/internal/auth"
)

func TestWAL_PersistenceAndReplay(t *testing.T) {
	tempDir := t.TempDir()
	walPath := filepath.Join(tempDir, "metadata.wal")

	// 1. Open WAL and record state
	wal, err := OpenWAL(walPath)
	if err != nil {
		t.Fatalf("failed to open WAL: %v", err)
	}

	store1 := NewStore()
	store1.SetWAL(wal)

	m1 := Manifest{
		FileID:   "file-wal-1",
		Size:     2048,
		ChunkIDs: []string{"chunk-101", "chunk-102"},
		ChunkLocations: map[string][]string{
			"chunk-101": {"http://localhost:8080"},
			"chunk-102": {"http://localhost:8080", "http://localhost:8081"},
		},
	}
	if err := store1.RecordPlacement(m1); err != nil {
		t.Fatalf("RecordPlacement failed: %v", err)
	}

	store1.UpdateChunkLocations("chunk-101", []string{"http://localhost:8080", "http://localhost:8082"})
	wal.Close()

	// 2. Crash / restart simulation: Create a brand new empty store
	store2 := NewStore()

	// Verify store2 is empty
	if _, found := store2.Lookup("file-wal-1"); found {
		t.Fatalf("new store should be empty initially")
	}

	// 3. Replay WAL into store2
	if err := ReplayWAL(walPath, store2); err != nil {
		t.Fatalf("ReplayWAL failed: %v", err)
	}

	// 4. Assert restored state
	mRestored, found := store2.Lookup("file-wal-1")
	if !found {
		t.Fatalf("expected file-wal-1 to be restored from WAL")
	}

	if mRestored.Size != m1.Size || len(mRestored.ChunkIDs) != 2 {
		t.Errorf("restored manifest mismatch: got %+v", mRestored)
	}

	locs101 := mRestored.ChunkLocations["chunk-101"]
	if len(locs101) != 2 || locs101[1] != "http://localhost:8082" {
		t.Errorf("updated chunk locations not restored correctly from WAL: got %v", locs101)
	}
}

func TestWAL_CredentialHashedPersistenceAndReplay(t *testing.T) {
	tempDir := t.TempDir()
	walPath := filepath.Join(tempDir, "keys.wal")

	wal, err := OpenWAL(walPath)
	if err != nil {
		t.Fatalf("failed to open WAL: %v", err)
	}

	auth1 := auth.NewDefaultAuthenticator()
	rawKey := "secret-dynamic-key-123"
	cred := auth1.AddRawKey(rawKey, []string{"tenant-wal"}, false)

	// Write Key WAL record (storing SHA-256 hash only)
	if err := wal.WriteRecord(WALRecord{
		Op:         OpRecordKey,
		Credential: cred,
	}); err != nil {
		t.Fatalf("WriteRecord failed: %v", err)
	}
	wal.Close()

	// Replay WAL into fresh authenticator
	auth2 := auth.NewDefaultAuthenticator()
	store := NewStore()
	if err := ReplayWAL(walPath, store, auth2); err != nil {
		t.Fatalf("ReplayWAL failed: %v", err)
	}

	// Validate with rawKey against auth2
	validated, ok := auth2.ValidateKey(rawKey)
	if !ok || validated == nil {
		t.Fatalf("expected rawKey to validate after WAL replay")
	}
	if !validated.CanAccessNamespace("tenant-wal") {
		t.Errorf("expected tenant-wal permission after WAL replay")
	}
}


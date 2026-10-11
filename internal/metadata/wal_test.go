package metadata

import (
	"os"
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

func TestWAL_TornTailRecovery(t *testing.T) {
	tempDir := t.TempDir()
	walPath := filepath.Join(tempDir, "torn_tail.wal")

	// 1. Write 2 valid records
	wal, err := OpenWAL(walPath)
	if err != nil {
		t.Fatalf("OpenWAL: %v", err)
	}

	m1 := Manifest{FileID: "valid-doc-1", Size: 100}
	m2 := Manifest{FileID: "valid-doc-2", Size: 200}
	_ = wal.WriteRecord(WALRecord{Op: OpRecordManifest, Manifest: m1})
	_ = wal.WriteRecord(WALRecord{Op: OpRecordManifest, Manifest: m2})
	wal.Close()

	// 2. Simulate a crash: Append corrupted / torn trailing bytes without closing newline
	tornData := []byte(`{"op":"RECORD_MANIFEST","manifest":{"file_id":"crashed-unfini`)
	f, err := os.OpenFile(walPath, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatalf("OpenFile: %v", err)
	}
	_, _ = f.Write(tornData)
	f.Close()

	// 3. Replay WAL - must NOT fail; must recover the 2 valid records
	store := NewStore()
	if err := ReplayWAL(walPath, store); err != nil {
		t.Fatalf("ReplayWAL failed on torn tail: %v", err)
	}

	if _, found := store.Lookup("valid-doc-1"); !found {
		t.Errorf("expected valid-doc-1 to be restored")
	}
	if _, found := store.Lookup("valid-doc-2"); !found {
		t.Errorf("expected valid-doc-2 to be restored")
	}
	if _, found := store.Lookup("crashed-unfini"); found {
		t.Errorf("crashed uncommitted record should not exist")
	}
}

func TestWAL_Compaction(t *testing.T) {
	tempDir := t.TempDir()
	walPath := filepath.Join(tempDir, "compact.wal")

	store := NewStore()
	wal, _ := OpenWAL(walPath)
	store.SetWAL(wal)

	// Write and overwrite
	_ = store.RecordPlacement(Manifest{FileID: "file-v1", Size: 50})
	_ = store.RecordPlacement(Manifest{FileID: "file-v1", Size: 150})
	_ = store.RecordPlacement(Manifest{FileID: "file-v2", Size: 250})
	_ = store.Delete("file-v1")
	wal.Close()

	// Compact WAL
	if err := CompactWAL(walPath, store, nil); err != nil {
		t.Fatalf("CompactWAL failed: %v", err)
	}

	// Replay compacted WAL into empty store
	freshStore := NewStore()
	if err := ReplayWAL(walPath, freshStore); err != nil {
		t.Fatalf("ReplayWAL from compacted failed: %v", err)
	}

	if _, found := freshStore.Lookup("file-v1"); found {
		t.Errorf("deleted file-v1 should not exist after compaction")
	}
	if m, found := freshStore.Lookup("file-v2"); !found || m.Size != 250 {
		t.Errorf("file-v2 not restored properly from compacted WAL")
	}
}


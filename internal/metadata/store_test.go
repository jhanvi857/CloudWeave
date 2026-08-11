package metadata

import (
	"testing"
)

func TestMetadataStore_RecordAndLookup(t *testing.T) {
	store := NewStore()

	m := Manifest{
		FileID:   "file-1",
		Size:     1024,
		ChunkIDs: []string{"chunk-a", "chunk-b"},
		ChunkLocations: map[string][]string{
			"chunk-a": {"node1", "node2"},
			"chunk-b": {"node2", "node3"},
		},
	}

	if err := store.RecordPlacement(m); err != nil {
		t.Fatalf("failed to record placement: %v", err)
	}

	got, found := store.Lookup("file-1")
	if !found {
		t.Fatalf("expected file-1 to be found")
	}

	if got.FileID != m.FileID || got.Size != m.Size || len(got.ChunkIDs) != 2 {
		t.Errorf("manifest mismatch: got %+v, want %+v", got, m)
	}

	// Verify deep copy / clone
	m.ChunkLocations["chunk-a"][0] = "mutated"
	gotAfterMutation, _ := store.Lookup("file-1")
	if gotAfterMutation.ChunkLocations["chunk-a"][0] == "mutated" {
		t.Errorf("store did not isolate internal memory from external mutations")
	}
}

func TestMetadataStore_UpdateChunkLocations(t *testing.T) {
	store := NewStore()

	m := Manifest{
		FileID:   "doc.txt",
		Size:     500,
		ChunkIDs: []string{"c1"},
		ChunkLocations: map[string][]string{
			"c1": {"node1"},
		},
	}
	_ = store.RecordPlacement(m)

	store.UpdateChunkLocations("c1", []string{"node1", "node2", "node3"})

	got, _ := store.Lookup("doc.txt")
	locs := got.ChunkLocations["c1"]
	if len(locs) != 3 || locs[1] != "node2" {
		t.Errorf("expected updated locations [node1 node2 node3], got %v", locs)
	}
}

func TestMetadataStore_Delete(t *testing.T) {
	store := NewStore()
	m := Manifest{FileID: "temp.bin", ChunkLocations: map[string][]string{}}
	_ = store.RecordPlacement(m)

	if !store.Delete("temp.bin") {
		t.Errorf("expected delete to return true")
	}

	if _, found := store.Lookup("temp.bin"); found {
		t.Errorf("expected deleted file to not be found")
	}
}

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

func TestMetadataStore_NamespacesAndMetadata(t *testing.T) {
	store := NewStore()

	m1 := Manifest{
		Namespace:   "tenant-a",
		FileID:      "report.pdf",
		Size:        1024,
		ContentType: "application/pdf",
		Metadata: map[string]string{
			"Author": "Alice",
		},
		ChunkIDs:       []string{"c1"},
		ChunkLocations: map[string][]string{"c1": {"n1"}},
	}
	m2 := Manifest{
		Namespace:   "tenant-b",
		FileID:      "report.pdf",
		Size:        2048,
		ContentType: "text/plain",
		Metadata: map[string]string{
			"Author": "Bob",
		},
		ChunkIDs:       []string{"c2"},
		ChunkLocations: map[string][]string{"c2": {"n2"}},
	}

	if err := store.RecordPlacement(m1); err != nil {
		t.Fatalf("failed to record m1: %v", err)
	}
	if err := store.RecordPlacement(m2); err != nil {
		t.Fatalf("failed to record m2: %v", err)
	}

	// LookupScoped tenant-a
	gotA, foundA := store.LookupScoped("tenant-a", "report.pdf")
	if !foundA || gotA.Size != 1024 || gotA.ContentType != "application/pdf" || gotA.Metadata["Author"] != "Alice" {
		t.Errorf("mismatch for tenant-a: %+v", gotA)
	}

	// LookupScoped tenant-b
	gotB, foundB := store.LookupScoped("tenant-b", "report.pdf")
	if !foundB || gotB.Size != 2048 || gotB.ContentType != "text/plain" || gotB.Metadata["Author"] != "Bob" {
		t.Errorf("mismatch for tenant-b: %+v", gotB)
	}

	// Delete tenant-a should not delete tenant-b
	if !store.DeleteScoped("tenant-a", "report.pdf") {
		t.Errorf("expected DeleteScoped tenant-a to succeed")
	}
	if _, found := store.LookupScoped("tenant-a", "report.pdf"); found {
		t.Errorf("tenant-a object should be deleted")
	}
	if _, found := store.LookupScoped("tenant-b", "report.pdf"); !found {
		t.Errorf("tenant-b object should still exist")
	}
}


package consensus

import (
	"testing"
	"time"

	"cloudWeave/internal/metadata"
)

func TestRaftConsensus_ProposeAndCommit(t *testing.T) {
	store := metadata.NewStore()
	raftNode := NewRaftNode("node-1", []string{"node-2", "node-3"}, store)
	raftNode.Start()
	defer raftNode.Stop()

	raftNode.ForceLeader()

	term, role := raftNode.GetState()
	if role != Leader || term != 1 {
		t.Fatalf("expected Leader at term 1, got %v term %d", role, term)
	}

	m := metadata.Manifest{
		FileID:   "raft-doc-1",
		Size:     1024,
		ChunkIDs: []string{"chunk-r1"},
		ChunkLocations: map[string][]string{
			"chunk-r1": {"node-1"},
		},
	}

	if err := raftNode.ProposeManifest(m); err != nil {
		t.Fatalf("failed to propose manifest: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	got, found := store.Lookup("raft-doc-1")
	if !found {
		t.Fatalf("expected manifest to be committed to store via Raft")
	}

	if got.Size != 1024 {
		t.Errorf("manifest size mismatch: got %d, want 1024", got.Size)
	}
}

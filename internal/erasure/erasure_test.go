package erasure

import (
	"bytes"
	"testing"
)

func TestErasureCoding_EncodeAndReconstruct(t *testing.T) {
	enc, err := NewEncoder(4, 2) // K=4, M=2
	if err != nil {
		t.Fatalf("failed to create encoder: %v", err)
	}

	payload := []byte("CloudWeave Reed-Solomon Erasure Coding Data Shard Reconstruction Test Payload 12345!")
	shards, err := enc.Encode(payload)
	if err != nil {
		t.Fatalf("failed to encode: %v", err)
	}

	if len(shards) != 6 {
		t.Fatalf("expected 6 total shards (4 data + 2 parity), got %d", len(shards))
	}

	// Simulate losing 2 shards (e.g., data shard 1 and parity shard 5)
	available := make(map[int][]byte)
	for _, s := range shards {
		if s.Index != 1 && s.Index != 5 {
			available[s.Index] = s.Data
		}
	}

	if len(available) != 4 {
		t.Fatalf("expected 4 available shards, got %d", len(available))
	}

	reconstructed, err := enc.Reconstruct(available, len(payload))
	if err != nil {
		t.Fatalf("reconstruction failed: %v", err)
	}

	if !bytes.Equal(reconstructed, payload) {
		t.Fatalf("reconstructed payload mismatch!\nGot:  %s\nWant: %s", string(reconstructed), string(payload))
	}
}

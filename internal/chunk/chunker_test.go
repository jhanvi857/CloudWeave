package chunk

import (
	"bytes"
	"math/rand"
	"testing"
)

func TestSplitReassemble_RoundTrip(t *testing.T) {
	original := make([]byte, 10*1024*1024+37) // deliberately not a clean multiple of chunkSize
	rand.New(rand.NewSource(42)).Read(original)

	chunks, err := Split(original, 4*1024*1024)
	if err != nil {
		t.Fatalf("Split failed: %v", err)
	}

	// Simulate out-of-order network delivery.
	rand.Shuffle(len(chunks), func(i, j int) {
		chunks[i], chunks[j] = chunks[j], chunks[i]
	})

	result, err := Reassemble(chunks)
	if err != nil {
		t.Fatalf("Reassemble failed: %v", err)
	}

	if !bytes.Equal(original, result) {
		t.Fatal("reassembled data does not match original")
	}
}

func TestReassemble_DetectsCorruption(t *testing.T) {
	chunks, _ := Split([]byte("hello world, this is a test file"), 8)
	chunks[1].Data[0] ^= 0xFF // corrupt one byte after chunking

	if _, err := Reassemble(chunks); err == nil {
		t.Fatal("expected integrity check to catch corrupted chunk, got nil error")
	}
}

func TestSplit_LastChunkShorter(t *testing.T) {
	chunks, err := Split(make([]byte, 10), 4)
	if err != nil {
		t.Fatalf("Split failed: %v", err)
	}
	if len(chunks) != 3 {
		t.Fatalf("expected 3 chunks, got %d", len(chunks))
	}
	if len(chunks[2].Data) != 2 {
		t.Fatalf("expected last chunk to have 2 bytes, got %d", len(chunks[2].Data))
	}
}

func TestSplitStream_RoundTrip(t *testing.T) {
	original := make([]byte, 5*1024*1024+123)
	rand.New(rand.NewSource(99)).Read(original)

	var collected []Chunk
	totalBytes, chunkIDs, err := SplitStream(bytes.NewReader(original), 1*1024*1024, func(c Chunk) error {
		collected = append(collected, c)
		return nil
	})
	if err != nil {
		t.Fatalf("SplitStream failed: %v", err)
	}

	if totalBytes != int64(len(original)) {
		t.Fatalf("expected %d bytes, got %d", len(original), totalBytes)
	}
	if len(chunkIDs) != len(collected) {
		t.Fatalf("expected %d chunkIDs, got %d", len(collected), len(chunkIDs))
	}

	reassembled, err := Reassemble(collected)
	if err != nil {
		t.Fatalf("Reassemble stream chunks failed: %v", err)
	}

	if !bytes.Equal(original, reassembled) {
		t.Fatal("reassembled stream data does not match original")
	}
}


package chunk

import (
	"bytes"
	"crypto/rand"
	"testing"
)

func TestCDC_SplitAndReassemble(t *testing.T) {
	data := make([]byte, 500*1024) // 500 KB payload
	rand.Read(data)

	chunks, err := SplitCDC(data, 16*1024, 64*1024)
	if err != nil {
		t.Fatalf("SplitCDC failed: %v", err)
	}

	if len(chunks) == 0 {
		t.Fatalf("expected chunks generated")
	}

	reassembled, err := Reassemble(chunks)
	if err != nil {
		t.Fatalf("Reassemble failed: %v", err)
	}

	if !bytes.Equal(reassembled, data) {
		t.Errorf("Reassembled content mismatch!")
	}
}

func TestCDC_DeduplicationContentBoundary(t *testing.T) {
	// Original 200KB payload
	baseData := make([]byte, 200*1024)
	rand.Read(baseData)

	// Modified payload with 10 bytes prepended at start
	modifiedData := append([]byte("MODIFIED__"), baseData...)

	chunks1, _ := SplitCDC(baseData, 16*1024, 64*1024)
	chunks2, _ := SplitCDC(modifiedData, 16*1024, 64*1024)

	// Build map of chunk IDs from dataset 1
	seen1 := make(map[string]bool)
	for _, c := range chunks1 {
		seen1[c.ID] = true
	}

	// Count duplicate matching chunk IDs in dataset 2
	sharedCount := 0
	for _, c := range chunks2 {
		if seen1[c.ID] {
			sharedCount++
		}
	}

	if sharedCount == 0 {
		t.Errorf("CDC failed to produce duplicate chunk IDs across shifted content insertions!")
	}
	t.Logf("CDC Deduplication Success: Shared %d identical chunks across shifted content", sharedCount)
}

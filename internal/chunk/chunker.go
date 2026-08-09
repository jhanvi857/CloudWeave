package chunk

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
)

func Split(data []byte, chunkSize int) ([]Chunk, error) {
	if chunkSize <= 0 {
		return nil, fmt.Errorf("Chunk size must be positive, got %d", chunkSize)
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("Cannot split empty data")
	}
	var chunks []Chunk
	for i, start := 0, 0; start < len(data); i, start = i+1, start+chunkSize {
		end := start + chunkSize
		if end > len(data) {
			end = len(data) // last chunk is shorter
		}
		piece := data[start:end]
		sum := sha256.Sum256(piece)
		chunks = append(chunks, Chunk{
			ID:    hex.EncodeToString(sum[:]),
			Data:  piece,
			Index: i,
		})
	}
	return chunks, nil
}

func Reassemble(chunks []Chunk) ([]byte, error) {
	if len(chunks) == 0 {
		return nil, fmt.Errorf("no chunks to reassemble")
	}
	ordered := make([]Chunk, len(chunks))
	copy(ordered, chunks)
	sort.Slice(ordered, func(i, j int) bool {
		return ordered[i].Index < ordered[j].Index
	})
	var buf bytes.Buffer
	for _, c := range ordered {
		if !verify(c) {
			return nil, fmt.Errorf("chunk %d (id=%s) failed integrity check", c.Index, c.ID)
		}
		buf.Write(c.Data)
	}
	return buf.Bytes(), nil
}

func verify(c Chunk) bool {
	sum := sha256.Sum256(c.Data)
	return hex.EncodeToString(sum[:]) == c.ID
}

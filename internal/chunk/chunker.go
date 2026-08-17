package chunk

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"sort"
	"sync"
)


var defaultBufferPool = sync.Pool{
	New: func() interface{} {
		b := make([]byte, 1024*1024)
		return &b
	},
}

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

// SplitStream reads data from an io.Reader in chunkSize increments, calculates SHA-256 for each chunk,
// and invokes processChunk callback. It returns total bytes read, chunk IDs, and any error encountered.
func SplitStream(r io.Reader, chunkSize int, processChunk func(c Chunk) error) (int64, []string, error) {
	if chunkSize <= 0 {
		return 0, nil, fmt.Errorf("chunk size must be positive, got %d", chunkSize)
	}

	var buf []byte
	if chunkSize == 1024*1024 {
		bufPtr := defaultBufferPool.Get().(*[]byte)
		buf = *bufPtr
		defer defaultBufferPool.Put(bufPtr)
	} else {
		buf = make([]byte, chunkSize)
	}

	var totalBytes int64
	var chunkIDs []string
	idx := 0

	for {
		n, err := io.ReadFull(r, buf)
		if n > 0 {
			totalBytes += int64(n)
			piece := make([]byte, n)
			copy(piece, buf[:n])
			sum := sha256.Sum256(piece)
			cID := hex.EncodeToString(sum[:])

			c := Chunk{
				ID:    cID,
				Data:  piece,
				Index: idx,
			}
			chunkIDs = append(chunkIDs, cID)
			idx++

			if processChunk != nil {
				if procErr := processChunk(c); procErr != nil {
					return totalBytes, chunkIDs, fmt.Errorf("chunk %d processing error: %w", c.Index, procErr)
				}
			}
		}

		if err == io.EOF || err == io.ErrUnexpectedEOF {
			break
		}
		if err != nil {
			return totalBytes, chunkIDs, fmt.Errorf("reading stream chunk %d: %w", idx, err)
		}
	}

	if totalBytes == 0 {
		return 0, nil, fmt.Errorf("cannot split empty data")
	}

	return totalBytes, chunkIDs, nil
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


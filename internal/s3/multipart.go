package s3

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sort"
	"sync"
	"time"
)

// PartInfo represents metadata for an uploaded part.
type PartInfo struct {
	PartNumber     int                 `json:"part_number"`
	ETag           string              `json:"etag"`
	Size           int64               `json:"size"`
	ChunkIDs       []string            `json:"chunk_ids"`
	ChunkLocations map[string][]string `json:"chunk_locations"`
}

// MultipartRecord tracks an in-progress multipart upload.
type MultipartRecord struct {
	UploadID  string           `json:"upload_id"`
	Bucket    string           `json:"bucket"`
	Key       string           `json:"key"`
	CreatedAt time.Time        `json:"created_at"`
	Parts     map[int]PartInfo `json:"parts"`
}

// MultipartStore provides thread-safe access to active multipart uploads.
type MultipartStore struct {
	mu      sync.RWMutex
	uploads map[string]*MultipartRecord
}

// NewMultipartStore initializes a new MultipartStore.
func NewMultipartStore() *MultipartStore {
	return &MultipartStore{
		uploads: make(map[string]*MultipartRecord),
	}
}

func generateUploadID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("mp_%d_%s", time.Now().UnixNano(), hex.EncodeToString(b))
}

// CreateUpload initializes a new multipart upload session.
func (m *MultipartStore) CreateUpload(bucket, key string) string {
	m.mu.Lock()
	defer m.mu.Unlock()

	uploadID := generateUploadID()
	rec := &MultipartRecord{
		UploadID:  uploadID,
		Bucket:    bucket,
		Key:       key,
		CreatedAt: time.Now().UTC(),
		Parts:     make(map[int]PartInfo),
	}
	m.uploads[uploadID] = rec
	return uploadID
}

// AddPart registers an uploaded part for an active multipart upload.
func (m *MultipartStore) AddPart(uploadID string, partNumber int, etag string, size int64, chunkIDs []string, chunkLocations map[string][]string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	rec, exists := m.uploads[uploadID]
	if !exists {
		return fmt.Errorf("NoSuchUpload")
	}

	cl := make(map[string][]string, len(chunkLocations))
	for k, v := range chunkLocations {
		cl[k] = append([]string(nil), v...)
	}

	rec.Parts[partNumber] = PartInfo{
		PartNumber:     partNumber,
		ETag:           etag,
		Size:           size,
		ChunkIDs:       append([]string(nil), chunkIDs...),
		ChunkLocations: cl,
	}
	return nil
}

// GetUpload retrieves an active multipart upload record.
func (m *MultipartStore) GetUpload(uploadID string) (*MultipartRecord, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	rec, exists := m.uploads[uploadID]
	if !exists {
		return nil, false
	}

	// Clone record
	parts := make(map[int]PartInfo, len(rec.Parts))
	for k, v := range rec.Parts {
		parts[k] = v
	}

	return &MultipartRecord{
		UploadID:  rec.UploadID,
		Bucket:    rec.Bucket,
		Key:       rec.Key,
		CreatedAt: rec.CreatedAt,
		Parts:     parts,
	}, true
}

// CompleteUpload verifies parts and combines them into final object chunk list.
func (m *MultipartStore) CompleteUpload(uploadID string, requestedParts []struct {
	PartNumber int
	ETag       string
}) (combinedChunkIDs []string, combinedChunkLocations map[string][]string, totalSize int64, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	rec, exists := m.uploads[uploadID]
	if !exists {
		return nil, nil, 0, fmt.Errorf("NoSuchUpload")
	}

	// Sort requested parts by PartNumber
	parts := append([]struct {
		PartNumber int
		ETag       string
	}(nil), requestedParts...)
	sort.Slice(parts, func(i, j int) bool {
		return parts[i].PartNumber < parts[j].PartNumber
	})

	combinedChunkLocations = make(map[string][]string)

	for _, reqPart := range parts {
		storedPart, found := rec.Parts[reqPart.PartNumber]
		if !found {
			return nil, nil, 0, fmt.Errorf("InvalidPart: part %d not found", reqPart.PartNumber)
		}

		combinedChunkIDs = append(combinedChunkIDs, storedPart.ChunkIDs...)
		totalSize += storedPart.Size

		for chunkID, locs := range storedPart.ChunkLocations {
			combinedChunkLocations[chunkID] = append([]string(nil), locs...)
		}
	}

	delete(m.uploads, uploadID)
	return combinedChunkIDs, combinedChunkLocations, totalSize, nil
}

// AbortUpload cancels a multipart upload and returns all chunk IDs to be cleaned up.
func (m *MultipartStore) AbortUpload(uploadID string) ([]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	rec, exists := m.uploads[uploadID]
	if !exists {
		return nil, fmt.Errorf("NoSuchUpload")
	}

	var allChunks []string
	for _, p := range rec.Parts {
		allChunks = append(allChunks, p.ChunkIDs...)
	}

	delete(m.uploads, uploadID)
	return allChunks, nil
}

// GetActiveChunkIDs returns all chunk IDs held by active, uncommitted multipart uploads.
func (m *MultipartStore) GetActiveChunkIDs() []string {
	if m == nil {
		return nil
	}
	m.mu.RLock()
	defer m.mu.RUnlock()

	var chunks []string
	for _, rec := range m.uploads {
		for _, p := range rec.Parts {
			chunks = append(chunks, p.ChunkIDs...)
		}
	}
	return chunks
}

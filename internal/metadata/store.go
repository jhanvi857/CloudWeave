package metadata

import (
	"fmt"
	"sync"
)

// Store provides thread-safe access to file manifests with optional WAL persistence.
type Store struct {
	mu        sync.RWMutex
	manifests map[string]Manifest
	wal       *WAL
}

// NewStore initializes a new MetadataStore.
func NewStore() *Store {
	return &Store{
		manifests: make(map[string]Manifest),
	}
}

func (s *Store) SetWAL(w *WAL) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.wal = w
}

// RecordPlacement saves or updates a file manifest.
func (s *Store) RecordPlacement(manifest Manifest) error {
	if manifest.FileID == "" {
		return fmt.Errorf("fileID cannot be empty")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.manifests[manifest.FileID] = manifest.Clone()

	if s.wal != nil {
		_ = s.wal.WriteRecord(WALRecord{
			Op:       OpRecordManifest,
			Manifest: manifest,
		})
	}
	return nil
}

// Lookup retrieves a file manifest by fileID.
func (s *Store) Lookup(fileID string) (Manifest, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	m, exists := s.manifests[fileID]
	if !exists {
		return Manifest{}, false
	}
	return m.Clone(), true
}

// Delete removes a file manifest from the store.
func (s *Store) Delete(fileID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.manifests[fileID]; !exists {
		return false
	}
	delete(s.manifests, fileID)

	if s.wal != nil {
		_ = s.wal.WriteRecord(WALRecord{
			Op:     OpDeleteManifest,
			FileID: fileID,
		})
	}
	return true
}

// GetAllManifests returns a snapshot of all manifests.
func (s *Store) GetAllManifests() map[string]Manifest {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]Manifest, len(s.manifests))
	for k, v := range s.manifests {
		result[k] = v.Clone()
	}
	return result
}

// UpdateChunkLocations updates the node locations for a specific chunk across all manifests containing it.
func (s *Store) UpdateChunkLocations(chunkID string, newLocations []string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for fileID, m := range s.manifests {
		if _, found := m.ChunkLocations[chunkID]; found {
			locs := make([]string, len(newLocations))
			copy(locs, newLocations)
			m.ChunkLocations[chunkID] = locs
			s.manifests[fileID] = m
		}
	}

	if s.wal != nil {
		_ = s.wal.WriteRecord(WALRecord{
			Op:        OpUpdateLocations,
			ChunkID:   chunkID,
			Locations: newLocations,
		})
	}
}

package metadata

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// ScopedKey builds a canonical manifest key scoped by namespace.
func ScopedKey(ns, fileID string) string {
	if ns == "" {
		ns = "default"
	}
	if strings.HasPrefix(fileID, ns+"/") {
		return fileID
	}
	return ns + "/" + fileID
}

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

// RecordPlacement saves or updates a file manifest, archiving previous versions.
func (s *Store) RecordPlacement(manifest Manifest) error {
	if manifest.FileID == "" {
		return fmt.Errorf("fileID cannot be empty")
	}
	if manifest.Namespace == "" {
		manifest.Namespace = "default"
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	key := ScopedKey(manifest.Namespace, manifest.FileID)

	if manifest.VersionID == "" {
		manifest.VersionID = fmt.Sprintf("v%d", time.Now().UnixNano())
	}

	if existing, exists := s.manifests[key]; exists {
		oldVersion := existing.Clone()
		history := append([]Manifest(nil), existing.Versions...)
		oldVersion.Versions = nil

		seenVerIDs := make(map[string]bool)
		for _, h := range history {
			seenVerIDs[h.VersionID] = true
		}
		if !seenVerIDs[oldVersion.VersionID] && oldVersion.VersionID != manifest.VersionID {
			history = append(history, oldVersion)
			seenVerIDs[oldVersion.VersionID] = true
		}
		for _, v := range manifest.Versions {
			if !seenVerIDs[v.VersionID] && v.VersionID != manifest.VersionID {
				history = append(history, v)
				seenVerIDs[v.VersionID] = true
			}
		}
		manifest.Versions = history
	}

	s.manifests[key] = manifest.Clone()

	if s.wal != nil {
		_ = s.wal.WriteRecord(WALRecord{
			Op:       OpRecordManifest,
			Manifest: manifest,
		})
	}
	return nil
}

// Lookup retrieves a file manifest by fileID, checking exact key and default namespace.
func (s *Store) Lookup(fileID string) (Manifest, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if m, exists := s.manifests[fileID]; exists {
		return m.Clone(), true
	}
	if m, exists := s.manifests["default/"+fileID]; exists {
		return m.Clone(), true
	}
	return Manifest{}, false
}

// LookupScoped retrieves a file manifest by namespace and fileID.
func (s *Store) LookupScoped(ns, fileID string) (Manifest, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := ScopedKey(ns, fileID)
	m, exists := s.manifests[key]
	if !exists {
		return Manifest{}, false
	}
	return m.Clone(), true
}

// ListVersions retrieves all historical manifest versions for a file key.
func (s *Store) ListVersions(ns, fileID string) ([]Manifest, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := ScopedKey(ns, fileID)
	m, exists := s.manifests[key]
	if !exists {
		key = fileID
		m, exists = s.manifests[key]
	}
	if !exists {
		return nil, false
	}

	all := make([]Manifest, 0, len(m.Versions)+1)
	for _, hist := range m.Versions {
		all = append(all, hist.Clone())
	}
	all = append(all, m.Clone())
	return all, true
}

// LookupVersion retrieves a specific file version by versionID.
func (s *Store) LookupVersion(ns, fileID, versionID string) (Manifest, bool) {
	versions, found := s.ListVersions(ns, fileID)
	if !found {
		return Manifest{}, false
	}
	for _, v := range versions {
		if v.VersionID == versionID {
			return v, true
		}
	}
	return Manifest{}, false
}

// Delete removes a file manifest from the store.
func (s *Store) Delete(fileID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	keyToDelete := fileID
	if _, exists := s.manifests[keyToDelete]; !exists {
		if _, exists := s.manifests["default/"+fileID]; exists {
			keyToDelete = "default/" + fileID
		} else {
			return false
		}
	}

	delete(s.manifests, keyToDelete)

	if s.wal != nil {
		_ = s.wal.WriteRecord(WALRecord{
			Op:     OpDeleteManifest,
			FileID: keyToDelete,
		})
	}
	return true
}

// DeleteScoped removes a file manifest for a specific namespace.
func (s *Store) DeleteScoped(ns, fileID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := ScopedKey(ns, fileID)
	if _, exists := s.manifests[key]; !exists {
		return false
	}
	delete(s.manifests, key)

	if s.wal != nil {
		_ = s.wal.WriteRecord(WALRecord{
			Op:     OpDeleteManifest,
			FileID: key,
		})
	}
	return true
}

// GetAllManifests returns a snapshot of all manifests across all namespaces.
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

	for key, m := range s.manifests {
		if _, found := m.ChunkLocations[chunkID]; found {
			locs := make([]string, len(newLocations))
			copy(locs, newLocations)
			m.ChunkLocations[chunkID] = locs
			s.manifests[key] = m
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

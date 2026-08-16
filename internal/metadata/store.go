package metadata

import (
	"fmt"
	"sort"
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

// BucketInfo represents metadata for an S3 bucket or namespace.
type BucketInfo struct {
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

// Store provides thread-safe access to file manifests with optional WAL persistence.
type Store struct {
	mu        sync.RWMutex
	manifests map[string]Manifest
	buckets   map[string]BucketInfo
	wal       *WAL
}

// NewStore initializes a new MetadataStore.
func NewStore() *Store {
	return &Store{
		manifests: make(map[string]Manifest),
		buckets: map[string]BucketInfo{
			"default": {Name: "default", CreatedAt: time.Now().UTC()},
		},
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

	if s.buckets == nil {
		s.buckets = make(map[string]BucketInfo)
	}
	if _, exists := s.buckets[manifest.Namespace]; !exists {
		s.buckets[manifest.Namespace] = BucketInfo{
			Name:      manifest.Namespace,
			CreatedAt: time.Now().UTC(),
		}
	}

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

// CreateBucket creates a new S3 bucket / namespace in metadata.
func (s *Store) CreateBucket(name string) error {
	if name == "" {
		return fmt.Errorf("bucket name cannot be empty")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.buckets == nil {
		s.buckets = make(map[string]BucketInfo)
	}

	if _, exists := s.buckets[name]; exists {
		return nil // idempotent create
	}

	s.buckets[name] = BucketInfo{
		Name:      name,
		CreatedAt: time.Now().UTC(),
	}

	if s.wal != nil {
		_ = s.wal.WriteRecord(WALRecord{
			Op:         OpCreateBucket,
			BucketName: name,
		})
	}
	return nil
}

// DeleteBucket removes an empty bucket.
func (s *Store) DeleteBucket(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.buckets[name]; !exists {
		return fmt.Errorf("no such bucket: %s", name)
	}

	// Verify bucket is empty
	for _, m := range s.manifests {
		if m.Namespace == name {
			return fmt.Errorf("bucket not empty")
		}
	}

	delete(s.buckets, name)

	if s.wal != nil {
		_ = s.wal.WriteRecord(WALRecord{
			Op:         OpDeleteBucket,
			BucketName: name,
		})
	}
	return nil
}

// BucketExists checks if a bucket exists or has stored objects.
func (s *Store) BucketExists(name string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if name == "" || name == "default" {
		return true
	}

	if _, exists := s.buckets[name]; exists {
		return true
	}

	// Implicit bucket check from stored manifests
	for _, m := range s.manifests {
		if m.Namespace == name {
			return true
		}
	}
	return false
}

// ListBuckets returns a snapshot of all registered buckets.
func (s *Store) ListBuckets() []BucketInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	bucketMap := make(map[string]BucketInfo)
	for k, v := range s.buckets {
		bucketMap[k] = v
	}

	// Include implicit buckets from manifests
	for _, m := range s.manifests {
		ns := m.Namespace
		if ns == "" {
			ns = "default"
		}
		if _, exists := bucketMap[ns]; !exists {
			bucketMap[ns] = BucketInfo{
				Name:      ns,
				CreatedAt: time.Now().UTC(),
			}
		}
	}

	result := make([]BucketInfo, 0, len(bucketMap))
	for _, b := range bucketMap {
		result = append(result, b)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})

	return result
}

// ListObjectsV2 returns filtered & paginated manifests for a bucket/namespace with optional S3 prefix & delimiter.
func (s *Store) ListObjectsV2(ns, prefix, delimiter, startAfter string, maxKeys int) (contents []Manifest, commonPrefixes []string, isTruncated bool, nextContinuationToken string) {
	if ns == "" {
		ns = "default"
	}
	if maxKeys <= 0 {
		maxKeys = 1000
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	var matching []Manifest
	for _, m := range s.manifests {
		if m.Namespace != ns && !(ns == "default" && m.Namespace == "") {
			continue
		}

		fileID := m.FileID
		if prefix != "" && !strings.HasPrefix(fileID, prefix) {
			continue
		}

		if startAfter != "" && fileID <= startAfter {
			continue
		}

		matching = append(matching, m.Clone())
	}

	sort.Slice(matching, func(i, j int) bool {
		return matching[i].FileID < matching[j].FileID
	})

	commonPrefixSet := make(map[string]bool)
	var filteredContents []Manifest

	for _, m := range matching {
		fileID := m.FileID
		if delimiter != "" {
			rel := fileID
			if prefix != "" {
				rel = strings.TrimPrefix(fileID, prefix)
			}
			idx := strings.Index(rel, delimiter)
			if idx >= 0 {
				cp := prefix + rel[:idx+len(delimiter)]
				commonPrefixSet[cp] = true
				continue
			}
		}
		filteredContents = append(filteredContents, m)
	}

	for cp := range commonPrefixSet {
		commonPrefixes = append(commonPrefixes, cp)
	}
	sort.Strings(commonPrefixes)

	if len(filteredContents) > maxKeys {
		isTruncated = true
		contents = filteredContents[:maxKeys]
		nextContinuationToken = contents[len(contents)-1].FileID
	} else {
		contents = filteredContents
	}

	return contents, commonPrefixes, isTruncated, nextContinuationToken
}

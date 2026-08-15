package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type DiskStore struct {
	baseDir string
	mu      sync.RWMutex
	cache   *ChunkCache
}

func NewDiskStore(baseDir string) (*DiskStore, error) {
	if baseDir == "" {
		return nil, fmt.Errorf("storage directory path cannot be empty")
	}
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return nil, fmt.Errorf("creating storage dir %q: %w", baseDir, err)
	}
	return &DiskStore{
		baseDir: baseDir,
		cache:   NewChunkCache(64 * 1024 * 1024), // 64MB default LRU cache
	}, nil
}

func (s *DiskStore) SetCacheSize(maxBytes int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cache = NewChunkCache(maxBytes)
}

func (s *DiskStore) GetCache() *ChunkCache {
	return s.cache
}

func (s *DiskStore) Put(chunkID string, data []byte) error {
	path := s.pathFor(chunkID)

	if s.cache != nil {
		if _, ok := s.cache.Get(chunkID); ok {
			return nil
		}
	}
	if s.Exists(chunkID) {
		if s.cache != nil {
			s.cache.Put(chunkID, data)
		}
		return nil
	}

	tmpPath := filepath.Join(s.baseDir, fmt.Sprintf(".%s-%d.tmp", chunkID, time.Now().UnixNano()))
	f, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("creating temp chunk file for %s: %w", chunkID, err)
	}

	if _, err := f.Write(data); err != nil {
		f.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("writing temp chunk %s: %w", chunkID, err)
	}

	if err := f.Sync(); err != nil {
		f.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("flushing chunk %s to disk: %w", chunkID, err)
	}

	if err := f.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("closing temp chunk %s: %w", chunkID, err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		if s.Exists(chunkID) {
			if s.cache != nil {
				s.cache.Put(chunkID, data)
			}
			return nil
		}
		return fmt.Errorf("renaming temp chunk %s: %w", chunkID, err)
	}

	if s.cache != nil {
		s.cache.Put(chunkID, data)
	}
	return nil
}

func (s *DiskStore) Get(chunkID string) ([]byte, error) {
	if data, ok := s.cache.Get(chunkID); ok {
		return data, nil
	}

	s.mu.RLock()
	data, err := os.ReadFile(s.pathFor(chunkID))
	s.mu.RUnlock()

	if err != nil {
		return nil, fmt.Errorf("reading chunk %s: %w", chunkID, err)
	}

	if s.cache != nil {
		s.cache.Put(chunkID, data)
	}
	return data, nil
}

func (s *DiskStore) Exists(chunkID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.ExistsUnlocked(chunkID)
}

func (s *DiskStore) ExistsUnlocked(chunkID string) bool {
	_, err := os.Stat(s.pathFor(chunkID))
	return err == nil
}

func (s *DiskStore) Delete(chunkID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.cache != nil {
		s.cache.Remove(chunkID)
	}

	path := s.pathFor(chunkID)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("deleting chunk %s: %w", chunkID, err)
	}
	return nil
}

func (s *DiskStore) ListChunks() ([]string, error) {
	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		return nil, fmt.Errorf("listing storage dir %s: %w", s.baseDir, err)
	}

	var chunkIDs []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if filepath.Ext(name) == ".wal" || filepath.Ext(name) == ".tmp" || (len(name) > 0 && name[0] == '.') {
			continue
		}
		chunkIDs = append(chunkIDs, name)
	}
	return chunkIDs, nil
}

func (s *DiskStore) pathFor(chunkID string) string {
	return filepath.Join(s.baseDir, chunkID)
}

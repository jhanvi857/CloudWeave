package storage

import (
	"fmt"
	"os"
	"path/filepath"
)

type DiskStore struct {
	baseDir string
}

func NewDiskStore(baseDir string) (*DiskStore, error) {
	if baseDir == "" {
		return nil, fmt.Errorf("storage directory path cannot be empty")
	}
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return nil, fmt.Errorf("creating storage dir %q: %w", baseDir, err)
	}
	return &DiskStore{baseDir: baseDir}, nil
}

func (s *DiskStore) Put(chunkID string, data []byte) error {
	path := s.pathFor(chunkID)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("writing chunk %s: %w", chunkID, err)
	}
	return os.Rename(tmp, path)
}

func (s *DiskStore) Get(chunkID string) ([]byte, error) {
	data, err := os.ReadFile(s.pathFor(chunkID))
	if err != nil {
		return nil, fmt.Errorf("reading chunk %s: %w", chunkID, err)
	}
	return data, nil
}

func (s *DiskStore) Exists(chunkID string) bool {
	_, err := os.Stat(s.pathFor(chunkID))
	return err == nil
}

func (s *DiskStore) pathFor(chunkID string) string {
	return filepath.Join(s.baseDir, chunkID)
}

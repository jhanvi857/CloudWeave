package metadata

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
)

type WALOpType string

const (
	OpRecordManifest   WALOpType = "RECORD_MANIFEST"
	OpUpdateLocations  WALOpType = "UPDATE_LOCATIONS"
	OpDeleteManifest   WALOpType = "DELETE_MANIFEST"
)

type WALRecord struct {
	Op       WALOpType `json:"op"`
	Manifest Manifest  `json:"manifest,omitempty"`
	ChunkID  string    `json:"chunk_id,omitempty"`
	Locations []string `json:"locations,omitempty"`
	FileID   string    `json:"file_id,omitempty"`
}

type WAL struct {
	mu   sync.Mutex
	file *os.File
	path string
}

func OpenWAL(walPath string) (*WAL, error) {
	dir := filepath.Dir(walPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("creating WAL directory: %w", err)
	}

	f, err := os.OpenFile(walPath, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0o644)
	if err != nil {
		return nil, fmt.Errorf("opening WAL file %s: %w", walPath, err)
	}

	return &WAL{
		file: f,
		path: walPath,
	}, nil
}

func (w *WAL) WriteRecord(rec WALRecord) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	data, err := json.Marshal(rec)
	if err != nil {
		return fmt.Errorf("marshaling WAL record: %w", err)
	}

	data = append(data, '\n')
	if _, err := w.file.Write(data); err != nil {
		return fmt.Errorf("writing to WAL file: %w", err)
	}

	return w.file.Sync()
}

func (w *WAL) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.file.Close()
}

func ReplayWAL(walPath string, store *Store) error {
	f, err := os.Open(walPath)
	if os.IsNotExist(err) {
		return nil // No WAL file exists yet
	}
	if err != nil {
		return fmt.Errorf("opening WAL for replay: %w", err)
	}
	defer f.Close()

	dec := json.NewDecoder(f)
	for {
		var rec WALRecord
		if err := dec.Decode(&rec); err == io.EOF {
			break
		} else if err != nil {
			return fmt.Errorf("decoding WAL line: %w", err)
		}

		switch rec.Op {
		case OpRecordManifest:
			_ = store.RecordPlacement(rec.Manifest)
		case OpUpdateLocations:
			store.UpdateChunkLocations(rec.ChunkID, rec.Locations)
		case OpDeleteManifest:
			store.Delete(rec.FileID)
		}
	}

	return nil
}

package metadata

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"

	"cloudWeave/internal/auth"
)

type WALOpType string

const (
	OpRecordManifest  WALOpType = "RECORD_MANIFEST"
	OpUpdateLocations WALOpType = "UPDATE_LOCATIONS"
	OpDeleteManifest  WALOpType = "DELETE_MANIFEST"
	OpRecordKey       WALOpType = "RECORD_KEY"
	OpDeleteKey       WALOpType = "DELETE_KEY"
	OpCreateBucket    WALOpType = "CREATE_BUCKET"
	OpDeleteBucket    WALOpType = "DELETE_BUCKET"
)

type WALRecord struct {
	Op         WALOpType       `json:"op"`
	Manifest   Manifest        `json:"manifest,omitempty"`
	ChunkID    string          `json:"chunk_id,omitempty"`
	Locations  []string        `json:"locations,omitempty"`
	FileID     string          `json:"file_id,omitempty"`
	Credential auth.Credential `json:"credential,omitempty"`
	BucketName string          `json:"bucket_name,omitempty"`
	CRC32      uint32          `json:"crc32,omitempty"`
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

	// Compute CRC32 checksum over the record fields (with CRC32 set to 0)
	rec.CRC32 = 0
	raw, err := json.Marshal(rec)
	if err != nil {
		return fmt.Errorf("marshaling WAL record: %w", err)
	}
	rec.CRC32 = crc32.ChecksumIEEE(raw)

	data, err := json.Marshal(rec)
	if err != nil {
		return fmt.Errorf("marshaling WAL record with crc32: %w", err)
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

// ReplayWAL reads and executes all committed records in the WAL.
// If a crash left a torn write at the tail of the WAL file, it gracefully truncates
// the torn bytes and recovers all prior valid committed records without failing.
func ReplayWAL(walPath string, store *Store, authOpts ...*auth.Authenticator) error {
	var authenticator *auth.Authenticator
	if len(authOpts) > 0 && authOpts[0] != nil {
		authenticator = authOpts[0]
	}

	f, err := os.OpenFile(walPath, os.O_RDWR, 0o644)
	if os.IsNotExist(err) {
		return nil // No WAL file exists yet
	}
	if err != nil {
		return fmt.Errorf("opening WAL for replay: %w", err)
	}
	defer f.Close()

	reader := bufio.NewReader(f)
	var lastCleanOffset int64
	replayedCount := 0

	for {
		lineStartOffset := lastCleanOffset
		line, readErr := reader.ReadBytes('\n')

		if len(line) > 0 {
			trimmed := bytes.TrimSpace(line)
			if len(trimmed) > 0 {
				var rec WALRecord
				if err := json.Unmarshal(trimmed, &rec); err != nil {
					// Invalid JSON: check if this is a torn tail at EOF
					if readErr == io.EOF || readErr == nil {
						log.Printf("[WAL] Detected torn write at tail of %s (offset %d); truncating and recovering %d valid records", walPath, lineStartOffset, replayedCount)
						_ = f.Truncate(lineStartOffset)
						break
					}
					return fmt.Errorf("decoding WAL line at offset %d: %w", lineStartOffset, err)
				}

				// If record contains CRC32, verify payload integrity
				if rec.CRC32 != 0 {
					storedCRC := rec.CRC32
					rec.CRC32 = 0
					raw, _ := json.Marshal(rec)
					calculatedCRC := crc32.ChecksumIEEE(raw)
					if storedCRC != calculatedCRC {
						if readErr == io.EOF {
							log.Printf("[WAL] Checksum mismatch on incomplete tail record in %s; truncating at offset %d", walPath, lineStartOffset)
							_ = f.Truncate(lineStartOffset)
							break
						}
						return fmt.Errorf("WAL record CRC32 mismatch at offset %d (stored: %d, calculated: %d)", lineStartOffset, storedCRC, calculatedCRC)
					}
					rec.CRC32 = storedCRC
				}

				applyRecord(rec, store, authenticator)
				replayedCount++
			}
			lastCleanOffset += int64(len(line))
		}

		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return fmt.Errorf("reading WAL stream: %w", readErr)
		}
	}

	return nil
}

func applyRecord(rec WALRecord, store *Store, authenticator *auth.Authenticator) {
	switch rec.Op {
	case OpRecordManifest:
		_ = store.RecordPlacement(rec.Manifest)
	case OpUpdateLocations:
		store.UpdateChunkLocations(rec.ChunkID, rec.Locations)
	case OpDeleteManifest:
		store.Delete(rec.FileID)
	case OpRecordKey:
		if authenticator != nil {
			authenticator.AddCredentialByHash(rec.Credential)
		}
	case OpDeleteKey:
		if authenticator != nil {
			authenticator.RevokeCredentialByHash(rec.Credential.KeyHash)
		}
	case OpCreateBucket:
		_ = store.CreateBucket(rec.BucketName)
	case OpDeleteBucket:
		_ = store.DeleteBucket(rec.BucketName)
	}
}

// CompactWAL snapshots active manifests and credentials to a clean new WAL file atomically.
func CompactWAL(walPath string, store *Store, authenticator *auth.Authenticator) error {
	tmpPath := walPath + ".compact.tmp"
	tmpWAL, err := OpenWAL(tmpPath)
	if err != nil {
		return fmt.Errorf("creating compact temp WAL: %w", err)
	}

	// 1. Snapshot active manifests
	if store != nil {
		allManifests := store.GetAllManifests()
		for _, m := range allManifests {
			if err := tmpWAL.WriteRecord(WALRecord{
				Op:       OpRecordManifest,
				Manifest: m,
			}); err != nil {
				tmpWAL.Close()
				_ = os.Remove(tmpPath)
				return fmt.Errorf("compacting manifest %s: %w", m.FileID, err)
			}
		}
	}

	// 2. Snapshot active credentials
	if authenticator != nil {
		creds := authenticator.GetAllCredentials()
		for _, c := range creds {
			if err := tmpWAL.WriteRecord(WALRecord{
				Op:         OpRecordKey,
				Credential: c,
			}); err != nil {
				tmpWAL.Close()
				_ = os.Remove(tmpPath)
				return fmt.Errorf("compacting credential: %w", err)
			}
		}
	}

	if err := tmpWAL.Close(); err != nil {
		return fmt.Errorf("closing compact temp WAL: %w", err)
	}

	return os.Rename(tmpPath, walPath)
}

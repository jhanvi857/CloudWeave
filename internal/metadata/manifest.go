package metadata

import (
	"cloudWeave/internal/vectorclock"
)

// Manifest defines the metadata record for a stored file.
type Manifest struct {
	FileID         string                   `json:"file_id"`
	Size           int64                    `json:"size"`
	ChunkIDs       []string                 `json:"chunk_ids"`
	ChunkLocations map[string][]string      `json:"chunk_locations"` // chunkID -> slice of node addresses
	Version        *vectorclock.VectorClock `json:"version,omitempty"`
}

// Clone creates a deep copy of the Manifest to prevent race conditions.
func (m Manifest) Clone() Manifest {
	cp := Manifest{
		FileID:         m.FileID,
		Size:           m.Size,
		ChunkIDs:       append([]string(nil), m.ChunkIDs...),
		ChunkLocations: make(map[string][]string, len(m.ChunkLocations)),
	}
	if m.Version != nil {
		cp.Version = m.Version.Clone()
	}
	for k, v := range m.ChunkLocations {
		cp.ChunkLocations[k] = append([]string(nil), v...)
	}
	return cp
}

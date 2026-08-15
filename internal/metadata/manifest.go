package metadata

import (
	"cloudWeave/internal/vectorclock"
)

// Manifest defines the metadata record for a stored file.
type Manifest struct {
	Namespace      string                   `json:"namespace,omitempty"`
	FileID         string                   `json:"file_id"`
	Size           int64                    `json:"size"`
	ChunkIDs       []string                 `json:"chunk_ids"`
	ChunkLocations map[string][]string      `json:"chunk_locations"` // chunkID -> slice of node addresses
	Version        *vectorclock.VectorClock `json:"version,omitempty"`
	VersionID      string                   `json:"version_id,omitempty"`
	Versions       []Manifest               `json:"versions,omitempty"` // Historical versions archive
	ContentType    string                   `json:"content_type,omitempty"`
	Metadata       map[string]string        `json:"metadata,omitempty"`
}

// Clone creates a deep copy of the Manifest to prevent race conditions.
func (m Manifest) Clone() Manifest {
	cp := Manifest{
		Namespace:      m.Namespace,
		FileID:         m.FileID,
		Size:           m.Size,
		ChunkIDs:       append([]string(nil), m.ChunkIDs...),
		ChunkLocations: make(map[string][]string, len(m.ChunkLocations)),
		VersionID:      m.VersionID,
		ContentType:    m.ContentType,
	}
	if m.Version != nil {
		cp.Version = m.Version.Clone()
	}
	for k, v := range m.ChunkLocations {
		cp.ChunkLocations[k] = append([]string(nil), v...)
	}
	if m.Metadata != nil {
		cp.Metadata = make(map[string]string, len(m.Metadata))
		for k, v := range m.Metadata {
			cp.Metadata[k] = v
		}
	}
	if len(m.Versions) > 0 {
		cp.Versions = make([]Manifest, len(m.Versions))
		for i, v := range m.Versions {
			cp.Versions[i] = v.Clone()
		}
	}
	return cp
}

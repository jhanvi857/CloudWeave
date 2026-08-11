package api

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"cloudWeave/internal/chunk"
	"cloudWeave/internal/metadata"
	"cloudWeave/internal/metrics"
)

const DefaultChunkSize = 1024 * 1024 // 1 MB default chunk size

// ChunkStorageEngine interface abstracts storing/fetching individual chunks locally or across quorum.
type ChunkStorageEngine interface {
	PutChunk(chunkID string, data []byte) ([]string, error) // returns node locations where chunk was stored
	GetChunk(chunkID string, locations []string) ([]byte, error)
}

// LocalStorageAdapter adapts DiskStore to ChunkStorageEngine interface.
type LocalStorageAdapter struct {
	PutFunc func(chunkID string, data []byte) error
	GetFunc func(chunkID string) ([]byte, error)
	NodeID  string
}

func (l *LocalStorageAdapter) PutChunk(chunkID string, data []byte) ([]string, error) {
	if err := l.PutFunc(chunkID, data); err != nil {
		return nil, err
	}
	return []string{l.NodeID}, nil
}

func (l *LocalStorageAdapter) GetChunk(chunkID string, locations []string) ([]byte, error) {
	return l.GetFunc(chunkID)
}

type APIHandler struct {
	metaStore *metadata.Store
	engine    ChunkStorageEngine
	chunkSize int
}

func NewAPIHandler(metaStore *metadata.Store, engine ChunkStorageEngine, chunkSize int) *APIHandler {
	if chunkSize <= 0 {
		chunkSize = DefaultChunkSize
	}
	return &APIHandler{
		metaStore: metaStore,
		engine:    engine,
		chunkSize: chunkSize,
	}
}

func (a *APIHandler) HandleFiles(w http.ResponseWriter, r *http.Request) {
	fileID := strings.TrimPrefix(r.URL.Path, "/files/")
	fileID = strings.TrimPrefix(fileID, "/")
	if fileID == "" {
		http.Error(w, "missing file id", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodPut, http.MethodPost:
		a.handlePutFile(w, r, fileID)
	case http.MethodGet:
		a.handleGetFile(w, r, fileID)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (a *APIHandler) handlePutFile(w http.ResponseWriter, r *http.Request, fileID string) {
	data, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "reading request body failed", http.StatusBadRequest)
		return
	}

	if len(data) == 0 {
		http.Error(w, "cannot store empty file", http.StatusBadRequest)
		return
	}

	chunks, err := chunk.Split(data, a.chunkSize)
	if err != nil {
		http.Error(w, fmt.Sprintf("chunking failed: %v", err), http.StatusInternalServerError)
		return
	}

	var chunkIDs []string
	chunkLocations := make(map[string][]string)

	for _, c := range chunks {
		locs, err := a.engine.PutChunk(c.ID, c.Data)
		if err != nil {
			log.Printf("failed to store chunk %s: %v", c.ID, err)
			http.Error(w, fmt.Sprintf("failed to store chunk %s", c.ID), http.StatusInternalServerError)
			return
		}
		chunkIDs = append(chunkIDs, c.ID)
		chunkLocations[c.ID] = locs
	}

	manifest := metadata.Manifest{
		FileID:         fileID,
		Size:           int64(len(data)),
		ChunkIDs:       chunkIDs,
		ChunkLocations: chunkLocations,
	}

	if err := a.metaStore.RecordPlacement(manifest); err != nil {
		log.Printf("failed to record manifest: %v", err)
		http.Error(w, "failed to record metadata", http.StatusInternalServerError)
		return
	}

	metrics.IncFileUploads()

	w.WriteHeader(http.StatusCreated)
	fmt.Fprintf(w, "File %s uploaded successfully (%d bytes, %d chunks)\n", fileID, len(data), len(chunks))
}

func (a *APIHandler) handleGetFile(w http.ResponseWriter, r *http.Request, fileID string) {
	manifest, found := a.metaStore.Lookup(fileID)
	if !found {
		http.Error(w, "file not found", http.StatusNotFound)
		return
	}

	metrics.IncFileDownloads()

	var fetchedChunks []chunk.Chunk
	for idx, chunkID := range manifest.ChunkIDs {
		locs := manifest.ChunkLocations[chunkID]
		data, err := a.engine.GetChunk(chunkID, locs)
		if err != nil {
			log.Printf("failed to retrieve chunk %s: %v", chunkID, err)
			http.Error(w, fmt.Sprintf("failed to retrieve chunk %s", chunkID), http.StatusInternalServerError)
			return
		}
		fetchedChunks = append(fetchedChunks, chunk.Chunk{
			ID:    chunkID,
			Data:  data,
			Index: idx,
		})
	}

	assembled, err := chunk.Reassemble(fetchedChunks)
	if err != nil {
		log.Printf("reassembly failed for file %s: %v", fileID, err)
		http.Error(w, "file corrupted or reassembly failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.WriteHeader(http.StatusOK)
	w.Write(assembled)
}

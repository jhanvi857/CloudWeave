package api

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
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
	case http.MethodDelete:
		a.handleDeleteFile(w, r, fileID)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (a *APIHandler) handlePutFile(w http.ResponseWriter, r *http.Request, fileID string) {
	if r.Body == nil {
		http.Error(w, "missing request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var chunkIDs []string
	chunkLocations := make(map[string][]string)

	totalBytes, chunkIDs, err := chunk.SplitStream(r.Body, a.chunkSize, func(c chunk.Chunk) error {
		locs, err := a.engine.PutChunk(c.ID, c.Data)
		if err != nil {
			log.Printf("failed to store chunk %s: %v", c.ID, err)
			return err
		}
		chunkLocations[c.ID] = locs
		return nil
	})

	if err != nil {
		log.Printf("streaming put file %s failed: %v", fileID, err)
		http.Error(w, fmt.Sprintf("upload failed: %v", err), http.StatusBadRequest)
		return
	}

	manifest := metadata.Manifest{
		FileID:         fileID,
		Size:           totalBytes,
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
	fmt.Fprintf(w, "File %s uploaded successfully (%d bytes, %d chunks)\n", fileID, totalBytes, len(chunkIDs))
}

func (a *APIHandler) handleDeleteFile(w http.ResponseWriter, r *http.Request, fileID string) {
	_, found := a.metaStore.Lookup(fileID)
	if !found {
		http.Error(w, "file not found", http.StatusNotFound)
		return
	}

	if deleted := a.metaStore.Delete(fileID); !deleted {
		http.Error(w, "failed to delete metadata", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "File %s deleted successfully\n", fileID)
}

func (a *APIHandler) handleGetFile(w http.ResponseWriter, r *http.Request, fileID string) {
	manifest, found := a.metaStore.Lookup(fileID)
	if !found {
		http.Error(w, "file not found", http.StatusNotFound)
		return
	}

	metrics.IncFileDownloads()

	rangeHeader := r.Header.Get("Range")
	if rangeHeader != "" {
		a.handleRangeGetFile(w, r, manifest, rangeHeader)
		return
	}

	// Full file streaming GET response
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", strconv.FormatInt(manifest.Size, 10))
	w.Header().Set("Accept-Ranges", "bytes")
	w.WriteHeader(http.StatusOK)

	flusher, _ := w.(http.Flusher)

	for idx, chunkID := range manifest.ChunkIDs {
		locs := manifest.ChunkLocations[chunkID]
		data, err := a.engine.GetChunk(chunkID, locs)
		if err != nil {
			log.Printf("failed to retrieve chunk %s (index %d): %v", chunkID, idx, err)
			return
		}
		if _, writeErr := w.Write(data); writeErr != nil {
			log.Printf("client disconnected during streaming read of file %s: %v", fileID, writeErr)
			return
		}
		if flusher != nil {
			flusher.Flush()
		}
	}
}

func (a *APIHandler) handleRangeGetFile(w http.ResponseWriter, r *http.Request, manifest metadata.Manifest, rangeHeader string) {
	start, end, err := parseRangeHeader(rangeHeader, manifest.Size)
	if err != nil {
		w.Header().Set("Content-Range", fmt.Sprintf("bytes */%d", manifest.Size))
		http.Error(w, "invalid range", http.StatusRequestedRangeNotSatisfiable)
		return
	}

	contentLength := end - start + 1
	startChunkIdx := int(start / int64(a.chunkSize))
	endChunkIdx := int(end / int64(a.chunkSize))

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Accept-Ranges", "bytes")
	w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, manifest.Size))
	w.Header().Set("Content-Length", strconv.FormatInt(contentLength, 10))
	w.WriteHeader(http.StatusPartialContent)

	flusher, _ := w.(http.Flusher)

	for chunkIdx := startChunkIdx; chunkIdx <= endChunkIdx && chunkIdx < len(manifest.ChunkIDs); chunkIdx++ {
		chunkID := manifest.ChunkIDs[chunkIdx]
		locs := manifest.ChunkLocations[chunkID]

		data, err := a.engine.GetChunk(chunkID, locs)
		if err != nil {
			log.Printf("failed to retrieve chunk %s for range request: %v", chunkID, err)
			return
		}

		chunkStart := int64(chunkIdx) * int64(a.chunkSize)
		chunkEnd := chunkStart + int64(len(data)) - 1

		sliceStart := int64(0)
		if start > chunkStart {
			sliceStart = start - chunkStart
		}

		sliceEnd := int64(len(data))
		if end < chunkEnd {
			sliceEnd = end - chunkStart + 1
		}

		if sliceStart < 0 || sliceStart > sliceEnd || sliceEnd > int64(len(data)) {
			continue
		}

		if _, writeErr := w.Write(data[sliceStart:sliceEnd]); writeErr != nil {
			log.Printf("client disconnected during range read of file %s: %v", manifest.FileID, writeErr)
			return
		}

		if flusher != nil {
			flusher.Flush()
		}
	}
}

// parseRangeHeader parses HTTP Range header "bytes=start-end" against total file size
func parseRangeHeader(rangeHeader string, totalSize int64) (int64, int64, error) {
	if !strings.HasPrefix(rangeHeader, "bytes=") {
		return 0, 0, fmt.Errorf("invalid range unit")
	}

	spec := strings.TrimPrefix(rangeHeader, "bytes=")
	parts := strings.Split(spec, "-")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid range format")
	}

	var start, end int64
	var err error

	if parts[0] == "" && parts[1] != "" {
		// Range: bytes=-num (last num bytes)
		suffixLen, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil || suffixLen <= 0 {
			return 0, 0, fmt.Errorf("invalid range suffix")
		}
		start = totalSize - suffixLen
		if start < 0 {
			start = 0
		}
		end = totalSize - 1
	} else if parts[0] != "" && parts[1] == "" {
		// Range: bytes=start-
		start, err = strconv.ParseInt(parts[0], 10, 64)
		if err != nil || start < 0 || start >= totalSize {
			return 0, 0, fmt.Errorf("invalid range start")
		}
		end = totalSize - 1
	} else if parts[0] != "" && parts[1] != "" {
		// Range: bytes=start-end
		start, err = strconv.ParseInt(parts[0], 10, 64)
		if err != nil || start < 0 || start >= totalSize {
			return 0, 0, fmt.Errorf("invalid range start")
		}
		end, err = strconv.ParseInt(parts[1], 10, 64)
		if err != nil || end < start {
			return 0, 0, fmt.Errorf("invalid range end")
		}
		if end >= totalSize {
			end = totalSize - 1
		}
	} else {
		return 0, 0, fmt.Errorf("empty range spec")
	}

	return start, end, nil
}


package transport

import (
	"cloudWeave/internal/storage"
	"io"
	"log"
	"net/http"
	"strings"
)

// maxChunkBodySize limits PUT body to 16 MiB to prevent memory exhaustion (finding #13).
const maxChunkBodySize = 16 * 1024 * 1024

type Server struct {
	store *storage.DiskStore
}

func NewServer(store *storage.DiskStore) *Server {
	return &Server{store: store}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/chunks/", s.handleChunk)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	return mux
}

func (s *Server) handleChunk(w http.ResponseWriter, r *http.Request) {
	chunkID := strings.TrimPrefix(r.URL.Path, "/chunks/")
	if chunkID == "" {
		http.Error(w, "missing chunk id", http.StatusBadRequest)
		return
	}

	// Validate chunk ID is a safe hex string (finding #8: path traversal protection)
	if !storage.ValidateChunkID(chunkID) {
		http.Error(w, "invalid chunk id", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodPut:
		// Limit body size to prevent memory exhaustion (finding #13)
		limitedBody := io.LimitReader(r.Body, maxChunkBodySize+1)
		data, err := io.ReadAll(limitedBody)
		if err != nil {
			http.Error(w, "reading body", http.StatusBadRequest)
			return
		}
		if int64(len(data)) > maxChunkBodySize {
			http.Error(w, "chunk body too large", http.StatusRequestEntityTooLarge)
			return
		}
		if err := s.store.Put(chunkID, data); err != nil {
			log.Printf("put %s failed: %v", chunkID, err)
			http.Error(w, "storage error", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)

	case http.MethodGet:
		if !s.store.Exists(chunkID) {
			http.Error(w, "chunk not found", http.StatusNotFound)
			return
		}
		data, err := s.store.Get(chunkID)
		if err != nil {
			log.Printf("get %s failed: %v", chunkID, err)
			http.Error(w, "storage error", http.StatusInternalServerError)
			return
		}
		w.Write(data)

	case http.MethodDelete:
		if err := s.store.Delete(chunkID); err != nil {
			log.Printf("delete %s failed: %v", chunkID, err)
			http.Error(w, "storage delete error", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}


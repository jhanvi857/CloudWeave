package transport

import (
	"cloudWeave/internal/storage"
	"io"
	"log"
	"net/http"
	"strings"
)

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

	switch r.Method {
	case http.MethodPut:
		data, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "reading body", http.StatusBadRequest)
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

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

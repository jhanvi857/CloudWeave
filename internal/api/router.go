package api

import (
	"net/http"

	"cloudWeave/internal/metrics"
)

func NewRouter(apiHandler *APIHandler, transportHandler http.Handler) http.Handler {
	mux := http.NewServeMux()

	if apiHandler != nil {
		mux.HandleFunc("/files/", apiHandler.HandleFiles)
	}

	if transportHandler != nil {
		mux.Handle("/chunks/", transportHandler)
	}

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Prometheus metrics endpoint
	mux.HandleFunc("/metrics", metrics.Handler())

	return mux
}

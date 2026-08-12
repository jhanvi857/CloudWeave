package api

import (
	"fmt"
	"net/http"

	"cloudWeave/internal/metrics"
)

type GCRunner interface {
	CollectGarbage() (int, error)
}

func NewRouter(apiHandler *APIHandler, transportHandler http.Handler, gcRunner GCRunner) http.Handler {
	mux := http.NewServeMux()

	if apiHandler != nil {
		mux.HandleFunc("/files/", apiHandler.HandleFiles)
	}

	if transportHandler != nil {
		mux.Handle("/chunks/", transportHandler)
	}

	if gcRunner != nil {
		mux.HandleFunc("/admin/gc", func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			swept, err := gcRunner.CollectGarbage()
			if err != nil {
				http.Error(w, fmt.Sprintf("GC execution failed: %v", err), http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, "GC sweep complete: %d orphan chunks removed\n", swept)
		})
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


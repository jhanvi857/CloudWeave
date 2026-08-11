package metrics

import (
	"fmt"
	"net/http"
	"sync/atomic"
)

type Metrics struct {
	FileUploadsTotal    uint64
	FileDownloadsTotal  uint64
	RepairedChunksTotal uint64
	ActiveNodesCount    int64
}

var DefaultMetrics = &Metrics{}

func IncFileUploads() {
	atomic.AddUint64(&DefaultMetrics.FileUploadsTotal, 1)
}

func IncFileDownloads() {
	atomic.AddUint64(&DefaultMetrics.FileDownloadsTotal, 1)
}

func AddRepairedChunks(count int) {
	if count > 0 {
		atomic.AddUint64(&DefaultMetrics.RepairedChunksTotal, uint64(count))
	}
}

func SetActiveNodes(count int) {
	atomic.StoreInt64(&DefaultMetrics.ActiveNodesCount, int64(count))
}

func Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		w.WriteHeader(http.StatusOK)

		fmt.Fprintf(w, "# HELP cloudweave_file_uploads_total Total number of file uploads.\n")
		fmt.Fprintf(w, "# TYPE cloudweave_file_uploads_total counter\n")
		fmt.Fprintf(w, "cloudweave_file_uploads_total %d\n\n", atomic.LoadUint64(&DefaultMetrics.FileUploadsTotal))

		fmt.Fprintf(w, "# HELP cloudweave_file_downloads_total Total number of file downloads.\n")
		fmt.Fprintf(w, "# TYPE cloudweave_file_downloads_total counter\n")
		fmt.Fprintf(w, "cloudweave_file_downloads_total %d\n\n", atomic.LoadUint64(&DefaultMetrics.FileDownloadsTotal))

		fmt.Fprintf(w, "# HELP cloudweave_repaired_chunks_total Total number of re-replicated repaired chunks.\n")
		fmt.Fprintf(w, "# TYPE cloudweave_repaired_chunks_total counter\n")
		fmt.Fprintf(w, "cloudweave_repaired_chunks_total %d\n\n", atomic.LoadUint64(&DefaultMetrics.RepairedChunksTotal))

		fmt.Fprintf(w, "# HELP cloudweave_active_nodes Current count of active storage nodes in cluster.\n")
		fmt.Fprintf(w, "# TYPE cloudweave_active_nodes gauge\n")
		fmt.Fprintf(w, "cloudweave_active_nodes %d\n", atomic.LoadInt64(&DefaultMetrics.ActiveNodesCount))
	}
}

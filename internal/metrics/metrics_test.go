package metrics

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMetrics_HandlerOutput(t *testing.T) {
	IncFileUploads()
	IncFileDownloads()
	AddRepairedChunks(5)
	SetActiveNodes(3)

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()

	Handler().ServeHTTP(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200 OK, got %d", res.StatusCode)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}

	out := string(body)
	expectedSubstrings := []string{
		"cloudweave_file_uploads_total",
		"cloudweave_file_downloads_total",
		"cloudweave_repaired_chunks_total 5",
		"cloudweave_active_nodes 3",
	}

	for _, exp := range expectedSubstrings {
		if !strings.Contains(out, exp) {
			t.Errorf("metrics output missing expected substring %q:\nGot:\n%s", exp, out)
		}
	}
}

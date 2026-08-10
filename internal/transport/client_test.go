package transport

import (
	"context"
	"net/http/httptest"
	"testing"

	"cloudWeave/internal/storage"
)

func TestClientServer_PutGetRoundTrip(t *testing.T) {
	store, err := storage.NewDiskStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewDiskStore: %v", err)
	}

	server := NewServer(store)
	ts := httptest.NewServer(server.Handler())
	defer ts.Close()

	client := NewClient(ts.URL)
	ctx := context.Background()

	want := []byte("some chunk data")
	if err := client.PutChunk(ctx, "chunk-1", want); err != nil {
		t.Fatalf("PutChunk: %v", err)
	}

	got, err := client.GetChunk(ctx, "chunk-1")
	if err != nil {
		t.Fatalf("GetChunk: %v", err)
	}
	if string(got) != string(want) {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestClient_GetMissingChunk(t *testing.T) {
	store, _ := storage.NewDiskStore(t.TempDir())
	server := NewServer(store)
	ts := httptest.NewServer(server.Handler())
	defer ts.Close()

	client := NewClient(ts.URL)
	if _, err := client.GetChunk(context.Background(), "does-not-exist"); err == nil {
		t.Fatal("expected error for missing chunk, got nil")
	}
}

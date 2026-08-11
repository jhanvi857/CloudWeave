package cluster

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"cloudWeave/internal/ring"
)

func TestCluster_FailureDetection(t *testing.T) {
	// 1. Setup 2 node servers
	ts1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts1.Close()

	ts2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	r := ring.New()
	var deadNodeCalled atomic.Value

	membership := NewMembership(r, func(deadAddr string) {
		deadNodeCalled.Store(deadAddr)
	})

	membership.AddNode(ts1.URL)
	membership.AddNode(ts2.URL)

	if len(membership.GetActiveNodes()) != 2 {
		t.Fatalf("expected 2 active nodes, got %d", len(membership.GetActiveNodes()))
	}

	checker := NewHeartbeatChecker(membership, 100*time.Millisecond, 300*time.Millisecond)
	checker.Start()
	defer checker.Stop()

	// Wait briefly to confirm heartbeats succeed
	time.Sleep(200 * time.Millisecond)
	if !membership.IsAlive(ts2.URL) {
		t.Fatalf("expected ts2 to be alive")
	}

	// 2. Kill ts2
	ts2.Close()

	// 3. Wait for failure detection timeout
	time.Sleep(600 * time.Millisecond)

	if membership.IsAlive(ts2.URL) {
		t.Fatalf("expected ts2 to be marked dead after shutdown")
	}

	val := deadNodeCalled.Load()
	if val == nil || val.(string) != ts2.URL {
		t.Fatalf("expected dead node callback to receive %s, got %v", ts2.URL, val)
	}

	if len(membership.GetActiveNodes()) != 1 {
		t.Errorf("expected 1 active node after failure, got %d", len(membership.GetActiveNodes()))
	}
}

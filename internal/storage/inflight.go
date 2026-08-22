package storage

import (
	"sync"
	"time"
)

// InFlightRegistry tracks chunk IDs participating in active upload sessions to prevent GC race conditions.
type InFlightRegistry struct {
	mu       sync.RWMutex
	chunks   map[string]int       // chunkID -> active session count
	expiries map[string]time.Time // chunkID -> safety expiration deadline
}

// NewInFlightRegistry initializes an InFlightRegistry.
func NewInFlightRegistry() *InFlightRegistry {
	return &InFlightRegistry{
		chunks:   make(map[string]int),
		expiries: make(map[string]time.Time),
	}
}

// Register marks one or more chunk IDs as currently in-flight with a 15-minute safety timeout.
func (r *InFlightRegistry) Register(chunkIDs ...string) {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	ttl := time.Now().Add(15 * time.Minute)
	for _, id := range chunkIDs {
		if id == "" {
			continue
		}
		r.chunks[id]++
		r.expiries[id] = ttl
	}
}

// Unregister releases reservation for chunk IDs when upload completes or fails.
func (r *InFlightRegistry) Unregister(chunkIDs ...string) {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, id := range chunkIDs {
		if count, exists := r.chunks[id]; exists {
			if count <= 1 {
				delete(r.chunks, id)
				delete(r.expiries, id)
			} else {
				r.chunks[id]--
			}
		}
	}
}

// IsInFlight returns true if the chunk ID is currently participating in an in-flight upload.
func (r *InFlightRegistry) IsInFlight(chunkID string) bool {
	if r == nil {
		return false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()

	if _, ok := r.chunks[chunkID]; ok {
		if exp, hasExp := r.expiries[chunkID]; hasExp && time.Now().Before(exp) {
			return true
		}
	}
	return false
}

// GetAllInFlight returns a snapshot of all active in-flight chunk IDs, purging expired entries.
func (r *InFlightRegistry) GetAllInFlight() []string {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	var active []string
	for id, exp := range r.expiries {
		if now.Before(exp) {
			active = append(active, id)
		} else {
			delete(r.chunks, id)
			delete(r.expiries, id)
		}
	}
	return active
}

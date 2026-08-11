package vectorclock

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

type ClockRelation int

const (
	Equal ClockRelation = iota
	Before
	After
	Concurrent
)

// VectorClock map of NodeID -> Logical Sequence Counter
type VectorClock struct {
	mu     sync.RWMutex
	Values map[string]uint64 `json:"values"`
}

func NewVectorClock() *VectorClock {
	return &VectorClock{
		Values: make(map[string]uint64),
	}
}

func (vc *VectorClock) Clone() *VectorClock {
	vc.mu.RLock()
	defer vc.mu.RUnlock()

	cp := NewVectorClock()
	for k, v := range vc.Values {
		cp.Values[k] = v
	}
	return cp
}

func (vc *VectorClock) Increment(nodeID string) {
	vc.mu.Lock()
	defer vc.mu.Unlock()

	if nodeID == "" {
		return
	}
	vc.Values[nodeID]++
}

func (vc *VectorClock) Get(nodeID string) uint64 {
	vc.mu.RLock()
	defer vc.mu.RUnlock()
	return vc.Values[nodeID]
}

// Compare compares this clock to another clock.
func (vc *VectorClock) Compare(other *VectorClock) ClockRelation {
	if other == nil {
		return After
	}

	vc.mu.RLock()
	other.mu.RLock()
	defer vc.mu.RUnlock()
	defer other.mu.RUnlock()

	allKeys := make(map[string]bool)
	for k := range vc.Values {
		allKeys[k] = true
	}
	for k := range other.Values {
		allKeys[k] = true
	}

	hasGreater := false
	hasLesser := false

	for k := range allKeys {
		v1 := vc.Values[k]
		v2 := other.Values[k]

		if v1 > v2 {
			hasGreater = true
		}
		if v1 < v2 {
			hasLesser = true
		}
	}

	if !hasGreater && !hasLesser {
		return Equal
	}
	if hasGreater && !hasLesser {
		return After
	}
	if !hasGreater && hasLesser {
		return Before
	}
	return Concurrent
}

// Merge combines another vector clock into this clock by taking element-wise max.
func (vc *VectorClock) Merge(other *VectorClock) {
	if other == nil {
		return
	}

	vc.mu.Lock()
	other.mu.RLock()
	defer vc.mu.Unlock()
	defer other.mu.RUnlock()

	for k, v2 := range other.Values {
		v1 := vc.Values[k]
		if v2 > v1 {
			vc.Values[k] = v2
		}
	}
}

func (vc *VectorClock) String() string {
	vc.mu.RLock()
	defer vc.mu.RUnlock()

	var keys []string
	for k := range vc.Values {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var parts []string
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s:%d", k, vc.Values[k]))
	}
	return "{" + strings.Join(parts, ", ") + "}"
}

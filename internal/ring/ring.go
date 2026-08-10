package ring

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"sort"
	"sync"
)

const defaultVirtualNodes = 150 // per physical node; higher = smoother distribution, more memory

type Ring struct {
	mu           sync.RWMutex
	virtualNodes int
	sortedHashes []uint32          // sorted ring positions, for binary search
	hashToNode   map[uint32]string // ring position -> physical node ID
	nodes        map[string]bool   // set of physical nodes currently in the ring
}

func New() *Ring {
	return &Ring{
		virtualNodes: defaultVirtualNodes,
		hashToNode:   make(map[uint32]string),
		nodes:        make(map[string]bool),
	}
}

func (r *Ring) AddNode(nodeID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.nodes[nodeID] {
		return // already present
	}
	r.nodes[nodeID] = true

	for i := 0; i < r.virtualNodes; i++ {
		h := hashKey(fmt.Sprintf("%s#%d", nodeID, i))
		r.hashToNode[h] = nodeID
	}
	r.rebuildSortedHashes()
}

func (r *Ring) RemoveNode(nodeID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.nodes[nodeID] {
		return
	}
	delete(r.nodes, nodeID)

	for i := 0; i < r.virtualNodes; i++ {
		h := hashKey(fmt.Sprintf("%s#%d", nodeID, i))
		delete(r.hashToNode, h)
	}
	r.rebuildSortedHashes()
}

// GetNodesForKey returns the n distinct physical nodes responsible for key,
// walking clockwise from the key's ring position. This is what the coordinator calls to find replica targets for a chunk.
func (r *Ring) GetNodesForKey(key string, n int) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if len(r.sortedHashes) == 0 {
		return nil
	}

	keyHash := hashKey(key)
	startIdx := sort.Search(len(r.sortedHashes), func(i int) bool {
		return r.sortedHashes[i] >= keyHash
	})

	seen := make(map[string]bool)
	var result []string

	for i := 0; i < len(r.sortedHashes) && len(result) < n; i++ {
		idx := (startIdx + i) % len(r.sortedHashes)
		nodeID := r.hashToNode[r.sortedHashes[idx]]
		if !seen[nodeID] {
			seen[nodeID] = true
			result = append(result, nodeID)
		}
	}
	return result
}

func (r *Ring) rebuildSortedHashes() {
	hashes := make([]uint32, 0, len(r.hashToNode))
	for h := range r.hashToNode {
		hashes = append(hashes, h)
	}
	sort.Slice(hashes, func(i, j int) bool { return hashes[i] < hashes[j] })
	r.sortedHashes = hashes
}

func hashKey(key string) uint32 {
	sum := sha256.Sum256([]byte(key))
	return binary.BigEndian.Uint32(sum[:4])
}

package consensus

import (
	"sync"

	"cloudWeave/internal/metadata"
)

type Role int

const (
	Follower Role = iota
	Candidate
	Leader
)

func (r Role) String() string {
	switch r {
	case Follower:
		return "Follower"
	case Candidate:
		return "Candidate"
	case Leader:
		return "Leader"
	default:
		return "Unknown"
	}
}

type LogEntry struct {
	Term      uint64             `json:"term"`
	Index     uint64             `json:"index"`
	Op        metadata.WALOpType `json:"op"`
	Manifest  metadata.Manifest  `json:"manifest"`
	ChunkID   string             `json:"chunk_id"`
	Locations []string           `json:"locations"`
}

type RaftNode struct {
	mu          sync.RWMutex
	nodeID      string
	peers       []string
	role        Role
	currentTerm uint64
	votedFor    string
	log         []LogEntry
	commitIndex uint64
	lastApplied uint64

	store     *metadata.Store
	applyChan chan LogEntry
	stopChan  chan struct{}
}

func NewRaftNode(nodeID string, peers []string, store *metadata.Store) *RaftNode {
	return &RaftNode{
		nodeID:      nodeID,
		peers:       peers,
		role:        Follower,
		currentTerm: 0,
		store:       store,
		log:         make([]LogEntry, 0),
		applyChan:   make(chan LogEntry, 100),
		stopChan:    make(chan struct{}),
	}
}

func (rn *RaftNode) Start() {
	go rn.applyLoop()
}

func (rn *RaftNode) Stop() {
	close(rn.stopChan)
}

func (rn *RaftNode) ProposeManifest(m metadata.Manifest) error {
	rn.mu.Lock()
	defer rn.mu.Unlock()

	entry := LogEntry{
		Term:     rn.currentTerm,
		Index:    uint64(len(rn.log) + 1),
		Op:       metadata.OpRecordManifest,
		Manifest: m,
	}

	rn.log = append(rn.log, entry)
	rn.commitIndex = entry.Index
	rn.applyChan <- entry
	return nil
}

func (rn *RaftNode) ProposeLocationUpdate(chunkID string, locs []string) error {
	rn.mu.Lock()
	defer rn.mu.Unlock()

	entry := LogEntry{
		Term:      rn.currentTerm,
		Index:     uint64(len(rn.log) + 1),
		Op:        metadata.OpUpdateLocations,
		ChunkID:   chunkID,
		Locations: locs,
	}

	rn.log = append(rn.log, entry)
	rn.commitIndex = entry.Index
	rn.applyChan <- entry
	return nil
}

func (rn *RaftNode) applyLoop() {
	for {
		select {
		case entry := <-rn.applyChan:
			rn.mu.Lock()
			rn.lastApplied = entry.Index
			rn.mu.Unlock()

			switch entry.Op {
			case metadata.OpRecordManifest:
				_ = rn.store.RecordPlacement(entry.Manifest)
			case metadata.OpUpdateLocations:
				rn.store.UpdateChunkLocations(entry.ChunkID, entry.Locations)
			}
		case <-rn.stopChan:
			return
		}
	}
}

func (rn *RaftNode) GetState() (uint64, Role) {
	rn.mu.RLock()
	defer rn.mu.RUnlock()
	return rn.currentTerm, rn.role
}

func (rn *RaftNode) ForceLeader() {
	rn.mu.Lock()
	defer rn.mu.Unlock()
	rn.role = Leader
	rn.currentTerm++
}

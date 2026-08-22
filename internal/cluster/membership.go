package cluster

import (
	"sync"
	"time"

	"cloudWeave/internal/ring"
)

type NodeInfo struct {
	Address             string
	LastSeen            time.Time
	IsAlive             bool
	ConsecutiveFailures int
}

type Membership struct {
	mu         sync.RWMutex
	ring       *ring.Ring
	nodes      map[string]*NodeInfo
	onNodeDead func(deadNodeAddr string)
}

func NewMembership(r *ring.Ring, onNodeDead func(string)) *Membership {
	return &Membership{
		ring:       r,
		nodes:      make(map[string]*NodeInfo),
		onNodeDead: onNodeDead,
	}
}

func (m *Membership) AddNode(addr string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	info, exists := m.nodes[addr]
	if !exists {
		info = &NodeInfo{Address: addr}
		m.nodes[addr] = info
	}
	info.LastSeen = time.Now()
	info.IsAlive = true
	info.ConsecutiveFailures = 0

	m.ring.AddNode(addr)
}

func (m *Membership) MarkAlive(addr string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	info, exists := m.nodes[addr]
	if exists {
		info.LastSeen = time.Now()
		info.ConsecutiveFailures = 0
		if !info.IsAlive {
			info.IsAlive = true
			m.ring.AddNode(addr)
		}
	}
}

func (m *Membership) RecordFailure(addr string, maxConsecutiveFailures int, deadTimeout time.Duration) bool {
	m.mu.Lock()
	dead := false

	info, exists := m.nodes[addr]
	if exists && info.IsAlive {
		info.ConsecutiveFailures++
		if info.ConsecutiveFailures >= maxConsecutiveFailures && time.Since(info.LastSeen) >= deadTimeout {
			info.IsAlive = false
			dead = true
			m.ring.RemoveNode(addr)
		}
	}
	onDeadCallback := m.onNodeDead
	m.mu.Unlock()

	if dead && onDeadCallback != nil {
		onDeadCallback(addr)
	}
	return dead
}

func (m *Membership) RemoveNode(addr string) {
	m.mu.Lock()
	removed := false

	info, exists := m.nodes[addr]
	if exists {
		if info.IsAlive {
			removed = true
		}
		delete(m.nodes, addr)
		m.ring.RemoveNode(addr)
	}
	onDeadCallback := m.onNodeDead
	m.mu.Unlock()

	if removed && onDeadCallback != nil {
		onDeadCallback(addr)
	}
}

func (m *Membership) MarkDead(addr string) {
	m.mu.Lock()
	dead := false

	info, exists := m.nodes[addr]
	if exists && info.IsAlive {
		info.IsAlive = false
		dead = true
		m.ring.RemoveNode(addr)
	}
	onDeadCallback := m.onNodeDead
	m.mu.Unlock()

	if dead && onDeadCallback != nil {
		onDeadCallback(addr)
	}
}

func (m *Membership) GetActiveNodes() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var active []string
	for addr, info := range m.nodes {
		if info.IsAlive {
			active = append(active, addr)
		}
	}
	return active
}

func (m *Membership) GetAllNodes() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var all []string
	for addr := range m.nodes {
		all = append(all, addr)
	}
	return all
}

func (m *Membership) IsAlive(addr string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	info, exists := m.nodes[addr]
	return exists && info.IsAlive
}

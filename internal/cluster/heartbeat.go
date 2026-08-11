package cluster

import (
	"context"
	"log"
	"net/http"
	"time"
)

type HeartbeatChecker struct {
	membership  *Membership
	interval    time.Duration
	deadTimeout time.Duration
	httpClient  *http.Client
	stopChan    chan struct{}
}

func NewHeartbeatChecker(membership *Membership, interval, deadTimeout time.Duration) *HeartbeatChecker {
	if interval <= 0 {
		interval = 1 * time.Second
	}
	if deadTimeout <= 0 {
		deadTimeout = 3 * time.Second
	}

	return &HeartbeatChecker{
		membership:  membership,
		interval:    interval,
		deadTimeout: deadTimeout,
		httpClient:  &http.Client{Timeout: 1 * time.Second},
		stopChan:    make(chan struct{}),
	}
}

func (h *HeartbeatChecker) Start() {
	ticker := time.NewTicker(h.interval)
	go func() {
		for {
			select {
			case <-ticker.C:
				h.checkNodes()
			case <-h.stopChan:
				ticker.Stop()
				return
			}
		}
	}()
}

func (h *HeartbeatChecker) Stop() {
	close(h.stopChan)
}

func (h *HeartbeatChecker) checkNodes() {
	allNodes := h.membership.GetAllNodes()
	for _, addr := range allNodes {
		go func(targetAddr string) {
			alive := h.pingNode(targetAddr)
			if alive {
				h.membership.MarkAlive(targetAddr)
			} else {
				h.membership.mu.RLock()
				info, exists := h.membership.nodes[targetAddr]
				var lastSeen time.Time
				if exists {
					lastSeen = info.LastSeen
				}
				h.membership.mu.RUnlock()

				if exists && time.Since(lastSeen) > h.deadTimeout {
					log.Printf("[Cluster] Node %s failed heartbeat check, marking DEAD", targetAddr)
					h.membership.MarkDead(targetAddr)
				}
			}
		}(addr)
	}
}

func (h *HeartbeatChecker) pingNode(addr string) bool {
	url := addr + "/health"
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false
	}

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

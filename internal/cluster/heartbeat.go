package cluster

import (
	"context"
	"log"
	"net/http"
	"time"
)

type HeartbeatChecker struct {
	membership         *Membership
	interval           time.Duration
	deadTimeout        time.Duration
	maxConsecutiveMiss int
	httpClient         *http.Client
	clusterSecret      string
	stopChan           chan struct{}
}

func NewHeartbeatChecker(membership *Membership, interval, deadTimeout time.Duration) *HeartbeatChecker {
	if interval <= 0 {
		interval = 2 * time.Second
	}
	if deadTimeout <= 0 {
		deadTimeout = 8 * time.Second
	}

	return &HeartbeatChecker{
		membership:         membership,
		interval:           interval,
		deadTimeout:        deadTimeout,
		maxConsecutiveMiss: 4, // Require 4 consecutive failures before declaring node dead (flap-damping)
		httpClient:         &http.Client{Timeout: 1500 * time.Millisecond},
		stopChan:           make(chan struct{}),
	}
}

func (h *HeartbeatChecker) SetMaxConsecutiveMisses(misses int) {
	if misses > 0 {
		h.maxConsecutiveMiss = misses
	}
}

func (h *HeartbeatChecker) SetHTTPClient(client *http.Client) {
	if client != nil {
		h.httpClient = client
	}
}

func (h *HeartbeatChecker) SetClusterSecret(secret string) {
	h.clusterSecret = secret
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
				dead := h.membership.RecordFailure(targetAddr, h.maxConsecutiveMiss, h.deadTimeout)
				if dead {
					log.Printf("[Cluster] Node %s declared DEAD after %d consecutive failed heartbeats (>%v)", targetAddr, h.maxConsecutiveMiss, h.deadTimeout)
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
	if h.clusterSecret != "" {
		req.Header.Set("X-Cluster-Secret", h.clusterSecret)
	}

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

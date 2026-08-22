package cluster

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"cloudWeave/internal/metadata"
)

// AntiEntropyReconciler periodically synchronizes metadata manifests with peer nodes to reconcile partitions or downtime.
type AntiEntropyReconciler struct {
	metaStore     *metadata.Store
	membership    *Membership
	localAddr     string
	httpClient    *http.Client
	clusterSecret string
	interval      time.Duration
	stopChan      chan struct{}
	mu            sync.Mutex
}

// NewAntiEntropyReconciler initializes a new AntiEntropyReconciler.
func NewAntiEntropyReconciler(metaStore *metadata.Store, membership *Membership, localAddr string, interval time.Duration) *AntiEntropyReconciler {
	if interval <= 0 {
		interval = 30 * time.Second
	}
	return &AntiEntropyReconciler{
		metaStore:  metaStore,
		membership: membership,
		localAddr:  localAddr,
		interval:   interval,
		httpClient: &http.Client{Timeout: 5 * time.Second},
		stopChan:   make(chan struct{}),
	}
}

func (a *AntiEntropyReconciler) SetHTTPClient(client *http.Client) {
	if client != nil {
		a.httpClient = client
	}
}

func (a *AntiEntropyReconciler) SetClusterSecret(secret string) {
	a.clusterSecret = secret
}

func (a *AntiEntropyReconciler) Start() {
	ticker := time.NewTicker(a.interval)
	go func() {
		for {
			select {
			case <-ticker.C:
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				_ = a.SyncOnce(ctx)
				cancel()
			case <-a.stopChan:
				ticker.Stop()
				return
			}
		}
	}()
}

func (a *AntiEntropyReconciler) Stop() {
	close(a.stopChan)
}

// SyncOnce performs a full anti-entropy sync cycle against all active peer nodes.
func (a *AntiEntropyReconciler) SyncOnce(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.membership == nil || a.metaStore == nil {
		return nil
	}

	peers := a.membership.GetActiveNodes()
	for _, peer := range peers {
		if peer == a.localAddr || peer == "" {
			continue
		}

		manifests, err := a.fetchPeerManifests(ctx, peer)
		if err != nil {
			log.Printf("[AntiEntropy] Failed to fetch manifests from peer %s: %v", peer, err)
			continue
		}

		for _, remoteM := range manifests {
			localM, found := a.metaStore.LookupScoped(remoteM.Namespace, remoteM.FileID)
			if !found {
				_ = a.metaStore.RecordPlacement(remoteM)
				log.Printf("[AntiEntropy] Reconciled missing manifest for %s/%s from peer %s", remoteM.Namespace, remoteM.FileID, peer)
			} else {
				if len(remoteM.Versions) > len(localM.Versions) || remoteM.VersionID != localM.VersionID {
					_ = a.metaStore.RecordPlacement(remoteM)
				}
			}
		}
	}
	return nil
}

func (a *AntiEntropyReconciler) fetchPeerManifests(ctx context.Context, peerAddr string) (map[string]metadata.Manifest, error) {
	url := peerAddr + "/internal/manifest"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	if a.clusterSecret != "" {
		req.Header.Set("X-Cluster-Secret", a.clusterSecret)
	}

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("peer %s returned status %d", peerAddr, resp.StatusCode)
	}

	var manifests map[string]metadata.Manifest
	if err := json.NewDecoder(resp.Body).Decode(&manifests); err != nil {
		return nil, fmt.Errorf("decoding peer manifests: %w", err)
	}
	return manifests, nil
}

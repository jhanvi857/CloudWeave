package coordinator

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"cloudWeave/internal/metadata"
	"cloudWeave/internal/ring"
	"cloudWeave/internal/storage"
)

type Coordinator struct {
	ring       *ring.Ring
	metaStore  *metadata.Store
	localAddr  string
	localStore *storage.DiskStore

	httpClient    *http.Client
	clusterSecret string

	N int // Replication factor
	W int // Write quorum
	R int // Read quorum

	StorageMode string // "replication" (default) or "erasure"
	K           int    // Data shards count for erasure coding (default 4)
	M           int    // Parity shards count for erasure coding (default 2)
}

func NewCoordinator(r *ring.Ring, meta *metadata.Store, localAddr string, localStore *storage.DiskStore, n, w, req int) *Coordinator {
	if n <= 0 {
		n = 3
	}
	if w <= 0 {
		w = 2
	}
	if req <= 0 {
		req = 2
	}
	return &Coordinator{
		ring:        r,
		metaStore:   meta,
		localAddr:   localAddr,
		localStore:  localStore,
		N:           n,
		W:           w,
		R:           req,
		StorageMode: "replication",
		K:           4,
		M:           2,
	}
}

func (c *Coordinator) SetStorageMode(mode string, k, m int) {
	if mode != "" {
		c.StorageMode = strings.ToLower(mode)
	}
	if k > 0 {
		c.K = k
	}
	if m > 0 {
		c.M = m
	}
}

func (c *Coordinator) SetHTTPClient(client *http.Client) {
	c.httpClient = client
}

func (c *Coordinator) SetClusterSecret(secret string) {
	c.clusterSecret = secret
}


// PutChunk satisfies api.ChunkStorageEngine interface by fanning out chunk to N nodes from ring or encoding via Reed-Solomon.
func (c *Coordinator) PutChunk(chunkID string, data []byte) ([]string, error) {
	if strings.EqualFold(c.StorageMode, "erasure") {
		return c.PutErasureChunk(chunkID, data)
	}

	targetNodes := c.ring.GetNodesForKey(chunkID, c.N)
	if len(targetNodes) == 0 {
		return nil, fmt.Errorf("no active nodes in hash ring to place chunk %s", chunkID)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return c.WriteChunk(ctx, chunkID, data, targetNodes)
}

// GetChunk satisfies api.ChunkStorageEngine interface by reading from R quorum.
func (c *Coordinator) GetChunk(chunkID string, locations []string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return c.ReadChunk(ctx, chunkID, locations)
}

func (c *Coordinator) GetRing() *ring.Ring {
	return c.ring
}

func (c *Coordinator) GetMetaStore() *metadata.Store {
	return c.metaStore
}

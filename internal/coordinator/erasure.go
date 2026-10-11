package coordinator

import (
	"context"
	"encoding/binary"
	"fmt"
	"time"

	"cloudWeave/internal/erasure"
	"cloudWeave/internal/transport"
)

// PutErasureChunk encodes chunk data into K data shards + M parity shards and places them across ring nodes.
func (c *Coordinator) PutErasureChunk(chunkID string, data []byte) ([]string, error) {
	k := c.K
	if k <= 0 {
		k = 4
	}
	m := c.M
	if m <= 0 {
		m = 2
	}

	enc, err := erasure.NewEncoder(k, m)
	if err != nil {
		return nil, fmt.Errorf("creating erasure encoder: %w", err)
	}

	// 4-byte original length prefix so reconstruction strips padding cleanly
	prefix := make([]byte, 4)
	binary.BigEndian.PutUint32(prefix, uint32(len(data)))
	payload := append(prefix, data...)

	shards, err := enc.Encode(payload)
	if err != nil {
		return nil, fmt.Errorf("erasure encoding chunk %s: %w", chunkID, err)
	}

	targetNodes := c.ring.GetNodesForKey(chunkID, k+m)
	if len(targetNodes) == 0 {
		return nil, fmt.Errorf("no active nodes in hash ring to place erasure chunk %s", chunkID)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var successfulLocations []string
	successfulShards := 0

	for i, shard := range shards {
		shardID := fmt.Sprintf("%s.s%d", chunkID, shard.Index)
		targetNode := targetNodes[i%len(targetNodes)]

		var writeErr error
		if targetNode == c.localAddr && c.localStore != nil {
			writeErr = c.localStore.Put(shardID, shard.Data)
		} else {
			client := transport.NewClientWithHTTPClientAndSecret(targetNode, c.httpClient, c.clusterSecret)
			reqCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
			writeErr = client.PutChunk(reqCtx, shardID, shard.Data)
			cancel()
		}

		if writeErr == nil {
			successfulShards++
			successfulLocations = append(successfulLocations, targetNode)
		}
	}

	if successfulShards < k {
		return successfulLocations, fmt.Errorf("erasure coding write failed for %s: got %d shards (needed %d)", chunkID, successfulShards, k)
	}

	return successfulLocations, nil
}

// ReadErasureChunk fetches at least K shards and reconstructs the original chunk data.
func (c *Coordinator) ReadErasureChunk(ctx context.Context, chunkID string, locations []string) ([]byte, error) {
	k := c.K
	if k <= 0 {
		k = 4
	}
	m := c.M
	if m <= 0 {
		m = 2
	}

	enc, err := erasure.NewEncoder(k, m)
	if err != nil {
		return nil, fmt.Errorf("creating erasure encoder: %w", err)
	}

	nodes := locations
	if len(nodes) == 0 {
		nodes = c.ring.GetNodesForKey(chunkID, k+m)
	}

	availableShards := make(map[int][]byte)

	for i := 0; i < k+m; i++ {
		shardID := fmt.Sprintf("%s.s%d", chunkID, i)
		targetNode := nodes[i%len(nodes)]

		var shardData []byte
		var readErr error
		if targetNode == c.localAddr && c.localStore != nil {
			shardData, readErr = c.localStore.Get(shardID)
		} else {
			client := transport.NewClientWithHTTPClientAndSecret(targetNode, c.httpClient, c.clusterSecret)
			reqCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
			shardData, readErr = client.GetChunk(reqCtx, shardID)
			cancel()
		}

		if readErr == nil && len(shardData) > 0 {
			availableShards[i] = shardData
			if len(availableShards) >= k {
				break
			}
		}
	}

	if len(availableShards) < k {
		return nil, fmt.Errorf("insufficient erasure shards for chunk %s: got %d, need %d", chunkID, len(availableShards), k)
	}

	reconstructed, err := enc.Reconstruct(availableShards, 0)
	if err != nil {
		return nil, fmt.Errorf("reconstructing erasure chunk %s: %w", chunkID, err)
	}

	if len(reconstructed) < 4 {
		return nil, fmt.Errorf("reconstructed data too short for chunk %s", chunkID)
	}

	origLen := int(binary.BigEndian.Uint32(reconstructed[:4]))
	if 4+origLen > len(reconstructed) {
		return nil, fmt.Errorf("corrupted length header in erasure chunk %s", chunkID)
	}

	return reconstructed[4 : 4+origLen], nil
}

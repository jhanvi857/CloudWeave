package coordinator

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"

	"cloudWeave/internal/transport"
)

func (c *Coordinator) ReadChunk(ctx context.Context, chunkID string, locations []string) ([]byte, error) {
	if strings.EqualFold(c.StorageMode, "erasure") {
		data, err := c.ReadErasureChunk(ctx, chunkID, locations)
		if err == nil {
			return data, nil
		}
	}

	nodes := locations
	if len(nodes) == 0 {
		nodes = c.ring.GetNodesForKey(chunkID, c.N)
	}

	if len(nodes) == 0 {
		return nil, fmt.Errorf("no target locations found for chunk %s", chunkID)
	}

	type readResult struct {
		data []byte
		err  error
		node string
	}

	ch := make(chan readResult, len(nodes))
	var wg sync.WaitGroup

	for _, node := range nodes {
		wg.Add(1)
		go func(targetNode string) {
			defer wg.Done()

			var data []byte
			var err error
			if targetNode == c.localAddr && c.localStore != nil {
				data, err = c.localStore.Get(chunkID)
			} else {
				client := transport.NewClientWithHTTPClientAndSecret(targetNode, c.httpClient, c.clusterSecret)
				reqCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
				defer cancel()
				data, err = client.GetChunk(reqCtx, chunkID)
			}

			ch <- readResult{data: data, err: err, node: targetNode}
		}(node)
	}

	wg.Wait()
	close(ch)

	var successfulData []byte
	successfulReads := 0
	var errs []string

	for res := range ch {
		if res.err == nil && len(res.data) > 0 {
			// Self-Verifying Integrity: verify SHA-256 for content-addressed 64-char hex chunk IDs
			if len(chunkID) == 64 {
				sum := sha256.Sum256(res.data)
				actualID := hex.EncodeToString(sum[:])
				if actualID != chunkID {
					errs = append(errs, fmt.Sprintf("%s: chunk integrity verification failed for %s (corrupted: got %s)", res.node, chunkID, actualID))
					continue
				}
			}

			successfulReads++
			if len(successfulData) == 0 {
				successfulData = res.data
			}
		} else if res.err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", res.node, res.err))
		}
	}

	if successfulReads < c.R || len(successfulData) == 0 {
		return nil, fmt.Errorf("read quorum failed for chunk %s: got %d successful reads (needed %d). Errors: %v",
			chunkID, successfulReads, c.R, errs)
	}

	return successfulData, nil
}

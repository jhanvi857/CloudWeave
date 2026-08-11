package coordinator

import (
	"context"
	"fmt"
	"sync"
	"time"

	"cloudWeave/internal/transport"
)

func (c *Coordinator) WriteChunk(ctx context.Context, chunkID string, data []byte, targetNodes []string) ([]string, error) {
	if len(targetNodes) == 0 {
		return nil, fmt.Errorf("no target nodes provided for chunk %s", chunkID)
	}

	type writeResult struct {
		node string
		err  error
	}

	ch := make(chan writeResult, len(targetNodes))
	var wg sync.WaitGroup

	for _, node := range targetNodes {
		wg.Add(1)
		go func(targetNode string) {
			defer wg.Done()

			var err error
			if targetNode == c.localAddr && c.localStore != nil {
				err = c.localStore.Put(chunkID, data)
			} else {
				client := transport.NewClient(targetNode)
				reqCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
				defer cancel()
				err = client.PutChunk(reqCtx, chunkID, data)
			}

			ch <- writeResult{node: targetNode, err: err}
		}(node)
	}

	wg.Wait()
	close(ch)

	var successfulNodes []string
	var errs []string

	for res := range ch {
		if res.err == nil {
			successfulNodes = append(successfulNodes, res.node)
		} else {
			errs = append(errs, fmt.Sprintf("%s: %v", res.node, res.err))
		}
	}

	if len(successfulNodes) < c.W {
		return successfulNodes, fmt.Errorf("write quorum failed for chunk %s: got %d ACKs (needed %d). Errors: %v",
			chunkID, len(successfulNodes), c.W, errs)
	}

	return successfulNodes, nil
}

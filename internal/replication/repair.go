package replication

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"cloudWeave/internal/metadata"
	"cloudWeave/internal/ring"
	"cloudWeave/internal/storage"
	"cloudWeave/internal/transport"
)

type RepairManager struct {
	metaStore  *metadata.Store
	ring       *ring.Ring
	N          int
	localAddr  string
	localStore *storage.DiskStore
	mu         sync.Mutex
}

func NewRepairManager(meta *metadata.Store, r *ring.Ring, n int, localAddr string, localStore *storage.DiskStore) *RepairManager {
	if n <= 0 {
		n = 3
	}
	return &RepairManager{
		metaStore:  meta,
		ring:       r,
		N:          n,
		localAddr:  localAddr,
		localStore: localStore,
	}
}

// RepairDeadNode scans all manifests for chunks stored on deadNodeAddr, finds surviving replicas, and replicates to missing target nodes.
func (rm *RepairManager) RepairDeadNode(deadNodeAddr string) (int, error) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	manifests := rm.metaStore.GetAllManifests()
	repairedCount := 0

	for _, m := range manifests {
		for chunkID, locs := range m.ChunkLocations {
			// Check if deadNodeAddr was holding this chunk
			hasDead := false
			var aliveLocs []string
			for _, loc := range locs {
				if loc == deadNodeAddr {
					hasDead = true
				} else {
					aliveLocs = append(aliveLocs, loc)
				}
			}

			if !hasDead && len(aliveLocs) >= rm.N {
				continue // No dead node involvement and already fully replicated
			}

			// Get current target nodes from Ring
			targetNodes := rm.ring.GetNodesForKey(chunkID, rm.N)
			if len(targetNodes) == 0 {
				continue
			}

			// Find missing nodes that need the chunk
			var neededTargets []string
			for _, target := range targetNodes {
				alreadyHas := false
				for _, alive := range aliveLocs {
					if alive == target {
						alreadyHas = true
						break
					}
				}
				if !alreadyHas {
					neededTargets = append(neededTargets, target)
				}
			}

			if len(aliveLocs) == 0 || len(neededTargets) == 0 {
				// No surviving replica available to copy from, or no target node needed
				continue
			}

			// Read chunk from first available surviving node
			chunkData, err := rm.fetchChunkFromSurvivors(chunkID, aliveLocs)
			if err != nil {
				log.Printf("[Repair] Failed to fetch chunk %s from survivors %v: %v", chunkID, aliveLocs, err)
				continue
			}

			// Replicate to needed targets
			var newLocations []string
			newLocations = append(newLocations, aliveLocs...)

			for _, targetNode := range neededTargets {
				err := rm.writeChunkToNode(chunkID, chunkData, targetNode)
				if err == nil {
					newLocations = append(newLocations, targetNode)
					repairedCount++
					log.Printf("[Repair] Successfully re-replicated chunk %s to target %s", chunkID, targetNode)
				} else {
					log.Printf("[Repair] Failed to re-replicate chunk %s to target %s: %v", chunkID, targetNode, err)
				}
			}

			// Update MetadataStore
			rm.metaStore.UpdateChunkLocations(chunkID, newLocations)
		}
	}

	return repairedCount, nil
}

func (rm *RepairManager) fetchChunkFromSurvivors(chunkID string, survivors []string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	for _, s := range survivors {
		if s == rm.localAddr && rm.localStore != nil {
			data, err := rm.localStore.Get(chunkID)
			if err == nil {
				return data, nil
			}
		} else {
			client := transport.NewClient(s)
			data, err := client.GetChunk(ctx, chunkID)
			if err == nil {
				return data, nil
			}
		}
	}
	return nil, fmt.Errorf("unable to retrieve chunk %s from any surviving node", chunkID)
}

func (rm *RepairManager) writeChunkToNode(chunkID string, data []byte, targetNode string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if targetNode == rm.localAddr && rm.localStore != nil {
		return rm.localStore.Put(chunkID, data)
	}

	client := transport.NewClient(targetNode)
	return client.PutChunk(ctx, chunkID, data)
}

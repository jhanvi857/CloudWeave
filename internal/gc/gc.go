package gc

import (
	"fmt"
	"log"

	"cloudWeave/internal/metadata"
	"cloudWeave/internal/storage"
)

// GarbageCollector performs mark-and-sweep GC on disk storage.
type GarbageCollector struct {
	metaStore *metadata.Store
	diskStore *storage.DiskStore
}

// NewGarbageCollector initializes a new GarbageCollector.
func NewGarbageCollector(metaStore *metadata.Store, diskStore *storage.DiskStore) *GarbageCollector {
	return &GarbageCollector{
		metaStore: metaStore,
		diskStore: diskStore,
	}
}

// CollectGarbage scans all manifests to build an active chunk ID set (Mark),
// then scans local disk chunks and deletes any unreferenced chunks (Sweep).
// Active chunks are collected across ALL namespaces to ensure deletion activity in one namespace
// never deletes chunks referenced by active manifests in another namespace.
func (g *GarbageCollector) CollectGarbage() (int, error) {
	return g.CollectGarbageForNamespace("")
}

// CollectGarbageForNamespace runs GC while ensuring active chunks across all namespaces are preserved.
func (g *GarbageCollector) CollectGarbageForNamespace(targetNamespace string) (int, error) {
	if g.metaStore == nil || g.diskStore == nil {
		return 0, fmt.Errorf("garbage collector not properly initialized")
	}

	// 1. Mark Phase: Gather all active chunk IDs referenced across ALL manifests in ALL namespaces
	manifests := g.metaStore.GetAllManifests()
	activeChunks := make(map[string]bool)

	for _, manifest := range manifests {
		for _, chunkID := range manifest.ChunkIDs {
			activeChunks[chunkID] = true
		}
		for _, hist := range manifest.Versions {
			for _, chunkID := range hist.ChunkIDs {
				activeChunks[chunkID] = true
			}
		}
	}

	// 2. Sweep Phase: Scan disk chunks and delete unreferenced ones
	localChunks, err := g.diskStore.ListChunks()
	if err != nil {
		return 0, fmt.Errorf("GC failed to list local chunks: %w", err)
	}

	deletedCount := 0
	for _, chunkID := range localChunks {
		if !activeChunks[chunkID] {
			if err := g.diskStore.Delete(chunkID); err != nil {
				log.Printf("[GC] Error deleting orphan chunk %s: %v", chunkID, err)
			} else {
				log.Printf("[GC] Swept orphan chunk %s from disk", chunkID)
				deletedCount++
			}
		}
	}

	return deletedCount, nil
}

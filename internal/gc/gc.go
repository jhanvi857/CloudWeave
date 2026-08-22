package gc

import (
	"fmt"
	"log"
	"time"

	"cloudWeave/internal/metadata"
	"cloudWeave/internal/storage"
)

// MultipartChunkProvider supplies chunk IDs currently held by active multipart uploads.
type MultipartChunkProvider interface {
	GetActiveChunkIDs() []string
}

// GarbageCollector performs mark-and-sweep GC on disk storage.
type GarbageCollector struct {
	metaStore         *metadata.Store
	diskStore         *storage.DiskStore
	inFlight          *storage.InFlightRegistry
	multipartProvider MultipartChunkProvider
	mtimeGrace        time.Duration
}

// NewGarbageCollector initializes a new GarbageCollector with an InFlightRegistry.
func NewGarbageCollector(metaStore *metadata.Store, diskStore *storage.DiskStore) *GarbageCollector {
	var reg *storage.InFlightRegistry
	if diskStore != nil {
		reg = diskStore.GetInFlightRegistry()
	}
	return &GarbageCollector{
		metaStore:  metaStore,
		diskStore:  diskStore,
		inFlight:   reg,
		mtimeGrace: 0,
	}
}

// SetInFlightRegistry configures an explicit in-flight upload registry.
func (g *GarbageCollector) SetInFlightRegistry(reg *storage.InFlightRegistry) {
	g.inFlight = reg
}

// SetMultipartProvider configures the active multipart chunk provider.
func (g *GarbageCollector) SetMultipartProvider(provider MultipartChunkProvider) {
	g.multipartProvider = provider
}

// SetMtimeGrace configures the safety grace period for recently modified chunk files.
func (g *GarbageCollector) SetMtimeGrace(grace time.Duration) {
	g.mtimeGrace = grace
}

// CollectGarbage scans all manifests and active in-flight uploads to build an active chunk ID set (Mark),
// then scans local disk chunks and deletes unreferenced chunks older than the grace period (Sweep).
func (g *GarbageCollector) CollectGarbage() (int, error) {
	return g.CollectGarbageForNamespace("")
}

// CollectGarbageForNamespace runs GC while ensuring active chunks across all namespaces and in-flight sessions are preserved.
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

	// Include in-flight streaming uploads from InFlightRegistry
	if g.inFlight != nil {
		for _, id := range g.inFlight.GetAllInFlight() {
			activeChunks[id] = true
		}
	}
	if g.diskStore != nil && g.diskStore.GetInFlightRegistry() != nil {
		for _, id := range g.diskStore.GetInFlightRegistry().GetAllInFlight() {
			activeChunks[id] = true
		}
	}

	// Include active multipart uploads
	if g.multipartProvider != nil {
		for _, id := range g.multipartProvider.GetActiveChunkIDs() {
			activeChunks[id] = true
		}
	}

	// 2. Sweep Phase: Scan disk chunks and delete unreferenced ones older than mtimeGrace
	localChunkInfos, err := g.diskStore.ListChunkInfos()
	if err != nil {
		return 0, fmt.Errorf("GC failed to list local chunks: %w", err)
	}

	now := time.Now()
	deletedCount := 0
	for _, info := range localChunkInfos {
		if activeChunks[info.ID] {
			continue // referenced by active manifest, version, in-flight upload, or multipart part
		}

		// Secondary protection: skip files modified within the mtimeGrace window
		if g.mtimeGrace > 0 && now.Sub(info.ModTime) < g.mtimeGrace {
			continue // newly written chunk protected by mtime grace period
		}

		if err := g.diskStore.Delete(info.ID); err != nil {
			log.Printf("[GC] Error deleting orphan chunk %s: %v", info.ID, err)
		} else {
			log.Printf("[GC] Swept orphan chunk %s from disk", info.ID)
			deletedCount++
		}
	}

	return deletedCount, nil
}

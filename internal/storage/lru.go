package storage

import (
	"container/list"
	"sync"
)

type cacheItem struct {
	key   string
	value []byte
	size  int64
}

// ChunkCache provides a thread-safe in-memory LRU cache for chunk payloads.
type ChunkCache struct {
	mu           sync.RWMutex
	maxBytes     int64
	currentBytes int64
	items        map[string]*list.Element
	evictList    *list.List
}

// NewChunkCache initializes a ChunkCache with maxBytes byte capacity.
// If maxBytes <= 0, defaults to 64MB (64 * 1024 * 1024).
func NewChunkCache(maxBytes int64) *ChunkCache {
	if maxBytes <= 0 {
		maxBytes = 64 * 1024 * 1024
	}
	return &ChunkCache{
		maxBytes:  maxBytes,
		items:     make(map[string]*list.Element),
		evictList: list.New(),
	}
}

// Get retrieves a chunk from cache if present, updating LRU recency.
func (c *ChunkCache) Get(key string) ([]byte, bool) {
	if c == nil {
		return nil, false
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, ok := c.items[key]; ok {
		c.evictList.MoveToFront(elem)
		item := elem.Value.(*cacheItem)
		// Return copy to prevent caller mutation of cached buffer
		cp := make([]byte, len(item.value))
		copy(cp, item.value)
		return cp, true
	}
	return nil, false
}

// Put inserts or updates a chunk in the LRU cache, evicting older chunks if full.
func (c *ChunkCache) Put(key string, value []byte) {
	if c == nil || int64(len(value)) > c.maxBytes {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	valSize := int64(len(value))

	if elem, ok := c.items[key]; ok {
		c.evictList.MoveToFront(elem)
		item := elem.Value.(*cacheItem)
		c.currentBytes += valSize - item.size
		cp := make([]byte, len(value))
		copy(cp, value)
		item.value = cp
		item.size = valSize
	} else {
		cp := make([]byte, len(value))
		copy(cp, value)
		item := &cacheItem{key: key, value: cp, size: valSize}
		elem := c.evictList.PushFront(item)
		c.items[key] = elem
		c.currentBytes += valSize
	}

	c.evict()
}

// Remove deletes a chunk key from cache.
func (c *ChunkCache) Remove(key string) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, ok := c.items[key]; ok {
		c.removeElement(elem)
	}
}

// Clear purges all entries from the cache.
func (c *ChunkCache) Clear() {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[string]*list.Element)
	c.evictList.Init()
	c.currentBytes = 0
}

func (c *ChunkCache) evict() {
	for c.currentBytes > c.maxBytes && c.evictList.Len() > 0 {
		elem := c.evictList.Back()
		if elem != nil {
			c.removeElement(elem)
		}
	}
}

func (c *ChunkCache) removeElement(elem *list.Element) {
	c.evictList.Remove(elem)
	item := elem.Value.(*cacheItem)
	delete(c.items, item.key)
	c.currentBytes -= item.size
}

// Stats returns current size in bytes and cached item count.
func (c *ChunkCache) Stats() (int64, int) {
	if c == nil {
		return 0, 0
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.currentBytes, len(c.items)
}

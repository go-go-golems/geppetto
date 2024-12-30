package embeddings

import (
	"container/list"
	"context"
	"sync"
)

// CacheEntry stores the embedding and its position in the LRU list
type cacheEntry struct {
	embedding []float32
	element   *list.Element
}

// CachedProvider wraps an embedding provider with LRU caching capabilities
type CachedProvider struct {
	provider Provider
	cache    map[string]cacheEntry
	lruList  *list.List
	maxSize  int
	mu       sync.RWMutex
}

// NewCachedProvider creates a new cached wrapper around an embedding provider
// maxSize determines how many embeddings to keep in cache (default 1000)
func NewCachedProvider(provider Provider, maxSize int) *CachedProvider {
	if maxSize <= 0 {
		maxSize = 1000 // reasonable default
	}
	return &CachedProvider{
		provider: provider,
		cache:    make(map[string]cacheEntry),
		lruList:  list.New(),
		maxSize:  maxSize,
	}
}

// GenerateEmbedding returns cached embeddings if available, otherwise generates new ones
func (c *CachedProvider) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	// Check cache first
	c.mu.RLock()
	if entry, ok := c.cache[text]; ok {
		// Move to front of LRU list
		c.mu.RUnlock()
		c.mu.Lock()
		c.lruList.MoveToFront(entry.element)
		c.mu.Unlock()
		return entry.embedding, nil
	}
	c.mu.RUnlock()

	// Generate new embedding
	embedding, err := c.provider.GenerateEmbedding(ctx, text)
	if err != nil {
		return nil, err
	}

	// Add to cache
	c.mu.Lock()
	defer c.mu.Unlock()

	// If we're at capacity, remove the least recently used item
	if c.lruList.Len() >= c.maxSize {
		oldest := c.lruList.Back()
		if oldest != nil {
			oldestKey := oldest.Value.(string)
			delete(c.cache, oldestKey)
			c.lruList.Remove(oldest)
		}
	}

	// Add new entry
	element := c.lruList.PushFront(text)
	c.cache[text] = cacheEntry{
		embedding: embedding,
		element:   element,
	}

	return embedding, nil
}

// GetModel delegates to the underlying provider
func (c *CachedProvider) GetModel() EmbeddingModel {
	return c.provider.GetModel()
}

// ClearCache removes all cached embeddings
func (c *CachedProvider) ClearCache() {
	c.mu.Lock()
	c.cache = make(map[string]cacheEntry)
	c.lruList.Init()
	c.mu.Unlock()
}

// Size returns the current number of cached embeddings
func (c *CachedProvider) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.lruList.Len()
}

// MaxSize returns the maximum number of embeddings that can be cached
func (c *CachedProvider) MaxSize() int {
	return c.maxSize
}

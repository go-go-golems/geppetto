package embeddings

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// DiskCacheEntry represents a single cached embedding with metadata
type DiskCacheEntry struct {
	Embedding  []float32 `json:"embedding"`
	TextPrefix string    `json:"text_prefix"` // First 100 chars
}

// DiskCacheProvider implements the Provider interface with disk persistence
type DiskCacheProvider struct {
	provider   Provider
	directory  string
	maxSize    int64 // in bytes
	maxEntries int   // LRU count
	mu         sync.RWMutex
}

type Option func(*DiskCacheProvider)

func WithDirectory(dir string) Option {
	return func(p *DiskCacheProvider) {
		if dir != "" {
			p.directory = dir
		}
	}
}

func WithMaxSize(size int64) Option {
	return func(p *DiskCacheProvider) {
		p.maxSize = size
	}
}

func WithMaxEntries(count int) Option {
	return func(p *DiskCacheProvider) {
		p.maxEntries = count
	}
}

// NewDiskCacheProvider creates a new disk-based cache provider
func NewDiskCacheProvider(provider Provider, opts ...Option) (*DiskCacheProvider, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	p := &DiskCacheProvider{
		provider:   provider,
		directory:  filepath.Join(homeDir, ".geppetto", "cache", "embeddings", provider.GetModel().Name),
		maxSize:    1 << 30, // 1GB default
		maxEntries: 10000,   // 10k entries default
	}

	for _, opt := range opts {
		opt(p)
	}

	if err := os.MkdirAll(p.directory, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	return p, nil
}

func (p *DiskCacheProvider) getCacheFilePath(text string) string {
	hash := sha256.Sum256([]byte(text))
	return filepath.Join(p.directory, hex.EncodeToString(hash[:]))
}

func (p *DiskCacheProvider) writeEntry(text string, entry *DiskCacheEntry) error {
	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal entry: %w", err)
	}

	path := p.getCacheFilePath(text)
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	return nil
}

func (p *DiskCacheProvider) readEntry(text string) (*DiskCacheEntry, error) {
	path := p.getCacheFilePath(text)

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read cache file: %w", err)
	}

	// Update access time
	now := time.Now()
	if err := os.Chtimes(path, now, now); err != nil {
		return nil, fmt.Errorf("failed to update file times: %w", err)
	}

	var entry DiskCacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		// Treat corrupted files as non-existent
		_ = os.Remove(path) // Best effort cleanup
		return nil, nil
	}

	return &entry, nil
}

func (p *DiskCacheProvider) enforceSize() error {
	entries, err := os.ReadDir(p.directory)
	if err != nil {
		return fmt.Errorf("failed to read cache directory: %w", err)
	}

	type fileInfo struct {
		path       string
		size       int64
		accessTime time.Time
	}

	var files []fileInfo
	var totalSize int64

	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}

		files = append(files, fileInfo{
			path:       filepath.Join(p.directory, entry.Name()),
			size:       info.Size(),
			accessTime: info.ModTime(),
		})
		totalSize += info.Size()
	}

	// Sort by access time (oldest first)
	sort.Slice(files, func(i, j int) bool {
		return files[i].accessTime.Before(files[j].accessTime)
	})

	// Remove oldest files until we're under limits
	for i := 0; i < len(files) && (len(files)-i > p.maxEntries || totalSize > p.maxSize); i++ {
		if err := os.Remove(files[i].path); err != nil {
			return fmt.Errorf("failed to remove cache file: %w", err)
		}
		totalSize -= files[i].size
	}

	return nil
}

func (p *DiskCacheProvider) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	p.mu.RLock()
	entry, err := p.readEntry(text)
	p.mu.RUnlock()
	if err != nil {
		return nil, err
	}
	if entry != nil {
		return entry.Embedding, nil
	}

	embedding, err := p.provider.GenerateEmbedding(ctx, text)
	if err != nil {
		return nil, err
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	// Store in cache
	prefix := text
	if len(prefix) > 100 {
		prefix = prefix[:100]
	}

	entry = &DiskCacheEntry{
		Embedding:  embedding,
		TextPrefix: prefix,
	}

	if err := p.writeEntry(text, entry); err != nil {
		return nil, err
	}

	if err := p.enforceSize(); err != nil {
		return nil, err
	}

	return embedding, nil
}

func (p *DiskCacheProvider) GetCachedEntry(text string) (*DiskCacheEntry, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.readEntry(text)
}

func (p *DiskCacheProvider) GetModel() EmbeddingModel {
	return p.provider.GetModel()
}

func (p *DiskCacheProvider) ClearCache() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if err := os.RemoveAll(p.directory); err != nil {
		return fmt.Errorf("failed to clear cache: %w", err)
	}

	return os.MkdirAll(p.directory, 0755)
}

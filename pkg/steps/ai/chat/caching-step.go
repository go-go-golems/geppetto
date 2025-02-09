package chat

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/go-go-golems/geppetto/pkg/conversation"
	"github.com/go-go-golems/geppetto/pkg/helpers"
	"github.com/go-go-golems/geppetto/pkg/steps"
	"github.com/rs/zerolog/log"
)

type CacheEntry struct {
	Messages []*conversation.Message   `json:"messages"`
	Input    conversation.Conversation `json:"input"`
	Created  time.Time                 `json:"created"`
}

type CachingStep struct {
	step       Step
	directory  string
	maxSize    int64
	maxEntries int
	mu         sync.RWMutex
}

type Option func(*CachingStep)

func WithCacheDirectory(dir string) Option {
	return func(p *CachingStep) {
		if dir != "" {
			p.directory = dir
		}
	}
}

func WithMaxSize(size int64) Option {
	return func(p *CachingStep) {
		p.maxSize = size
	}
}

func WithMaxEntries(count int) Option {
	return func(p *CachingStep) {
		p.maxEntries = count
	}
}

var _ steps.Step[conversation.Conversation, *conversation.Message] = &CachingStep{}

func NewCachingStep(step Step, opts ...Option) (*CachingStep, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	s := &CachingStep{
		step:       step,
		directory:  filepath.Join(homeDir, ".geppetto", "cache", "chat"),
		maxSize:    1 << 30, // 1GB default
		maxEntries: 10000,   // 10k entries default
	}

	for _, opt := range opts {
		opt(s)
	}

	if err := os.MkdirAll(s.directory, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory %s: %w", s.directory, err)
	}

	return s, nil
}

func (c *CachingStep) getCacheFilePath(input conversation.Conversation) (string, error) {
	hash := input.HashBytes()
	return filepath.Join(c.directory, hex.EncodeToString(hash)), nil
}

func (c *CachingStep) writeEntry(input conversation.Conversation, messages []*conversation.Message) error {
	entry := &CacheEntry{
		Messages: messages,
		Input:    input,
		Created:  time.Now(),
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal entry: %w", err)
	}

	// Check size before writing
	if int64(len(data)) > c.maxSize {
		return fmt.Errorf("entry size %d exceeds maximum size %d", len(data), c.maxSize)
	}

	path, err := c.getCacheFilePath(input)
	if err != nil {
		return err
	}

	log.Debug().
		Str("path", path).
		Int("messageCount", len(messages)).
		Int("dataSize", len(data)).
		Time("created", entry.Created).
		Msg("writing cache entry")

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	// Enforce size limits after writing
	if err := c.enforceSize(); err != nil {
		log.Error().Err(err).Msg("failed to enforce cache size limits")
	}

	return nil
}

func (c *CachingStep) readEntry(input conversation.Conversation) (*CacheEntry, error) {
	path, err := c.getCacheFilePath(input)
	if err != nil {
		return nil, err
	}

	log.Debug().
		Str("path", path).
		Msg("attempting to read cache entry")

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			log.Debug().
				Str("path", path).
				Msg("cache miss - file does not exist")
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read cache file: %w", err)
	}

	// Update access time
	now := time.Now()
	if err := os.Chtimes(path, now, now); err != nil {
		return nil, fmt.Errorf("failed to update file times: %w", err)
	}

	var entry CacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		// Treat corrupted files as non-existent
		log.Debug().
			Str("path", path).
			Err(err).
			Msg("cache entry corrupted, removing file")
		_ = os.Remove(path) // Best effort cleanup
		return nil, nil
	}

	log.Debug().
		Str("path", path).
		Int("messageCount", len(entry.Messages)).
		Time("created", entry.Created).
		Msg("successfully read cache entry")

	return &entry, nil
}

func (c *CachingStep) enforceSize() error {
	entries, err := os.ReadDir(c.directory)
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
			path:       filepath.Join(c.directory, entry.Name()),
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
	for i := 0; i < len(files) && (len(files)-i > c.maxEntries || totalSize > c.maxSize); i++ {
		if err := os.Remove(files[i].path); err != nil {
			return fmt.Errorf("failed to remove cache file: %w", err)
		}
		totalSize -= files[i].size
	}

	return nil
}

func createCachedStepResult(messages []*conversation.Message) steps.StepResult[*conversation.Message] {
	c := make(chan helpers.Result[*conversation.Message], len(messages))
	for _, msg := range messages {
		c <- helpers.NewValueResult(msg)
	}
	close(c)
	return steps.NewStepResult(c)
}

func (c *CachingStep) Start(ctx context.Context, input conversation.Conversation) (steps.StepResult[*conversation.Message], error) {
	c.mu.RLock()
	entry, err := c.readEntry(input)
	c.mu.RUnlock()
	if err != nil {
		return nil, err
	}
	if entry != nil {
		return createCachedStepResult(entry.Messages), nil
	}

	// Cache miss, call underlying step
	result, err := c.step.Start(ctx, input)
	if err != nil {
		return nil, err
	}

	// Collect all messages
	var messages []*conversation.Message
	for r := range result.GetChannel() {
		if r.Error() != nil {
			return nil, r.Error()
		}
		messages = append(messages, r.Unwrap())
	}

	// Cache the messages
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.writeEntry(input, messages); err != nil {
		return nil, err
	}

	return createCachedStepResult(messages), nil
}

func (c *CachingStep) AddPublishedTopic(publisher message.Publisher, topic string) error {
	return c.step.AddPublishedTopic(publisher, topic)
}

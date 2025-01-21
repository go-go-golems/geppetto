package chat

import (
	"container/list"
	"context"
	"sync"
	"time"

	"encoding/hex"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/go-go-golems/geppetto/pkg/conversation"
	"github.com/go-go-golems/geppetto/pkg/helpers"
	"github.com/go-go-golems/geppetto/pkg/steps"
)

type MemoryCacheEntry struct {
	Messages []*conversation.Message
	Input    conversation.Conversation
	Created  time.Time
	element  *list.Element // for LRU tracking
}

type MemoryCachingStep struct {
	step    Step
	cache   map[string]MemoryCacheEntry
	lruList *list.List
	maxSize int
	mu      sync.RWMutex
}

type MemoryOption func(*MemoryCachingStep)

func WithMemoryMaxSize(size int) MemoryOption {
	return func(p *MemoryCachingStep) {
		p.maxSize = size
	}
}

var _ steps.Step[conversation.Conversation, *conversation.Message] = &MemoryCachingStep{}

func NewMemoryCachingStep(step Step, opts ...MemoryOption) (*MemoryCachingStep, error) {
	s := &MemoryCachingStep{
		step:    step,
		cache:   make(map[string]MemoryCacheEntry),
		lruList: list.New(),
		maxSize: 1000, // reasonable default
	}

	for _, opt := range opts {
		opt(s)
	}

	return s, nil
}

func (c *MemoryCachingStep) getCacheKey(input conversation.Conversation) (string, error) {
	return hex.EncodeToString(input.HashBytes()), nil
}

func (c *MemoryCachingStep) writeEntry(key string, input conversation.Conversation, messages []*conversation.Message) {
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
	element := c.lruList.PushFront(key)
	c.cache[key] = MemoryCacheEntry{
		Messages: messages,
		Input:    input,
		Created:  time.Now(),
		element:  element,
	}
}

func (c *MemoryCachingStep) readEntry(key string) (*MemoryCacheEntry, error) {
	if entry, ok := c.cache[key]; ok {
		// Move to front of LRU list
		c.lruList.MoveToFront(entry.element)
		return &entry, nil
	}
	return nil, nil
}

func createMemoryCachedStepResult(messages []*conversation.Message) steps.StepResult[*conversation.Message] {
	c := make(chan helpers.Result[*conversation.Message], len(messages))
	for _, msg := range messages {
		c <- helpers.NewValueResult(msg)
	}
	close(c)
	return steps.NewStepResult(c)
}

func (c *MemoryCachingStep) Start(ctx context.Context, input conversation.Conversation) (steps.StepResult[*conversation.Message], error) {
	key, err := c.getCacheKey(input)
	if err != nil {
		return nil, err
	}

	c.mu.RLock()
	entry, err := c.readEntry(key)
	c.mu.RUnlock()
	if err != nil {
		return nil, err
	}
	if entry != nil {
		return createMemoryCachedStepResult(entry.Messages), nil
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
	c.writeEntry(key, input, messages)
	c.mu.Unlock()

	return createMemoryCachedStepResult(messages), nil
}

func (c *MemoryCachingStep) AddPublishedTopic(publisher message.Publisher, topic string) error {
	return c.step.AddPublishedTopic(publisher, topic)
}

func (c *MemoryCachingStep) ClearCache() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache = make(map[string]MemoryCacheEntry)
	c.lruList.Init()
}

func (c *MemoryCachingStep) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.lruList.Len()
}

func (c *MemoryCachingStep) MaxSize() int {
	return c.maxSize
}

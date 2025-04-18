package steps

import (
	"container/list"
	"context"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/chat"
	"sync"
	"time"

	"encoding/hex"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/go-go-golems/geppetto/pkg/conversation"
	"github.com/go-go-golems/geppetto/pkg/events"
	"github.com/go-go-golems/geppetto/pkg/helpers"
	"github.com/go-go-golems/geppetto/pkg/steps"
	"github.com/google/uuid"
)

type MemoryCacheEntry struct {
	Conversation conversation.Conversation
	Input        conversation.Conversation
	Created      time.Time
	element      *list.Element // for LRU tracking
}

type MemoryCachingStep struct {
	step                chat.Step
	cache               map[string]MemoryCacheEntry
	lruList             *list.List
	maxSize             int
	mu                  sync.RWMutex
	subscriptionManager *events.PublisherManager
}

type MemoryOption func(*MemoryCachingStep) error

func WithMemoryMaxSize(size int) MemoryOption {
	return func(p *MemoryCachingStep) error {
		p.maxSize = size
		return nil
	}
}

func WithMemorySubscriptionManager(subscriptionManager *events.PublisherManager) MemoryOption {
	return func(p *MemoryCachingStep) error {
		p.subscriptionManager = subscriptionManager
		return nil
	}
}

func WithMemoryStepOptions(options ...chat.StepOption) MemoryOption {
	return func(p *MemoryCachingStep) error {
		// Apply step options to the caching step too
		for _, option := range options {
			err := option(p.step)
			if err != nil {
				panic(err)
			}
		}
		return nil
	}
}

var _ steps.Step[conversation.Conversation, *conversation.Message] = &MemoryCachingStep{}

func NewMemoryCachingStep(step chat.Step, opts ...MemoryOption) (*MemoryCachingStep, error) {
	s := &MemoryCachingStep{
		step:                step,
		cache:               make(map[string]MemoryCacheEntry),
		lruList:             list.New(),
		maxSize:             1000, // reasonable default
		subscriptionManager: events.NewPublisherManager(),
	}

	for _, opt := range opts {
		err := opt(s)
		if err != nil {
			return nil, err
		}
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
		Conversation: messages,
		Input:        input,
		Created:      time.Now(),
		element:      element,
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
		// Create metadata for cache hit
		metadata := events.EventMetadata{
			ID:       conversation.NewNodeID(),
			ParentID: conversation.NullNode,
		}
		stepMetadata := &steps.StepMetadata{
			StepID:     uuid.New(),
			Type:       "cache-hit",
			InputType:  "conversation.Conversation",
			OutputType: "string",
			Metadata: map[string]interface{}{
				"cache_type": "memory",
				"created":    entry.Created,
			},
		}

		conversationString := entry.Conversation.ToString()

		// Publish final event for cache hit
		c.subscriptionManager.PublishBlind(events.NewStartEvent(metadata, stepMetadata))
		c.subscriptionManager.PublishBlind(events.NewPartialCompletionEvent(metadata, stepMetadata, conversationString, conversationString))
		c.subscriptionManager.PublishBlind(events.NewFinalEvent(metadata, stepMetadata, conversationString))
		return createMemoryCachedStepResult(entry.Conversation), nil
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

	// Create metadata for cache write
	metadata := events.EventMetadata{
		ID:       conversation.NewNodeID(),
		ParentID: conversation.NullNode,
	}
	stepMetadata := &steps.StepMetadata{
		StepID:     uuid.New(),
		Type:       "cache-write",
		InputType:  "conversation.Conversation",
		OutputType: "string",
		Metadata: map[string]interface{}{
			"cache_type": "memory",
		},
	}

	// Publish final event for cache write
	c.subscriptionManager.PublishBlind(events.NewFinalEvent(metadata, stepMetadata, ""))

	return createMemoryCachedStepResult(messages), nil
}

func (c *MemoryCachingStep) AddPublishedTopic(publisher message.Publisher, topic string) error {
	c.subscriptionManager.RegisterPublisher(topic, publisher)
	return nil
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

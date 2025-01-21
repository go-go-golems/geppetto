package chat

import (
	"context"
	"sync"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/go-go-golems/geppetto/pkg/conversation"
	"github.com/go-go-golems/geppetto/pkg/helpers"
	"github.com/go-go-golems/geppetto/pkg/steps"
)

type MockStep struct {
	messages []*conversation.Message
	mu       sync.Mutex
	index    int
}

var _ steps.Step[conversation.Conversation, *conversation.Message] = &MockStep{}

func NewMockStep(messages []*conversation.Message) *MockStep {
	return &MockStep{
		messages: messages,
		index:    0,
	}
}

func (s *MockStep) Start(ctx context.Context, input conversation.Conversation) (steps.StepResult[*conversation.Message], error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.messages) == 0 {
		return steps.ResolveNone[*conversation.Message](), nil
	}

	// Get the current message
	message := s.messages[s.index]

	// Update index in a round-robin fashion
	s.index = (s.index + 1) % len(s.messages)

	// Create a channel and send the message
	c := make(chan helpers.Result[*conversation.Message], 1)
	c <- helpers.NewValueResult(message)
	close(c)

	return steps.NewStepResult(c), nil
}

func (s *MockStep) AddPublishedTopic(publisher message.Publisher, topic string) error {
	// No-op for mock step
	return nil
}

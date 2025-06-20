package chat

import (
	"context"

	"github.com/go-go-golems/geppetto/pkg/conversation"
)

// SimpleChatStep provides a simplified interface for LLM inference without the complex step mechanism
type SimpleChatStep interface {
	// RunInference executes the LLM call directly and returns the response message
	RunInference(ctx context.Context, messages conversation.Conversation) (*conversation.Message, error)
}

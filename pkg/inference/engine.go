package inference

import (
	"context"

	"github.com/go-go-golems/geppetto/pkg/conversation"
)

// Engine represents an AI inference engine that can process conversations
// and return AI-generated responses. Engines handle provider-specific logic
// for services like OpenAI, Claude, etc.
type Engine interface {
	// RunInference processes a conversation and returns an AI-generated message.
	// The engine handles both streaming and non-streaming modes based on configuration.
	// Events are published through all registered EventSinks during inference.
	RunInference(ctx context.Context, messages conversation.Conversation) (*conversation.Message, error)
}

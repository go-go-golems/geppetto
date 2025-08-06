package engine

import (
	"context"

	"github.com/go-go-golems/geppetto/pkg/conversation"
)

// Engine represents an AI inference engine that can process conversations
// and return AI-generated responses. Engines handle provider-specific logic
// for services like OpenAI, Claude, etc.
type Engine interface {
	// RunInference processes a conversation and returns the full updated conversation.
	// The engine handles both streaming and non-streaming modes based on configuration.
	// Events are published through all registered EventSinks during inference.
	// The returned conversation includes all original messages plus the AI response(s).
	RunInference(ctx context.Context, messages conversation.Conversation) (conversation.Conversation, error)
}

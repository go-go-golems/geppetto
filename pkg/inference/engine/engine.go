package engine

import (
	"context"

	"github.com/go-go-golems/geppetto/pkg/conversation"
)

// Engine represents an AI inference engine that processes conversations
// and returns AI-generated responses. All provider-specific engines implement this.
type Engine interface {
	// RunInference processes an inference context and returns the updated context.
	// The engine handles provider-specific API calls but does NOT handle tool orchestration.
	// Tool calls in the response should be preserved as-is for helper processing.
	RunInference(ctx context.Context, conv conversation.InferenceContext) (conversation.InferenceContext, error)
}

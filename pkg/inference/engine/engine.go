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

// EngineWithTools extends Engine to support tool calling capabilities
// This interface is implemented by engines that can handle tool definitions
// and tool calling in their inference process.
type EngineWithTools interface {
	Engine
	
	// GetSupportedToolFeatures returns what tool features this engine supports
	GetSupportedToolFeatures() ToolFeatures
	
	// PrepareToolsForRequest converts tools to provider-specific format
	PrepareToolsForRequest(tools []ToolDefinition, config ToolConfig) (interface{}, error)
}

// StreamingEngine extends Engine to support streaming inference
type StreamingEngine interface {
	Engine
	
	// RunInferenceStream processes with streaming support
	RunInferenceStream(ctx context.Context, messages conversation.Conversation, chunkHandler StreamChunkHandler) error
}

// EngineWithToolsAndStreaming combines tool support and streaming
type EngineWithToolsAndStreaming interface {
	EngineWithTools
	StreamingEngine
}

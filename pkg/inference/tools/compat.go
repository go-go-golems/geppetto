package tools

import (
	"context"

	"github.com/go-go-golems/geppetto/pkg/conversation"
	"github.com/go-go-golems/geppetto/pkg/inference/engine"
)

// LegacyEngineAdapter adapts old engines to work with the new tool system
type LegacyEngineAdapter struct {
	baseEngine engine.Engine
}

// NewLegacyEngineAdapter creates an adapter for engines that don't support tools
func NewLegacyEngineAdapter(baseEngine engine.Engine) *LegacyEngineAdapter {
	return &LegacyEngineAdapter{
		baseEngine: baseEngine,
	}
}

// RunInference delegates to the base engine
func (a *LegacyEngineAdapter) RunInference(ctx context.Context, messages conversation.Conversation) (conversation.Conversation, error) {
	return a.baseEngine.RunInference(ctx, messages)
}

// GetSupportedToolFeatures returns empty features for legacy engines
func (a *LegacyEngineAdapter) GetSupportedToolFeatures() ToolFeatures {
	return ToolFeatures{
		SupportsParallelCalls: false,
		SupportsToolChoice:    false,
		SupportsSystemTools:   false,
		SupportsStreaming:     false,
		Limits: ProviderLimits{
			MaxToolsPerRequest:      0,
			MaxToolNameLength:       0,
			MaxTotalSizeBytes:       0,
			SupportedParameterTypes: []string{},
		},
		SupportedChoiceTypes: []ToolChoice{},
	}
}

// PrepareToolsForRequest returns nil for legacy engines
func (a *LegacyEngineAdapter) PrepareToolsForRequest(tools []ToolDefinition, config ToolConfig) (interface{}, error) {
	// Legacy engines don't support tools
	return nil, nil
}

// RunInferenceStream provides streaming support if the base engine supports it
func (a *LegacyEngineAdapter) RunInferenceStream(ctx context.Context, messages conversation.Conversation, chunkHandler StreamChunkHandler) error {
	// Check if the base engine supports streaming
	if streamingEngine, ok := a.baseEngine.(interface {
		RunInferenceStream(ctx context.Context, messages conversation.Conversation, chunkHandler func(chunk interface{}) error) error
	}); ok {
		// Adapt the chunk handler
		adaptedHandler := func(chunk interface{}) error {
			// Convert legacy chunk format to our StreamChunk format
			// This is a simplified conversion - real implementation would depend on the legacy format
			streamChunk := StreamChunk{
				Type:       ChunkTypeContent,
				Content:    "", // Would extract content from chunk
				IsComplete: true,
			}
			return chunkHandler(streamChunk)
		}
		
		return streamingEngine.RunInferenceStream(ctx, messages, adaptedHandler)
	}
	
	// Fallback: simulate streaming by running inference and sending the result as one chunk
	result, err := a.baseEngine.RunInference(ctx, messages)
	if err != nil {
		return err
	}
	
	// Extract content from the last message and send as a chunk
	if len(result) > len(messages) {
		lastMessage := result[len(result)-1]
		if chatContent, ok := lastMessage.Content.(*conversation.ChatMessageContent); ok {
			chunk := StreamChunk{
				Type:       ChunkTypeContent,
				Content:    chatContent.Text,
				IsComplete: true,
			}
			if err := chunkHandler(chunk); err != nil {
				return err
			}
		}
	}
	
	// Send completion chunk
	return chunkHandler(StreamChunk{
		Type:       ChunkTypeComplete,
		IsComplete: true,
	})
}

// EngineWrapper wraps any engine to provide tool support via orchestration
type EngineWrapper struct {
	baseEngine   engine.Engine
	orchestrator *InferenceOrchestrator
}

// NewEngineWrapper creates a wrapper that adds tool support to any engine
func NewEngineWrapper(baseEngine engine.Engine, registry ToolRegistry, config ToolConfig) *EngineWrapper {
	// Create an adapter for engines that don't support tools natively
	var toolEngine Engine
	if _, ok := baseEngine.(engine.EngineWithTools); ok {
		// Engine already supports tools, use it directly
		toolEngine = baseEngine.(Engine)
	} else {
		// Engine doesn't support tools, wrap it
		toolEngine = NewLegacyEngineAdapter(baseEngine)
	}
	
	orchestrator := NewInferenceOrchestrator(toolEngine, registry, config)
	
	return &EngineWrapper{
		baseEngine:   baseEngine,
		orchestrator: orchestrator,
	}
}

// RunInference runs inference with tool orchestration
func (w *EngineWrapper) RunInference(ctx context.Context, messages conversation.Conversation) (conversation.Conversation, error) {
	return w.orchestrator.RunInference(ctx, messages)
}

// RunInferenceStream runs streaming inference with tool orchestration
func (w *EngineWrapper) RunInferenceStream(ctx context.Context, messages conversation.Conversation, chunkHandler StreamChunkHandler) error {
	return w.orchestrator.RunInferenceStream(ctx, messages, chunkHandler)
}

// GetOrchestrator returns the underlying orchestrator for advanced usage
func (w *EngineWrapper) GetOrchestrator() *InferenceOrchestrator {
	return w.orchestrator
}

// Ensure compatibility with engine interfaces
var _ engine.Engine = (*LegacyEngineAdapter)(nil)
var _ Engine = (*LegacyEngineAdapter)(nil)
var _ engine.Engine = (*EngineWrapper)(nil)

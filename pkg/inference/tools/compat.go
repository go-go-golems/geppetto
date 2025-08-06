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
	if engineWithTools, ok := baseEngine.(engine.EngineWithTools); ok {
		// Engine already supports tools, create an adapter for it
		toolEngine = NewEngineWithToolsAdapter(engineWithTools)
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

// EngineWithToolsAdapter adapts engine.EngineWithTools to tools.Engine
type EngineWithToolsAdapter struct {
	baseEngine engine.EngineWithTools
}

// NewEngineWithToolsAdapter creates an adapter for engines that implement EngineWithTools
func NewEngineWithToolsAdapter(baseEngine engine.EngineWithTools) *EngineWithToolsAdapter {
	return &EngineWithToolsAdapter{
		baseEngine: baseEngine,
	}
}

// RunInference delegates to the base engine
func (a *EngineWithToolsAdapter) RunInference(ctx context.Context, messages conversation.Conversation) (conversation.Conversation, error) {
	return a.baseEngine.RunInference(ctx, messages)
}

// ConfigureTools configures tools on the underlying engine if it supports direct configuration
func (a *EngineWithToolsAdapter) ConfigureTools(tools []ToolDefinition, config ToolConfig) error {
	// Convert tools.ToolDefinition to engine.ToolDefinition
	var engineTools []engine.ToolDefinition
	for _, tool := range tools {
		engineTool := engine.ToolDefinition{
			Name:        tool.Name,
			Description: tool.Description,
			Parameters:  tool.Parameters,
			Examples:    convertExamples(tool.Examples),
			Tags:        tool.Tags,
			Version:     tool.Version,
		}
		engineTools = append(engineTools, engineTool)
	}
	
	// Convert tools.ToolConfig to engine.ToolConfig
	engineConfig := engine.ToolConfig{
		Enabled:           config.Enabled,
		ToolChoice:        engine.ToolChoice(config.ToolChoice),
		MaxIterations:     config.MaxIterations,
		ExecutionTimeout:  config.ExecutionTimeout,
		MaxParallelTools:  config.MaxParallelTools,
		AllowedTools:      config.AllowedTools,
		ToolErrorHandling: engine.ToolErrorHandling(config.ToolErrorHandling),
		RetryConfig:       engine.RetryConfig(config.RetryConfig),
	}
	
	// Check if the base engine has a ConfigureTools method (like OpenAI engine)
	if configurableEngine, ok := a.baseEngine.(interface {
		ConfigureTools([]engine.ToolDefinition, engine.ToolConfig)
	}); ok {
		configurableEngine.ConfigureTools(engineTools, engineConfig)
		return nil
	}
	
	// Otherwise, just prepare tools (engines that don't need explicit configuration)
	_, err := a.baseEngine.PrepareToolsForRequest(engineTools, engineConfig)
	return err
}

// GetSupportedToolFeatures delegates to the base engine
func (a *EngineWithToolsAdapter) GetSupportedToolFeatures() ToolFeatures {
	baseFeatures := a.baseEngine.GetSupportedToolFeatures()
	
	// Convert engine.ToolFeatures to tools.ToolFeatures
	return ToolFeatures{
		SupportsParallelCalls: baseFeatures.SupportsParallelCalls,
		SupportsToolChoice:    baseFeatures.SupportsToolChoice,
		SupportsSystemTools:   baseFeatures.SupportsSystemTools,
		SupportsStreaming:     baseFeatures.SupportsStreaming,
		Limits: ProviderLimits{
			MaxToolsPerRequest:      baseFeatures.Limits.MaxToolsPerRequest,
			MaxToolNameLength:       baseFeatures.Limits.MaxToolNameLength,
			MaxTotalSizeBytes:       baseFeatures.Limits.MaxTotalSizeBytes,
			SupportedParameterTypes: baseFeatures.Limits.SupportedParameterTypes,
		},
		SupportedChoiceTypes: convertToolChoices(baseFeatures.SupportedChoiceTypes),
	}
}

// PrepareToolsForRequest delegates to the base engine
func (a *EngineWithToolsAdapter) PrepareToolsForRequest(tools []ToolDefinition, config ToolConfig) (interface{}, error) {
	// Convert tools.ToolDefinition to engine.ToolDefinition
	var engineTools []engine.ToolDefinition
	for _, tool := range tools {
		engineTool := engine.ToolDefinition{
			Name:        tool.Name,
			Description: tool.Description,
			Parameters:  tool.Parameters,
			Examples:    convertExamples(tool.Examples),
			Tags:        tool.Tags,
			Version:     tool.Version,
		}
		engineTools = append(engineTools, engineTool)
	}
	
	// Convert tools.ToolConfig to engine.ToolConfig
	engineConfig := engine.ToolConfig{
		Enabled:           config.Enabled,
		ToolChoice:        engine.ToolChoice(config.ToolChoice),
		MaxIterations:     config.MaxIterations,
		ExecutionTimeout:  config.ExecutionTimeout,
		MaxParallelTools:  config.MaxParallelTools,
		AllowedTools:      config.AllowedTools,
		ToolErrorHandling: engine.ToolErrorHandling(config.ToolErrorHandling),
		RetryConfig:       engine.RetryConfig(config.RetryConfig),
	}
	
	return a.baseEngine.PrepareToolsForRequest(engineTools, engineConfig)
}

// RunInferenceStream provides streaming support if the base engine supports it
func (a *EngineWithToolsAdapter) RunInferenceStream(ctx context.Context, messages conversation.Conversation, chunkHandler StreamChunkHandler) error {
	// Check if the base engine supports streaming
	if streamingEngine, ok := a.baseEngine.(engine.StreamingEngine); ok {
		// Convert our StreamChunkHandler to engine.StreamChunkHandler
		engineHandler := func(chunk engine.StreamChunk) error {
			// Convert engine.StreamChunk to tools.StreamChunk
			toolsChunk := StreamChunk{
				Type:       ChunkType(chunk.Type),
				Content:    chunk.Content,
				IsComplete: chunk.IsComplete,
			}
			
			// Convert tool call if present
			if chunk.ToolCall != nil {
				toolsChunk.ToolCall = &PartialToolCall{
					ID:        chunk.ToolCall.ID,
					Name:      chunk.ToolCall.Name,
					Arguments: chunk.ToolCall.Arguments,
				}
			}
			
			return chunkHandler(toolsChunk)
		}
		
		return streamingEngine.RunInferenceStream(ctx, messages, engineHandler)
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

// Helper functions for type conversion
func convertToolChoices(choices []engine.ToolChoice) []ToolChoice {
	var converted []ToolChoice
	for _, choice := range choices {
		converted = append(converted, ToolChoice(choice))
	}
	return converted
}

func convertExamples(examples []ToolExample) []engine.ToolExample {
	var converted []engine.ToolExample
	for _, ex := range examples {
		converted = append(converted, engine.ToolExample{
			Input:       ex.Input,
			Output:      ex.Output,
			Description: ex.Description,
		})
	}
	return converted
}

// Ensure compatibility with engine interfaces
var _ engine.Engine = (*LegacyEngineAdapter)(nil)
var _ Engine = (*LegacyEngineAdapter)(nil)
var _ Engine = (*EngineWithToolsAdapter)(nil)
var _ engine.Engine = (*EngineWrapper)(nil)

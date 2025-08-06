package tools

import (
	"context"
	"fmt"

	"github.com/go-go-golems/geppetto/pkg/conversation"
	"github.com/rs/zerolog/log"
)

// Engine interface for tool-enabled inference engines
type Engine interface {
	// RunInference processes a conversation and returns the full updated conversation.
	// Does NOT handle tool execution - that's the orchestrator's job.
	RunInference(ctx context.Context, messages conversation.Conversation) (conversation.Conversation, error)
	
	// RunInferenceStream processes with streaming support
	RunInferenceStream(ctx context.Context, messages conversation.Conversation, chunkHandler StreamChunkHandler) error
	
	// GetSupportedToolFeatures returns what tool features this engine supports
	GetSupportedToolFeatures() ToolFeatures
	
	// PrepareToolsForRequest converts tools to provider-specific format
	PrepareToolsForRequest(tools []ToolDefinition, config ToolConfig) (interface{}, error)
}

// StreamChunkHandler processes streaming chunks that may include partial tool calls
type StreamChunkHandler func(chunk StreamChunk) error

// StreamChunk represents a piece of streaming response
type StreamChunk struct {
	Type       ChunkType        `json:"type"`
	Content    string           `json:"content,omitempty"`
	ToolCall   *PartialToolCall `json:"tool_call,omitempty"`
	IsComplete bool             `json:"is_complete"`
}

// ChunkType defines the type of streaming chunk
type ChunkType string

const (
	ChunkTypeContent  ChunkType = "content"
	ChunkTypeToolCall ChunkType = "tool_call"
	ChunkTypeComplete ChunkType = "complete"
)

// PartialToolCall represents a partial tool call during streaming
type PartialToolCall struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // May be partial JSON
}

// ToolFeatures describes what tool features an engine supports
type ToolFeatures struct {
	SupportsParallelCalls bool           `json:"supports_parallel_calls"`
	SupportsToolChoice    bool           `json:"supports_tool_choice"`
	SupportsSystemTools   bool           `json:"supports_system_tools"`
	SupportsStreaming     bool           `json:"supports_streaming"`
	Limits                ProviderLimits `json:"limits"`
	SupportedChoiceTypes  []ToolChoice   `json:"supported_choice_types"`
}

// ProviderLimits defines provider-specific limitations
type ProviderLimits struct {
	MaxToolsPerRequest      int      `json:"max_tools_per_request"`
	MaxToolNameLength       int      `json:"max_tool_name_length"`
	MaxTotalSizeBytes       int      `json:"max_total_size_bytes"`
	SupportedParameterTypes []string `json:"supported_parameter_types"`
}

// InferenceOrchestrator handles tool execution iteration and coordination
type InferenceOrchestrator struct {
	engine       Engine
	toolRegistry ToolRegistry
	toolConfig   ToolConfig
	executor     ToolExecutor
}

// NewInferenceOrchestrator creates a new orchestrator with the given components
func NewInferenceOrchestrator(engine Engine, registry ToolRegistry, config ToolConfig) *InferenceOrchestrator {
	executor := NewDefaultToolExecutor(config)
	
	return &InferenceOrchestrator{
		engine:       engine,
		toolRegistry: registry,
		toolConfig:   config,
		executor:     executor,
	}
}

// configureToolsOnEngine configures the available tools on the underlying engine
func (o *InferenceOrchestrator) configureToolsOnEngine() error {
	// Get all tools from registry
	allTools := o.toolRegistry.ListTools()
	
	// Filter tools based on configuration
	var allowedTools []ToolDefinition
	if len(o.toolConfig.AllowedTools) > 0 {
		// Only include explicitly allowed tools
		allowedMap := make(map[string]bool)
		for _, name := range o.toolConfig.AllowedTools {
			allowedMap[name] = true
		}
		
		for _, tool := range allTools {
			if allowedMap[tool.Name] {
				allowedTools = append(allowedTools, tool)
			}
		}
	} else {
		// Include all tools
		allowedTools = allTools
	}
	
	if len(allowedTools) == 0 {
		log.Debug().Msg("No tools available for configuration")
		return nil
	}
	
	log.Debug().Int("tool_count", len(allowedTools)).Msg("Configuring tools on engine")
	
	// Check if the engine supports direct tool configuration
	if configurableEngine, ok := o.engine.(interface {
		ConfigureTools([]ToolDefinition, ToolConfig) error
	}); ok {
		// Engine supports ConfigureTools method, use it
		err := configurableEngine.ConfigureTools(allowedTools, o.toolConfig)
		if err != nil {
			return fmt.Errorf("failed to configure tools on engine: %w", err)
		}
	} else {
		// Fallback to PrepareToolsForRequest
		_, err := o.engine.PrepareToolsForRequest(allowedTools, o.toolConfig)
		if err != nil {
			return fmt.Errorf("failed to prepare tools for engine: %w", err)
		}
	}
	
	return nil
}

// RunInference orchestrates the inference process with tool calling
func (o *InferenceOrchestrator) RunInference(ctx context.Context, messages conversation.Conversation) (conversation.Conversation, error) {
	if !o.toolConfig.Enabled {
		// Tools disabled, just run the engine
		return o.engine.RunInference(ctx, messages)
	}
	
	// Configure tools on the engine before running inference
	err := o.configureToolsOnEngine()
	if err != nil {
		return messages, fmt.Errorf("failed to configure tools on engine: %w", err)
	}
	
	currentConversation := messages
	
	for iteration := 0; iteration < o.toolConfig.MaxIterations; iteration++ {
		// Run inference with current conversation
		response, err := o.engine.RunInference(ctx, currentConversation)
		if err != nil {
			return currentConversation, fmt.Errorf("inference failed at iteration %d: %w", iteration, err)
		}
		
		// Update conversation with the response
		currentConversation = response
		
		// Check if the last message contains tool calls
		toolCalls := o.extractToolCallsFromConversation(response)
		if len(toolCalls) == 0 {
			// No tool calls, we're done
			return currentConversation, nil
		}
		
		// Execute tool calls
		results, err := o.executor.ExecuteToolCalls(ctx, toolCalls, o.toolRegistry)
		if err != nil {
			return currentConversation, fmt.Errorf("tool execution failed at iteration %d: %w", iteration, err)
		}
		
		// Add tool results to conversation
		currentConversation = o.addToolResultsToConversation(currentConversation, results)
		
		// Continue to next iteration
	}
	
	return currentConversation, fmt.Errorf("maximum iterations (%d) reached without completion", o.toolConfig.MaxIterations)
}

// RunInferenceStream orchestrates streaming inference with tool calling
func (o *InferenceOrchestrator) RunInferenceStream(ctx context.Context, messages conversation.Conversation, handler StreamChunkHandler) error {
	if !o.toolConfig.Enabled {
		// Tools disabled, just stream the engine
		return o.engine.RunInferenceStream(ctx, messages, handler)
	}
	
	// For streaming with tools, we need to collect chunks and handle tool calls
	// This is a simplified implementation - a more sophisticated version would
	// stream partial tool calls and handle them incrementally
	
	var chunks []StreamChunk
	
	// Collect chunks from the engine
	engineHandler := func(chunk StreamChunk) error {
		chunks = append(chunks, chunk)
		
		// Forward content chunks immediately
		if chunk.Type == ChunkTypeContent {
			return handler(chunk)
		}
		
		return nil
	}
	
	currentConversation := messages
	
	for iteration := 0; iteration < o.toolConfig.MaxIterations; iteration++ {
		chunks = nil // Reset chunks for this iteration
		
		// Stream from engine
		err := o.engine.RunInferenceStream(ctx, currentConversation, engineHandler)
		if err != nil {
			return fmt.Errorf("streaming inference failed at iteration %d: %w", iteration, err)
		}
		
		// Process collected chunks to build conversation
		response := o.buildConversationFromChunks(currentConversation, chunks)
		currentConversation = response
		
		// Check for tool calls
		toolCalls := o.extractToolCallsFromConversation(response)
		if len(toolCalls) == 0 {
			// No tool calls, send completion and finish
			return handler(StreamChunk{
				Type:       ChunkTypeComplete,
				IsComplete: true,
			})
		}
		
		// Execute tool calls
		results, err := o.executor.ExecuteToolCalls(ctx, toolCalls, o.toolRegistry)
		if err != nil {
			return fmt.Errorf("tool execution failed during streaming at iteration %d: %w", iteration, err)
		}
		
		// Add tool results to conversation
		currentConversation = o.addToolResultsToConversation(currentConversation, results)
		
		// Continue to next iteration
	}
	
	return fmt.Errorf("maximum iterations (%d) reached during streaming", o.toolConfig.MaxIterations)
}

// extractToolCallsFromConversation extracts tool calls from the last assistant message
func (o *InferenceOrchestrator) extractToolCallsFromConversation(conv conversation.Conversation) []ToolCall {
	if len(conv) == 0 {
		return nil
	}
	
	lastMessage := conv[len(conv)-1]
	
	// Check if it's an assistant message by examining the content
	if chatContent, ok := lastMessage.Content.(*conversation.ChatMessageContent); ok {
		if chatContent.Role != conversation.RoleAssistant {
			return nil
		}
	} else {
		// For non-chat content, assume it could contain tool calls
		// In practice, you'd want more sophisticated role checking
	}
	
	var toolCalls []ToolCall
	
	// Handle both single and multiple content
	if chatContent, ok := lastMessage.Content.(*conversation.ChatMessageContent); ok {
		// For now, assume tool calls are not in ChatMessageContent
		// This would need to be extended based on how tool calls are represented
		_ = chatContent
	} else if toolUse, ok := lastMessage.Content.(*conversation.ToolUseContent); ok {
		toolCall := ToolCall{
			ID:        toolUse.ToolID,
			Name:      toolUse.Name,
			Arguments: toolUse.Input,
		}
		toolCalls = append(toolCalls, toolCall)
	}
	
	return toolCalls
}

// addToolResultsToConversation adds tool execution results to the conversation
func (o *InferenceOrchestrator) addToolResultsToConversation(conv conversation.Conversation, results []*ToolResult) conversation.Conversation {
	for _, result := range results {
		var content conversation.MessageContent
		
		if result.Error != "" {
			content = &conversation.ToolResultContent{
				ToolID: result.ID,
				Result: fmt.Sprintf("Error: %s", result.Error),
			}
		} else {
			resultStr := fmt.Sprintf("%v", result.Result)
			content = &conversation.ToolResultContent{
				ToolID: result.ID,
				Result: resultStr,
			}
		}
		
		// Add as a user message (tool results are typically from user perspective)
		toolMessage := &conversation.Message{
			Content: content,
		}
		
		conv = append(conv, toolMessage)
	}
	
	return conv
}

// buildConversationFromChunks builds a conversation from streaming chunks
func (o *InferenceOrchestrator) buildConversationFromChunks(baseConv conversation.Conversation, chunks []StreamChunk) conversation.Conversation {
	// This is a simplified implementation
	// In practice, you'd want to properly reconstruct the conversation from chunks
	
	var textContent string
	var toolCalls []ToolCall
	
	for _, chunk := range chunks {
		switch chunk.Type {
		case ChunkTypeContent:
			textContent += chunk.Content
		case ChunkTypeToolCall:
			if chunk.ToolCall != nil && chunk.IsComplete {
				toolCall := ToolCall{
					ID:        chunk.ToolCall.ID,
					Name:      chunk.ToolCall.Name,
					Arguments: []byte(chunk.ToolCall.Arguments),
				}
				toolCalls = append(toolCalls, toolCall)
			}
		}
	}
	
	// Create assistant message with content and tool calls
	var content conversation.MessageContent
	
	if textContent != "" && len(toolCalls) == 0 {
		// Simple text response
		content = &conversation.ChatMessageContent{
			Role: conversation.RoleAssistant,
			Text: textContent,
		}
	} else if len(toolCalls) == 1 && textContent == "" {
		// Single tool call
		toolCall := toolCalls[0]
		content = &conversation.ToolUseContent{
			ToolID: toolCall.ID,
			Name:   toolCall.Name,
			Input:  toolCall.Arguments,
		}
	} else if textContent != "" {
		// Text with tool calls - for now just use text
		content = &conversation.ChatMessageContent{
			Role: conversation.RoleAssistant,
			Text: textContent,
		}
	}
	
	if content != nil {
		assistantMessage := &conversation.Message{
			Content: content,
		}
		
		result := append(baseConv, assistantMessage)
		return result
	}
	
	return baseConv
}

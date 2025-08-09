package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-go-golems/geppetto/pkg/conversation"
	"github.com/go-go-golems/geppetto/pkg/events"
)

// ToolConfig contains configuration for tool calling middleware
type ToolConfig struct {
	MaxIterations int           // Maximum number of tool calling iterations to prevent infinite loops
	Timeout       time.Duration // Timeout for individual tool executions
	ToolFilter    []string      // Allow specific tools only (empty means all tools allowed)
}

// DefaultToolConfig returns a sensible default tool configuration
func DefaultToolConfig() ToolConfig {
	return ToolConfig{
		MaxIterations: 5,
		Timeout:       30 * time.Second,
		ToolFilter:    nil, // Allow all tools
	}
}

// ToolDescription describes a tool available for the AI to use
type ToolDescription struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"` // JSON Schema for parameters
}

// Toolbox interface for tool execution and descriptions
type Toolbox interface {
	// ExecuteTool executes a tool with the given name and arguments
	ExecuteTool(ctx context.Context, name string, arguments map[string]interface{}) (interface{}, error)

	// GetToolDescriptions returns descriptions of all available tools
	GetToolDescriptions() []ToolDescription
}

// ToolCall represents a tool call extracted from an AI response
type ToolCall struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

// ToolResult represents the result of executing a tool
type ToolResult struct {
	ID      string `json:"id"`
	Content string `json:"content,omitempty"`
	Error   string `json:"error,omitempty"`
}

// ToMessage converts a ToolResult to a conversation Message
func (tr ToolResult) ToMessage() *conversation.Message {
	content := &conversation.ToolResultContent{
		ToolID: tr.ID,
		Result: tr.Content,
	}
	if tr.Error != "" {
		content.Result = fmt.Sprintf("Error: %s", tr.Error)
	}

	return conversation.NewMessage(content)
}

// NewToolMiddleware creates middleware that handles function calling workflows for OpenAI/Claude
func NewToolMiddleware(toolbox Toolbox, config ToolConfig) Middleware {
	return func(next HandlerFunc) HandlerFunc {
		return func(ctx context.Context, conv conversation.InferenceContext) (conversation.InferenceContext, error) {
			return executeToolWorkflow(ctx, conv, toolbox, config, next)
		}
	}
}

// executeToolWorkflow handles the complete tool calling workflow
func executeToolWorkflow(
	ctx context.Context,
	conv conversation.InferenceContext,
	toolbox Toolbox,
	config ToolConfig,
	next HandlerFunc,
) (conversation.InferenceContext, error) {
	iterations := 0
	currentConv := conv

	for iterations < config.MaxIterations {
		enriched := addToolContext(currentConv, toolbox)
		resultConversation, err := next(ctx, enriched)
		if err != nil {
			return conversation.InferenceContext{}, fmt.Errorf("tool inference failed: %w", err)
		}
		if len(resultConversation.Messages) == 0 {
			return conversation.InferenceContext{}, fmt.Errorf("empty conversation returned from inference")
		}
		aiResponse := resultConversation.Messages[len(resultConversation.Messages)-1]

		// Check if response contains tool calls
		toolCalls := extractToolCalls(aiResponse)
		if len(toolCalls) == 0 {
			// No more tool calls, return complete conversation
			return resultConversation, nil
		}

		// Filter tools if ToolFilter is specified
		if len(config.ToolFilter) > 0 {
			toolCalls = filterToolCalls(toolCalls, config.ToolFilter)
		}

		if len(toolCalls) == 0 {
			// All tool calls were filtered out, return conversation
			return resultConversation, nil
		}

		// Publish tool calling events
		for _, toolCall := range toolCalls {
			eventToolCall := events.ToolCall{
				ID:    toolCall.ID,
				Name:  toolCall.Name,
				Input: fmt.Sprintf("%v", toolCall.Arguments), // Convert to string for event
			}
			// TODO: Add proper event metadata and step metadata
			// events.Dispatch(ctx, events.NewToolCallEvent(metadata, stepMetadata, eventToolCall))
			_ = eventToolCall // Placeholder until event system is integrated
		}

		// Execute all tool calls
		toolResults, err := executeToolCalls(ctx, toolCalls, toolbox, config.Timeout)
		if err != nil {
			return conversation.InferenceContext{}, fmt.Errorf("tool execution failed: %w", err)
		}
		currentConv = resultConversation
		for _, result := range toolResults {
			currentConv.Messages = append(currentConv.Messages, result.ToMessage())
		}

		// Publish tool result events
		for _, result := range toolResults {
			eventToolResult := events.ToolResult{
				ID:     result.ID,
				Result: result.Content,
			}
			if result.Error != "" {
				eventToolResult.Result = result.Error
			}
			// TODO: Add proper event metadata and step metadata
			// events.Dispatch(ctx, events.NewToolResultEvent(metadata, stepMetadata, eventToolResult))
			_ = eventToolResult // Placeholder until event system is integrated
		}

		iterations++
	}

	return currentConv, fmt.Errorf("tool calling exceeded maximum iterations (%d)", config.MaxIterations)
}

// addToolContext adds tool descriptions to the conversation if not already present
func addToolContext(conv conversation.InferenceContext, toolbox Toolbox) conversation.InferenceContext {
	if len(conv.Tools) > 0 {
		return conv
	}
	descs := toolbox.GetToolDescriptions()
	for _, d := range descs {
		conv.Tools = append(conv.Tools, conversation.ToolDefinition{Name: d.Name, Description: d.Description, Parameters: d.Parameters})
	}
	return conv
}

// extractToolCalls extracts tool calls from an AI response message
func extractToolCalls(message *conversation.Message) []ToolCall {
	var toolCalls []ToolCall

	switch content := message.Content.(type) {
	case *conversation.ToolUseContent:
		// Single tool use content
		var args map[string]interface{}
		if err := json.Unmarshal(content.Input, &args); err != nil {
			// If we can't parse arguments, create empty map
			args = make(map[string]interface{})
		}

		toolCalls = append(toolCalls, ToolCall{
			ID:        content.ToolID,
			Name:      content.Name,
			Arguments: args,
		})

	case *conversation.ChatMessageContent:
		// Check metadata for tool calls (OpenAI style)
		if message.Metadata != nil {
			if toolCallsData, exists := message.Metadata["tool_calls"]; exists {
				// Try to extract tool calls from metadata
				if toolCallsJson, err := json.Marshal(toolCallsData); err == nil {
					var openaiToolCalls []struct {
						ID       string `json:"id"`
						Type     string `json:"type"`
						Function struct {
							Name      string `json:"name"`
							Arguments string `json:"arguments"`
						} `json:"function"`
					}
					if err := json.Unmarshal(toolCallsJson, &openaiToolCalls); err == nil {
						for _, openaiCall := range openaiToolCalls {
							var args map[string]interface{}
							if err := json.Unmarshal([]byte(openaiCall.Function.Arguments), &args); err != nil {
								args = make(map[string]interface{})
							}

							toolCalls = append(toolCalls, ToolCall{
								ID:        openaiCall.ID,
								Name:      openaiCall.Function.Name,
								Arguments: args,
							})
						}
					}
				}
			}
		}
	}

	return toolCalls
}

// filterToolCalls filters tool calls based on the allowed tool filter
func filterToolCalls(toolCalls []ToolCall, allowedTools []string) []ToolCall {
	if len(allowedTools) == 0 {
		return toolCalls // No filter applied
	}

	allowedSet := make(map[string]bool)
	for _, tool := range allowedTools {
		allowedSet[tool] = true
	}

	var filtered []ToolCall
	for _, toolCall := range toolCalls {
		if allowedSet[toolCall.Name] {
			filtered = append(filtered, toolCall)
		}
	}

	return filtered
}

// executeToolCalls executes all tool calls with timeout handling
func executeToolCalls(ctx context.Context, toolCalls []ToolCall, toolbox Toolbox, timeout time.Duration) ([]ToolResult, error) {
	results := make([]ToolResult, len(toolCalls))

	for i, call := range toolCalls {
		// Create context with timeout for each tool call
		childCtx, cancel := context.WithTimeout(ctx, timeout)

		result, err := toolbox.ExecuteTool(childCtx, call.Name, call.Arguments)
		cancel() // Always cancel the context

		if err != nil {
			results[i] = ToolResult{
				ID:    call.ID,
				Error: err.Error(),
			}
		} else {
			// Convert result to string representation
			resultStr := fmt.Sprintf("%v", result)
			if resultBytes, err := json.Marshal(result); err == nil {
				resultStr = string(resultBytes)
			}

			results[i] = ToolResult{
				ID:      call.ID,
				Content: resultStr,
			}
		}
	}

	return results, nil
}

// MockToolbox provides a simple implementation for testing
type MockToolbox struct {
	tools        map[string]func(context.Context, map[string]interface{}) (interface{}, error)
	descriptions []ToolDescription
}

// NewMockToolbox creates a new mock toolbox for testing
func NewMockToolbox() *MockToolbox {
	return &MockToolbox{
		tools:        make(map[string]func(context.Context, map[string]interface{}) (interface{}, error)),
		descriptions: []ToolDescription{},
	}
}

// RegisterTool registers a tool function with the mock toolbox
func (mt *MockToolbox) RegisterTool(name, description string, parameters map[string]interface{}, fn func(context.Context, map[string]interface{}) (interface{}, error)) {
	mt.tools[name] = fn
	mt.descriptions = append(mt.descriptions, ToolDescription{
		Name:        name,
		Description: description,
		Parameters:  parameters,
	})
}

// ExecuteTool executes a tool with the given name and arguments
func (mt *MockToolbox) ExecuteTool(ctx context.Context, name string, arguments map[string]interface{}) (interface{}, error) {
	fn, exists := mt.tools[name]
	if !exists {
		return nil, fmt.Errorf("tool %q not found", name)
	}
	return fn(ctx, arguments)
}

// GetToolDescriptions returns descriptions of all available tools
func (mt *MockToolbox) GetToolDescriptions() []ToolDescription {
	return mt.descriptions
}

// Ensure MockToolbox implements Toolbox interface
var _ Toolbox = (*MockToolbox)(nil)

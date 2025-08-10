package middleware

import (
    "context"
    "encoding/json"
    "fmt"
    "time"

    "github.com/go-go-golems/geppetto/pkg/conversation"
    "github.com/go-go-golems/geppetto/pkg/turns"
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

// NewToolMiddleware creates middleware that handles function calling workflows for OpenAI/Claude.
// NOTE: Turn-based stub version; tool execution on blocks to be implemented.
func NewToolMiddleware(toolbox Toolbox, config ToolConfig) Middleware {
    return func(next HandlerFunc) HandlerFunc {
        return func(ctx context.Context, t *turns.Turn) (*turns.Turn, error) {
            // Prevent infinite loops
            iterations := 0
            current := t
            for iterations < config.MaxIterations {
                // Run engine step to possibly produce tool_call blocks
                updated, err := next(ctx, current)
                if err != nil {
                    return nil, fmt.Errorf("tool middleware engine step failed: %w", err)
                }

                // Extract pending tool calls
                toolCalls := extractPendingToolCalls(updated)
                if len(toolCalls) == 0 {
                    return updated, nil
                }

                // Filter tools if configured
                if len(config.ToolFilter) > 0 {
                    toolCalls = filterToolCalls(toolCalls, config.ToolFilter)
                }
                if len(toolCalls) == 0 {
                    return updated, nil
                }

                // Execute tool calls with timeout per call
                results, err := executeToolCallsTurn(ctx, toolCalls, toolbox, config.Timeout)
                if err != nil {
                    return nil, fmt.Errorf("tool execution failed: %w", err)
                }

                // Append tool_use blocks to the same turn
                appendToolResultsBlocks(updated, results)

                // Continue loop with the same updated turn to let engine consume tool results
                current = updated
                iterations++
            }
            return current, fmt.Errorf("tool calling exceeded maximum iterations (%d)", config.MaxIterations)
        }
    }
}

// executeToolWorkflowTurns is a minimal Turn-based placeholder that simply delegates to next.
// Full block-level tool execution will be implemented following the design.
func executeToolWorkflowTurns(
    ctx context.Context,
    t *turns.Turn,
    toolbox Toolbox,
    config ToolConfig,
    next HandlerFunc,
) (*turns.Turn, error) {
    // Kept for compatibility; defer to NewToolMiddleware logic above
    return NewToolMiddleware(toolbox, config)(next)(ctx, t)
}

// extractPendingToolCalls finds tool_call blocks without a corresponding tool_use block with same id
func extractPendingToolCalls(t *turns.Turn) []ToolCall {
    if t == nil {
        return nil
    }
    used := make(map[string]bool)
    for _, b := range t.Blocks {
        if b.Kind == turns.BlockKindToolUse {
            if id, ok := b.Payload["id"].(string); ok && id != "" {
                used[id] = true
            }
        }
    }
    var calls []ToolCall
    for _, b := range t.Blocks {
        if b.Kind != turns.BlockKindToolCall {
            continue
        }
        id, _ := b.Payload["id"].(string)
        if id == "" || used[id] {
            continue
        }
        name, _ := b.Payload["name"].(string)
        // args may be an object or json.RawMessage string
        var args map[string]interface{}
        if raw := b.Payload["args"]; raw != nil {
            switch v := raw.(type) {
            case map[string]interface{}:
                args = v
            case string:
                _ = json.Unmarshal([]byte(v), &args)
            case json.RawMessage:
                _ = json.Unmarshal(v, &args)
            default:
                // attempt generic marshal/unmarshal
                if bts, err := json.Marshal(v); err == nil {
                    _ = json.Unmarshal(bts, &args)
                }
            }
        }
        if args == nil {
            args = map[string]interface{}{}
        }
        calls = append(calls, ToolCall{ID: id, Name: name, Arguments: args})
    }
    return calls
}

// executeToolCallsTurn executes the tool calls with per-call timeout
func executeToolCallsTurn(ctx context.Context, calls []ToolCall, toolbox Toolbox, timeout time.Duration) ([]ToolResult, error) {
    results := make([]ToolResult, len(calls))
    for i, call := range calls {
        cctx := ctx
        cancel := func() {}
        if timeout > 0 {
            cctx, cancel = context.WithTimeout(ctx, timeout)
        }
        res, err := toolbox.ExecuteTool(cctx, call.Name, call.Arguments)
        cancel()
        if err != nil {
            results[i] = ToolResult{ID: call.ID, Error: err.Error()}
            continue
        }
        // Try JSON encode first; fallback to fmt
        content := ""
        if b, err := json.Marshal(res); err == nil {
            content = string(b)
        } else {
            content = fmt.Sprintf("%v", res)
        }
        results[i] = ToolResult{ID: call.ID, Content: content}
    }
    return results, nil
}

// appendToolResultsBlocks appends tool_use blocks from results
func appendToolResultsBlocks(t *turns.Turn, results []ToolResult) {
    for _, r := range results {
        payload := map[string]any{"id": r.ID}
        if r.Error != "" {
            payload["result"] = fmt.Sprintf("Error: %s", r.Error)
        } else {
            payload["result"] = r.Content
        }
        turns.AppendBlock(t, turns.Block{Kind: turns.BlockKindToolUse, Payload: payload})
    }
}

// executeToolWorkflow handles the complete tool calling workflow
// Deprecated conversation-based workflow kept for reference but not compiled in Turn mode

// addToolContext adds tool descriptions to the conversation if not already present
func addToolContext(messages conversation.Conversation, toolbox Toolbox) conversation.Conversation {
	// For now, we'll rely on the AI provider (OpenAI/Claude) to handle tool descriptions
	// through their API. This function can be extended later to inject tool context
	// into system messages if needed.
	return messages
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

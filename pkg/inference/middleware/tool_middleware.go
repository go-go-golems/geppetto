package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-go-golems/geppetto/pkg/inference/toolblocks"
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
				toolCalls_ := toolblocks.ExtractPendingToolCalls(updated)
				// adapt to local ToolCall type
				var toolCalls []ToolCall
				for _, c := range toolCalls_ {
					toolCalls = append(toolCalls, ToolCall{ID: c.ID, Name: c.Name, Arguments: c.Arguments})
				}
				if len(toolCalls) == 0 {
					return updated, nil
				}

				// Filter tools: prefer per-Turn agent-mode allowed tools if present, else use static config
				allowed := config.ToolFilter
				if updated != nil {
					if v, ok, err := turns.KeyAgentModeAllowedTools.Get(updated.Data); err != nil {
						return nil, fmt.Errorf("get agentmode allowed tools: %w", err)
					} else if ok {
						allowed = v
					}
				}
				if len(allowed) > 0 {
					toolCalls = filterToolCalls(toolCalls, allowed)
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
				// convert to shared results and append via toolblocks
				var shared []toolblocks.ToolResult
				for _, r := range results {
					shared = append(shared, toolblocks.ToolResult{ID: r.ID, Content: r.Content, Error: r.Error})
				}
				toolblocks.AppendToolResultsBlocks(updated, shared)

				// Continue loop with the same updated turn to let engine consume tool results
				current = updated
				iterations++
			}
			// New Turn semantics: hitting MaxIterations is a soft cap.
			// Return the current turn without error so callers can decide next steps.
			return current, nil
		}
	}
}

// executeToolWorkflowTurns is a minimal Turn-based placeholder that simply delegates to next.
// Full block-level tool execution will be implemented following the design.
// Deprecated: conversation-based tool workflow helper removed in Turn-based path

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
// executeToolCalls kept in conversation mode; Turn path uses executeToolCallsTurn

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

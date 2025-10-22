package tools

import (
    "context"
)

// ToolExecutor handles the execution of tool calls
type ToolExecutor interface {
	ExecuteToolCall(ctx context.Context, toolCall ToolCall, registry ToolRegistry) (*ToolResult, error)
	ExecuteToolCalls(ctx context.Context, toolCalls []ToolCall, registry ToolRegistry) ([]*ToolResult, error)
}

// DefaultToolExecutor wraps BaseToolExecutor with default hooks
type DefaultToolExecutor struct {
    *BaseToolExecutor
}

// NewDefaultToolExecutor creates a new DefaultToolExecutor
func NewDefaultToolExecutor(config ToolConfig) *DefaultToolExecutor {
    base := NewBaseToolExecutor(config)
    return &DefaultToolExecutor{BaseToolExecutor: base}
}

// ExecuteToolCall delegates to BaseToolExecutor
func (e *DefaultToolExecutor) ExecuteToolCall(ctx context.Context, toolCall ToolCall, registry ToolRegistry) (*ToolResult, error) {
    return e.BaseToolExecutor.ExecuteToolCall(ctx, toolCall, registry)
}

// ExecuteToolCalls delegates to BaseToolExecutor
func (e *DefaultToolExecutor) ExecuteToolCalls(ctx context.Context, toolCalls []ToolCall, registry ToolRegistry) ([]*ToolResult, error) {
    return e.BaseToolExecutor.ExecuteToolCalls(ctx, toolCalls, registry)
}

// pow is a simple integer power function to avoid importing math
// pow helper moved to base_executor.go

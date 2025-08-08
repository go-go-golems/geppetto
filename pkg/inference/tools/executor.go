package tools

import (
	"context"
	"fmt"
	"sync"
	"time"
    
    "encoding/json"
    "github.com/go-go-golems/geppetto/pkg/events"
)

// ToolExecutor handles the execution of tool calls
type ToolExecutor interface {
	ExecuteToolCall(ctx context.Context, toolCall ToolCall, registry ToolRegistry) (*ToolResult, error)
	ExecuteToolCalls(ctx context.Context, toolCalls []ToolCall, registry ToolRegistry) ([]*ToolResult, error)
}

// DefaultToolExecutor is the default implementation of ToolExecutor
type DefaultToolExecutor struct {
	config ToolConfig
}

// NewDefaultToolExecutor creates a new DefaultToolExecutor
func NewDefaultToolExecutor(config ToolConfig) *DefaultToolExecutor {
	return &DefaultToolExecutor{
		config: config,
	}
}

// ExecuteToolCall executes a single tool call
func (e *DefaultToolExecutor) ExecuteToolCall(ctx context.Context, toolCall ToolCall, registry ToolRegistry) (*ToolResult, error) {
	start := time.Now()
	
	// Get the tool definition
	toolDef, err := registry.GetTool(toolCall.Name)
	if err != nil {
		return &ToolResult{
			ID:       toolCall.ID,
			Error:    fmt.Sprintf("tool not found: %s", toolCall.Name),
			Duration: time.Since(start),
		}, nil
	}
	
	// Check if tool is allowed
	if !e.config.IsToolAllowed(toolCall.Name) {
		return &ToolResult{
			ID:       toolCall.ID,
			Error:    fmt.Sprintf("tool not allowed: %s", toolCall.Name),
			Duration: time.Since(start),
		}, nil
	}
	
    // Publish ToolCall event prior to execution (best effort)
    argStr := ""
    if len(toolCall.Arguments) > 0 {
        // Compact JSON for logging/event payload
        var tmp interface{}
        if err := json.Unmarshal(toolCall.Arguments, &tmp); err == nil {
            if b, err2 := json.Marshal(tmp); err2 == nil {
                argStr = string(b)
            }
        } else {
            argStr = string(toolCall.Arguments)
        }
    }
    events.PublishEventToContext(ctx, events.NewToolCallExecuteEvent(
        events.EventMetadata{}, nil,
        events.ToolCall{ID: toolCall.ID, Name: toolCall.Name, Input: argStr},
    ))

    // Execute with timeout and retries
	result, err := e.executeWithRetry(ctx, toolCall, toolDef)
	
    result.ID = toolCall.ID
    result.Duration = time.Since(start)

    // Publish ToolResult event
    resultStr := ""
    if result != nil && result.Result != nil {
        if b, err := json.Marshal(result.Result); err == nil {
            resultStr = string(b)
        } else {
            resultStr = fmt.Sprintf("%v", result.Result)
        }
    }
    if result != nil && result.Error != "" {
        // Include error in result payload string for visibility
        if resultStr == "" {
            resultStr = fmt.Sprintf("Error: %s", result.Error)
        } else {
            resultStr = fmt.Sprintf("%s | Error: %s", resultStr, result.Error)
        }
    }
    events.PublishEventToContext(ctx, events.NewToolCallExecutionResultEvent(
        events.EventMetadata{}, nil,
        events.ToolResult{ID: toolCall.ID, Result: resultStr},
    ))

    return result, err
}

// ExecuteToolCalls executes multiple tool calls, potentially in parallel
func (e *DefaultToolExecutor) ExecuteToolCalls(ctx context.Context, toolCalls []ToolCall, registry ToolRegistry) ([]*ToolResult, error) {
	if len(toolCalls) == 0 {
		return nil, nil
	}
	
	if len(toolCalls) == 1 {
		result, err := e.ExecuteToolCall(ctx, toolCalls[0], registry)
		return []*ToolResult{result}, err
	}
	
	// Determine if we should execute in parallel
	maxParallel := e.config.MaxParallelTools
	if maxParallel <= 1 || len(toolCalls) == 1 {
		// Execute sequentially
		return e.executeSequentially(ctx, toolCalls, registry)
	}
	
	// Execute in parallel with limits
	return e.executeInParallel(ctx, toolCalls, registry, maxParallel)
}

// executeSequentially executes tool calls one by one
func (e *DefaultToolExecutor) executeSequentially(ctx context.Context, toolCalls []ToolCall, registry ToolRegistry) ([]*ToolResult, error) {
	results := make([]*ToolResult, len(toolCalls))
	
	for i, toolCall := range toolCalls {
		result, err := e.ExecuteToolCall(ctx, toolCall, registry)
		if err != nil {
			return results, err
		}
		results[i] = result
		
		// Check if we should abort on error
		if result.Error != "" && e.config.ToolErrorHandling == ToolErrorAbort {
			return results, fmt.Errorf("tool execution aborted due to error in %s: %s", toolCall.Name, result.Error)
		}
	}
	
	return results, nil
}

// executeInParallel executes tool calls in parallel with concurrency limits
func (e *DefaultToolExecutor) executeInParallel(ctx context.Context, toolCalls []ToolCall, registry ToolRegistry, maxParallel int) ([]*ToolResult, error) {
	results := make([]*ToolResult, len(toolCalls))
	errors := make([]error, len(toolCalls))
	
	// Use a semaphore to limit concurrency
	sem := make(chan struct{}, maxParallel)
	var wg sync.WaitGroup
	
	for i, toolCall := range toolCalls {
		wg.Add(1)
		
		go func(index int, tc ToolCall) {
			defer wg.Done()
			
			// Acquire semaphore
			sem <- struct{}{}
			defer func() { <-sem }()
			
			result, err := e.ExecuteToolCall(ctx, tc, registry)
			results[index] = result
			errors[index] = err
		}(i, toolCall)
	}
	
	wg.Wait()
	
	// Check for errors
	for i, err := range errors {
		if err != nil {
			return results, err
		}
		
		// Check if we should abort on tool error
		if results[i].Error != "" && e.config.ToolErrorHandling == ToolErrorAbort {
			return results, fmt.Errorf("tool execution aborted due to error in %s: %s", toolCalls[i].Name, results[i].Error)
		}
	}
	
	return results, nil
}

// executeWithRetry executes a tool call with retry logic
func (e *DefaultToolExecutor) executeWithRetry(ctx context.Context, toolCall ToolCall, toolDef *ToolDefinition) (*ToolResult, error) {
	var lastErr error
	retries := 0
	
	for attempt := 0; attempt <= e.config.RetryConfig.MaxRetries; attempt++ {
		if attempt > 0 {
			// Apply backoff
			backoff := time.Duration(float64(e.config.RetryConfig.BackoffBase) * 
				pow(e.config.RetryConfig.BackoffFactor, float64(attempt-1)))
			
			select {
			case <-ctx.Done():
				return &ToolResult{
					Error:   "context cancelled during retry backoff",
					Retries: retries,
				}, ctx.Err()
			case <-time.After(backoff):
				// Continue with retry
			}
			retries++
		}
		
		// Create a timeout context for this execution
		execCtx := ctx
		if e.config.ExecutionTimeout > 0 {
			var cancel context.CancelFunc
			execCtx, cancel = context.WithTimeout(ctx, e.config.ExecutionTimeout)
			defer cancel()
		}
		
		// Execute the tool
		result, err := e.executeTool(execCtx, toolCall, toolDef)
		if err == nil && result.Error == "" {
			// Success
			result.Retries = retries
			return result, nil
		}
		
		if err != nil {
			lastErr = err
		} else {
			lastErr = fmt.Errorf("tool execution error: %s", result.Error)
		}
		
		// Only retry if the error handling policy allows it
		if e.config.ToolErrorHandling != ToolErrorRetry {
			break
		}
	}
	
	// All retries exhausted
	return &ToolResult{
		Error:   fmt.Sprintf("execution failed after %d retries: %v", retries, lastErr),
		Retries: retries,
	}, nil
}

// executeTool performs the actual tool execution
func (e *DefaultToolExecutor) executeTool(ctx context.Context, toolCall ToolCall, toolDef *ToolDefinition) (*ToolResult, error) {
	// Check for context cancellation
	select {
	case <-ctx.Done():
		return &ToolResult{
			Error: "execution cancelled",
		}, ctx.Err()
	default:
	}
	
	// Execute the tool function
	result, err := toolDef.Function.Execute(toolCall.Arguments)
	if err != nil {
		return &ToolResult{
			Error: err.Error(),
		}, nil
	}
	
	return &ToolResult{
		Result: result,
	}, nil
}

// pow is a simple integer power function to avoid importing math
func pow(base float64, exp float64) float64 {
	if exp == 0 {
		return 1
	}
	if exp == 1 {
		return base
	}
	
	result := base
	for i := 1; i < int(exp); i++ {
		result *= base
	}
	return result
}

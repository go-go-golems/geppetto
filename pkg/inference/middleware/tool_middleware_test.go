package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/go-go-golems/geppetto/pkg/conversation"
	"github.com/go-go-golems/geppetto/pkg/turns"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MultiResponseMockEngine provides a mock implementation with multiple responses for testing
type MultiResponseMockEngine struct {
	adders    []func(*turns.Turn)
	callCount int
}

func NewMultiResponseMockEngine(adders ...func(*turns.Turn)) *MultiResponseMockEngine {
	return &MultiResponseMockEngine{adders: adders}
}

func (me *MultiResponseMockEngine) RunInference(ctx context.Context, t *turns.Turn) (*turns.Turn, error) {
	if me.callCount >= len(me.adders) {
		return nil, fmt.Errorf("no more mock responses available")
	}
	adder := me.adders[me.callCount]
	me.callCount++
	if adder != nil {
		adder(t)
	}
	return t, nil
}

func (me *MultiResponseMockEngine) Reset()            { me.callCount = 0 }
func (me *MultiResponseMockEngine) GetCallCount() int { return me.callCount }

// Helper function to create a tool call in message metadata (OpenAI style)
func createMessageWithToolCall(toolID, toolName string, arguments map[string]interface{}) *conversation.Message {
	argsJson, _ := json.Marshal(arguments)

	toolCall := map[string]interface{}{
		"id":   toolID,
		"type": "function",
		"function": map[string]interface{}{
			"name":      toolName,
			"arguments": string(argsJson),
		},
	}

	metadata := map[string]interface{}{
		"tool_calls": []interface{}{toolCall},
	}

	return conversation.NewChatMessage(
		conversation.RoleAssistant,
		fmt.Sprintf("I'll use the %s tool", toolName),
		conversation.WithMetadata(metadata),
	)
}

// Helper function to create a ToolUseContent message (Claude style)
func createToolUseMessage(toolID, toolName string, arguments map[string]interface{}) *conversation.Message {
	argsJson, _ := json.Marshal(arguments)

	content := &conversation.ToolUseContent{
		ToolID: toolID,
		Name:   toolName,
		Input:  argsJson,
		Type:   "function",
	}

	return conversation.NewMessage(content)
}

func TestToolMiddleware_NoToolCalls(t *testing.T) {
	// Create mock engine that returns a simple text response
    mockEngine := NewMultiResponseMockEngine(func(t *turns.Turn) {
        turns.AppendBlock(t, turns.NewAssistantTextBlock("Hello, how can I help you?"))
    })

	// Create mock toolbox
	toolbox := NewMockToolbox()
	toolbox.RegisterTool("calculator", "Performs calculations", map[string]interface{}{},
		func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
			return "42", nil
		})

	// Create tool middleware
	config := DefaultToolConfig()
	middleware := NewToolMiddleware(toolbox, config)
	handler := middleware(engineHandlerFunc(mockEngine))

	// Test with no tool calls
    turn := &turns.Turn{}
    turns.AppendBlock(turn, turns.NewUserTextBlock("Hello"))
	_, err := handler(context.Background(), turn)
	require.NoError(t, err)
	assert.Equal(t, 1, mockEngine.GetCallCount())
}

func TestToolMiddleware_SingleToolCall_OpenAIStyle(t *testing.T) {
	// Create tool call response (OpenAI style)
	// conversation-based fixture no longer used

	// Create final response after tool execution
	// conversation-based fixture no longer used

	mockEngine := NewMultiResponseMockEngine(
		func(t *turns.Turn) {
            args := map[string]any{"operation": "add", "a": 5.0, "b": 3.0}
            turns.AppendBlock(t, turns.NewToolCallBlock("call_123", "calculator", args))
		},
        func(t *turns.Turn) {
            turns.AppendBlock(t, turns.NewAssistantTextBlock("The result is 8"))
        },
	)

	// Create mock toolbox
	toolbox := NewMockToolbox()
	toolbox.RegisterTool("calculator", "Performs calculations", map[string]interface{}{},
		func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
			op := args["operation"].(string)
			a := args["a"].(float64)
			b := args["b"].(float64)

			if op == "add" {
				return a + b, nil
			}
			return nil, fmt.Errorf("unknown operation: %s", op)
		})

	// Create tool middleware
	config := DefaultToolConfig()
	middleware := NewToolMiddleware(toolbox, config)
	handler := middleware(engineHandlerFunc(mockEngine))

	// Test with tool call
    turn := &turns.Turn{}
    turns.AppendBlock(turn, turns.NewUserTextBlock("What is 5 + 3?"))
	_, err := handler(context.Background(), turn)
	require.NoError(t, err)
	assert.Equal(t, 2, mockEngine.GetCallCount())
}

func TestToolMiddleware_SingleToolCall_ClaudeStyle(t *testing.T) {
	// Create tool use response (Claude style)
	// conversation-based fixture no longer used

	// Create final response after tool execution
	// conversation-based fixture no longer used

	mockEngine := NewMultiResponseMockEngine(
		func(t *turns.Turn) {
            args := map[string]any{"location": "San Francisco"}
            turns.AppendBlock(t, turns.NewToolCallBlock("call_456", "weather", args))
		},
        func(t *turns.Turn) {
            turns.AppendBlock(t, turns.NewAssistantTextBlock("The weather in San Francisco is sunny and 72°F"))
        },
	)

	// Create mock toolbox
	toolbox := NewMockToolbox()
	toolbox.RegisterTool("weather", "Gets weather information", map[string]interface{}{},
		func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
			location := args["location"].(string)
			return fmt.Sprintf("Weather in %s: sunny, 72°F", location), nil
		})

	// Create tool middleware
	config := DefaultToolConfig()
	middleware := NewToolMiddleware(toolbox, config)
	handler := middleware(engineHandlerFunc(mockEngine))

	// Test with tool call
    turn := &turns.Turn{}
    turns.AppendBlock(turn, turns.NewUserTextBlock("What's the weather in San Francisco?"))
	_, err := handler(context.Background(), turn)
	require.NoError(t, err)
	assert.Equal(t, 2, mockEngine.GetCallCount())
}

func TestToolMiddleware_MultipleToolCalls(t *testing.T) {
	// Create response with multiple tool calls
	argsJson1, _ := json.Marshal(map[string]interface{}{"a": 5.0, "b": 3.0})
	argsJson2, _ := json.Marshal(map[string]interface{}{"location": "New York"})

	toolCalls := []interface{}{
		map[string]interface{}{
			"id":   "call_1",
			"type": "function",
			"function": map[string]interface{}{
				"name":      "calculator",
				"arguments": string(argsJson1),
			},
		},
		map[string]interface{}{
			"id":   "call_2",
			"type": "function",
			"function": map[string]interface{}{
				"name":      "weather",
				"arguments": string(argsJson2),
			},
		},
	}

	_ = toolCalls // kept for reference; not used in Turn-based tests

	// conversation-based fixture no longer used

	// conversation-based fixture no longer used

	mockEngine := NewMultiResponseMockEngine(
		func(t *turns.Turn) {
            args1 := map[string]any{"a": 5.0, "b": 3.0}
            args2 := map[string]any{"location": "New York"}
            turns.AppendBlock(t, turns.NewToolCallBlock("call_1", "calculator", args1))
            turns.AppendBlock(t, turns.NewToolCallBlock("call_2", "weather", args2))
		},
        func(t *turns.Turn) {
            turns.AppendBlock(t, turns.NewAssistantTextBlock("The calculation result is 8 and the weather in New York is cloudy"))
        },
	)

	// Create mock toolbox
	toolbox := NewMockToolbox()
	toolbox.RegisterTool("calculator", "Performs calculations", map[string]interface{}{},
		func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
			a := args["a"].(float64)
			b := args["b"].(float64)
			return a + b, nil
		})
	toolbox.RegisterTool("weather", "Gets weather information", map[string]interface{}{},
		func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
			location := args["location"].(string)
			return fmt.Sprintf("Weather in %s: cloudy", location), nil
		})

	// Create tool middleware
	config := DefaultToolConfig()
	middleware := NewToolMiddleware(toolbox, config)
	handler := middleware(engineHandlerFunc(mockEngine))

	// Test with multiple tool calls
    turn := &turns.Turn{}
    turns.AppendBlock(turn, turns.NewUserTextBlock("Calculate 5+3 and get weather for NYC"))
	_, err := handler(context.Background(), turn)
	require.NoError(t, err)
	assert.Equal(t, 2, mockEngine.GetCallCount())
}

func TestToolMiddleware_MaxIterationsLimit(t *testing.T) {
	// Create engine that always returns tool calls (infinite loop scenario)
	// conversation-based fixture no longer used

	mockEngine := NewMultiResponseMockEngine(
		func(t *turns.Turn) {
            turns.AppendBlock(t, turns.NewToolCallBlock("call_loop_1", "infinite_tool", map[string]any{"value": 1}))
		},
        func(t *turns.Turn) {
            turns.AppendBlock(t, turns.NewToolCallBlock("call_loop_2", "infinite_tool", map[string]any{"value": 1}))
        },
        func(t *turns.Turn) {
            turns.AppendBlock(t, turns.NewToolCallBlock("call_loop_3", "infinite_tool", map[string]any{"value": 1}))
        },
        func(t *turns.Turn) {
            turns.AppendBlock(t, turns.NewToolCallBlock("call_loop_4", "infinite_tool", map[string]any{"value": 1}))
        },
	)

	// Create mock toolbox
	toolbox := NewMockToolbox()
	toolbox.RegisterTool("infinite_tool", "A tool that creates infinite loops", map[string]interface{}{},
		func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
			return "loop result", nil
		})

	// Create tool middleware with low max iterations
	config := ToolConfig{
		MaxIterations: 3,
		Timeout:       1 * time.Second,
		ToolFilter:    nil,
	}
	middleware := NewToolMiddleware(toolbox, config)
	handler := middleware(engineHandlerFunc(mockEngine))

	// Test that it stops after max iterations
    turn := &turns.Turn{}
    turns.AppendBlock(turn, turns.NewUserTextBlock("Start infinite loop"))
	result, err := handler(context.Background(), turn)
	// New Turn semantics: loop exits successfully when no pending tool calls remain,
	// even if multiple iterations occurred. No error expected here.
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 3, mockEngine.GetCallCount())
}

func TestToolMiddleware_ToolFilter(t *testing.T) {
	// Create tool call for filtered tool
	// conversation-based fixture no longer used

	// Since tool is filtered, it should proceed to final response
	// conversation-based fixture no longer used

	mockEngine := NewMultiResponseMockEngine(
		func(t *turns.Turn) {
            args := map[string]any{"data": "test"}
            turns.AppendBlock(t, turns.NewToolCallBlock("call_filtered", "blocked_tool", args))
		},
        func(t *turns.Turn) {
            turns.AppendBlock(t, turns.NewAssistantTextBlock("I can't use that tool"))
        },
	)

	// Create mock toolbox
	toolbox := NewMockToolbox()
	toolbox.RegisterTool("blocked_tool", "A blocked tool", map[string]interface{}{},
		func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
			return "should not execute", nil
		})
	toolbox.RegisterTool("allowed_tool", "An allowed tool", map[string]interface{}{},
		func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
			return "allowed result", nil
		})

	// Create tool middleware with filter
	config := ToolConfig{
		MaxIterations: 5,
		Timeout:       5 * time.Second,
		ToolFilter:    []string{"allowed_tool"}, // Only allow specific tool
	}
	middleware := NewToolMiddleware(toolbox, config)
	handler := middleware(engineHandlerFunc(mockEngine))

	// Test with filtered tool call
    turn := &turns.Turn{}
    turns.AppendBlock(turn, turns.NewUserTextBlock("Use blocked tool"))
	_, err := handler(context.Background(), turn)
	require.NoError(t, err)
	assert.Equal(t, 1, mockEngine.GetCallCount())
}

func TestToolMiddleware_ToolExecutionError(t *testing.T) {
	// Create tool call response
	// conversation-based fixture no longer used

	// Create final response after tool error
	// conversation-based fixture no longer used

	mockEngine := NewMultiResponseMockEngine(
		func(t *turns.Turn) {
            turns.AppendBlock(t, turns.NewToolCallBlock("call_error", "error_tool", map[string]any{"should_error": true}))
		},
        func(t *turns.Turn) {
            turns.AppendBlock(t, turns.NewAssistantTextBlock("There was an error with the tool"))
        },
	)

	// Create mock toolbox with error tool
	toolbox := NewMockToolbox()
	toolbox.RegisterTool("error_tool", "A tool that errors", map[string]interface{}{},
		func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
			return nil, fmt.Errorf("simulated tool error")
		})

	// Create tool middleware
	config := DefaultToolConfig()
	middleware := NewToolMiddleware(toolbox, config)
	handler := middleware(engineHandlerFunc(mockEngine))

	// Test with error tool call
    turn := &turns.Turn{}
    turns.AppendBlock(turn, turns.NewUserTextBlock("Use error tool"))
	_, err := handler(context.Background(), turn)
	require.NoError(t, err)
	assert.Equal(t, 2, mockEngine.GetCallCount())
}

func TestToolMiddleware_TimeoutHandling(t *testing.T) {
	// Create tool call response
	// conversation-based fixture no longer used

	// Create final response after tool timeout
	// conversation-based fixture no longer used

	mockEngine := NewMultiResponseMockEngine(
		func(t *turns.Turn) {
            turns.AppendBlock(t, turns.NewToolCallBlock("call_timeout", "slow_tool", map[string]any{"delay": 2.0}))
		},
        func(t *turns.Turn) {
            turns.AppendBlock(t, turns.NewAssistantTextBlock("The tool timed out"))
        },
	)

	// Create mock toolbox with slow tool
	toolbox := NewMockToolbox()
	toolbox.RegisterTool("slow_tool", "A slow tool", map[string]interface{}{},
		func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
			delay := time.Duration(args["delay"].(float64)) * time.Second

			select {
			case <-time.After(delay):
				return "slow result", nil
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		})

	// Create tool middleware with short timeout
	config := ToolConfig{
		MaxIterations: 5,
		Timeout:       100 * time.Millisecond, // Very short timeout
		ToolFilter:    nil,
	}
	middleware := NewToolMiddleware(toolbox, config)
	handler := middleware(engineHandlerFunc(mockEngine))

	// Test with slow tool call
    turn := &turns.Turn{}
    turns.AppendBlock(turn, turns.NewUserTextBlock("Use slow tool"))
	_, err := handler(context.Background(), turn)
	require.NoError(t, err)
}

func TestExtractToolCalls_OpenAIStyle(t *testing.T) {
	// Test OpenAI style tool calls in metadata
	toolCall := createMessageWithToolCall("test_id", "test_tool", map[string]interface{}{
		"param1": "value1",
		"param2": 42,
	})

	extracted := extractToolCalls(toolCall)

	require.Len(t, extracted, 1)
	assert.Equal(t, "test_id", extracted[0].ID)
	assert.Equal(t, "test_tool", extracted[0].Name)
	assert.Equal(t, "value1", extracted[0].Arguments["param1"])
	assert.Equal(t, float64(42), extracted[0].Arguments["param2"]) // JSON unmarshaling converts to float64
}

func TestExtractToolCalls_ClaudeStyle(t *testing.T) {
	// Test Claude style tool use content
	toolUse := createToolUseMessage("claude_id", "claude_tool", map[string]interface{}{
		"location": "Paris",
		"units":    "metric",
	})

	extracted := extractToolCalls(toolUse)

	require.Len(t, extracted, 1)
	assert.Equal(t, "claude_id", extracted[0].ID)
	assert.Equal(t, "claude_tool", extracted[0].Name)
	assert.Equal(t, "Paris", extracted[0].Arguments["location"])
	assert.Equal(t, "metric", extracted[0].Arguments["units"])
}

func TestExtractToolCalls_NoToolCalls(t *testing.T) {
	// Test regular chat message with no tool calls
	regularMessage := conversation.NewChatMessage(
		conversation.RoleAssistant,
		"This is a regular response with no tool calls",
	)

	extracted := extractToolCalls(regularMessage)
	assert.Len(t, extracted, 0)
}

func TestFilterToolCalls(t *testing.T) {
	toolCalls := []ToolCall{
		{ID: "1", Name: "allowed_tool1", Arguments: map[string]interface{}{}},
		{ID: "2", Name: "blocked_tool", Arguments: map[string]interface{}{}},
		{ID: "3", Name: "allowed_tool2", Arguments: map[string]interface{}{}},
	}

	// Test with filter
	filtered := filterToolCalls(toolCalls, []string{"allowed_tool1", "allowed_tool2"})
	require.Len(t, filtered, 2)
	assert.Equal(t, "allowed_tool1", filtered[0].Name)
	assert.Equal(t, "allowed_tool2", filtered[1].Name)

	// Test with empty filter (should return all)
	allFiltered := filterToolCalls(toolCalls, []string{})
	assert.Len(t, allFiltered, 3)

	// Test with no matches
	noneFiltered := filterToolCalls(toolCalls, []string{"nonexistent_tool"})
	assert.Len(t, noneFiltered, 0)
}

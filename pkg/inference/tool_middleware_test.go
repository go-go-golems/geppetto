package inference

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/go-go-golems/geppetto/pkg/conversation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MultiResponseMockEngine provides a mock implementation with multiple responses for testing
type MultiResponseMockEngine struct {
	responses []*conversation.Message
	callCount int
}

// NewMultiResponseMockEngine creates a new mock engine with predefined responses
func NewMultiResponseMockEngine(responses ...*conversation.Message) *MultiResponseMockEngine {
	return &MultiResponseMockEngine{
		responses: responses,
		callCount: 0,
	}
}

// RunInference implements the Engine interface
func (me *MultiResponseMockEngine) RunInference(ctx context.Context, messages conversation.Conversation) (*conversation.Message, error) {
	if me.callCount >= len(me.responses) {
		return nil, fmt.Errorf("no more mock responses available")
	}

	response := me.responses[me.callCount]
	me.callCount++
	return response, nil
}

// Reset resets the mock engine's call count
func (me *MultiResponseMockEngine) Reset() {
	me.callCount = 0
}

// GetCallCount returns the number of times RunInference was called
func (me *MultiResponseMockEngine) GetCallCount() int {
	return me.callCount
}

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
	textResponse := conversation.NewChatMessage(
		conversation.RoleAssistant,
		"Hello, how can I help you?",
	)
	mockEngine := NewMultiResponseMockEngine(textResponse)

	// Create mock toolbox
	toolbox := NewMockToolbox()
	toolbox.RegisterTool("calculator", "Performs calculations", map[string]interface{}{},
		func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
			return "42", nil
		})

	// Create tool middleware
	config := DefaultToolConfig()
	middleware := NewToolMiddleware(toolbox, config)
	handler := middleware(EngineHandler(mockEngine))

	// Test with no tool calls
	messages := conversation.NewConversation(
		conversation.NewChatMessage(conversation.RoleUser, "Hello"),
	)

	result, err := handler(context.Background(), messages)

	require.NoError(t, err)
	assert.Len(t, result, 2) // Original user message + AI response
	assert.Equal(t, "Hello, how can I help you?", result[1].Content.(*conversation.ChatMessageContent).Text)
	assert.Equal(t, 1, mockEngine.GetCallCount())
}

func TestToolMiddleware_SingleToolCall_OpenAIStyle(t *testing.T) {
	// Create tool call response (OpenAI style)
	toolCallResponse := createMessageWithToolCall("call_123", "calculator", map[string]interface{}{
		"operation": "add",
		"a":         5.0,
		"b":         3.0,
	})

	// Create final response after tool execution
	finalResponse := conversation.NewChatMessage(
		conversation.RoleAssistant,
		"The result is 8",
	)

	mockEngine := NewMultiResponseMockEngine(toolCallResponse, finalResponse)

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
	handler := middleware(EngineHandler(mockEngine))

	// Test with tool call
	messages := conversation.NewConversation(
		conversation.NewChatMessage(conversation.RoleUser, "What is 5 + 3?"),
	)

	result, err := handler(context.Background(), messages)

	require.NoError(t, err)
	assert.Len(t, result, 4)                      // User + AI tool call + tool result + final AI response
	assert.Equal(t, 2, mockEngine.GetCallCount()) // Called twice: once for tool call, once for final response

	// Check tool result message
	toolResultMsg := result[2]
	toolResult, ok := toolResultMsg.Content.(*conversation.ToolResultContent)
	require.True(t, ok)
	assert.Equal(t, "call_123", toolResult.ToolID)
	assert.Contains(t, toolResult.Result, "8") // Result contains the calculation
}

func TestToolMiddleware_SingleToolCall_ClaudeStyle(t *testing.T) {
	// Create tool use response (Claude style)
	toolUseResponse := createToolUseMessage("call_456", "weather", map[string]interface{}{
		"location": "San Francisco",
	})

	// Create final response after tool execution
	finalResponse := conversation.NewChatMessage(
		conversation.RoleAssistant,
		"The weather in San Francisco is sunny and 72°F",
	)

	mockEngine := NewMultiResponseMockEngine(toolUseResponse, finalResponse)

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
	handler := middleware(EngineHandler(mockEngine))

	// Test with tool call
	messages := conversation.NewConversation(
		conversation.NewChatMessage(conversation.RoleUser, "What's the weather in San Francisco?"),
	)

	result, err := handler(context.Background(), messages)

	require.NoError(t, err)
	assert.Len(t, result, 4) // User + AI tool use + tool result + final AI response
	assert.Equal(t, 2, mockEngine.GetCallCount())

	// Check tool result message
	toolResultMsg := result[2]
	toolResult, ok := toolResultMsg.Content.(*conversation.ToolResultContent)
	require.True(t, ok)
	assert.Equal(t, "call_456", toolResult.ToolID)
	assert.Contains(t, toolResult.Result, "sunny")
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

	metadata := map[string]interface{}{
		"tool_calls": toolCalls,
	}

	toolCallResponse := conversation.NewChatMessage(
		conversation.RoleAssistant,
		"I'll get the calculation and weather for you",
		conversation.WithMetadata(metadata),
	)

	finalResponse := conversation.NewChatMessage(
		conversation.RoleAssistant,
		"The calculation result is 8 and the weather in New York is cloudy",
	)

	mockEngine := NewMultiResponseMockEngine(toolCallResponse, finalResponse)

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
	handler := middleware(EngineHandler(mockEngine))

	// Test with multiple tool calls
	messages := conversation.NewConversation(
		conversation.NewChatMessage(conversation.RoleUser, "Calculate 5+3 and get weather for NYC"),
	)

	result, err := handler(context.Background(), messages)

	require.NoError(t, err)
	assert.Len(t, result, 5) // User + AI tool calls + 2 tool results + final AI response
	assert.Equal(t, 2, mockEngine.GetCallCount())

	// Check both tool result messages
	toolResult1 := result[2].Content.(*conversation.ToolResultContent)
	toolResult2 := result[3].Content.(*conversation.ToolResultContent)

	assert.Equal(t, "call_1", toolResult1.ToolID)
	assert.Equal(t, "call_2", toolResult2.ToolID)
	assert.Contains(t, toolResult1.Result, "8")
	assert.Contains(t, toolResult2.Result, "cloudy")
}

func TestToolMiddleware_MaxIterationsLimit(t *testing.T) {
	// Create engine that always returns tool calls (infinite loop scenario)
	toolCallResponse := createMessageWithToolCall("call_loop", "infinite_tool", map[string]interface{}{
		"value": 1,
	})

	mockEngine := NewMultiResponseMockEngine()
	// Set up multiple identical responses to simulate infinite loop
	for i := 0; i < 10; i++ {
		mockEngine.responses = append(mockEngine.responses, toolCallResponse)
	}

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
	handler := middleware(EngineHandler(mockEngine))

	// Test that it stops after max iterations
	messages := conversation.NewConversation(
		conversation.NewChatMessage(conversation.RoleUser, "Start infinite loop"),
	)

	result, err := handler(context.Background(), messages)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeded maximum iterations (3)")
	assert.Nil(t, result)
	assert.Equal(t, 3, mockEngine.GetCallCount()) // Should stop after 3 iterations
}

func TestToolMiddleware_ToolFilter(t *testing.T) {
	// Create tool call for filtered tool
	toolCallResponse := createMessageWithToolCall("call_filtered", "blocked_tool", map[string]interface{}{
		"data": "test",
	})

	// Since tool is filtered, it should proceed to final response
	finalResponse := conversation.NewChatMessage(
		conversation.RoleAssistant,
		"I can't use that tool",
	)

	mockEngine := NewMultiResponseMockEngine(toolCallResponse, finalResponse)

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
	handler := middleware(EngineHandler(mockEngine))

	// Test with filtered tool call
	messages := conversation.NewConversation(
		conversation.NewChatMessage(conversation.RoleUser, "Use blocked tool"),
	)

	result, err := handler(context.Background(), messages)

	require.NoError(t, err)
	assert.Len(t, result, 2)                      // User + AI response (no tool execution)
	assert.Equal(t, 1, mockEngine.GetCallCount()) // Only called once since tool was filtered
}

func TestToolMiddleware_ToolExecutionError(t *testing.T) {
	// Create tool call response
	toolCallResponse := createMessageWithToolCall("call_error", "error_tool", map[string]interface{}{
		"should_error": true,
	})

	// Create final response after tool error
	finalResponse := conversation.NewChatMessage(
		conversation.RoleAssistant,
		"There was an error with the tool",
	)

	mockEngine := NewMultiResponseMockEngine(toolCallResponse, finalResponse)

	// Create mock toolbox with error tool
	toolbox := NewMockToolbox()
	toolbox.RegisterTool("error_tool", "A tool that errors", map[string]interface{}{},
		func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
			return nil, fmt.Errorf("simulated tool error")
		})

	// Create tool middleware
	config := DefaultToolConfig()
	middleware := NewToolMiddleware(toolbox, config)
	handler := middleware(EngineHandler(mockEngine))

	// Test with error tool call
	messages := conversation.NewConversation(
		conversation.NewChatMessage(conversation.RoleUser, "Use error tool"),
	)

	result, err := handler(context.Background(), messages)

	require.NoError(t, err)
	assert.Len(t, result, 4) // User + AI tool call + tool error result + final AI response
	assert.Equal(t, 2, mockEngine.GetCallCount())

	// Check tool result contains error
	toolResultMsg := result[2]
	toolResult, ok := toolResultMsg.Content.(*conversation.ToolResultContent)
	require.True(t, ok)
	assert.Equal(t, "call_error", toolResult.ToolID)
	assert.Contains(t, toolResult.Result, "Error: simulated tool error")
}

func TestToolMiddleware_TimeoutHandling(t *testing.T) {
	// Create tool call response
	toolCallResponse := createMessageWithToolCall("call_timeout", "slow_tool", map[string]interface{}{
		"delay": 2,
	})

	// Create final response after tool timeout
	finalResponse := conversation.NewChatMessage(
		conversation.RoleAssistant,
		"The tool timed out",
	)

	mockEngine := NewMultiResponseMockEngine(toolCallResponse, finalResponse)

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
	handler := middleware(EngineHandler(mockEngine))

	// Test with slow tool call
	messages := conversation.NewConversation(
		conversation.NewChatMessage(conversation.RoleUser, "Use slow tool"),
	)

	result, err := handler(context.Background(), messages)

	require.NoError(t, err)
	assert.Len(t, result, 4) // User + AI tool call + tool timeout result + final AI response
	assert.Equal(t, 2, mockEngine.GetCallCount())

	// Check tool result contains timeout error
	toolResultMsg := result[2]
	toolResult, ok := toolResultMsg.Content.(*conversation.ToolResultContent)
	require.True(t, ok)
	assert.Equal(t, "call_timeout", toolResult.ToolID)
	assert.Contains(t, toolResult.Result, "Error:")
	assert.Contains(t, toolResult.Result, "context deadline exceeded")
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

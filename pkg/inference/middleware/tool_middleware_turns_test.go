package middleware

import (
	"context"
	"testing"
	"time"

	"github.com/go-go-golems/geppetto/pkg/inference/toolblocks"
	"github.com/go-go-golems/geppetto/pkg/turns"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockNextHandler simulates an engine step:
// - If no tool_call exists yet, it appends a tool_call for tool "echo" with id "call_1".
// - If a tool_use for id "call_1" exists, it appends an assistant llm_text block.
func mockNextHandler() HandlerFunc {
	return func(ctx context.Context, t *turns.Turn) (*turns.Turn, error) {
		hasToolCall := false
		hasToolUse := false
		for _, b := range t.Blocks {
			if b.Kind == turns.BlockKindToolCall {
				hasToolCall = true
			}
			if b.Kind == turns.BlockKindToolUse {
				hasToolUse = true
			}
		}

		if !hasToolCall {
			turns.AppendBlock(t, turns.NewToolCallBlock("call_1", "echo", map[string]any{"text": "hello"}))
			return t, nil
		}

		if hasToolUse {
			turns.AppendBlock(t, turns.NewAssistantTextBlock("done"))
		}
		return t, nil
	}
}

func TestExecuteAndAppendToolResults_Turns(t *testing.T) {
	// Prepare toolbox with echo
	tb := NewMockToolbox()
	tb.RegisterTool("echo", "Echo tool", map[string]any{"text": map[string]any{"type": "string"}}, func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
		return args["text"], nil
	})

	// Build a turn with a tool_call
	turn := &turns.Turn{}
	turns.AppendBlock(turn, turns.NewToolCallBlock("call_1", "echo", map[string]any{"text": "hi"}))

	pending := toolblocks.ExtractPendingToolCalls(turn)
	require.Len(t, pending, 1)
	calls := []ToolCall{{ID: pending[0].ID, Name: pending[0].Name, Arguments: pending[0].Arguments}}

	results, err := executeToolCallsTurn(context.Background(), calls, tb, 2*time.Second)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "call_1", results[0].ID)
	assert.Empty(t, results[0].Error)
	assert.Contains(t, results[0].Content, "hi")

	var shared []toolblocks.ToolResult
	for _, r := range results {
		shared = append(shared, toolblocks.ToolResult{ID: r.ID, Content: r.Content, Error: r.Error})
	}
	toolblocks.AppendToolResultsBlocks(turn, shared)
	// Expect a tool_use block appended
	foundToolUse := false
	for _, b := range turn.Blocks {
		if b.Kind == turns.BlockKindToolUse {
			foundToolUse = true
			assert.Equal(t, "call_1", b.Payload[turns.PayloadKeyID])
			break
		}
	}
	assert.True(t, foundToolUse)
}

func TestToolMiddleware_EndToEnd_Turns(t *testing.T) {
	// Toolbox with echo implementation
	tb := NewMockToolbox()
	tb.RegisterTool("echo", "Echo tool", map[string]any{"text": map[string]any{"type": "string"}}, func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
		return args["text"], nil
	})

	mw := NewToolMiddleware(tb, ToolConfig{MaxIterations: 3, Timeout: 1 * time.Second})
	handler := mw(mockNextHandler())

	// Seed with a user block
	turn := &turns.Turn{}
	turns.AppendBlock(turn, turns.NewUserTextBlock("please echo hello"))

	updated, err := handler(context.Background(), turn)
	require.NoError(t, err)

	// Expect tool_call -> tool_use -> llm_text in any order preserving logical sequence
	hasCall := false
	hasUse := false
	hasText := false
	for _, b := range updated.Blocks {
		switch b.Kind {
		case turns.BlockKindToolCall:
			hasCall = true
		case turns.BlockKindToolUse:
			hasUse = true
		case turns.BlockKindLLMText:
			hasText = true
		case turns.BlockKindUser, turns.BlockKindSystem, turns.BlockKindReasoning, turns.BlockKindOther:
			// ignore
		}
	}
	assert.True(t, hasCall)
	assert.True(t, hasUse)
	assert.True(t, hasText)
}

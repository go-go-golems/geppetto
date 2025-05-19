package claude

import (
	"github.com/go-go-golems/geppetto/pkg/events"
	"testing"

	"github.com/go-go-golems/geppetto/pkg/steps"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/claude/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContentBlockMerger(t *testing.T) {
	toolCallResult := "{\"operation\": \"add\", \"a\": 5, \"b\": 3}"
	finalToolCallText := "Here's the result: Tool Call: calculator\nID: tool_1\n" + toolCallResult + " is the sum."
	tests := []struct {
		name           string
		events         []api.StreamingEvent
		expectedEvents []events.Event
		expectedError  string
		checkMetadata  func(*testing.T, map[string]interface{})
		checkResponse  func(*testing.T, *api.MessageResponse)
	}{
		{
			name: "Test NewContentBlockMerger initialization",
			events: []api.StreamingEvent{
				{Type: api.MessageStartType, Message: &api.MessageResponse{}},
			},
			expectedEvents: []events.Event{
				events.NewStartEvent(events.EventMetadata{}, &steps.StepMetadata{}),
			},
		},
		{
			name: "Test Add method with MessageStartType event",
			events: []api.StreamingEvent{
				{
					Type: api.MessageStartType,
					Message: &api.MessageResponse{
						Model: "claude-2",
						Usage: api.Usage{InputTokens: 10, OutputTokens: 20},
						ID:    "msg_123",
						Role:  "assistant",
					},
				},
			},
			expectedEvents: []events.Event{
				events.NewStartEvent(events.EventMetadata{}, &steps.StepMetadata{}),
			},
			checkMetadata: func(t *testing.T, metadata map[string]interface{}) {
				assert.Equal(t, "claude-2", metadata[ModelMetadataSlug])
				assert.Equal(t, "msg_123", metadata[MessageIdMetadataSlug])
				assert.Equal(t, "assistant", metadata[RoleMetadataSlug])
				assert.NotContains(t, metadata, StopReasonMetadataSlug)
				assert.NotContains(t, metadata, StopSequenceMetadataSlug)
			},
		},
		{
			name: "Test Add method with MessageStopType event (with stop reason)",
			events: []api.StreamingEvent{
				{Type: api.MessageStartType, Message: &api.MessageResponse{}},
				{
					Type: api.MessageStopType,
					Message: &api.MessageResponse{
						StopReason: "end_turn",
						Usage:      api.Usage{InputTokens: 15, OutputTokens: 25},
					},
				},
			},
			expectedEvents: []events.Event{
				events.NewStartEvent(events.EventMetadata{}, &steps.StepMetadata{}),
				events.NewFinalEvent(events.EventMetadata{}, &steps.StepMetadata{}, ""),
			},
			checkMetadata: func(t *testing.T, metadata map[string]interface{}) {
				assert.Equal(t, "end_turn", metadata[StopReasonMetadataSlug])
			},
		},
		{
			name: "Test single text content block",
			events: []api.StreamingEvent{
				{Type: api.MessageStartType, Message: &api.MessageResponse{}},
				{
					Type:         api.ContentBlockStartType,
					Index:        0,
					ContentBlock: &api.ContentBlock{Type: api.ContentTypeText},
				},
				{
					Type:  api.ContentBlockDeltaType,
					Index: 0,
					Delta: &api.Delta{
						Type: api.TextDeltaType,
						Text: "Hello, ",
					},
				},
				{
					Type:  api.ContentBlockDeltaType,
					Index: 0,
					Delta: &api.Delta{
						Type: api.TextDeltaType,
						Text: "world!",
					},
				},
				{
					Type:  api.ContentBlockStopType,
					Index: 0,
				},
				{Type: api.MessageStopType,
					Message: &api.MessageResponse{
						Usage:      api.Usage{InputTokens: 5, OutputTokens: 10},
						StopReason: "end_turn",
					},
				},
			},
			expectedEvents: []events.Event{
				events.NewStartEvent(events.EventMetadata{}, &steps.StepMetadata{}),
				events.NewPartialCompletionEvent(events.EventMetadata{}, &steps.StepMetadata{}, "Hello, ", "Hello, "),
				events.NewPartialCompletionEvent(events.EventMetadata{}, &steps.StepMetadata{}, "world!", "Hello, world!"),
				events.NewPartialCompletionEvent(events.EventMetadata{}, &steps.StepMetadata{}, "", "Hello, world!"),
				events.NewFinalEvent(events.EventMetadata{}, &steps.StepMetadata{}, "Hello, world!"),
			},
			checkResponse: func(t *testing.T, response *api.MessageResponse) {
				assert.Len(t, response.Content, 1)
				assert.Equal(t, api.ContentTypeText, response.Content[0].Type())
				assert.Equal(t, "Hello, world!", response.Content[0].(api.TextContent).Text)
			},
		},
		{
			name: "Test multiple content blocks (text and tool use)",
			events: []api.StreamingEvent{
				{Type: api.MessageStartType, Message: &api.MessageResponse{}},
				{
					Type:         api.ContentBlockStartType,
					Index:        0,
					ContentBlock: &api.ContentBlock{Type: api.ContentTypeText},
				},
				{
					Type:  api.ContentBlockDeltaType,
					Index: 0,
					Delta: &api.Delta{
						Type: api.TextDeltaType,
						Text: "Here's the result: ",
					},
				},
				{
					Type:  api.ContentBlockStopType,
					Index: 0,
				},
				{
					Type:  api.ContentBlockStartType,
					Index: 1,
					ContentBlock: &api.ContentBlock{
						Type: api.ContentTypeToolUse,
						ID:   "tool_1",
						Name: "calculator",
					},
				},
				{
					Type:  api.ContentBlockDeltaType,
					Index: 1,
					Delta: &api.Delta{
						Type:        api.InputJSONDeltaType,
						PartialJSON: toolCallResult,
					},
				},
				{
					Type:  api.ContentBlockStopType,
					Index: 1,
				},
				{
					Type:         api.ContentBlockStartType,
					Index:        2,
					ContentBlock: &api.ContentBlock{Type: api.ContentTypeText},
				},
				{
					Type:  api.ContentBlockDeltaType,
					Index: 2,
					Delta: &api.Delta{
						Type: api.TextDeltaType,
						Text: " is the sum.",
					},
				},
				{
					Type:  api.ContentBlockStopType,
					Index: 2,
				},
				{Type: api.MessageStopType,
					Message: &api.MessageResponse{
						Usage:      api.Usage{InputTokens: 5, OutputTokens: 10},
						StopReason: "end_turn",
					},
				},
			},
			expectedEvents: []events.Event{
				events.NewStartEvent(events.EventMetadata{}, &steps.StepMetadata{}),
				events.NewPartialCompletionEvent(events.EventMetadata{}, &steps.StepMetadata{}, "Here's the result: ", "Here's the result: "),
				events.NewPartialCompletionEvent(events.EventMetadata{}, &steps.StepMetadata{}, "", "Here's the result: "),
				events.NewToolCallEvent(events.EventMetadata{}, &steps.StepMetadata{}, events.ToolCall{
					ID:    "tool_1",
					Name:  "calculator",
					Input: toolCallResult,
				}),
				events.NewPartialCompletionEvent(events.EventMetadata{}, &steps.StepMetadata{}, " is the sum.", finalToolCallText),
				events.NewPartialCompletionEvent(events.EventMetadata{}, &steps.StepMetadata{}, "", finalToolCallText),
				events.NewFinalEvent(events.EventMetadata{}, &steps.StepMetadata{}, finalToolCallText),
			},
			checkResponse: func(t *testing.T, response *api.MessageResponse) {
				assert.Len(t, response.Content, 3)
				assert.Equal(t, api.ContentTypeText, response.Content[0].Type())
				assert.Equal(t, "Here's the result: ", response.Content[0].(api.TextContent).Text)
				assert.Equal(t, api.ContentTypeToolUse, response.Content[1].Type())
				toolUseContent := response.Content[1].(api.ToolUseContent)
				assert.Equal(t, "tool_1", toolUseContent.ID)
				assert.Equal(t, "calculator", toolUseContent.Name)
				assert.Equal(t, toolCallResult, toolUseContent.Input)
				assert.Equal(t, api.ContentTypeText, response.Content[2].Type())
				assert.Equal(t, " is the sum.", response.Content[2].(api.TextContent).Text)
			},
		},
		{
			name: "Test server tool use and web search result blocks",
			events: []api.StreamingEvent{
				{Type: api.MessageStartType, Message: &api.MessageResponse{}},
				{
					Type:         api.ContentBlockStartType,
					Index:        0,
					ContentBlock: &api.ContentBlock{Type: api.ContentTypeServerToolUse, ID: "search_1", Name: "web_search"},
				},
				{
					Type:  api.ContentBlockDeltaType,
					Index: 0,
					Delta: &api.Delta{Type: api.InputJSONDeltaType, PartialJSON: "{\"query\":\"golang\"}"},
				},
				{Type: api.ContentBlockStopType, Index: 0},
				{
					Type:         api.ContentBlockStartType,
					Index:        1,
					ContentBlock: &api.ContentBlock{Type: api.ContentTypeWebSearchToolResult, ToolUseID: "search_1"},
				},
				{
					Type:  api.ContentBlockDeltaType,
					Index: 1,
					Delta: &api.Delta{Type: api.InputJSONDeltaType, PartialJSON: "[{\"snippet\":\"result\"}]"},
				},
				{Type: api.ContentBlockStopType, Index: 1},
				{Type: api.MessageStopType, Message: &api.MessageResponse{}},
			},
			expectedEvents: []events.Event{
				events.NewStartEvent(events.EventMetadata{}, &steps.StepMetadata{}),
				events.NewToolCallEvent(events.EventMetadata{}, &steps.StepMetadata{}, events.ToolCall{ID: "search_1", Name: "web_search", Input: "{\"query\":\"golang\"}"}),
				events.NewToolResultEvent(events.EventMetadata{}, &steps.StepMetadata{}, events.ToolResult{ID: "search_1", Result: "[{\"snippet\":\"result\"}]"}),
				events.NewFinalEvent(events.EventMetadata{}, &steps.StepMetadata{}, ""),
			},
			checkResponse: func(t *testing.T, response *api.MessageResponse) {
				require.Len(t, response.Content, 2)
				stuc := response.Content[0].(api.ServerToolUseContent)
				assert.Equal(t, "search_1", stuc.ID)
				assert.Equal(t, "web_search", stuc.Name)
				assert.Equal(t, "{\"query\":\"golang\"}", stuc.Input)
				wsrc := response.Content[1].(api.WebSearchToolResultContent)
				assert.Equal(t, "search_1", wsrc.ToolUseID)
				assert.Equal(t, "[{\"snippet\":\"result\"}]", string(wsrc.Content))
			},
		},
		{
			name: "Test thinking block",
			events: []api.StreamingEvent{
				{Type: api.MessageStartType, Message: &api.MessageResponse{}},
				{
					Type:         api.ContentBlockStartType,
					Index:        0,
					ContentBlock: &api.ContentBlock{Type: api.ContentTypeThinking},
				},
				{
					Type:  api.ContentBlockDeltaType,
					Index: 0,
					Delta: &api.Delta{Type: api.SignatureDeltaType, Signature: "sig1"},
				},
				{
					Type:  api.ContentBlockDeltaType,
					Index: 0,
					Delta: &api.Delta{Type: api.ThinkingDeltaType, Thinking: "pondering"},
				},
				{Type: api.ContentBlockStopType, Index: 0},
				{Type: api.MessageStopType, Message: &api.MessageResponse{}},
			},
			expectedEvents: []events.Event{
				events.NewStartEvent(events.EventMetadata{}, &steps.StepMetadata{}),
				events.NewFinalEvent(events.EventMetadata{}, &steps.StepMetadata{}, ""),
			},
			checkResponse: func(t *testing.T, response *api.MessageResponse) {
				require.Len(t, response.Content, 1)
				tc := response.Content[0].(api.ThinkingContent)
				assert.Equal(t, "sig1", tc.Signature)
				assert.Equal(t, "pondering", tc.Thinking)
			},
		},
		{
			name: "Test redacted thinking block",
			events: []api.StreamingEvent{
				{Type: api.MessageStartType, Message: &api.MessageResponse{}},
				{
					Type:         api.ContentBlockStartType,
					Index:        0,
					ContentBlock: &api.ContentBlock{Type: api.ContentTypeRedactedThinking},
				},
				{
					Type:  api.ContentBlockDeltaType,
					Index: 0,
					Delta: &api.Delta{Type: api.TextDeltaType, Text: "[redacted]"},
				},
				{Type: api.ContentBlockStopType, Index: 0},
				{Type: api.MessageStopType, Message: &api.MessageResponse{}},
			},
			expectedEvents: []events.Event{
				events.NewStartEvent(events.EventMetadata{}, &steps.StepMetadata{}),
				events.NewPartialCompletionEvent(events.EventMetadata{}, &steps.StepMetadata{}, "[redacted]", "[redacted]"),
				events.NewFinalEvent(events.EventMetadata{}, &steps.StepMetadata{}, ""),
			},
			checkResponse: func(t *testing.T, response *api.MessageResponse) {
				require.Len(t, response.Content, 1)
				rtc := response.Content[0].(api.RedactedThinkingContent)
				assert.Equal(t, "[redacted]", rtc.Data)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metadata := events.EventMetadata{}
			stepMetadata := &steps.StepMetadata{
				Metadata: make(map[string]interface{}),
			}
			merger := NewContentBlockMerger(metadata, stepMetadata)

			var events_ []events.Event
			var err error

			for _, event := range tt.events {
				newEvents, newErr := merger.Add(event)
				events_ = append(events_, newEvents...)
				if newErr != nil {
					err = newErr
					break
				}
			}

			if tt.expectedError != "" {
				assert.EqualError(t, err, tt.expectedError)
			} else {
				assert.NoError(t, err)
				require.Equal(t, len(tt.expectedEvents), len(events_), "Number of events mismatch")
				for i, expectedEvent := range tt.expectedEvents {
					assert.Equal(t, expectedEvent.Type(), events_[i].Type(), "Event type mismatch at index %d", i)

					switch expected := expectedEvent.(type) {
					case *events.EventPartialCompletion:
						actual, ok := events_[i].(*events.EventPartialCompletion)
						require.True(t, ok, "Event at index %d is not EventPartialCompletion", i)
						assert.Equal(t, expected.Delta, actual.Delta, "Delta mismatch at index %d", i)
						assert.Equal(t, expected.Completion, actual.Completion, "Completion mismatch at index %d", i)
					case *events.EventToolCall:
						actual, ok := events_[i].(*events.EventToolCall)
						require.True(t, ok, "Event at index %d is not EventToolCall", i)
						assert.Equal(t, expected.ToolCall, actual.ToolCall, "ToolCall mismatch at index %d", i)
					case *events.EventFinal:
						actual, ok := events_[i].(*events.EventFinal)
						require.True(t, ok, "Event at index %d is not EventFinal", i)
						assert.Equal(t, expected.Text, actual.Text, "Final text mismatch at index %d", i)
					}
				}

				if tt.checkMetadata != nil {
					tt.checkMetadata(t, stepMetadata.Metadata)
				}

				if tt.checkResponse != nil {
					tt.checkResponse(t, merger.Response())
				}
			}
		})
	}
}

package claude

import (
	"encoding/json"
	"testing"

	"github.com/go-go-golems/geppetto/pkg/events"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/claude/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContentBlockMerger(t *testing.T) {
	toolCallResult := "{\"operation\": \"add\", \"a\": 5, \"b\": 3}"
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
				events.NewProviderCallStartedEvent(events.EventMetadata{}, events.BuildClaudeProviderCallCorrelation("claude", "", 0)),
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
				events.NewProviderCallStartedEvent(events.EventMetadata{}, events.BuildClaudeProviderCallCorrelation("claude", "", 0)),
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
				events.NewProviderCallStartedEvent(events.EventMetadata{}, events.BuildClaudeProviderCallCorrelation("claude", "", 0)),
				events.NewProviderCallFinishedEvent(events.EventMetadata{}, events.BuildClaudeProviderCallCorrelation("claude", "", 0), "end_turn", "", nil, nil, false),
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
				events.NewProviderCallStartedEvent(events.EventMetadata{}, events.BuildClaudeProviderCallCorrelation("claude", "", 0)),
				events.NewTextSegmentStartedEvent(events.EventMetadata{}, events.BuildClaudeSegmentCorrelation("claude", "claude-provider-call-0", 0, events.SegmentTypeText), ""),
				events.NewTextDeltaEvent(events.EventMetadata{}, events.BuildClaudeSegmentCorrelation("claude", "claude-provider-call-0", 0, events.SegmentTypeText), "Hello, ", "Hello, ", 0),
				events.NewTextDeltaEvent(events.EventMetadata{}, events.BuildClaudeSegmentCorrelation("claude", "claude-provider-call-0", 0, events.SegmentTypeText), "world!", "Hello, world!", 0),
				events.NewTextSegmentFinishedEvent(events.EventMetadata{}, events.BuildClaudeSegmentCorrelation("claude", "claude-provider-call-0", 0, events.SegmentTypeText), "Hello, world!", "content_block_stop"),
				events.NewProviderCallFinishedEvent(events.EventMetadata{}, events.BuildClaudeProviderCallCorrelation("claude", "", 0), "end_turn", "", nil, nil, false),
			},
			checkResponse: func(t *testing.T, response *api.MessageResponse) {
				assert.Len(t, response.Content, 1)
				assert.Equal(t, api.ContentTypeText, response.Content[0].Type())
				assert.Equal(t, "Hello, world!", response.Content[0].(api.TextContent).Text)
			},
		},
		{
			name: "Test tool use stop does not duplicate finalized text",
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
						Text: "I'll inspect the schema first.",
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
						Name: "sql_doc",
					},
				},
				{
					Type:  api.ContentBlockDeltaType,
					Index: 1,
					Delta: &api.Delta{
						Type:        api.InputJSONDeltaType,
						PartialJSON: `{"topic":"inventory"}`,
					},
				},
				{
					Type:  api.ContentBlockStopType,
					Index: 1,
				},
				{
					Type: api.MessageDeltaType,
					Delta: &api.Delta{
						StopReason: "tool_use",
					},
				},
				{Type: api.MessageStopType},
			},
			expectedEvents: []events.Event{
				events.NewProviderCallStartedEvent(events.EventMetadata{}, events.BuildClaudeProviderCallCorrelation("claude", "", 0)),
				events.NewTextSegmentStartedEvent(events.EventMetadata{}, events.BuildClaudeSegmentCorrelation("claude", "claude-provider-call-0", 0, events.SegmentTypeText), ""),
				events.NewTextDeltaEvent(events.EventMetadata{}, events.BuildClaudeSegmentCorrelation("claude", "claude-provider-call-0", 0, events.SegmentTypeText), "I'll inspect the schema first.", "I'll inspect the schema first.", 0),
				events.NewTextSegmentFinishedEvent(events.EventMetadata{}, events.BuildClaudeSegmentCorrelation("claude", "claude-provider-call-0", 0, events.SegmentTypeText), "I'll inspect the schema first.", "content_block_stop"),
				events.NewToolCallStartedEvent(events.EventMetadata{}, events.BuildClaudeSegmentCorrelation("claude", "claude-provider-call-0", 1, events.SegmentTypeTool), "tool_1", "sql_doc"),
				events.NewToolCallArgumentsDeltaEvent(events.EventMetadata{}, events.BuildClaudeSegmentCorrelation("claude", "claude-provider-call-0", 1, events.SegmentTypeTool), "tool_1", `{"topic":"inventory"}`, `{"topic":"inventory"}`, 0),
				events.NewToolCallRequestedEvent(events.EventMetadata{}, events.BuildClaudeSegmentCorrelation("claude", "claude-provider-call-0", 1, events.SegmentTypeTool), "tool_1", "sql_doc", `{"topic":"inventory"}`),
				events.NewProviderCallMetadataUpdatedEvent(events.EventMetadata{}, events.BuildClaudeProviderCallCorrelation("claude", "", 0), "tool_use", "", nil),
				events.NewProviderCallFinishedEvent(events.EventMetadata{}, events.BuildClaudeProviderCallCorrelation("claude", "", 0), "tool_use", "", nil, nil, true),
			},
			checkMetadata: func(t *testing.T, metadata map[string]interface{}) {
				assert.Equal(t, "tool_use", metadata[StopReasonMetadataSlug])
			},
			checkResponse: func(t *testing.T, response *api.MessageResponse) {
				assert.Len(t, response.Content, 2)
				assert.Equal(t, "I'll inspect the schema first.", response.Content[0].(api.TextContent).Text)
				toolUseContent := response.Content[1].(api.ToolUseContent)
				assert.Equal(t, "tool_1", toolUseContent.ID)
				assert.Equal(t, "sql_doc", toolUseContent.Name)
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
				events.NewProviderCallStartedEvent(events.EventMetadata{}, events.BuildClaudeProviderCallCorrelation("claude", "", 0)),
				events.NewTextSegmentStartedEvent(events.EventMetadata{}, events.BuildClaudeSegmentCorrelation("claude", "claude-provider-call-0", 0, events.SegmentTypeText), ""),
				events.NewTextDeltaEvent(events.EventMetadata{}, events.BuildClaudeSegmentCorrelation("claude", "claude-provider-call-0", 0, events.SegmentTypeText), "Here's the result: ", "Here's the result: ", 0),
				events.NewTextSegmentFinishedEvent(events.EventMetadata{}, events.BuildClaudeSegmentCorrelation("claude", "claude-provider-call-0", 0, events.SegmentTypeText), "Here's the result: ", "content_block_stop"),
				events.NewToolCallStartedEvent(events.EventMetadata{}, events.BuildClaudeSegmentCorrelation("claude", "claude-provider-call-0", 1, events.SegmentTypeTool), "tool_1", "calculator"),
				events.NewToolCallArgumentsDeltaEvent(events.EventMetadata{}, events.BuildClaudeSegmentCorrelation("claude", "claude-provider-call-0", 1, events.SegmentTypeTool), "tool_1", toolCallResult, toolCallResult, 0),
				events.NewToolCallRequestedEvent(events.EventMetadata{}, events.BuildClaudeSegmentCorrelation("claude", "claude-provider-call-0", 1, events.SegmentTypeTool), "tool_1", "calculator", toolCallResult),
				events.NewTextSegmentStartedEvent(events.EventMetadata{}, events.BuildClaudeSegmentCorrelation("claude", "claude-provider-call-0", 2, events.SegmentTypeText), ""),
				events.NewTextDeltaEvent(events.EventMetadata{}, events.BuildClaudeSegmentCorrelation("claude", "claude-provider-call-0", 2, events.SegmentTypeText), " is the sum.", " is the sum.", 0),
				events.NewTextSegmentFinishedEvent(events.EventMetadata{}, events.BuildClaudeSegmentCorrelation("claude", "claude-provider-call-0", 2, events.SegmentTypeText), " is the sum.", "content_block_stop"),
				events.NewProviderCallFinishedEvent(events.EventMetadata{}, events.BuildClaudeProviderCallCorrelation("claude", "", 0), "end_turn", "", nil, nil, false),
			},
			checkResponse: func(t *testing.T, response *api.MessageResponse) {
				assert.Len(t, response.Content, 3)
				assert.Equal(t, api.ContentTypeText, response.Content[0].Type())
				assert.Equal(t, "Here's the result: ", response.Content[0].(api.TextContent).Text)
				assert.Equal(t, api.ContentTypeToolUse, response.Content[1].Type())
				toolUseContent := response.Content[1].(api.ToolUseContent)
				assert.Equal(t, "tool_1", toolUseContent.ID)
				assert.Equal(t, "calculator", toolUseContent.Name)
				assert.Equal(t, json.RawMessage(toolCallResult), toolUseContent.Input)
				assert.Equal(t, api.ContentTypeText, response.Content[2].Type())
				assert.Equal(t, " is the sum.", response.Content[2].(api.TextContent).Text)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metadata := events.EventMetadata{}
			merger := NewContentBlockMerger(metadata)

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
					case *events.EventTextDelta:
						actual, ok := events_[i].(*events.EventTextDelta)
						require.True(t, ok, "Event at index %d is not EventTextDelta", i)
						assert.Equal(t, expected.Delta, actual.Delta, "Text delta mismatch at index %d", i)
						assert.Equal(t, expected.Text, actual.Text, "Text mismatch at index %d", i)
					case *events.EventTextSegmentFinished:
						actual, ok := events_[i].(*events.EventTextSegmentFinished)
						require.True(t, ok, "Event at index %d is not EventTextSegmentFinished", i)
						assert.Equal(t, expected.Text, actual.Text, "Segment text mismatch at index %d", i)
					case *events.EventToolCallArgumentsDelta:
						actual, ok := events_[i].(*events.EventToolCallArgumentsDelta)
						require.True(t, ok, "Event at index %d is not EventToolCallArgumentsDelta", i)
						assert.Equal(t, expected.Delta, actual.Delta, "Tool args delta mismatch at index %d", i)
						assert.Equal(t, expected.Arguments, actual.Arguments, "Tool args mismatch at index %d", i)
					case *events.EventToolCallRequested:
						actual, ok := events_[i].(*events.EventToolCallRequested)
						require.True(t, ok, "Event at index %d is not EventToolCallRequested", i)
						assert.Equal(t, expected.ToolCallID, actual.ToolCallID, "Tool ID mismatch at index %d", i)
						assert.Equal(t, expected.ToolName, actual.ToolName, "Tool name mismatch at index %d", i)
						assert.Equal(t, expected.Input, actual.Input, "Tool input mismatch at index %d", i)
					}
				}

				if tt.checkMetadata != nil {
					// No StepMetadata anymore; expose via merger.metadata.Extra
					tt.checkMetadata(t, merger.metadata.Extra)
				}

				if tt.checkResponse != nil {
					tt.checkResponse(t, merger.Response())
				}
			}
		})
	}
}

func TestContentBlockMergerToolUseMessageDeltaMetadataPreservedWithoutEvent(t *testing.T) {
	metadata := events.EventMetadata{}
	merger := NewContentBlockMerger(metadata)

	stream := []api.StreamingEvent{
		{
			Type: api.MessageStartType,
			Message: &api.MessageResponse{
				ID:    "msg_1",
				Model: "claude-test",
				Role:  "assistant",
				Usage: api.Usage{InputTokens: 7},
			},
		},
		{
			Type:         api.ContentBlockStartType,
			Index:        0,
			ContentBlock: &api.ContentBlock{Type: api.ContentTypeText},
		},
		{
			Type:  api.ContentBlockDeltaType,
			Index: 0,
			Delta: &api.Delta{Type: api.TextDeltaType, Text: "I'll inspect first."},
		},
		{Type: api.ContentBlockStopType, Index: 0},
		{
			Type:  api.ContentBlockStartType,
			Index: 1,
			ContentBlock: &api.ContentBlock{
				Type: api.ContentTypeToolUse,
				ID:   "tool_1",
				Name: "sql_doc",
			},
		},
		{
			Type:  api.ContentBlockDeltaType,
			Index: 1,
			Delta: &api.Delta{Type: api.InputJSONDeltaType, PartialJSON: `{"topic":"inventory"}`},
		},
		{Type: api.ContentBlockStopType, Index: 1},
		{
			Type:  api.MessageDeltaType,
			Delta: &api.Delta{StopReason: "tool_use"},
			Usage: &api.Usage{
				OutputTokens:             13,
				CacheCreationInputTokens: 2,
				CacheReadInputTokens:     5,
			},
		},
		{Type: api.MessageStopType},
	}

	var allEvents []events.Event
	for _, ev := range stream {
		generated, err := merger.Add(ev)
		require.NoError(t, err)
		allEvents = append(allEvents, generated...)
	}

	require.Len(t, allEvents, 9)
	assert.Equal(t, events.EventTypeToolCallRequested, allEvents[6].Type())
	assert.Equal(t, events.EventTypeProviderCallMetadataUpdated, allEvents[7].Type())
	assert.Equal(t, events.EventTypeProviderCallFinished, allEvents[8].Type())

	gotMeta := merger.Metadata()
	require.NotNil(t, gotMeta.StopReason)
	assert.Equal(t, "tool_use", *gotMeta.StopReason)
	require.NotNil(t, gotMeta.Usage)
	assert.Equal(t, 7, gotMeta.Usage.InputTokens)
	assert.Equal(t, 13, gotMeta.Usage.OutputTokens)
	assert.Equal(t, 2, gotMeta.Usage.CacheCreationInputTokens)
	assert.Equal(t, 5, gotMeta.Usage.CacheReadInputTokens)
	require.NotNil(t, gotMeta.DurationMs)

	response := merger.Response()
	require.NotNil(t, response)
	assert.Equal(t, "tool_use", response.StopReason)
	assert.Equal(t, 7, response.Usage.InputTokens)
	assert.Equal(t, 13, response.Usage.OutputTokens)
	assert.Equal(t, 2, response.Usage.CacheCreationInputTokens)
	assert.Equal(t, 5, response.Usage.CacheReadInputTokens)
}

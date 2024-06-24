package claude

import (
	"github.com/go-go-golems/geppetto/pkg/steps"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/chat"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/claude/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestContentBlockMerger(t *testing.T) {
	tests := []struct {
		name           string
		events         []api.StreamingEvent
		expectedResult *chat.EventPartialCompletion
		expectedError  string
		checkMetadata  func(*testing.T, map[string]interface{})
		checkResponse  func(*testing.T, *api.MessageResponse)
	}{
		{
			name: "Test NewContentBlockMerger initialization",
			events: []api.StreamingEvent{
				{Type: api.MessageStartType, Message: &api.MessageResponse{}},
			},
			expectedResult: &chat.EventPartialCompletion{
				Event: chat.Event{
					Type: chat.EventTypePartial,
				},
				Delta:      "",
				Completion: "",
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
			expectedResult: &chat.EventPartialCompletion{
				Event: chat.Event{
					Type: chat.EventTypePartial,
				},
				Delta:      "",
				Completion: "",
			},
			checkMetadata: func(t *testing.T, metadata map[string]interface{}) {
				assert.Equal(t, "claude-2", metadata[ModelMetadataSlug])
				assert.Equal(t, api.Usage{InputTokens: 10, OutputTokens: 20}, metadata[UsageMetadataSlug])
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
			expectedResult: &chat.EventPartialCompletion{
				Event: chat.Event{
					Type: chat.EventTypeFinal,
				},
				Delta:      "",
				Completion: "",
			},
			checkMetadata: func(t *testing.T, metadata map[string]interface{}) {
				assert.Equal(t, "end_turn", metadata[StopReasonMetadataSlug])
				assert.Equal(t, api.Usage{InputTokens: 15, OutputTokens: 25}, metadata[UsageMetadataSlug])
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
			expectedResult: &chat.EventPartialCompletion{
				Event: chat.Event{
					Type: chat.EventTypeFinal,
				},
				Delta:      "",
				Completion: "Hello, world!",
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
						PartialJSON: "{\"operation\": \"add\", \"a\": 5, \"b\": 3}",
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
			expectedResult: &chat.EventPartialCompletion{
				Event: chat.Event{
					Type: chat.EventTypeFinal,
				},
				Delta:      "",
				Completion: "Here's the result: Tool Call: calculator\nID: tool_1\n is the sum.",
			},
			checkResponse: func(t *testing.T, response *api.MessageResponse) {
				assert.Len(t, response.Content, 3)
				assert.Equal(t, api.ContentTypeText, response.Content[0].Type())
				assert.Equal(t, "Here's the result: ", response.Content[0].(api.TextContent).Text)
				assert.Equal(t, api.ContentTypeToolUse, response.Content[1].Type())
				toolUseContent := response.Content[1].(api.ToolUseContent)
				assert.Equal(t, "tool_1", toolUseContent.ID)
				assert.Equal(t, "calculator", toolUseContent.Name)
				assert.Equal(t, "{\"operation\": \"add\", \"a\": 5, \"b\": 3}", toolUseContent.Input)
				assert.Equal(t, api.ContentTypeText, response.Content[2].Type())
				assert.Equal(t, " is the sum.", response.Content[2].(api.TextContent).Text)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metadata := chat.EventMetadata{}
			stepMetadata := &steps.StepMetadata{
				Metadata: make(map[string]interface{}),
			}
			merger := NewContentBlockMerger(metadata, stepMetadata)

			var result *chat.EventPartialCompletion
			var err error

			for _, event := range tt.events {
				result, err = merger.Add(event)
			}

			if tt.expectedError != "" {
				assert.EqualError(t, err, tt.expectedError)
			} else {
				assert.NoError(t, err)
				require.NotNil(t, result)
				assert.Equal(t, tt.expectedResult.Type, result.Type)
				assert.Equal(t, tt.expectedResult.Delta, result.Delta)
				assert.Equal(t, tt.expectedResult.Completion, result.Completion)

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

package events

import "github.com/rs/zerolog"

type ToolCall struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Input string `json:"input"`
}

// TODO(manuel, 2024-07-04) Handle multiple tool calls
type EventToolCall struct {
	EventImpl
	ToolCall ToolCall `json:"tool_call"`
}

func NewToolCallEvent(metadata EventMetadata, toolCall ToolCall) *EventToolCall {
	return &EventToolCall{
		EventImpl: EventImpl{
			Type_:     EventTypeToolCall,
			Metadata_: metadata,
			payload:   nil,
		},
		ToolCall: toolCall,
	}
}

var _ Event = &EventToolCall{}

type ToolResult struct {
	ID     string `json:"id"`
	Name   string `json:"name,omitempty"`
	Result string `json:"result"`
}

type EventToolResult struct {
	EventImpl
	ToolResult ToolResult `json:"tool_result"`
}

func NewToolResultEvent(metadata EventMetadata, toolResult ToolResult) *EventToolResult {
	return &EventToolResult{
		EventImpl: EventImpl{
			Type_:     EventTypeToolResult,
			Metadata_: metadata,
			payload:   nil,
		},
		ToolResult: toolResult,
	}
}

var _ Event = &EventToolResult{}

// EventToolCallExecute captures the intent to execute a tool locally
type EventToolCallExecute struct {
	EventImpl
	ToolCall ToolCall `json:"tool_call"`
}

func NewToolCallExecuteEvent(metadata EventMetadata, toolCall ToolCall) *EventToolCallExecute {
	return &EventToolCallExecute{
		EventImpl: EventImpl{
			Type_:     EventTypeToolCallExecute,
			Metadata_: metadata,
			payload:   nil,
		},
		ToolCall: toolCall,
	}
}

var _ Event = &EventToolCallExecute{}

// EventToolCallExecutionResult captures the result of executing a tool locally
type EventToolCallExecutionResult struct {
	EventImpl
	ToolResult ToolResult `json:"tool_result"`
}

func NewToolCallExecutionResultEvent(metadata EventMetadata, toolResult ToolResult) *EventToolCallExecutionResult {
	return &EventToolCallExecutionResult{
		EventImpl: EventImpl{
			Type_:     EventTypeToolCallExecutionResult,
			Metadata_: metadata,
			payload:   nil,
		},
		ToolResult: toolResult,
	}
}

var _ Event = &EventToolCallExecutionResult{}

// MetadataToolCallsSlug is the slug used to store ToolCall metadata as returned by the openai API
// TODO(manuel, 2024-07-04) This needs to deleted once we have a good way to do tool calling
const MetadataToolCallsSlug = "tool-calls"

func (tc ToolCall) MarshalZerologObject(ev *zerolog.Event) {
	ev.Str("id", tc.ID).Str("name", tc.Name).Str("input", tc.Input)
}

func (e EventToolCall) MarshalZerologObject(ev *zerolog.Event) {
	e.EventImpl.MarshalZerologObject(ev)
	ev.Object("tool_call", e.ToolCall)
}

func (tr ToolResult) MarshalZerologObject(ev *zerolog.Event) {
	ev.Str("id", tr.ID).Str("result", tr.Result)
	if tr.Name != "" {
		ev.Str("name", tr.Name)
	}
}

func (e EventToolResult) MarshalZerologObject(ev *zerolog.Event) {
	e.EventImpl.MarshalZerologObject(ev)
	ev.Object("tool_result", e.ToolResult)
}

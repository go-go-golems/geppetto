package events

import (
	"encoding/json"
	"fmt"

	"github.com/go-go-golems/geppetto/pkg/conversation"
	"github.com/rs/zerolog"
)

type EventType string

const (
	// EventTypeStart to EventTypeFinal are for text completion, actually
	EventTypeStart             EventType = "start"
	EventTypeFinal             EventType = "final"
	EventTypePartialCompletion EventType = "partial"

	// TODO(manuel, 2024-07-04) I'm not sure if this is needed
	EventTypeStatus EventType = "status"

	// TODO(manuel, 2024-07-04) Should potentially have a EventTypeText for a block stop here
	// Model requested a tool call (received from provider stream)
	EventTypeToolCall   EventType = "tool-call"
	EventTypeToolResult EventType = "tool-result"

	// Execution-phase events (we are actually executing tools locally)
	EventTypeToolCallExecute         EventType = "tool-call-execute"
	EventTypeToolCallExecutionResult EventType = "tool-call-execution-result"
	EventTypeError                   EventType = "error"
	EventTypeInterrupt               EventType = "interrupt"

    // Informational/logging events (emitted by engines, middlewares or tools)
    EventTypeLog  EventType = "log"
    EventTypeInfo EventType = "info"
)

type Event interface {
	Type() EventType
	Metadata() EventMetadata
	Payload() []byte
}

// MetadataSettingsSlug retained for compatibility in EventMetadata.Extra
const MetadataSettingsSlug = "settings"

type EventImpl struct {
	Type_     EventType     `json:"type"`
	Error_    error         `json:"error,omitempty"`
	Metadata_ EventMetadata `json:"meta,omitempty"`

	// store payload if the event was deserialized from JSON (see NewEventFromJson), not further used
	payload []byte
}

func (e *EventImpl) MarshalZerologObject(ev *zerolog.Event) {
	ev.Str("type", string(e.Type_))

	if e.Error_ != nil {
		ev.Err(e.Error_)
	}

    ev.Object("meta", e.Metadata_)

}

func (e *EventImpl) Type() EventType {
	return e.Type_
}

func (e *EventImpl) Error() error {
	return e.Error_
}

func (e *EventImpl) Metadata() EventMetadata {
	return e.Metadata_
}

func (e *EventImpl) Payload() []byte {
	return e.payload
}

var _ Event = &EventImpl{}

type EventPartialCompletionStart struct {
	EventImpl
}

func NewStartEvent(metadata EventMetadata) *EventPartialCompletionStart {
	return &EventPartialCompletionStart{
		EventImpl: EventImpl{
			Type_:     EventTypeStart,
			Metadata_: metadata,
			payload:   nil,
		},
	}
}

var _ Event = &EventPartialCompletionStart{}

type EventInterrupt struct {
	EventImpl
	Text string `json:"text"`
	// TODO(manuel, 2024-07-04) Add all collected tool calls so far
}

func NewInterruptEvent(metadata EventMetadata, text string) *EventInterrupt {
	return &EventInterrupt{
		EventImpl: EventImpl{
			Type_:     EventTypeInterrupt,
			Metadata_: metadata,
			payload:   nil,
		},
		Text: text,
	}
}

var _ Event = &EventInterrupt{}

type EventFinal struct {
	EventImpl
	Text string `json:"text"`
	// TODO(manuel, 2024-07-04) Add all collected tool calls so far
}

func NewFinalEvent(metadata EventMetadata, text string) *EventFinal {
	return &EventFinal{
		EventImpl: EventImpl{
			Type_:     EventTypeFinal,
			Metadata_: metadata,
			payload:   nil,
		},
		Text: text,
	}
}

var _ Event = &EventFinal{}

type EventError struct {
	EventImpl
	ErrorString string `json:"error_string"`
}

func NewErrorEvent(metadata EventMetadata, err error) *EventError {
	return &EventError{
		EventImpl: EventImpl{
			Type_:     EventTypeError,
			Metadata_: metadata,
			payload:   nil,
		},
		ErrorString: err.Error(),
	}
}

var _ Event = &EventError{}

// TODO(manuel, 2024-07-05) This might be possible to delete
type EventText struct {
	EventImpl
	Text string `json:"text"`
	// TODO(manuel, 2024-06-04) Add ToolCall information here, and potentially multiple responses (see the claude API that allows multiple content blocks)
	// This is currently stored in the metadata uder the MetadataToolCallsSlug (see chat-with-tools-step.go in openai)
}

func NewTextEvent(metadata EventMetadata, text string) *EventText {
	return &EventText{
		EventImpl: EventImpl{
			Type_:     EventTypeStart,
			Metadata_: metadata,
			payload:   nil,
		},
		Text: text,
	}
}

var _ Event = &EventText{}

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

// TODO(manuel, 2024-07-03) Then, we can add those to the openai step as well, and to the UI, and then we should have a good way to do auto tool calling

// EventPartialCompletion is the event type for textual partial completion. We don't support partial tool completion.
type EventPartialCompletion struct {
	EventImpl
	Delta string `json:"delta"`
	// This is the complete completion string so far (when using openai, this is currently also the toolcall json)
	Completion string `json:"completion"`

	// TODO(manuel, 2024-06-04) This might need partial tool completion if it is of interest,
	// this is less important than adding tool call information to the result above
}

func NewPartialCompletionEvent(metadata EventMetadata, delta string, completion string) *EventPartialCompletion {
	return &EventPartialCompletion{
		EventImpl: EventImpl{
			Type_:     EventTypePartialCompletion,
			Metadata_: metadata,
			payload:   nil,
		},
		Delta:      delta,
		Completion: completion,
	}
}

var _ Event = &EventPartialCompletion{}

// MetadataToolCallsSlug is the slug used to store ToolCall metadata as returned by the openai API
// TODO(manuel, 2024-07-04) This needs to deleted once we have a good way to do tool calling
const MetadataToolCallsSlug = "tool-calls"

// EventMetadata contains all the information that is passed along with watermill message,
// specific to chat steps.
type EventMetadata struct {
	conversation.LLMMessageMetadata
	ID       conversation.NodeID `json:"message_id" yaml:"message_id" mapstructure:"message_id"`
    // Correlation identifiers
    RunID   string `json:"run_id,omitempty" yaml:"run_id,omitempty" mapstructure:"run_id"`
    TurnID  string `json:"turn_id,omitempty" yaml:"turn_id,omitempty" mapstructure:"turn_id"`
    // Extra carries provider-specific/context values
    Extra    map[string]interface{} `json:"extra,omitempty" yaml:"extra,omitempty" mapstructure:"extra"`
}

func (em EventMetadata) MarshalZerologObject(e *zerolog.Event) {
	e.Str("message_id", em.ID.String())
    if em.RunID != "" {
        e.Str("run_id", em.RunID)
    }
    if em.TurnID != "" {
        e.Str("turn_id", em.TurnID)
    }
	if em.Engine != "" {
		e.Str("engine", em.Engine)
	}
	if em.StopReason != nil && *em.StopReason != "" {
		e.Str("stop_reason", *em.StopReason)
	}
	if em.Usage != nil {
		e.Int("input_tokens", em.Usage.InputTokens)
		e.Int("output_tokens", em.Usage.OutputTokens)
	}
    if len(em.Extra) > 0 {
        e.Dict("extra", zerolog.Dict().Fields(em.Extra))
    }
}

// Extra metadata keys for correlation
const (
    MetaKeyRunID  = "run_id"
    MetaKeyTurnID = "turn_id"
)

func NewEventFromJson(b []byte) (Event, error) {
	var e *EventImpl
	err := json.Unmarshal(b, &e)
	if err != nil {
		return nil, err
	}

	e.payload = b

	switch e.Type_ {
	case EventTypeStart:
		ret, ok := ToTypedEvent[EventPartialCompletionStart](e)
		if !ok {
			return nil, fmt.Errorf("could not cast event to EventPartialCompletionStart")
		}
		return ret, nil
	case EventTypePartialCompletion:
		ret, ok := ToTypedEvent[EventPartialCompletion](e)
		if !ok {
			return nil, fmt.Errorf("could not cast event to EventPartialCompletion")
		}
		return ret, nil
	case EventTypeToolCall:
		ret, ok := ToTypedEvent[EventToolCall](e)
		if !ok {
			return nil, fmt.Errorf("could not cast event to EventToolCall")
		}
		return ret, nil
	case EventTypeToolResult:
		ret, ok := ToTypedEvent[EventToolResult](e)
		if !ok {
			return nil, fmt.Errorf("could not cast event to EventToolResult")
		}
		return ret, nil
	case EventTypeToolCallExecute:
		ret, ok := ToTypedEvent[EventToolCallExecute](e)
		if !ok {
			return nil, fmt.Errorf("could not cast event to EventToolCallExecute")
		}
		return ret, nil
	case EventTypeToolCallExecutionResult:
		ret, ok := ToTypedEvent[EventToolCallExecutionResult](e)
		if !ok {
			return nil, fmt.Errorf("could not cast event to EventToolCallExecutionResult")
		}
		return ret, nil
	case EventTypeError:
		ret, ok := ToTypedEvent[EventError](e)
		if !ok {
			return nil, fmt.Errorf("could not cast event to EventError")
		}
		return ret, nil
	case EventTypeInterrupt:
		ret, ok := ToTypedEvent[EventInterrupt](e)
		if !ok {
			return nil, fmt.Errorf("could not cast event to EventInterrupt")
		}
		return ret, nil
	case EventTypeFinal:
		ret, ok := ToTypedEvent[EventFinal](e)
		if !ok {
			return nil, fmt.Errorf("could not cast event to EventFinal")
		}
		return ret, nil

	case EventTypeStatus:
    case EventTypeLog:
        ret, ok := ToTypedEvent[EventLog](e)
        if !ok {
            return nil, fmt.Errorf("could not cast event to EventLog")
        }
        return ret, nil
    case EventTypeInfo:
        ret, ok := ToTypedEvent[EventInfo](e)
        if !ok {
            return nil, fmt.Errorf("could not cast event to EventInfo")
        }
        return ret, nil
	}

	return e, nil
}

func ToTypedEvent[T any](e Event) (*T, bool) {
	var ret *T
	err := json.Unmarshal(e.Payload(), &ret)
	if err != nil {
		return nil, false
	}

	return ret, true
}

func (e *EventImpl) ToText() (EventText, bool) {
	ret, ok := ToTypedEvent[EventText](e)
	if !ok || ret == nil {
		return EventText{}, false
	}
	return *ret, true
}

func (e *EventImpl) ToPartialCompletion() (EventPartialCompletion, bool) {
	ret, ok := ToTypedEvent[EventPartialCompletion](e)
	if !ok || ret == nil {
		return EventPartialCompletion{}, false
	}
	return *ret, true
}

func (e *EventImpl) ToToolCall() (EventToolCall, bool) {
	ret, ok := ToTypedEvent[EventToolCall](e)
	if !ok || ret == nil {
		return EventToolCall{}, false
	}
	return *ret, true
}

func (e EventPartialCompletionStart) MarshalZerologObject(ev *zerolog.Event) {
	e.EventImpl.MarshalZerologObject(ev)
}

func (e EventInterrupt) MarshalZerologObject(ev *zerolog.Event) {
	e.EventImpl.MarshalZerologObject(ev)
	ev.Str("text", e.Text)
}

func (e EventFinal) MarshalZerologObject(ev *zerolog.Event) {
	e.EventImpl.MarshalZerologObject(ev)
	ev.Str("text", e.Text)
}

func (e EventError) MarshalZerologObject(ev *zerolog.Event) {
	e.EventImpl.MarshalZerologObject(ev)
	ev.Str("error", e.ErrorString)
}

func (e EventText) MarshalZerologObject(ev *zerolog.Event) {
	e.EventImpl.MarshalZerologObject(ev)
	ev.Str("text", e.Text)
}

func (tc ToolCall) MarshalZerologObject(ev *zerolog.Event) {
	ev.Str("id", tc.ID).Str("name", tc.Name).Str("input", tc.Input)
}

func (e EventToolCall) MarshalZerologObject(ev *zerolog.Event) {
	e.EventImpl.MarshalZerologObject(ev)
	ev.Object("tool_call", e.ToolCall)
}

func (tr ToolResult) MarshalZerologObject(ev *zerolog.Event) {
	ev.Str("id", tr.ID).Str("result", tr.Result)
}

func (e EventToolResult) MarshalZerologObject(ev *zerolog.Event) {
	e.EventImpl.MarshalZerologObject(ev)
	ev.Object("tool_result", e.ToolResult)
}

func (e EventPartialCompletion) MarshalZerologObject(ev *zerolog.Event) {
	e.EventImpl.MarshalZerologObject(ev)
	ev.Str("delta", e.Delta).Str("completion", e.Completion)
}

// EventLog represents a generic log record emitted during inference (by engine, middleware or tools)
type EventLog struct {
    EventImpl
    Level   string                 `json:"level"`
    Message string                 `json:"message"`
    Fields  map[string]interface{} `json:"fields,omitempty"`
}

func NewLogEvent(metadata EventMetadata, level string, message string, fields map[string]interface{}) *EventLog {
    return &EventLog{
        EventImpl: EventImpl{
            Type_:     EventTypeLog,
            Metadata_: metadata,
            payload:   nil,
        },
        Level:   level,
        Message: message,
        Fields:  fields,
    }
}

var _ Event = &EventLog{}

func (e EventLog) MarshalZerologObject(ev *zerolog.Event) {
    e.EventImpl.MarshalZerologObject(ev)
    ev.Str("level", e.Level).Str("message", e.Message)
    if len(e.Fields) > 0 {
        ev.Dict("fields", zerolog.Dict().Fields(e.Fields))
    }
}

// EventInfo is a lightweight informational message for user-facing notifications
type EventInfo struct {
    EventImpl
    Message string                 `json:"message"`
    Data    map[string]interface{} `json:"data,omitempty"`
}

func NewInfoEvent(metadata EventMetadata, message string, data map[string]interface{}) *EventInfo {
    return &EventInfo{
        EventImpl: EventImpl{
            Type_:     EventTypeInfo,
            Metadata_: metadata,
            payload:   nil,
        },
        Message: message,
        Data:    data,
    }
}

var _ Event = &EventInfo{}

func (e EventInfo) MarshalZerologObject(ev *zerolog.Event) {
    e.EventImpl.MarshalZerologObject(ev)
    ev.Str("message", e.Message)
    if len(e.Data) > 0 {
        ev.Dict("data", zerolog.Dict().Fields(e.Data))
    }
}

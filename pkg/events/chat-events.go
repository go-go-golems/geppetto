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
    EventTypeToolCallExecute          EventType = "tool-call-execute"
    EventTypeToolCallExecutionResult  EventType = "tool-call-execution-result"
	EventTypeError      EventType = "error"
	EventTypeInterrupt  EventType = "interrupt"
)

type Event interface {
	Type() EventType
	Metadata() EventMetadata
    StepMetadata() *StepMetadata
	Payload() []byte
}

// MetadataSettingsSlug is the slug used to store settings metadata inside StepMetadata.Metadata
const MetadataSettingsSlug = "settings"

// StepMetadata contains execution context for steps/engines used in events
type StepMetadata struct {
    StepID     conversation.NodeID     `json:"step_id" yaml:"step_id" mapstructure:"step_id"`
    Type       string                  `json:"type" yaml:"type" mapstructure:"type"`
    InputType  string                  `json:"input_type" yaml:"input_type" mapstructure:"input_type"`
    OutputType string                  `json:"output_type" yaml:"output_type" mapstructure:"output_type"`
    Metadata   map[string]interface{}  `json:"meta" yaml:"meta" mapstructure:"meta"`
}

func (sm StepMetadata) MarshalZerologObject(e *zerolog.Event) {
    e.Str("step_id", sm.StepID.String())
    e.Str("type", sm.Type)
    e.Str("input_type", sm.InputType)
    e.Str("output_type", sm.OutputType)

    if len(sm.Metadata) > 0 {
        e.Dict("meta", zerolog.Dict().Fields(sm.Metadata))
    }
}

type EventImpl struct {
    Type_     EventType     `json:"type"`
    Error_    error         `json:"error,omitempty"`
    Metadata_ EventMetadata `json:"meta,omitempty"`
    Step_     *StepMetadata `json:"step,omitempty"`

	// store payload if the event was deserialized from JSON (see NewEventFromJson), not further used
	payload []byte
}

func (e *EventImpl) MarshalZerologObject(ev *zerolog.Event) {
	ev.Str("type", string(e.Type_))

	if e.Error_ != nil {
		ev.Err(e.Error_)
	}

	if e.Metadata_ != (EventMetadata{}) {
		ev.Object("meta", e.Metadata_)
	}

	if e.Step_ != nil {
		ev.Object("step", e.Step_)
	}
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

func (e *EventImpl) StepMetadata() *StepMetadata {
	return e.Step_
}

func (e *EventImpl) Payload() []byte {
	return e.payload
}

var _ Event = &EventImpl{}

type EventPartialCompletionStart struct {
	EventImpl
}

func NewStartEvent(metadata EventMetadata, stepMetadata *StepMetadata) *EventPartialCompletionStart {
	return &EventPartialCompletionStart{
		EventImpl: EventImpl{
			Type_:     EventTypeStart,
			Step_:     stepMetadata,
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

func NewInterruptEvent(metadata EventMetadata, stepMetadata *StepMetadata, text string) *EventInterrupt {
	return &EventInterrupt{
		EventImpl: EventImpl{
			Type_:     EventTypeInterrupt,
			Step_:     stepMetadata,
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

func NewFinalEvent(metadata EventMetadata, stepMetadata *StepMetadata, text string) *EventFinal {
	return &EventFinal{
		EventImpl: EventImpl{
			Type_:     EventTypeFinal,
			Step_:     stepMetadata,
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

func NewErrorEvent(metadata EventMetadata, stepMetadata *StepMetadata, err error) *EventError {
	return &EventError{
		EventImpl: EventImpl{
			Type_:     EventTypeError,
			Step_:     stepMetadata,
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

func NewTextEvent(metadata EventMetadata, stepMetadata *StepMetadata, text string) *EventText {
	return &EventText{
		EventImpl: EventImpl{
			Type_:     EventTypeStart,
			Step_:     stepMetadata,
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

func NewToolCallEvent(metadata EventMetadata, stepMetadata *StepMetadata, toolCall ToolCall) *EventToolCall {
	return &EventToolCall{
		EventImpl: EventImpl{
			Type_:     EventTypeToolCall,
			Step_:     stepMetadata,
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

func NewToolResultEvent(metadata EventMetadata, stepMetadata *StepMetadata, toolResult ToolResult) *EventToolResult {
	return &EventToolResult{
		EventImpl: EventImpl{
			Type_:     EventTypeToolResult,
			Step_:     stepMetadata,
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

func NewToolCallExecuteEvent(metadata EventMetadata, stepMetadata *StepMetadata, toolCall ToolCall) *EventToolCallExecute {
    return &EventToolCallExecute{
        EventImpl: EventImpl{
            Type_:     EventTypeToolCallExecute,
            Step_:     stepMetadata,
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

func NewToolCallExecutionResultEvent(metadata EventMetadata, stepMetadata *StepMetadata, toolResult ToolResult) *EventToolCallExecutionResult {
    return &EventToolCallExecutionResult{
        EventImpl: EventImpl{
            Type_:     EventTypeToolCallExecutionResult,
            Step_:     stepMetadata,
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

func NewPartialCompletionEvent(metadata EventMetadata, stepMetadata *StepMetadata, delta string, completion string) *EventPartialCompletion {
	return &EventPartialCompletion{
		EventImpl: EventImpl{
			Type_:     EventTypePartialCompletion,
			Step_:     stepMetadata,
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
	ParentID conversation.NodeID `json:"parent_id" yaml:"parent_id" mapstructure:"parent_id"`
}

func (em EventMetadata) MarshalZerologObject(e *zerolog.Event) {
	e.Str("message_id", em.ID.String())
	e.Str("parent_id", em.ParentID.String())
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
}

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

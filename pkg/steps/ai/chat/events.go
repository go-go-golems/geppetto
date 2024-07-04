package chat

import (
	"encoding/json"
	"fmt"
	"github.com/go-go-golems/bobatea/pkg/conversation"
	"github.com/go-go-golems/geppetto/pkg/steps"
)

type EventType string

const (
	// EventTypeStart to EventTypeFinal are for text completion, actually
	EventTypeStart             EventType = "start"
	EventTypeStatus            EventType = "status"
	EventTypeFinal             EventType = "final"
	EventTypePartialCompletion EventType = "partial"

	// TODO(manuel, 2024-07-04) Should potentially have a EventTypeText for a block stop here
	EventTypeToolCall   EventType = "tool-call"
	EventTypeToolResult EventType = "tool-result"
	EventTypeError      EventType = "error"
	EventTypeInterrupt  EventType = "interrupt"
)

type Event interface {
	Type() EventType
	Metadata() EventMetadata
	StepMetadata() *steps.StepMetadata
	Payload() []byte
}

type EventImpl struct {
	Type_     EventType           `json:"type"`
	Error_    error               `json:"error,omitempty"`
	Metadata_ EventMetadata       `json:"meta,omitempty"`
	Step_     *steps.StepMetadata `json:"step,omitempty"`

	// store payload if the event was deserialized from JSON (see NewEventFromJson), not further used
	payload []byte
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

func (e *EventImpl) StepMetadata() *steps.StepMetadata {
	return e.Step_
}

func (e *EventImpl) Payload() []byte {
	return e.payload
}

var _ Event = &EventImpl{}

type EventPartialCompletionStart struct {
	EventImpl
}

func NewStartEvent(metadata EventMetadata, stepMetadata *steps.StepMetadata) *EventPartialCompletionStart {
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

func NewInterruptEvent(metadata EventMetadata, stepMetadata *steps.StepMetadata, text string) *EventInterrupt {
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

func NewFinalEvent(metadata EventMetadata, stepMetadata *steps.StepMetadata, text string) *EventFinal {
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
	Error_ string `json:"error"`
}

func NewErrorEvent(metadata EventMetadata, stepMetadata *steps.StepMetadata, err string) *EventError {
	return &EventError{
		EventImpl: EventImpl{
			Type_:     EventTypeError,
			Step_:     stepMetadata,
			Metadata_: metadata,
			payload:   nil,
		},
		Error_: err,
	}
}

var _ Event = &EventError{}

type EventText struct {
	EventImpl
	Text string `json:"text"`
	// TODO(manuel, 2024-06-04) Add ToolCall information here, and potentially multiple responses (see the claude API that allows multiple content blocks)
	// This is currently stored in the metadata uder the MetadataToolCallsSlug (see chat-with-tools-step.go in openai)
}

func NewTextEvent(metadata EventMetadata, stepMetadata *steps.StepMetadata, text string) *EventText {
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

func NewToolCallEvent(metadata EventMetadata, stepMetadata *steps.StepMetadata, toolCall ToolCall) *EventToolCall {
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

func NewToolResultEvent(metadata EventMetadata, stepMetadata *steps.StepMetadata, toolResult ToolResult) *EventToolResult {
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

func NewPartialCompletionEvent(metadata EventMetadata, stepMetadata *steps.StepMetadata, delta string, completion string) *EventPartialCompletion {
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
	ID       conversation.NodeID `json:"message_id"`
	ParentID conversation.NodeID `json:"parent_id"`
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
	case EventTypeToolCall:
		return &EventToolCall{EventImpl: *e}, nil
	case EventTypeToolResult:
		return &EventToolResult{EventImpl: *e}, nil
	case EventTypeError, EventTypeInterrupt:
		return e, nil
	default:
		return nil, nil
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

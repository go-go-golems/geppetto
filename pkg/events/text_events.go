package events

import "github.com/rs/zerolog"

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

// EventThinkingPartial mirrors EventPartialCompletion but is dedicated to reasoning/summary text
type EventThinkingPartial struct {
	EventImpl
	Delta      string `json:"delta"`
	Completion string `json:"completion"`
}

func NewThinkingPartialEvent(metadata EventMetadata, delta string, completion string) *EventThinkingPartial {
	return &EventThinkingPartial{
		EventImpl:  EventImpl{Type_: EventTypePartialThinking, Metadata_: metadata},
		Delta:      delta,
		Completion: completion,
	}
}

var _ Event = &EventThinkingPartial{}

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

func (e EventPartialCompletion) MarshalZerologObject(ev *zerolog.Event) {
	e.EventImpl.MarshalZerologObject(ev)
	ev.Str("delta", e.Delta).Str("completion", e.Completion)
}

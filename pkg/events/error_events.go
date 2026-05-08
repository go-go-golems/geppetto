package events

import "github.com/rs/zerolog"

type EventInterrupt struct {
	EventImpl
	Text string `json:"text"`
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

func (e EventInterrupt) MarshalZerologObject(ev *zerolog.Event) {
	e.EventImpl.MarshalZerologObject(ev)
	ev.Str("text", e.Text)
}

func (e EventError) MarshalZerologObject(ev *zerolog.Event) {
	e.EventImpl.MarshalZerologObject(ev)
	ev.Str("error", e.ErrorString)
}

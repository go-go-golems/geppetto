package events

import "github.com/rs/zerolog"

type EventRunStarted struct {
	EventImpl
	Correlation_ Correlation `json:"correlation"`
	Prompt       string      `json:"prompt,omitempty"`
}

func NewRunStartedEvent(metadata EventMetadata, corr Correlation, prompt string) *EventRunStarted {
	return &EventRunStarted{EventImpl: EventImpl{Type_: EventTypeRunStarted, Metadata_: metadata}, Correlation_: corr, Prompt: prompt}
}

func (e *EventRunStarted) Correlation() Correlation { return e.Correlation_ }

var _ CorrelatedEvent = &EventRunStarted{}

type EventRunFinished struct {
	EventImpl
	Correlation_ Correlation `json:"correlation"`
	Status       string      `json:"status,omitempty"`
}

func NewRunFinishedEvent(metadata EventMetadata, corr Correlation, status string) *EventRunFinished {
	return &EventRunFinished{EventImpl: EventImpl{Type_: EventTypeRunFinished, Metadata_: metadata}, Correlation_: corr, Status: status}
}

func (e *EventRunFinished) Correlation() Correlation { return e.Correlation_ }

var _ CorrelatedEvent = &EventRunFinished{}

type EventRunStopped struct {
	EventImpl
	Correlation_ Correlation `json:"correlation"`
	Reason       string      `json:"reason,omitempty"`
}

func NewRunStoppedEvent(metadata EventMetadata, corr Correlation, reason string) *EventRunStopped {
	return &EventRunStopped{EventImpl: EventImpl{Type_: EventTypeRunStopped, Metadata_: metadata}, Correlation_: corr, Reason: reason}
}

func (e *EventRunStopped) Correlation() Correlation { return e.Correlation_ }

var _ CorrelatedEvent = &EventRunStopped{}

type EventRunFailed struct {
	EventImpl
	Correlation_ Correlation `json:"correlation"`
	ErrorString  string      `json:"error_string,omitempty"`
}

func NewRunFailedEvent(metadata EventMetadata, corr Correlation, err error) *EventRunFailed {
	errorString := ""
	if err != nil {
		errorString = err.Error()
	}
	return &EventRunFailed{EventImpl: EventImpl{Type_: EventTypeRunFailed, Metadata_: metadata, Error_: err}, Correlation_: corr, ErrorString: errorString}
}

func (e *EventRunFailed) Correlation() Correlation { return e.Correlation_ }

var _ CorrelatedEvent = &EventRunFailed{}

type EventProviderCallStarted struct {
	EventImpl
	Correlation_ Correlation `json:"correlation"`
}

func NewProviderCallStartedEvent(metadata EventMetadata, corr Correlation) *EventProviderCallStarted {
	return &EventProviderCallStarted{EventImpl: EventImpl{Type_: EventTypeProviderCallStarted, Metadata_: metadata}, Correlation_: corr}
}

func (e *EventProviderCallStarted) Correlation() Correlation { return e.Correlation_ }

var _ CorrelatedEvent = &EventProviderCallStarted{}

type EventProviderCallMetadataUpdated struct {
	EventImpl
	Correlation_ Correlation `json:"correlation"`
	StopReason   string      `json:"stop_reason,omitempty"`
	StopSequence string      `json:"stop_sequence,omitempty"`
	Usage        *Usage      `json:"usage,omitempty"`
}

func NewProviderCallMetadataUpdatedEvent(metadata EventMetadata, corr Correlation, stopReason, stopSequence string, usage *Usage) *EventProviderCallMetadataUpdated {
	return &EventProviderCallMetadataUpdated{EventImpl: EventImpl{Type_: EventTypeProviderCallMetadataUpdated, Metadata_: metadata}, Correlation_: corr, StopReason: stopReason, StopSequence: stopSequence, Usage: usage}
}

func (e *EventProviderCallMetadataUpdated) Correlation() Correlation { return e.Correlation_ }

var _ CorrelatedEvent = &EventProviderCallMetadataUpdated{}

type EventProviderCallFinished struct {
	EventImpl
	Correlation_ Correlation `json:"correlation"`
	StopReason   string      `json:"stop_reason,omitempty"`
	FinishClass  string      `json:"finish_class,omitempty"`
	Usage        *Usage      `json:"usage,omitempty"`
	DurationMs   *int64      `json:"duration_ms,omitempty"`
	HasToolCalls bool        `json:"has_tool_calls,omitempty"`
}

func NewProviderCallFinishedEvent(metadata EventMetadata, corr Correlation, stopReason, finishClass string, usage *Usage, durationMs *int64, hasToolCalls bool) *EventProviderCallFinished {
	return &EventProviderCallFinished{EventImpl: EventImpl{Type_: EventTypeProviderCallFinished, Metadata_: metadata}, Correlation_: corr, StopReason: stopReason, FinishClass: finishClass, Usage: usage, DurationMs: durationMs, HasToolCalls: hasToolCalls}
}

func (e *EventProviderCallFinished) Correlation() Correlation { return e.Correlation_ }

var _ CorrelatedEvent = &EventProviderCallFinished{}

type EventTextSegmentStarted struct {
	EventImpl
	Correlation_ Correlation `json:"correlation"`
	Role         string      `json:"role,omitempty"`
}

func NewTextSegmentStartedEvent(metadata EventMetadata, corr Correlation, role string) *EventTextSegmentStarted {
	return &EventTextSegmentStarted{EventImpl: EventImpl{Type_: EventTypeTextSegmentStarted, Metadata_: metadata}, Correlation_: corr, Role: role}
}

func (e *EventTextSegmentStarted) Correlation() Correlation { return e.Correlation_ }

var _ CorrelatedEvent = &EventTextSegmentStarted{}

type EventTextDelta struct {
	EventImpl
	Correlation_ Correlation `json:"correlation"`
	Delta        string      `json:"delta"`
	Text         string      `json:"text"`
	Sequence     int64       `json:"sequence,omitempty"`
}

func NewTextDeltaEvent(metadata EventMetadata, corr Correlation, delta, text string, sequence int64) *EventTextDelta {
	return &EventTextDelta{EventImpl: EventImpl{Type_: EventTypeTextDelta, Metadata_: metadata}, Correlation_: corr, Delta: delta, Text: text, Sequence: sequence}
}

func (e *EventTextDelta) Correlation() Correlation { return e.Correlation_ }

var _ CorrelatedEvent = &EventTextDelta{}

type EventTextSegmentFinished struct {
	EventImpl
	Correlation_ Correlation `json:"correlation"`
	Text         string      `json:"text"`
	FinishReason string      `json:"finish_reason,omitempty"`
}

func NewTextSegmentFinishedEvent(metadata EventMetadata, corr Correlation, text, finishReason string) *EventTextSegmentFinished {
	return &EventTextSegmentFinished{EventImpl: EventImpl{Type_: EventTypeTextSegmentFinished, Metadata_: metadata}, Correlation_: corr, Text: text, FinishReason: finishReason}
}

func (e *EventTextSegmentFinished) Correlation() Correlation { return e.Correlation_ }

var _ CorrelatedEvent = &EventTextSegmentFinished{}

type EventReasoningSegmentStarted struct {
	EventImpl
	Correlation_ Correlation `json:"correlation"`
	Source       string      `json:"source,omitempty"`
}

func NewReasoningSegmentStartedEvent(metadata EventMetadata, corr Correlation, source string) *EventReasoningSegmentStarted {
	return &EventReasoningSegmentStarted{EventImpl: EventImpl{Type_: EventTypeReasoningSegmentStarted, Metadata_: metadata}, Correlation_: corr, Source: source}
}

func (e *EventReasoningSegmentStarted) Correlation() Correlation { return e.Correlation_ }

var _ CorrelatedEvent = &EventReasoningSegmentStarted{}

type EventReasoningDelta struct {
	EventImpl
	Correlation_ Correlation `json:"correlation"`
	Delta        string      `json:"delta"`
	Text         string      `json:"text"`
	Sequence     int64       `json:"sequence,omitempty"`
}

func NewReasoningDeltaEvent(metadata EventMetadata, corr Correlation, delta, text string, sequence int64) *EventReasoningDelta {
	return &EventReasoningDelta{EventImpl: EventImpl{Type_: EventTypeReasoningDelta, Metadata_: metadata}, Correlation_: corr, Delta: delta, Text: text, Sequence: sequence}
}

func (e *EventReasoningDelta) Correlation() Correlation { return e.Correlation_ }

var _ CorrelatedEvent = &EventReasoningDelta{}

type EventReasoningSegmentFinished struct {
	EventImpl
	Correlation_ Correlation `json:"correlation"`
	Text         string      `json:"text,omitempty"`
	FinishReason string      `json:"finish_reason,omitempty"`
}

func NewReasoningSegmentFinishedEvent(metadata EventMetadata, corr Correlation, text, finishReason string) *EventReasoningSegmentFinished {
	return &EventReasoningSegmentFinished{EventImpl: EventImpl{Type_: EventTypeReasoningSegmentFinished, Metadata_: metadata}, Correlation_: corr, Text: text, FinishReason: finishReason}
}

func (e *EventReasoningSegmentFinished) Correlation() Correlation { return e.Correlation_ }

var _ CorrelatedEvent = &EventReasoningSegmentFinished{}

func (e EventTextDelta) MarshalZerologObject(ev *zerolog.Event) {
	e.EventImpl.MarshalZerologObject(ev)
	ev.Str("delta", e.Delta).Str("text", e.Text)
}

func (e EventTextSegmentFinished) MarshalZerologObject(ev *zerolog.Event) {
	e.EventImpl.MarshalZerologObject(ev)
	ev.Str("text", e.Text).Str("finish_reason", e.FinishReason)
}

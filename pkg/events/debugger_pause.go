package events

// EventDebuggerPause is emitted when the tool loop pauses for step-mode debugging.
type EventDebuggerPause struct {
	EventImpl
	PauseID    string         `json:"pause_id"`
	Phase      string         `json:"phase"`
	Summary    string         `json:"summary"`
	DeadlineMs int64          `json:"deadline_ms"`
	Extra      map[string]any `json:"extra,omitempty"`
}

func NewDebuggerPauseEvent(metadata EventMetadata, pauseID string, phase string, summary string, deadlineMs int64, extra map[string]any) *EventDebuggerPause {
	if extra == nil {
		extra = map[string]any{}
	}
	return &EventDebuggerPause{
		EventImpl: EventImpl{
			Type_:     EventTypeDebuggerPause,
			Metadata_: metadata,
			payload:   nil,
		},
		PauseID:    pauseID,
		Phase:      phase,
		Summary:    summary,
		DeadlineMs: deadlineMs,
		Extra:      extra,
	}
}

var _ Event = &EventDebuggerPause{}

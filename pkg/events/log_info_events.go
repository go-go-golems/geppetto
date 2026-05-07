package events

import "github.com/rs/zerolog"

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

// EventAgentModeSwitch: exported custom event with analysis and new mode
// Message carries a short title; Data should include "from", "to", and optionally "analysis"
type EventAgentModeSwitch struct {
	EventImpl
	Message string                 `json:"message"`
	Data    map[string]interface{} `json:"data,omitempty"`
}

func NewAgentModeSwitchEvent(metadata EventMetadata, from string, to string, analysis string) *EventAgentModeSwitch {
	data := map[string]interface{}{"from": from, "to": to}
	if analysis != "" {
		data["analysis"] = analysis
	}
	return &EventAgentModeSwitch{
		EventImpl: EventImpl{Type_: EventTypeAgentModeSwitch, Metadata_: metadata},
		Message:   "agentmode: mode switched",
		Data:      data,
	}
}

var _ Event = &EventAgentModeSwitch{}

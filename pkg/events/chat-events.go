package events

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type EventType string

const (
	// EventTypeStart to EventTypeFinal are for text completion, actually
	EventTypeStart             EventType = "start"
	EventTypeFinal             EventType = "final"
	EventTypePartialCompletion EventType = "partial"
	// Separate partial stream for reasoning/summary thinking text
	EventTypePartialThinking EventType = "partial-thinking"

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

	// Debugger pause event (step-mode)
	EventTypeDebuggerPause EventType = "debugger.pause"

	// Agent-mode custom event (exported so UIs can act upon it)
	EventTypeAgentModeSwitch EventType = "agent-mode-switch"

	// Web search progress events (built-in/server tools)
	EventTypeWebSearchStarted   EventType = "web-search-started"
	EventTypeWebSearchSearching EventType = "web-search-searching"
	EventTypeWebSearchOpenPage  EventType = "web-search-open-page"
	EventTypeWebSearchDone      EventType = "web-search-done"

	// Citation annotations attached to output text
	EventTypeCitation EventType = "citation"

	// File search progress events
	EventTypeFileSearchStarted   EventType = "file-search-started"
	EventTypeFileSearchSearching EventType = "file-search-searching"
	EventTypeFileSearchDone      EventType = "file-search-done"

	// Code interpreter events
	EventTypeCodeInterpreterStarted      EventType = "code-interpreter-started"
	EventTypeCodeInterpreterInterpreting EventType = "code-interpreter-interpreting"
	EventTypeCodeInterpreterDone         EventType = "code-interpreter-done"
	EventTypeCodeInterpreterCodeDelta    EventType = "code-interpreter-code-delta"
	EventTypeCodeInterpreterCodeDone     EventType = "code-interpreter-code-done"

	// MCP tools
	EventTypeMCPArgsDelta      EventType = "mcp-args-delta"
	EventTypeMCPArgsDone       EventType = "mcp-args-done"
	EventTypeMCPInProgress     EventType = "mcp-in-progress"
	EventTypeMCPCompleted      EventType = "mcp-completed"
	EventTypeMCPFailed         EventType = "mcp-failed"
	EventTypeMCPListInProgress EventType = "mcp-list-tools-in-progress"
	EventTypeMCPListCompleted  EventType = "mcp-list-tools-completed"
	EventTypeMCPListFailed     EventType = "mcp-list-tools-failed"

	// Image generation built-in
	EventTypeImageGenInProgress   EventType = "image-generation-in-progress"
	EventTypeImageGenGenerating   EventType = "image-generation-generating"
	EventTypeImageGenPartialImage EventType = "image-generation-partial-image"
	EventTypeImageGenCompleted    EventType = "image-generation-completed"

	// Normalized tool results
	EventTypeToolSearchResults EventType = "tool-search-results"
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

// SetPayload stores the raw JSON payload on the event implementation.
// This is used by NewEventFromJson and external decoders.
func (e *EventImpl) SetPayload(b []byte) {
	e.payload = b
}

var _ Event = &EventImpl{}

// EventMetadata contains all the information that is passed along with watermill message,
// specific to chat steps.
type EventMetadata struct {
	LLMInferenceData
	ID uuid.UUID `json:"message_id" yaml:"message_id" mapstructure:"message_id"`
	// Correlation identifiers
	SessionID   string `json:"session_id,omitempty" yaml:"session_id,omitempty" mapstructure:"session_id"`
	InferenceID string `json:"inference_id,omitempty" yaml:"inference_id,omitempty" mapstructure:"inference_id"`
	TurnID      string `json:"turn_id,omitempty" yaml:"turn_id,omitempty" mapstructure:"turn_id"`
	// Extra carries provider-specific/context values
	Extra map[string]interface{} `json:"extra,omitempty" yaml:"extra,omitempty" mapstructure:"extra"`
}

func (em EventMetadata) MarshalZerologObject(e *zerolog.Event) {
	e.Str("message_id", em.ID.String())
	if em.SessionID != "" {
		e.Str("session_id", em.SessionID)
	}
	if em.InferenceID != "" {
		e.Str("inference_id", em.InferenceID)
	}
	if em.TurnID != "" {
		e.Str("turn_id", em.TurnID)
	}
	if em.Model != "" {
		e.Str("model", em.Model)
	}
	if em.Temperature != nil {
		e.Float64("temperature", *em.Temperature)
	}
	if em.TopP != nil {
		e.Float64("top_p", *em.TopP)
	}
	if em.MaxTokens != nil {
		e.Int("max_tokens", *em.MaxTokens)
	}
	if em.StopReason != nil && *em.StopReason != "" {
		e.Str("stop_reason", *em.StopReason)
	}
	if em.Usage != nil {
		e.Int("input_tokens", em.Usage.InputTokens)
		e.Int("output_tokens", em.Usage.OutputTokens)
		if em.Usage.CachedTokens > 0 {
			e.Int("cached_tokens", em.Usage.CachedTokens)
		}
		if em.Usage.CacheCreationInputTokens > 0 {
			e.Int("cache_creation_input_tokens", em.Usage.CacheCreationInputTokens)
		}
		if em.Usage.CacheReadInputTokens > 0 {
			e.Int("cache_read_input_tokens", em.Usage.CacheReadInputTokens)
		}
	}
	if em.DurationMs != nil {
		e.Int64("duration_ms", *em.DurationMs)
	}
	if len(em.Extra) > 0 {
		e.Dict("extra", zerolog.Dict().Fields(em.Extra))
	}
}

// Extra metadata keys for correlation
const (
	MetaKeySessionID   = "session_id"
	MetaKeyInferenceID = "inference_id"
	MetaKeyTurnID      = "turn_id"
)

func NewEventFromJson(b []byte) (Event, error) {
	// First, read minimal header to get type.
	var hdr struct {
		Type EventType `json:"type"`
	}
	_ = json.Unmarshal(b, &hdr)

	// If an external decoder is registered, try it first.
	if hdr.Type != "" {
		if dec := lookupDecoder(string(hdr.Type)); dec != nil {
			if ev, err := dec(b); err == nil && ev != nil {
				// Ensure payload is available on embedded EventImpl if present
				if impl, ok := ev.(*EventImpl); ok {
					impl.SetPayload(b)
				} else if setter, ok := ev.(interface{ SetPayload([]byte) }); ok {
					setter.SetPayload(b)
				}
				return ev, nil
			}
		}
	}

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
	case EventTypePartialThinking:
		ret, ok := ToTypedEvent[EventThinkingPartial](e)
		if !ok {
			return nil, fmt.Errorf("could not cast event to EventThinkingPartial")
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
	case EventTypeDebuggerPause:
		ret, ok := ToTypedEvent[EventDebuggerPause](e)
		if !ok {
			return nil, fmt.Errorf("could not cast event to EventDebuggerPause")
		}
		return ret, nil
	case EventTypeAgentModeSwitch:
		ret, ok := ToTypedEvent[EventAgentModeSwitch](e)
		if !ok {
			return nil, fmt.Errorf("could not cast event to EventAgentModeSwitch")
		}
		return ret, nil
	case EventTypeWebSearchStarted:
		ret, ok := ToTypedEvent[EventWebSearchStarted](e)
		if !ok {
			return nil, fmt.Errorf("could not cast event to EventWebSearchStarted")
		}
		return ret, nil
	case EventTypeWebSearchSearching:
		ret, ok := ToTypedEvent[EventWebSearchSearching](e)
		if !ok {
			return nil, fmt.Errorf("could not cast event to EventWebSearchSearching")
		}
		return ret, nil
	case EventTypeWebSearchOpenPage:
		ret, ok := ToTypedEvent[EventWebSearchOpenPage](e)
		if !ok {
			return nil, fmt.Errorf("could not cast event to EventWebSearchOpenPage")
		}
		return ret, nil
	case EventTypeWebSearchDone:
		ret, ok := ToTypedEvent[EventWebSearchDone](e)
		if !ok {
			return nil, fmt.Errorf("could not cast event to EventWebSearchDone")
		}
		return ret, nil
	case EventTypeCitation:
		ret, ok := ToTypedEvent[EventCitation](e)
		if !ok {
			return nil, fmt.Errorf("could not cast event to EventCitation")
		}
		return ret, nil
	case EventTypeFileSearchStarted:
		ret, ok := ToTypedEvent[EventFileSearchStarted](e)
		if !ok {
			return nil, fmt.Errorf("could not cast event to EventFileSearchStarted")
		}
		return ret, nil
	case EventTypeFileSearchSearching:
		ret, ok := ToTypedEvent[EventFileSearchSearching](e)
		if !ok {
			return nil, fmt.Errorf("could not cast event to EventFileSearchSearching")
		}
		return ret, nil
	case EventTypeFileSearchDone:
		ret, ok := ToTypedEvent[EventFileSearchDone](e)
		if !ok {
			return nil, fmt.Errorf("could not cast event to EventFileSearchDone")
		}
		return ret, nil
	case EventTypeCodeInterpreterStarted:
		ret, ok := ToTypedEvent[EventCodeInterpreterStarted](e)
		if !ok {
			return nil, fmt.Errorf("could not cast event to EventCodeInterpreterStarted")
		}
		return ret, nil
	case EventTypeCodeInterpreterInterpreting:
		ret, ok := ToTypedEvent[EventCodeInterpreterInterpreting](e)
		if !ok {
			return nil, fmt.Errorf("could not cast event to EventCodeInterpreterInterpreting")
		}
		return ret, nil
	case EventTypeCodeInterpreterDone:
		ret, ok := ToTypedEvent[EventCodeInterpreterDone](e)
		if !ok {
			return nil, fmt.Errorf("could not cast event to EventCodeInterpreterDone")
		}
		return ret, nil
	case EventTypeCodeInterpreterCodeDelta:
		ret, ok := ToTypedEvent[EventCodeInterpreterCodeDelta](e)
		if !ok {
			return nil, fmt.Errorf("could not cast event to EventCodeInterpreterCodeDelta")
		}
		return ret, nil
	case EventTypeCodeInterpreterCodeDone:
		ret, ok := ToTypedEvent[EventCodeInterpreterCodeDone](e)
		if !ok {
			return nil, fmt.Errorf("could not cast event to EventCodeInterpreterCodeDone")
		}
		return ret, nil
	case EventTypeMCPArgsDelta:
		ret, ok := ToTypedEvent[EventMCPArgsDelta](e)
		if !ok {
			return nil, fmt.Errorf("could not cast event to EventMCPArgsDelta")
		}
		return ret, nil
	case EventTypeMCPArgsDone:
		ret, ok := ToTypedEvent[EventMCPArgsDone](e)
		if !ok {
			return nil, fmt.Errorf("could not cast event to EventMCPArgsDone")
		}
		return ret, nil
	case EventTypeMCPInProgress:
		ret, ok := ToTypedEvent[EventMCPInProgress](e)
		if !ok {
			return nil, fmt.Errorf("could not cast event to EventMCPInProgress")
		}
		return ret, nil
	case EventTypeMCPCompleted:
		ret, ok := ToTypedEvent[EventMCPCompleted](e)
		if !ok {
			return nil, fmt.Errorf("could not cast event to EventMCPCompleted")
		}
		return ret, nil
	case EventTypeMCPFailed:
		ret, ok := ToTypedEvent[EventMCPFailed](e)
		if !ok {
			return nil, fmt.Errorf("could not cast event to EventMCPFailed")
		}
		return ret, nil
	case EventTypeMCPListInProgress:
		ret, ok := ToTypedEvent[EventMCPListInProgress](e)
		if !ok {
			return nil, fmt.Errorf("could not cast event to EventMCPListInProgress")
		}
		return ret, nil
	case EventTypeMCPListCompleted:
		ret, ok := ToTypedEvent[EventMCPListCompleted](e)
		if !ok {
			return nil, fmt.Errorf("could not cast event to EventMCPListCompleted")
		}
		return ret, nil
	case EventTypeMCPListFailed:
		ret, ok := ToTypedEvent[EventMCPListFailed](e)
		if !ok {
			return nil, fmt.Errorf("could not cast event to EventMCPListFailed")
		}
		return ret, nil
	case EventTypeImageGenInProgress:
		ret, ok := ToTypedEvent[EventImageGenInProgress](e)
		if !ok {
			return nil, fmt.Errorf("could not cast event to EventImageGenInProgress")
		}
		return ret, nil
	case EventTypeImageGenGenerating:
		ret, ok := ToTypedEvent[EventImageGenGenerating](e)
		if !ok {
			return nil, fmt.Errorf("could not cast event to EventImageGenGenerating")
		}
		return ret, nil
	case EventTypeImageGenPartialImage:
		ret, ok := ToTypedEvent[EventImageGenPartialImage](e)
		if !ok {
			return nil, fmt.Errorf("could not cast event to EventImageGenPartialImage")
		}
		return ret, nil
	case EventTypeImageGenCompleted:
		ret, ok := ToTypedEvent[EventImageGenCompleted](e)
		if !ok {
			return nil, fmt.Errorf("could not cast event to EventImageGenCompleted")
		}
		return ret, nil
	case EventTypeToolSearchResults:
		ret, ok := ToTypedEvent[EventToolSearchResults](e)
		if !ok {
			return nil, fmt.Errorf("could not cast event to EventToolSearchResults")
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

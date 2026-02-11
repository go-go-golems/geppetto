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

	// Reasoning text (not summary)
	EventTypeReasoningTextDelta EventType = "reasoning-text-delta"
	EventTypeReasoningTextDone  EventType = "reasoning-text-done"

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

// MetadataToolCallsSlug is the slug used to store ToolCall metadata as returned by the openai API
// TODO(manuel, 2024-07-04) This needs to deleted once we have a good way to do tool calling
const MetadataToolCallsSlug = "tool-calls"

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
	case EventTypeReasoningTextDelta:
		ret, ok := ToTypedEvent[EventReasoningTextDelta](e)
		if !ok {
			return nil, fmt.Errorf("could not cast event to EventReasoningTextDelta")
		}
		return ret, nil
	case EventTypeReasoningTextDone:
		ret, ok := ToTypedEvent[EventReasoningTextDone](e)
		if !ok {
			return nil, fmt.Errorf("could not cast event to EventReasoningTextDone")
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

// --- Web search custom events ---

type EventWebSearchStarted struct {
	EventImpl
	ItemID string `json:"item_id,omitempty"`
	Query  string `json:"query,omitempty"`
}

func NewWebSearchStarted(metadata EventMetadata, itemID, query string) *EventWebSearchStarted {
	return &EventWebSearchStarted{EventImpl: EventImpl{Type_: EventTypeWebSearchStarted, Metadata_: metadata}, ItemID: itemID, Query: query}
}

type EventWebSearchSearching struct {
	EventImpl
	ItemID string `json:"item_id,omitempty"`
}

func NewWebSearchSearching(metadata EventMetadata, itemID string) *EventWebSearchSearching {
	return &EventWebSearchSearching{EventImpl: EventImpl{Type_: EventTypeWebSearchSearching, Metadata_: metadata}, ItemID: itemID}
}

type EventWebSearchOpenPage struct {
	EventImpl
	ItemID string `json:"item_id,omitempty"`
	URL    string `json:"url,omitempty"`
}

func NewWebSearchOpenPage(metadata EventMetadata, itemID, url string) *EventWebSearchOpenPage {
	return &EventWebSearchOpenPage{EventImpl: EventImpl{Type_: EventTypeWebSearchOpenPage, Metadata_: metadata}, ItemID: itemID, URL: url}
}

type EventWebSearchDone struct {
	EventImpl
	ItemID string `json:"item_id,omitempty"`
}

func NewWebSearchDone(metadata EventMetadata, itemID string) *EventWebSearchDone {
	return &EventWebSearchDone{EventImpl: EventImpl{Type_: EventTypeWebSearchDone, Metadata_: metadata}, ItemID: itemID}
}

// Citation event attached to streamed output text
type EventCitation struct {
	EventImpl
	Title           string `json:"title,omitempty"`
	URL             string `json:"url,omitempty"`
	StartIndex      *int   `json:"start_index,omitempty"`
	EndIndex        *int   `json:"end_index,omitempty"`
	OutputIndex     *int   `json:"output_index,omitempty"`
	ContentIndex    *int   `json:"content_index,omitempty"`
	AnnotationIndex *int   `json:"annotation_index,omitempty"`
}

func NewCitation(metadata EventMetadata, title, url string, start, end, outputIdx, contentIdx, annIdx *int) *EventCitation {
	return &EventCitation{EventImpl: EventImpl{Type_: EventTypeCitation, Metadata_: metadata}, Title: title, URL: url, StartIndex: start, EndIndex: end, OutputIndex: outputIdx, ContentIndex: contentIdx, AnnotationIndex: annIdx}
}

// File search custom events
type EventFileSearchStarted struct {
	EventImpl
	ItemID string `json:"item_id,omitempty"`
}

func NewFileSearchStarted(metadata EventMetadata, itemID string) *EventFileSearchStarted {
	return &EventFileSearchStarted{EventImpl: EventImpl{Type_: EventTypeFileSearchStarted, Metadata_: metadata}, ItemID: itemID}
}

type EventFileSearchSearching struct {
	EventImpl
	ItemID string `json:"item_id,omitempty"`
}

func NewFileSearchSearching(metadata EventMetadata, itemID string) *EventFileSearchSearching {
	return &EventFileSearchSearching{EventImpl: EventImpl{Type_: EventTypeFileSearchSearching, Metadata_: metadata}, ItemID: itemID}
}

type EventFileSearchDone struct {
	EventImpl
	ItemID string `json:"item_id,omitempty"`
}

func NewFileSearchDone(metadata EventMetadata, itemID string) *EventFileSearchDone {
	return &EventFileSearchDone{EventImpl: EventImpl{Type_: EventTypeFileSearchDone, Metadata_: metadata}, ItemID: itemID}
}

// Code interpreter custom events
type EventCodeInterpreterStarted struct {
	EventImpl
	ItemID string `json:"item_id,omitempty"`
}

func NewCodeInterpreterStarted(metadata EventMetadata, itemID string) *EventCodeInterpreterStarted {
	return &EventCodeInterpreterStarted{EventImpl: EventImpl{Type_: EventTypeCodeInterpreterStarted, Metadata_: metadata}, ItemID: itemID}
}

type EventCodeInterpreterInterpreting struct {
	EventImpl
	ItemID string `json:"item_id,omitempty"`
}

func NewCodeInterpreterInterpreting(metadata EventMetadata, itemID string) *EventCodeInterpreterInterpreting {
	return &EventCodeInterpreterInterpreting{EventImpl: EventImpl{Type_: EventTypeCodeInterpreterInterpreting, Metadata_: metadata}, ItemID: itemID}
}

type EventCodeInterpreterDone struct {
	EventImpl
	ItemID string `json:"item_id,omitempty"`
}

func NewCodeInterpreterDone(metadata EventMetadata, itemID string) *EventCodeInterpreterDone {
	return &EventCodeInterpreterDone{EventImpl: EventImpl{Type_: EventTypeCodeInterpreterDone, Metadata_: metadata}, ItemID: itemID}
}

type EventCodeInterpreterCodeDelta struct {
	EventImpl
	ItemID string `json:"item_id,omitempty"`
	Delta  string `json:"delta"`
}

func NewCodeInterpreterCodeDelta(metadata EventMetadata, itemID, delta string) *EventCodeInterpreterCodeDelta {
	return &EventCodeInterpreterCodeDelta{EventImpl: EventImpl{Type_: EventTypeCodeInterpreterCodeDelta, Metadata_: metadata}, ItemID: itemID, Delta: delta}
}

type EventCodeInterpreterCodeDone struct {
	EventImpl
	ItemID string `json:"item_id,omitempty"`
	Code   string `json:"code"`
}

func NewCodeInterpreterCodeDone(metadata EventMetadata, itemID, code string) *EventCodeInterpreterCodeDone {
	return &EventCodeInterpreterCodeDone{EventImpl: EventImpl{Type_: EventTypeCodeInterpreterCodeDone, Metadata_: metadata}, ItemID: itemID, Code: code}
}

// Reasoning text
type EventReasoningTextDelta struct {
	EventImpl
	Delta string `json:"delta"`
}

func NewReasoningTextDelta(metadata EventMetadata, delta string) *EventReasoningTextDelta {
	return &EventReasoningTextDelta{EventImpl: EventImpl{Type_: EventTypeReasoningTextDelta, Metadata_: metadata}, Delta: delta}
}

type EventReasoningTextDone struct {
	EventImpl
	Text string `json:"text"`
}

func NewReasoningTextDone(metadata EventMetadata, text string) *EventReasoningTextDone {
	return &EventReasoningTextDone{EventImpl: EventImpl{Type_: EventTypeReasoningTextDone, Metadata_: metadata}, Text: text}
}

// MCP
type EventMCPArgsDelta struct {
	EventImpl
	ItemID string `json:"item_id,omitempty"`
	Delta  string `json:"delta"`
}

func NewMCPArgsDelta(metadata EventMetadata, itemID, delta string) *EventMCPArgsDelta {
	return &EventMCPArgsDelta{EventImpl: EventImpl{Type_: EventTypeMCPArgsDelta, Metadata_: metadata}, ItemID: itemID, Delta: delta}
}

type EventMCPArgsDone struct {
	EventImpl
	ItemID    string `json:"item_id,omitempty"`
	Arguments string `json:"arguments"`
}

func NewMCPArgsDone(metadata EventMetadata, itemID, args string) *EventMCPArgsDone {
	return &EventMCPArgsDone{EventImpl: EventImpl{Type_: EventTypeMCPArgsDone, Metadata_: metadata}, ItemID: itemID, Arguments: args}
}

type EventMCPInProgress struct {
	EventImpl
	ItemID string `json:"item_id,omitempty"`
}

func NewMCPInProgress(metadata EventMetadata, itemID string) *EventMCPInProgress {
	return &EventMCPInProgress{EventImpl: EventImpl{Type_: EventTypeMCPInProgress, Metadata_: metadata}, ItemID: itemID}
}

type EventMCPCompleted struct {
	EventImpl
	ItemID string `json:"item_id,omitempty"`
}

func NewMCPCompleted(metadata EventMetadata, itemID string) *EventMCPCompleted {
	return &EventMCPCompleted{EventImpl: EventImpl{Type_: EventTypeMCPCompleted, Metadata_: metadata}, ItemID: itemID}
}

type EventMCPFailed struct {
	EventImpl
	ItemID string `json:"item_id,omitempty"`
}

func NewMCPFailed(metadata EventMetadata, itemID string) *EventMCPFailed {
	return &EventMCPFailed{EventImpl: EventImpl{Type_: EventTypeMCPFailed, Metadata_: metadata}, ItemID: itemID}
}

type EventMCPListInProgress struct {
	EventImpl
	ItemID string `json:"item_id,omitempty"`
}

func NewMCPListInProgress(metadata EventMetadata, itemID string) *EventMCPListInProgress {
	return &EventMCPListInProgress{EventImpl: EventImpl{Type_: EventTypeMCPListInProgress, Metadata_: metadata}, ItemID: itemID}
}

type EventMCPListCompleted struct {
	EventImpl
	ItemID string `json:"item_id,omitempty"`
}

func NewMCPListCompleted(metadata EventMetadata, itemID string) *EventMCPListCompleted {
	return &EventMCPListCompleted{EventImpl: EventImpl{Type_: EventTypeMCPListCompleted, Metadata_: metadata}, ItemID: itemID}
}

type EventMCPListFailed struct {
	EventImpl
	ItemID string `json:"item_id,omitempty"`
}

func NewMCPListFailed(metadata EventMetadata, itemID string) *EventMCPListFailed {
	return &EventMCPListFailed{EventImpl: EventImpl{Type_: EventTypeMCPListFailed, Metadata_: metadata}, ItemID: itemID}
}

// Image generation
type EventImageGenInProgress struct {
	EventImpl
	ItemID string `json:"item_id,omitempty"`
}

func NewImageGenInProgress(metadata EventMetadata, itemID string) *EventImageGenInProgress {
	return &EventImageGenInProgress{EventImpl: EventImpl{Type_: EventTypeImageGenInProgress, Metadata_: metadata}, ItemID: itemID}
}

type EventImageGenGenerating struct {
	EventImpl
	ItemID string `json:"item_id,omitempty"`
}

func NewImageGenGenerating(metadata EventMetadata, itemID string) *EventImageGenGenerating {
	return &EventImageGenGenerating{EventImpl: EventImpl{Type_: EventTypeImageGenGenerating, Metadata_: metadata}, ItemID: itemID}
}

type EventImageGenPartialImage struct {
	EventImpl
	ItemID             string `json:"item_id,omitempty"`
	PartialImageBase64 string `json:"partial_image_base64,omitempty"`
}

func NewImageGenPartialImage(metadata EventMetadata, itemID, b64 string) *EventImageGenPartialImage {
	return &EventImageGenPartialImage{EventImpl: EventImpl{Type_: EventTypeImageGenPartialImage, Metadata_: metadata}, ItemID: itemID, PartialImageBase64: b64}
}

type EventImageGenCompleted struct {
	EventImpl
	ItemID string `json:"item_id,omitempty"`
}

func NewImageGenCompleted(metadata EventMetadata, itemID string) *EventImageGenCompleted {
	return &EventImageGenCompleted{EventImpl: EventImpl{Type_: EventTypeImageGenCompleted, Metadata_: metadata}, ItemID: itemID}
}

// Normalized results
type SearchResult struct {
	URL        string         `json:"url,omitempty"`
	Title      string         `json:"title,omitempty"`
	Snippet    string         `json:"snippet,omitempty"`
	Extensions map[string]any `json:"ext,omitempty"`
}
type EventToolSearchResults struct {
	EventImpl
	Tool    string         `json:"tool"`
	ItemID  string         `json:"item_id,omitempty"`
	Results []SearchResult `json:"results"`
}

func NewToolSearchResults(metadata EventMetadata, tool, itemID string, res []SearchResult) *EventToolSearchResults {
	return &EventToolSearchResults{EventImpl: EventImpl{Type_: EventTypeToolSearchResults, Metadata_: metadata}, Tool: tool, ItemID: itemID, Results: res}
}

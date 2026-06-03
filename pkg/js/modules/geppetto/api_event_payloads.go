package geppetto

import (
	"time"

	"github.com/go-go-golems/geppetto/pkg/events"
)

func encodeGeppettoEventPayload(ev events.Event) map[string]any {
	meta := ev.Metadata()
	payload := map[string]any{
		"type":        string(ev.Type()),
		"timestampMs": time.Now().UnixMilli(),
	}
	if meta.SessionID != "" {
		payload["sessionId"] = meta.SessionID
	}
	if meta.InferenceID != "" {
		payload["inferenceId"] = meta.InferenceID
	}
	if meta.TurnID != "" {
		payload["turnId"] = meta.TurnID
	}
	if len(meta.Extra) > 0 {
		payload["metaExtra"] = cloneJSONValue(meta.Extra)
	}

	switch e := ev.(type) {
	case events.CorrelatedEvent:
		payload["correlation"] = cloneJSONValue(e.Correlation())
	}

	switch e := ev.(type) {
	case *events.EventRunStarted:
		payload["prompt"] = e.Prompt
	case *events.EventRunFinished:
		payload["status"] = e.Status
	case *events.EventRunStopped:
		payload["reason"] = e.Reason
	case *events.EventRunFailed:
		payload["error"] = e.ErrorString
		payload["message"] = e.ErrorString
	case *events.EventProviderCallMetadataUpdated:
		payload["stopReason"] = e.StopReason
		payload["stopSequence"] = e.StopSequence
		if e.Usage != nil {
			payload["usage"] = cloneJSONValue(e.Usage)
		}
	case *events.EventProviderCallFinished:
		payload["stopReason"] = e.StopReason
		payload["finishClass"] = e.FinishClass
		payload["hasToolCalls"] = e.HasToolCalls
		if e.Usage != nil {
			payload["usage"] = cloneJSONValue(e.Usage)
		}
		if e.DurationMs != nil {
			payload["durationMs"] = *e.DurationMs
		}
	case *events.EventTextSegmentStarted:
		payload["role"] = e.Role
	case *events.EventTextDelta:
		payload["delta"] = e.Delta
		payload["text"] = e.Text
		payload["sequence"] = e.Sequence
	case *events.EventTextSegmentFinished:
		payload["text"] = e.Text
		payload["finishReason"] = e.FinishReason
	case *events.EventReasoningSegmentStarted:
		if e.Source != "" {
			payload["source"] = e.Source
		}
	case *events.EventReasoningDelta:
		payload["delta"] = e.Delta
		payload["text"] = e.Text
		payload["sequence"] = e.Sequence
		if e.Source != "" {
			payload["source"] = e.Source
		}
	case *events.EventReasoningSegmentFinished:
		payload["text"] = e.Text
		payload["finishReason"] = e.FinishReason
		if e.Source != "" {
			payload["source"] = e.Source
		}
	case *events.EventToolCallStarted:
		payload["toolCall"] = map[string]any{
			"id":   e.ToolCallID,
			"name": e.ToolName,
		}
	case *events.EventToolCallArgumentsDelta:
		payload["toolCall"] = map[string]any{
			"id":        e.ToolCallID,
			"delta":     e.Delta,
			"arguments": e.Arguments,
			"sequence":  e.Sequence,
		}
	case *events.EventToolCallRequested:
		payload["toolCall"] = map[string]any{
			"id":    e.ToolCallID,
			"name":  e.ToolName,
			"input": e.Input,
		}
	case *events.EventToolExecutionStarted:
		payload["toolCall"] = map[string]any{
			"id":    e.ToolCallID,
			"name":  e.ToolName,
			"input": e.Input,
		}
	case *events.EventToolResultReady:
		payload["toolResult"] = map[string]any{
			"id":     e.ToolCallID,
			"name":   e.ToolName,
			"result": e.Result,
			"status": e.Status,
		}
	case *events.EventToolCallFinished:
		payload["toolCall"] = map[string]any{
			"id":     e.ToolCallID,
			"name":   e.ToolName,
			"status": e.Status,
		}
	case *events.EventError:
		payload["error"] = e.ErrorString
		payload["message"] = e.ErrorString
	case *events.EventInterrupt:
		payload["text"] = e.Text
	case *events.EventLog:
		payload["level"] = e.Level
		payload["message"] = e.Message
		if len(e.Fields) > 0 {
			payload["fields"] = cloneJSONValue(e.Fields)
		}
	case *events.EventInfo:
		payload["message"] = e.Message
		if len(e.Data) > 0 {
			payload["data"] = cloneJSONValue(e.Data)
		}
	case *events.EventAgentModeSwitch:
		payload["message"] = e.Message
		if len(e.Data) > 0 {
			payload["data"] = cloneJSONValue(e.Data)
		}
	}
	if raw := ev.Payload(); len(raw) > 0 {
		payload["rawPayload"] = string(raw)
	}
	return payload
}

func eventEmitterNamesForPayload(payload map[string]any) []string {
	names := []string{"event"}
	eventType, _ := payload["type"].(string)
	if eventType == "" {
		return names
	}
	if eventType == string(events.EventTypeError) {
		eventType = "inference-error"
	}
	if eventType != "event" {
		names = append(names, eventType)
	}
	return names
}

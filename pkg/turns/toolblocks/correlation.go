package toolblocks

import (
	"encoding/json"
	"strings"

	"github.com/go-go-golems/geppetto/pkg/events"
	"github.com/go-go-golems/geppetto/pkg/turns"
)

const PayloadKeyCorrelation = "correlation"

// NewToolCallBlockWithCorrelation returns a tool_call block that preserves the
// canonical provider/request correlation needed by host-side execution events.
func NewToolCallBlockWithCorrelation(id string, name string, args any, corr events.Correlation) turns.Block {
	b := turns.NewToolCallBlock(id, name, args)
	AttachToolCallCorrelation(&b, corr)
	return b
}

// AttachToolCallCorrelation stores canonical correlation on a tool_call block.
// The stored value is debug/persistence friendly JSON data; consumers must still
// treat IDs as opaque strings.
func AttachToolCallCorrelation(b *turns.Block, corr events.Correlation) {
	if b == nil {
		return
	}
	if corr.ToolCallID == "" {
		if id, _ := b.Payload[turns.PayloadKeyID].(string); id != "" {
			corr.ToolCallID = id
		}
	}
	if isEmptyCorrelation(corr) {
		return
	}
	if b.Payload == nil {
		b.Payload = map[string]any{}
	}
	b.Payload[PayloadKeyCorrelation] = corr
}

func toolCallCorrelationFromBlock(b turns.Block) (events.Correlation, bool) {
	return decodeCorrelation(b.Payload[PayloadKeyCorrelation])
}

func decodeCorrelation(raw any) (events.Correlation, bool) {
	switch v := raw.(type) {
	case nil:
		return events.Correlation{}, false
	case events.Correlation:
		return v, !isEmptyCorrelation(v)
	case map[string]any:
		corr := events.Correlation{
			SessionID:      stringFromMap(v, "session_id", "sessionId"),
			RunID:          stringFromMap(v, "run_id", "runId"),
			TurnID:         stringFromMap(v, "turn_id", "turnId"),
			ProviderCallID: stringFromMap(v, "provider_call_id", "providerCallId"),
			SegmentID:      stringFromMap(v, "segment_id", "segmentId"),
			ToolCallID:     stringFromMap(v, "tool_call_id", "toolCallId"),
		}
		return corr, !isEmptyCorrelation(corr)
	case map[string]string:
		corr := events.Correlation{
			SessionID:      firstString(v, "session_id", "sessionId"),
			RunID:          firstString(v, "run_id", "runId"),
			TurnID:         firstString(v, "turn_id", "turnId"),
			ProviderCallID: firstString(v, "provider_call_id", "providerCallId"),
			SegmentID:      firstString(v, "segment_id", "segmentId"),
			ToolCallID:     firstString(v, "tool_call_id", "toolCallId"),
		}
		return corr, !isEmptyCorrelation(corr)
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return events.Correlation{}, false
		}
		var corr events.Correlation
		if err := json.Unmarshal(b, &corr); err != nil {
			return events.Correlation{}, false
		}
		return corr, !isEmptyCorrelation(corr)
	}
}

func isEmptyCorrelation(corr events.Correlation) bool {
	return strings.TrimSpace(corr.SessionID) == "" &&
		strings.TrimSpace(corr.RunID) == "" &&
		strings.TrimSpace(corr.TurnID) == "" &&
		strings.TrimSpace(corr.ProviderCallID) == "" &&
		strings.TrimSpace(corr.SegmentID) == "" &&
		strings.TrimSpace(corr.ToolCallID) == ""
}

func stringFromMap(m map[string]any, keys ...string) string {
	for _, key := range keys {
		if v, ok := m[key]; ok {
			if s, ok := v.(string); ok {
				return s
			}
		}
	}
	return ""
}

func firstString(m map[string]string, keys ...string) string {
	for _, key := range keys {
		if v := m[key]; v != "" {
			return v
		}
	}
	return ""
}

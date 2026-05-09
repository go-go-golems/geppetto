package events

import (
	"encoding/json"
	"testing"
)

func sampleCorrelation() Correlation {
	return Correlation{
		SessionID:      "session-1",
		RunID:          "run-1",
		TurnID:         "turn-1",
		ProviderCallID: "provider-call-1",
		SegmentID:      "segment-1",
		ToolCallID:     "tool-1",
	}
}

func TestCanonicalEventsRoundTripCorrelation(t *testing.T) {
	metadata := EventMetadata{SessionID: "session-1", InferenceID: "inference-1", TurnID: "turn-1"}
	corr := sampleCorrelation()
	duration := int64(42)
	usage := &Usage{InputTokens: 10, OutputTokens: 20, CacheCreationInputTokens: 1, CacheReadInputTokens: 2}

	tests := []struct {
		name  string
		event Event
	}{
		{"run-started", NewRunStartedEvent(metadata, corr, "prompt")},
		{"run-finished", NewRunFinishedEvent(metadata, corr, "finished")},
		{"provider-started", NewProviderCallStartedEvent(metadata, corr)},
		{"provider-metadata", NewProviderCallMetadataUpdatedEvent(metadata, corr, "tool_use", "", usage)},
		{"provider-finished", NewProviderCallFinishedEvent(metadata, corr, "tool_use", "tool_calls_pending", usage, &duration, true)},
		{"text-started", NewTextSegmentStartedEvent(metadata, corr, "assistant")},
		{"text-delta", NewTextDeltaEvent(metadata, corr, "hello", "hello", 1)},
		{"text-finished", NewTextSegmentFinishedEvent(metadata, corr, "hello", "content_block_stop")},
		{"reasoning-started", NewReasoningSegmentStartedEvent(metadata, corr, "provider")},
		{"reasoning-delta", NewReasoningDeltaEvent(metadata, corr, "why", "why", 1)},
		{"reasoning-finished", NewReasoningSegmentFinishedEvent(metadata, corr, "why", "done")},
		{"tool-started", NewToolCallStartedEvent(metadata, corr, "tool-1", "sql_doc")},
		{"tool-args", NewToolCallArgumentsDeltaEvent(metadata, corr, "tool-1", `{"a":`, `{"a":`, 1)},
		{"tool-requested", NewToolCallRequestedEvent(metadata, corr, "tool-1", "sql_doc", `{"a":1}`)},
		{"tool-execution", NewToolExecutionStartedEvent(metadata, corr, "tool-1", "sql_doc", `{"a":1}`)},
		{"tool-result", NewToolResultReadyEvent(metadata, corr, "tool-1", "sql_doc", `{"ok":true}`, "success")},
		{"tool-finished", NewToolCallFinishedEvent(metadata, corr, "tool-1", "sql_doc", "completed")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := json.Marshal(tt.event)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			decoded, err := NewEventFromJson(b)
			if err != nil {
				t.Fatalf("decode %s: %v\n%s", tt.name, err, string(b))
			}
			if decoded.Type() != tt.event.Type() {
				t.Fatalf("type mismatch: got %s want %s", decoded.Type(), tt.event.Type())
			}
			if err := ValidateCanonicalEvent(decoded); err != nil {
				t.Fatalf("validate canonical event: %v", err)
			}

			correlated, ok := decoded.(CorrelatedEvent)
			if !ok {
				t.Fatalf("decoded event %T does not implement CorrelatedEvent", decoded)
			}
			got := correlated.Correlation()
			if got.RunID != corr.RunID || got.ProviderCallID != corr.ProviderCallID || got.SegmentID != corr.SegmentID || got.ToolCallID != corr.ToolCallID {
				t.Fatalf("correlation mismatch: got %+v want %+v", got, corr)
			}
		})
	}
}

func TestValidateCanonicalEventRejectsMissingCorrelationFields(t *testing.T) {
	metadata := EventMetadata{SessionID: "session-1", InferenceID: "inference-1", TurnID: "turn-1"}

	if err := ValidateCanonicalEvent(NewTextDeltaEvent(metadata, Correlation{RunID: "run-1", ProviderCallID: "provider-1"}, "x", "x", 1)); err == nil {
		t.Fatalf("expected text delta without segment_id to fail validation")
	}

	if err := ValidateCanonicalEvent(NewProviderCallFinishedEvent(metadata, Correlation{RunID: "run-1"}, "end_turn", "completed", nil, nil, false)); err == nil {
		t.Fatalf("expected provider finish without provider_call_id to fail validation")
	}

	if err := ValidateCanonicalEvent(NewToolCallRequestedEvent(metadata, Correlation{RunID: "run-1"}, "", "sql_doc", "{}")); err == nil {
		t.Fatalf("expected tool request without tool_call_id to fail validation")
	}

	if err := ValidateCanonicalEvent(NewTextDeltaEvent(metadata, Correlation{RunID: "run-1", SegmentID: "segment-1"}, "x", "x", 1)); err == nil {
		t.Fatalf("expected text delta without provider_call_id to fail validation")
	}
}

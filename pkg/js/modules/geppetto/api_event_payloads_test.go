package geppetto

import (
	"errors"
	"testing"

	"github.com/go-go-golems/geppetto/pkg/events"
)

func TestEncodeGeppettoEventPayloadLifecycleAndProviderFields(t *testing.T) {
	duration := int64(42)
	usage := &events.Usage{InputTokens: 10, OutputTokens: 20, CachedTokens: 3}
	meta := events.EventMetadata{SessionID: "session-1", InferenceID: "inference-1", TurnID: "turn-1"}
	corr := events.Correlation{SessionID: "session-1", RunID: "run-1", TurnID: "turn-1", ProviderCallID: "provider-1"}

	cases := []struct {
		name string
		ev   events.Event
		want map[string]any
	}{
		{
			name: "run-started",
			ev:   events.NewRunStartedEvent(meta, corr, "hello"),
			want: map[string]any{"type": "run-started", "prompt": "hello"},
		},
		{
			name: "run-finished",
			ev:   events.NewRunFinishedEvent(meta, corr, "success"),
			want: map[string]any{"type": "run-finished", "status": "success"},
		},
		{
			name: "run-stopped",
			ev:   events.NewRunStoppedEvent(meta, corr, "cancelled"),
			want: map[string]any{"type": "run-stopped", "reason": "cancelled"},
		},
		{
			name: "run-failed",
			ev:   events.NewRunFailedEvent(meta, corr, errors.New("boom")),
			want: map[string]any{"type": "run-failed", "error": "boom", "message": "boom"},
		},
		{
			name: "provider-call-metadata-updated",
			ev:   events.NewProviderCallMetadataUpdatedEvent(meta, corr, "stop", "seq", usage),
			want: map[string]any{"type": "provider-call-metadata-updated", "stopReason": "stop", "stopSequence": "seq"},
		},
		{
			name: "provider-call-finished",
			ev:   events.NewProviderCallFinishedEvent(meta, corr, "stop", "normal", usage, &duration, true),
			want: map[string]any{"type": "provider-call-finished", "stopReason": "stop", "finishClass": "normal", "durationMs": int64(42), "hasToolCalls": true},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := encodeGeppettoEventPayload(tc.ev)
			for key, want := range tc.want {
				if got[key] != want {
					t.Fatalf("payload[%s] = %#v, want %#v in %#v", key, got[key], want, got)
				}
			}
			if got["sessionId"] != "session-1" || got["inferenceId"] != "inference-1" || got["turnId"] != "turn-1" {
				t.Fatalf("missing metadata ids in %#v", got)
			}
			corrMap, ok := got["correlation"].(map[string]any)
			if !ok || corrMap["run_id"] != "run-1" {
				t.Fatalf("missing correlation in %#v", got)
			}
		})
	}
}

func TestEncodeGeppettoEventPayloadSegmentLogInfoAndToolFields(t *testing.T) {
	meta := events.EventMetadata{SessionID: "session-1"}
	corr := events.Correlation{SessionID: "session-1", SegmentID: "segment-1", ToolCallID: "tool-1"}

	cases := []struct {
		name string
		ev   events.Event
		want map[string]any
	}{
		{
			name: "text-segment-started",
			ev:   events.NewTextSegmentStartedEvent(meta, corr, "assistant"),
			want: map[string]any{"type": "text-segment-started", "role": "assistant"},
		},
		{
			name: "reasoning-segment-started",
			ev:   events.NewReasoningSegmentStartedEvent(meta, corr, "summary"),
			want: map[string]any{"type": "reasoning-segment-started", "source": "summary"},
		},
		{
			name: "log",
			ev:   events.NewLogEvent(meta, "warn", "check", map[string]any{"key": "value"}),
			want: map[string]any{"type": "log", "level": "warn", "message": "check"},
		},
		{
			name: "info",
			ev:   events.NewInfoEvent(meta, "heads up", map[string]any{"ok": true}),
			want: map[string]any{"type": "info", "message": "heads up"},
		},
		{
			name: "agent-mode-switch",
			ev:   events.NewAgentModeSwitchEvent(meta, "search", "answer", "done"),
			want: map[string]any{"type": "agent-mode-switch", "message": "agentmode: mode switched"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := encodeGeppettoEventPayload(tc.ev)
			for key, want := range tc.want {
				if got[key] != want {
					t.Fatalf("payload[%s] = %#v, want %#v in %#v", key, got[key], want, got)
				}
			}
		})
	}
}

func TestEventEmitterNamesForPayloadOrderingAndErrorMapping(t *testing.T) {
	if got := eventEmitterNamesForPayload(map[string]any{"type": "text-delta"}); len(got) != 2 || got[0] != "event" || got[1] != "text-delta" {
		t.Fatalf("text-delta names = %#v", got)
	}
	if got := eventEmitterNamesForPayload(map[string]any{"type": "error"}); len(got) != 2 || got[0] != "event" || got[1] != "inference-error" {
		t.Fatalf("error names = %#v", got)
	}
}

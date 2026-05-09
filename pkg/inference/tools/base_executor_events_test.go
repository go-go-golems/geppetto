package tools

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/go-go-golems/geppetto/pkg/events"
)

type captureEventSink struct {
	events []events.Event
}

func (s *captureEventSink) PublishEvent(event events.Event) error {
	s.events = append(s.events, event)
	return nil
}

func TestBaseToolExecutorPublishesCanonicalToolLifecycleEvents(t *testing.T) {
	registry := NewInMemoryToolRegistry()
	def, err := NewToolFromFunc("hello", "returns hello", func() (map[string]string, error) {
		return map[string]string{"message": "hello"}, nil
	})
	if err != nil {
		t.Fatalf("NewToolFromFunc: %v", err)
	}
	if err := registry.RegisterTool("hello", *def); err != nil {
		t.Fatalf("RegisterTool: %v", err)
	}

	sink := &captureEventSink{}
	ctx := events.WithEventSinks(context.Background(), sink)
	ctx = WithCurrentToolCorrelation(ctx, events.Correlation{RunID: "run-1", ProviderCallID: "provider-call-1", ToolCallID: "call-1"})
	executor := NewBaseToolExecutor(DefaultToolConfig())

	result, err := executor.ExecuteToolCall(ctx, ToolCall{ID: "call-1", Name: "hello", Arguments: json.RawMessage(`{}`)}, registry)
	if err != nil {
		t.Fatalf("ExecuteToolCall: %v", err)
	}
	if result == nil || result.Error != "" {
		t.Fatalf("unexpected result: %#v", result)
	}

	if len(sink.events) != 3 {
		t.Fatalf("expected 3 lifecycle events, got %d: %#v", len(sink.events), sink.events)
	}
	if _, ok := sink.events[0].(*events.EventToolExecutionStarted); !ok {
		t.Fatalf("event 0 type = %T, want *EventToolExecutionStarted", sink.events[0])
	}
	ready, ok := sink.events[1].(*events.EventToolResultReady)
	if !ok {
		t.Fatalf("event 1 type = %T, want *EventToolResultReady", sink.events[1])
	}
	if ready.Status != "success" || ready.ToolCallID != "call-1" || ready.ToolName != "hello" {
		t.Fatalf("unexpected result-ready payload: %#v", ready)
	}
	if _, ok := sink.events[2].(*events.EventToolCallFinished); !ok {
		t.Fatalf("event 2 type = %T, want *EventToolCallFinished", sink.events[2])
	}
	for i, event := range sink.events {
		if err := events.ValidateCanonicalEvent(event); err != nil {
			t.Fatalf("event %d should validate as canonical: %v", i, err)
		}
	}
}

func TestBaseToolExecutorPublishesCanonicalEventsFromToolCallCorrelation(t *testing.T) {
	t.Parallel()

	registry := NewInMemoryToolRegistry()
	def, err := NewToolFromFunc("hello", "returns hello", func() (map[string]string, error) {
		return map[string]string{"message": "hello"}, nil
	})
	if err != nil {
		t.Fatalf("NewToolFromFunc: %v", err)
	}
	if err := registry.RegisterTool("hello", *def); err != nil {
		t.Fatalf("RegisterTool: %v", err)
	}

	sink := &captureEventSink{}
	ctx := events.WithEventSinks(context.Background(), sink)
	executor := NewBaseToolExecutor(DefaultToolConfig())
	corr := events.Correlation{
		SessionID:      "session-1",
		RunID:          "run-1",
		TurnID:         "turn-1",
		ProviderCallID: "provider-call-1",
		SegmentID:      "segment-1",
		ToolCallID:     "call-1",
	}

	result, err := executor.ExecuteToolCall(ctx, ToolCall{ID: "call-1", Name: "hello", Arguments: json.RawMessage(`{}`), Correlation: corr}, registry)
	if err != nil {
		t.Fatalf("ExecuteToolCall: %v", err)
	}
	if result == nil || result.Error != "" {
		t.Fatalf("unexpected result: %#v", result)
	}
	if len(sink.events) != 3 {
		t.Fatalf("expected 3 lifecycle events, got %d: %#v", len(sink.events), sink.events)
	}
	for i, event := range sink.events {
		correlated, ok := event.(events.CorrelatedEvent)
		if !ok {
			t.Fatalf("event %d is not correlated: %T", i, event)
		}
		if got := correlated.Correlation(); got != corr {
			t.Fatalf("event %d correlation = %#v, want %#v", i, got, corr)
		}
		if err := events.ValidateCanonicalEvent(event); err != nil {
			t.Fatalf("event %d should validate as canonical: %v", i, err)
		}
	}
}

func TestToolExecutionCorrelationSources(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		ctx  context.Context
		call ToolCall
		want events.Correlation
	}{
		{
			name: "context correlation",
			ctx: WithCurrentToolCorrelation(context.Background(), events.Correlation{
				RunID:          "run-1",
				ProviderCallID: "provider-call-1",
				ToolCallID:     "call-1",
			}),
			call: ToolCall{ID: "call-1", Name: "hello"},
			want: events.Correlation{RunID: "run-1", ProviderCallID: "provider-call-1", ToolCallID: "call-1"},
		},
		{
			name: "tool call correlation",
			ctx:  context.Background(),
			call: ToolCall{ID: "call-1", Name: "hello", Correlation: events.Correlation{
				SessionID:      "session-1",
				RunID:          "run-1",
				TurnID:         "turn-1",
				ProviderCallID: "provider-call-1",
				SegmentID:      "segment-1",
				ToolCallID:     "call-1",
			}},
			want: events.Correlation{SessionID: "session-1", RunID: "run-1", TurnID: "turn-1", ProviderCallID: "provider-call-1", SegmentID: "segment-1", ToolCallID: "call-1"},
		},
		{
			name: "tool call overrides batch context",
			ctx: WithCurrentToolCorrelation(context.Background(), events.Correlation{
				RunID:          "run-1",
				ProviderCallID: "provider-call-1",
				ToolCallID:     "wrong-call",
			}),
			call: ToolCall{ID: "call-2", Name: "hello", Correlation: events.Correlation{
				RunID:          "run-1",
				ProviderCallID: "provider-call-2",
				ToolCallID:     "call-2",
			}},
			want: events.Correlation{RunID: "run-1", ProviderCallID: "provider-call-2", ToolCallID: "call-2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			corr := toolExecutionCorrelation(tt.ctx, tt.call)
			if corr != tt.want {
				t.Fatalf("correlation = %#v, want %#v", corr, tt.want)
			}
		})
	}
}

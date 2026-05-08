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
		switch event.(type) {
		case *events.EventToolCallExecute, *events.EventToolCallExecutionResult:
			t.Fatalf("event %d used legacy tool execution type %T", i, event)
		}
	}
}

func TestToolExecutionCorrelationCanComeFromContext(t *testing.T) {
	base := events.Correlation{
		Provider:       "gemini",
		ProviderCallID: "provider-call-1",
		ToolCallID:     "call-1",
		CorrelationKey: "provider:tool:call-1",
	}
	ctx := WithCurrentToolCorrelation(context.Background(), base)

	corr := toolExecutionCorrelation(ctx, ToolCall{ID: "call-1", Name: "hello"})
	if corr.Provider != "gemini" || corr.ProviderCallID != "provider-call-1" {
		t.Fatalf("provider correlation was not preserved: %#v", corr)
	}
	if corr.ToolCallID != "call-1" || corr.CorrelationKey != "provider:tool:call-1" {
		t.Fatalf("tool correlation was not preserved: %#v", corr)
	}
	if corr.SegmentType != events.SegmentTypeTool || corr.StreamKind != events.StreamKindToolCall {
		t.Fatalf("tool execution correlation was not normalized: %#v", corr)
	}
}

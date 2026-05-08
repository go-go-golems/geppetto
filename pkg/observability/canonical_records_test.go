package observability

import (
	"testing"

	"github.com/go-go-golems/geppetto/pkg/events"
)

func TestDerivedRecordsFromProviderCallFinished(t *testing.T) {
	duration := int64(123)
	usage := &events.Usage{InputTokens: 10, OutputTokens: 5, CachedTokens: 2}
	corr := events.BuildProviderCallCorrelation("openai", "inf-1", "", 2, "resp-1")
	event := events.NewProviderCallFinishedEvent(events.EventMetadata{InferenceID: "inf-1", TurnID: "turn-1"}, corr, "tool_calls", "tool_calls_pending", usage, &duration, true)

	records := DerivedRecordsFromEvent(Record{Provider: "openai", Model: "gpt-test"}, event)
	if len(records) != 1 {
		t.Fatalf("expected one provider-call result record, got %d", len(records))
	}
	rec := records[0]
	if rec.Stage != StageProviderCallResultFinalized || rec.Kind != RecordKindProviderCallResult {
		t.Fatalf("unexpected provider-call result stage/kind: %#v", rec)
	}
	if rec.ProviderCallID != corr.ProviderCallID || rec.CorrelationKey != corr.CorrelationKey {
		t.Fatalf("correlation not copied: %#v", rec)
	}
	if rec.StopReason != "tool_calls" || rec.FinishClass != "tool_calls_pending" || !rec.HasToolCalls {
		t.Fatalf("provider-call result fields not copied: %#v", rec)
	}
	if rec.Usage == nil || rec.Usage.InputTokens != 10 || rec.DurationMs == nil || *rec.DurationMs != duration {
		t.Fatalf("usage/duration not copied: %#v", rec)
	}
}

func TestDerivedRecordsFromSegmentEvents(t *testing.T) {
	corr := events.Correlation{
		Provider:             "openai_responses",
		ProviderCallID:       "provider-call-1",
		ProviderCallIndex:    1,
		SegmentID:            "segment-1",
		SegmentIndex:         3,
		SegmentType:          events.SegmentTypeText,
		StreamKind:           events.StreamKindContent,
		CorrelationKey:       "segment-key",
		ParentCorrelationKey: "provider-key",
	}
	event := events.NewTextSegmentFinishedEvent(events.EventMetadata{TurnID: "turn-1"}, corr, "hello", "stop")

	records := DerivedRecordsFromEvent(Record{Provider: "openai_responses"}, event)
	if len(records) != 1 {
		t.Fatalf("expected one segment record, got %d", len(records))
	}
	rec := records[0]
	if rec.Stage != StageSegmentFinished || rec.Kind != RecordKindSegment {
		t.Fatalf("unexpected segment stage/kind: %#v", rec)
	}
	if rec.SegmentID != "segment-1" || rec.SegmentType != events.SegmentTypeText || rec.StreamKind != events.StreamKindContent {
		t.Fatalf("segment correlation not copied: %#v", rec)
	}
	if rec.TextLen != len("hello") || rec.SegmentStatus != "stop" {
		t.Fatalf("segment text/status not copied: %#v", rec)
	}
}

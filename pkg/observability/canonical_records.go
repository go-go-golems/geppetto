package observability

import "github.com/go-go-golems/geppetto/pkg/events"

// EnrichRecordFromEvent copies typed canonical event correlation and lifecycle
// metadata into a neutral observability Record. It intentionally does not read
// EventMetadata.Extra; Extra is debug payload only in the new vocabulary.
func EnrichRecordFromEvent(rec *Record, event events.Event) {
	if rec == nil || event == nil {
		return
	}
	if rec.Kind == "" {
		rec.Kind = RecordKindCanonicalEvent
	}
	metadata := event.Metadata()
	if rec.Model == "" {
		rec.Model = metadata.Model
	}
	if rec.SessionID == "" {
		rec.SessionID = metadata.SessionID
	}
	if rec.InferenceID == "" {
		rec.InferenceID = metadata.InferenceID
	}
	if rec.TurnID == "" {
		rec.TurnID = metadata.TurnID
	}
	if rec.MessageID == "" {
		rec.MessageID = metadata.ID.String()
	}

	correlated, ok := event.(events.CorrelatedEvent)
	if ok {
		applyCorrelation(rec, correlated.Correlation())
	}

	switch e := event.(type) {
	case *events.EventProviderCallMetadataUpdated:
		rec.StopReason = e.StopReason
		rec.Usage = e.Usage
	case *events.EventProviderCallFinished:
		rec.StopReason = e.StopReason
		rec.FinishClass = e.FinishClass
		rec.Usage = e.Usage
		rec.DurationMs = e.DurationMs
		rec.HasToolCalls = e.HasToolCalls
	case *events.EventTextSegmentStarted:
		rec.SegmentType = events.SegmentTypeText
		rec.StreamKind = events.StreamKindContent
	case *events.EventTextDelta:
		rec.TextLen = len(e.Text)
		rec.SegmentType = events.SegmentTypeText
		rec.StreamKind = events.StreamKindContent
	case *events.EventTextSegmentFinished:
		rec.TextLen = len(e.Text)
		rec.SegmentStatus = e.FinishReason
		rec.SegmentType = events.SegmentTypeText
		rec.StreamKind = events.StreamKindContent
	case *events.EventReasoningSegmentStarted:
		rec.SegmentType = events.SegmentTypeReasoning
		rec.StreamKind = events.StreamKindReasoning
	case *events.EventReasoningDelta:
		rec.TextLen = len(e.Text)
		rec.SegmentType = events.SegmentTypeReasoning
		rec.StreamKind = events.StreamKindReasoning
	case *events.EventReasoningSegmentFinished:
		rec.TextLen = len(e.Text)
		rec.SegmentStatus = e.FinishReason
		rec.SegmentType = events.SegmentTypeReasoning
		rec.StreamKind = events.StreamKindReasoning
	case *events.EventToolCallStarted:
		rec.SegmentType = events.SegmentTypeTool
		rec.StreamKind = events.StreamKindToolCall
		rec.ToolCallID = e.ToolCallID
	case *events.EventToolCallArgumentsDelta:
		rec.ToolCallID = e.ToolCallID
		rec.TextLen = len(e.Arguments)
		rec.SegmentType = events.SegmentTypeTool
		rec.StreamKind = events.StreamKindToolCall
	case *events.EventToolCallRequested:
		rec.ToolCallID = e.ToolCallID
		rec.TextLen = len(e.Input)
		rec.SegmentType = events.SegmentTypeTool
		rec.StreamKind = events.StreamKindToolCall
	case *events.EventToolExecutionStarted:
		rec.ToolCallID = e.ToolCallID
		rec.TextLen = len(e.Input)
		rec.SegmentType = events.SegmentTypeTool
		rec.StreamKind = events.StreamKindToolCall
	case *events.EventToolResultReady:
		rec.ToolCallID = e.ToolCallID
		rec.TextLen = len(e.Result)
		rec.SegmentType = events.SegmentTypeTool
		rec.StreamKind = events.StreamKindToolCall
	case *events.EventToolCallFinished:
		rec.ToolCallID = e.ToolCallID
		rec.SegmentStatus = e.Status
		rec.SegmentType = events.SegmentTypeTool
		rec.StreamKind = events.StreamKindToolCall
	}
}

// DerivedRecordsFromEvent emits provider-call result and segment lifecycle rows
// in addition to compact canonical event rows. Callers provide a base record
// with provider/model/session/message context; the derived records fill typed
// correlation and lifecycle-specific fields.
func DerivedRecordsFromEvent(base Record, event events.Event) []Record {
	if event == nil {
		return nil
	}
	//nolint:exhaustive // Only canonical provider-call result and segment events derive extra observability rows.
	switch event.Type() {
	case events.EventTypeProviderCallFinished:
		rec := base
		rec.Stage = StageProviderCallResultFinalized
		rec.Kind = RecordKindProviderCallResult
		rec.EventType = string(event.Type())
		EnrichRecordFromEvent(&rec, event)
		return []Record{rec}
	case events.EventTypeTextSegmentStarted, events.EventTypeReasoningSegmentStarted, events.EventTypeToolCallStarted:
		rec := base
		rec.Stage = StageSegmentStarted
		rec.Kind = RecordKindSegment
		rec.EventType = string(event.Type())
		EnrichRecordFromEvent(&rec, event)
		return []Record{rec}
	case events.EventTypeTextDelta, events.EventTypeReasoningDelta, events.EventTypeToolCallArgumentsDelta:
		rec := base
		rec.Stage = StageSegmentUpdated
		rec.Kind = RecordKindSegment
		rec.EventType = string(event.Type())
		EnrichRecordFromEvent(&rec, event)
		return []Record{rec}
	case events.EventTypeTextSegmentFinished, events.EventTypeReasoningSegmentFinished, events.EventTypeToolCallRequested, events.EventTypeToolCallFinished:
		rec := base
		rec.Stage = StageSegmentFinished
		rec.Kind = RecordKindSegment
		rec.EventType = string(event.Type())
		EnrichRecordFromEvent(&rec, event)
		return []Record{rec}
	default:
		return nil
	}
}

func applyCorrelation(rec *Record, corr events.Correlation) {
	if corr.SessionID != "" {
		rec.SessionID = corr.SessionID
	}
	if corr.RunID != "" {
		rec.RunID = corr.RunID
	}
	if corr.TurnID != "" {
		rec.TurnID = corr.TurnID
	}
	if corr.ProviderCallID != "" {
		rec.ProviderCallID = corr.ProviderCallID
	}
	if corr.ToolCallID != "" {
		rec.ToolCallID = corr.ToolCallID
	}
	if corr.SegmentID != "" {
		rec.SegmentID = corr.SegmentID
	}
}

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
	case *events.EventTextDelta:
		rec.TextLen = len(e.Text)
	case *events.EventTextSegmentFinished:
		rec.TextLen = len(e.Text)
		rec.SegmentStatus = e.FinishReason
	case *events.EventReasoningDelta:
		rec.TextLen = len(e.Text)
	case *events.EventReasoningSegmentFinished:
		rec.TextLen = len(e.Text)
		rec.SegmentStatus = e.FinishReason
	case *events.EventToolCallStarted:
		rec.ToolCallID = e.ToolCallID
	case *events.EventToolCallArgumentsDelta:
		rec.ToolCallID = e.ToolCallID
		rec.TextLen = len(e.Arguments)
	case *events.EventToolCallRequested:
		rec.ToolCallID = e.ToolCallID
		rec.TextLen = len(e.Input)
	case *events.EventToolExecutionStarted:
		rec.ToolCallID = e.ToolCallID
		rec.TextLen = len(e.Input)
	case *events.EventToolResultReady:
		rec.ToolCallID = e.ToolCallID
		rec.TextLen = len(e.Result)
	case *events.EventToolCallFinished:
		rec.ToolCallID = e.ToolCallID
		rec.SegmentStatus = e.Status
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
	if corr.Provider != "" {
		rec.Provider = corr.Provider
	}
	if corr.Model != "" {
		rec.Model = corr.Model
	}
	if corr.SessionID != "" {
		rec.SessionID = corr.SessionID
	}
	if corr.RunID != "" {
		rec.RunID = corr.RunID
	}
	if corr.InferenceID != "" {
		rec.InferenceID = corr.InferenceID
	}
	if corr.TurnID != "" {
		rec.TurnID = corr.TurnID
	}
	if corr.ResponseID != "" {
		rec.ResponseID = corr.ResponseID
	}
	if corr.ItemID != "" {
		rec.ItemID = corr.ItemID
	}
	if corr.ProviderCallID != "" {
		rec.ProviderCallID = corr.ProviderCallID
	}
	if corr.ProviderCallIndex != 0 {
		v := int(corr.ProviderCallIndex)
		rec.ProviderCallIndex = &v
	}
	if corr.OutputIndex != nil {
		v := int(*corr.OutputIndex)
		rec.OutputIndex = &v
	}
	if corr.SummaryIndex != nil {
		v := int(*corr.SummaryIndex)
		rec.SummaryIndex = &v
	}
	if corr.ChoiceIndex != nil {
		v := int(*corr.ChoiceIndex)
		rec.ChoiceIndex = &v
	}
	if corr.ContentBlockIndex != nil {
		v := int(*corr.ContentBlockIndex)
		rec.ContentBlockIndex = &v
	}
	if corr.StreamKind != "" {
		rec.StreamKind = corr.StreamKind
	}
	if corr.CorrelationKey != "" {
		rec.CorrelationKey = corr.CorrelationKey
	}
	if corr.ParentCorrelationKey != "" {
		rec.ParentCorrelationKey = corr.ParentCorrelationKey
	}
	if corr.ToolCallID != "" {
		rec.ToolCallID = corr.ToolCallID
	}
	if corr.ToolCallIndex != nil {
		v := int(*corr.ToolCallIndex)
		rec.ToolCallIndex = &v
	}
	if corr.SegmentID != "" {
		rec.SegmentID = corr.SegmentID
	}
	if corr.SegmentIndex != 0 {
		v := int(corr.SegmentIndex)
		rec.SegmentIndex = &v
	}
	if corr.SegmentType != "" {
		rec.SegmentType = corr.SegmentType
	}
}

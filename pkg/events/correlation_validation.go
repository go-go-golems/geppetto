package events

import "fmt"

// ValidateCanonicalEvent verifies the non-negotiable identity fields for the
// hard-cutover event vocabulary. Correlation is intentionally small: run,
// provider-call, segment, and tool identities are explicit IDs. There are no
// fallback join keys.
func ValidateCanonicalEvent(event Event) error {
	correlated, ok := event.(CorrelatedEvent)
	if !ok {
		return fmt.Errorf("event %s does not expose typed correlation", event.Type())
	}

	corr := correlated.Correlation()

	//nolint:exhaustive // Validation groups canonical scopes and future events must be added deliberately.
	switch event.Type() {
	case EventTypeRunStarted, EventTypeRunFinished, EventTypeRunStopped, EventTypeRunFailed:
		if corr.RunID == "" {
			return fmt.Errorf("event %s missing run_id", event.Type())
		}
	case EventTypeProviderCallStarted, EventTypeProviderCallMetadataUpdated, EventTypeProviderCallFinished:
		if corr.RunID == "" {
			return fmt.Errorf("event %s missing run_id", event.Type())
		}
		if corr.ProviderCallID == "" {
			return fmt.Errorf("event %s missing provider_call_id", event.Type())
		}
	case EventTypeTextSegmentStarted, EventTypeTextDelta, EventTypeTextSegmentFinished,
		EventTypeReasoningSegmentStarted, EventTypeReasoningDelta, EventTypeReasoningSegmentFinished:
		if corr.RunID == "" {
			return fmt.Errorf("event %s missing run_id", event.Type())
		}
		if corr.ProviderCallID == "" {
			return fmt.Errorf("event %s missing provider_call_id", event.Type())
		}
		if corr.SegmentID == "" {
			return fmt.Errorf("event %s missing segment_id", event.Type())
		}
	case EventTypeToolCallStarted, EventTypeToolCallArgumentsDelta, EventTypeToolCallRequested,
		EventTypeToolExecutionStarted, EventTypeToolResultReady, EventTypeToolCallFinished:
		if corr.RunID == "" {
			return fmt.Errorf("event %s missing run_id", event.Type())
		}
		if corr.ToolCallID == "" {
			return fmt.Errorf("event %s missing tool_call_id", event.Type())
		}
	}

	return nil
}

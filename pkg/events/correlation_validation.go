package events

import "fmt"

// ValidateCanonicalEvent verifies the non-negotiable identity fields for the new
// hard-cutover event vocabulary. Provider adapters and downstream bridges can use
// this as a cheap guard before publishing/transporting canonical events.
func ValidateCanonicalEvent(event Event) error {
	correlated, ok := event.(CorrelatedEvent)
	if !ok {
		return fmt.Errorf("event %s does not expose typed correlation", event.Type())
	}

	corr := correlated.Correlation()
	if corr.CorrelationKey == "" {
		return fmt.Errorf("event %s missing correlation_key", event.Type())
	}

	//nolint:exhaustive // Validation groups canonical scopes and lets run/future events use the common correlation_key guard.
	switch event.Type() {
	case EventTypeProviderCallStarted, EventTypeProviderCallMetadataUpdated, EventTypeProviderCallFinished:
		if corr.ProviderCallID == "" {
			return fmt.Errorf("event %s missing provider_call_id", event.Type())
		}
	case EventTypeTextSegmentStarted, EventTypeTextDelta, EventTypeTextSegmentFinished,
		EventTypeReasoningSegmentStarted, EventTypeReasoningDelta, EventTypeReasoningSegmentFinished:
		if corr.SegmentID == "" {
			return fmt.Errorf("event %s missing segment_id", event.Type())
		}
	case EventTypeToolCallStarted, EventTypeToolCallArgumentsDelta, EventTypeToolCallRequested,
		EventTypeToolExecutionStarted, EventTypeToolResultReady, EventTypeToolCallFinished:
		if corr.ToolCallID == "" {
			return fmt.Errorf("event %s missing tool_call_id", event.Type())
		}
	default:
		// Run lifecycle and future canonical event types only require the common
		// typed correlation_key unless they are added to one of the stricter
		// scopes above.
	}

	return nil
}

package events

// EventSink represents a destination for inference events emitted while an inference
// is running (including streaming deltas and the final completion).
//
// Intended use:
// - UX/telemetry: live timelines, WebSocket broadcasts, logging, tracing, metrics
// - Debugging: capture raw payloads and intermediate diagnostics
//
// Non-goal / warning:
//   - Avoid committing durable application state based on partial/streaming events.
//     Streaming output can be malformed, incomplete, or later superseded.
//     Prefer doing validation + persistence at a clear RunInference boundary
//     (e.g. when the final completion is known, or after RunInference returns).
type EventSink interface {
	// PublishEvent publishes an event to the sink.
	// Returns an error if the event could not be published.
	PublishEvent(event Event) error
}

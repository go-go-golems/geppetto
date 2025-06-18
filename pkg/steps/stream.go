package steps

import (
	"context"
)

// Event is a minimal interface to avoid circular imports with events package
type Event interface {
	Type() string
	Payload() []byte
}

// StreamableStep extends the Step interface to support streaming events.
// This allows steps to emit partial results, tool calls, and other events
// during execution rather than just returning a final result.
type StreamableStep[I, O any] interface {
	Step[I, O]
	
	// Stream starts the step and returns a channel of events.
	// The channel will be closed when the step completes or is cancelled.
	// The cancel function can be called to stop the step execution.
	Stream(ctx context.Context, input I) (<-chan Event, func(), error)
}



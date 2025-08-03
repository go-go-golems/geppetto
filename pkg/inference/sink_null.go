package inference

import "github.com/go-go-golems/geppetto/pkg/events"

// NullSink is a no-op EventSink implementation that discards all events.
// Useful for testing or when event publishing is not desired.
type NullSink struct{}

// NewNullSink creates a new NullSink instance.
func NewNullSink() *NullSink {
	return &NullSink{}
}

// PublishEvent discards the event and always returns nil.
func (n *NullSink) PublishEvent(event events.Event) error {
	// No-op: discard the event
	return nil
}

var _ EventSink = (*NullSink)(nil)

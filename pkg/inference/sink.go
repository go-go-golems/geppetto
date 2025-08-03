package inference

import "github.com/go-go-golems/geppetto/pkg/events"

// EventSink represents a destination for inference events.
// Implementations can publish events to different backends like watermill,
// logging systems, or other event processing systems.
type EventSink interface {
	// PublishEvent publishes an event to the sink.
	// Returns an error if the event could not be published.
	PublishEvent(event events.Event) error
}

package inference

import (
	"encoding/json"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/go-go-golems/geppetto/pkg/events"
	"github.com/rs/zerolog/log"
)

// WatermillSink publishes events to a watermill Publisher.
// This allows events to be distributed through the watermill message bus
// to multiple subscribers.
type WatermillSink struct {
	publisher message.Publisher
	topic     string
}

// NewWatermillSink creates a new WatermillSink that publishes to the given
// publisher and topic.
func NewWatermillSink(publisher message.Publisher, topic string) *WatermillSink {
	return &WatermillSink{
		publisher: publisher,
		topic:     topic,
	}
}

// PublishEvent publishes the event to the watermill publisher.
// The event is serialized to JSON and sent as a watermill message.
func (w *WatermillSink) PublishEvent(event events.Event) error {
	// Serialize the event to JSON
	payload, err := json.Marshal(event)
	if err != nil {
		log.Error().Err(err).Msg("Failed to marshal event to JSON")
		return err
	}

	// Create watermill message
	msg := message.NewMessage(watermill.NewUUID(), payload)

	// Publish to the topic
	err = w.publisher.Publish(w.topic, msg)
	if err != nil {
		log.Error().Err(err).Str("topic", w.topic).Msg("Failed to publish event to watermill")
		return err
	}

	log.Trace().Str("topic", w.topic).Str("event_type", string(event.Type())).Msg("Published event to watermill")
	return nil
}

var _ EventSink = (*WatermillSink)(nil)

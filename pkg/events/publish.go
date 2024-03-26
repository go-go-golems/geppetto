package events

import (
	"encoding/json"
	"fmt"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/rs/zerolog/log"
	"sync"
)

// NOTE(manuel, 2024-03-24) This might be worth moving / integrating into the event router
// It sounds also logical that this is the thing that would add sequence numbers to events?

// PublisherManager is used to distribute messages to a set of Publishers.
// As such, you "subscribe" a publisher to the given topic.
// When you Publish a message, it will get distributed to all publishers
// on the channel they were subscribed with.
//
// The Manager also keeps a sequence number for each outgoing message,
// in the order they are handled by Publish.
type PublisherManager struct {
	Publishers     map[string][]message.Publisher
	sequenceNumber uint64
	mutex          sync.Mutex
}

func NewPublisherManager() *PublisherManager {
	return &PublisherManager{
		Publishers: make(map[string][]message.Publisher),
	}
}

func (s *PublisherManager) SubscribePublisher(topic string, sub message.Publisher) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.Publishers[topic] = append(s.Publishers[topic], sub)
}

// Publish distributes a message to all Publishers across all topics.
// Serializing the payload to JSON is done by Publish itself.
//
// Returns an error for any processing or distribution issues.
func (s *PublisherManager) Publish(payload interface{}) error {
	// lock for the sequence number hash
	s.mutex.Lock()
	defer s.mutex.Unlock()

	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	msg := message.NewMessage(watermill.NewUUID(), b)
	msg.Metadata.Set("sequence_number", fmt.Sprintf("%d", s.sequenceNumber))
	s.sequenceNumber++

	for topic, subs := range s.Publishers {
		for _, sub := range subs {
			err = sub.Publish(topic, msg)
			if err != nil {
				log.Warn().Err(err).Msg("failed to publish")
			}
		}
	}

	return nil
}

func (s *PublisherManager) PublishBlind(payload interface{}) {
	err := s.Publish(payload)
	if err != nil {
		log.Warn().Err(err).Msg("failed to publish")
	}
}

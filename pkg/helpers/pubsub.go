package helpers

import (
	"encoding/json"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/rs/zerolog/log"
)

type PublisherManager struct {
	Publishers map[string][]message.Publisher
}

func NewPublisherManager() *PublisherManager {
	return &PublisherManager{
		Publishers: make(map[string][]message.Publisher),
	}
}

func (s *PublisherManager) AddPublishedTopic(topic string, sub message.Publisher) {
	s.Publishers[topic] = append(s.Publishers[topic], sub)
}

func (s *PublisherManager) Publish(payload interface{}) error {
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	msg := message.NewMessage(watermill.NewUUID(), b)

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

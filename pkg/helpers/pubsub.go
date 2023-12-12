package helpers

import (
	"encoding/json"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/rs/zerolog/log"
)

type SubscriptionManager struct {
	subscriptions map[string][]message.Publisher
}

func NewSubscriptionManager() *SubscriptionManager {
	return &SubscriptionManager{
		subscriptions: make(map[string][]message.Publisher),
	}
}

func (s *SubscriptionManager) AddSubscription(topic string, sub message.Publisher) {
	s.subscriptions[topic] = append(s.subscriptions[topic], sub)
}

func (s *SubscriptionManager) Publish(payload interface{}) error {
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	msg := message.NewMessage(watermill.NewUUID(), b)

	for topic, subs := range s.subscriptions {
		for _, sub := range subs {
			err = sub.Publish(topic, msg)
			if err != nil {
				log.Warn().Err(err).Msg("failed to publish")
			}
		}
	}

	return nil
}

func (s *SubscriptionManager) PublishBlind(payload interface{}) {
	err := s.Publish(payload)
	if err != nil {
		log.Warn().Err(err).Msg("failed to publish")
	}
}

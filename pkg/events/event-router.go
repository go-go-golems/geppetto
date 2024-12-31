package events

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	"github.com/rs/zerolog/log"
)

type EventRouter struct {
	logger     watermill.LoggerAdapter
	Publisher  message.Publisher
	Subscriber message.Subscriber
	router     *message.Router
	verbose    bool
}

type EventRouterOption func(*EventRouter)

func WithLogger(logger watermill.LoggerAdapter) EventRouterOption {
	return func(r *EventRouter) {
		r.logger = logger
	}
}

func WithPublisher(publisher message.Publisher) EventRouterOption {
	return func(r *EventRouter) {
		r.Publisher = publisher
	}
}

func WithSubscriber(subscriber message.Subscriber) EventRouterOption {
	return func(r *EventRouter) {
		r.Subscriber = subscriber
	}
}

func WithVerbose(verbose bool) EventRouterOption {
	return func(r *EventRouter) {
		r.verbose = verbose
	}
}

func NewEventRouter(options ...EventRouterOption) (*EventRouter, error) {
	ret := &EventRouter{
		logger: watermill.NopLogger{},
	}

	for _, o := range options {
		o(ret)
	}

	goPubSub := gochannel.NewGoChannel(gochannel.Config{
		BlockPublishUntilSubscriberAck: true,
	}, ret.logger)
	ret.Publisher = goPubSub
	ret.Subscriber = goPubSub

	router, err := message.NewRouter(message.RouterConfig{}, ret.logger)
	if err != nil {
		return nil, err
	}

	ret.router = router

	return ret, nil
}

func (e *EventRouter) Close() error {
	err := e.Publisher.Close()
	if err != nil {
		log.Error().Err(err).Msg("Failed to close pubsub")
		// not returning just yet
	}

	return nil
}

func (e *EventRouter) AddHandler(name string, topic string, f func(msg *message.Message) error) {
	e.router.AddNoPublisherHandler(name, topic, e.Subscriber, f)
}

func (e *EventRouter) DumpRawEvents(msg *message.Message) error {
	defer msg.Ack()

	var s map[string]interface{}
	err := json.Unmarshal(msg.Payload, &s)
	if err != nil {
		return err
	}
	if !e.verbose {
		s["id"] = s["meta"].(map[string]interface{})["message_id"]
		s["step_type"] = s["step"].(map[string]interface{})["type"]
		delete(s, "meta")
		delete(s, "step")
	}
	s_, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(s_))
	return nil
}

func (e *EventRouter) Running() chan struct{} {
	return e.router.Running()
}

func (e *EventRouter) Run(ctx context.Context) error {
	return e.router.Run(ctx)
}

func (e *EventRouter) RunHandlers(ctx context.Context) error {
	return e.router.RunHandlers(ctx)
}

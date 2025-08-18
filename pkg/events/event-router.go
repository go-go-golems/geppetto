package events

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	"github.com/rs/zerolog/log"

	"github.com/go-go-golems/geppetto/pkg/helpers"
)

// ChatEventHandler defines an interface for handling different chat events.
// Moved from pinocchio/cmd/experiments/web-ui/client/client.go
type ChatEventHandler interface {
	HandlePartialCompletion(ctx context.Context, e *EventPartialCompletion) error
	HandleText(ctx context.Context, e *EventText) error
	HandleFinal(ctx context.Context, e *EventFinal) error
	HandleError(ctx context.Context, e *EventError) error
	HandleInterrupt(ctx context.Context, e *EventInterrupt) error
	// Add other event types as needed
}

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
		r.logger = helpers.NewWatermill(log.Logger)
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
	log.Debug().Msg("Closing publisher")
	err := e.Publisher.Close()
	if err != nil {
		log.Error().Err(err).Msg("Failed to close pubsub")
		// not returning just yet
	}
	log.Debug().Msg("Publisher closed")

	// XXX(2025-03-30, manuel): I am not sure if this is fully correct, but 09-sonnet-event-router-investigation.md surfaced the
	// fact that the watermill router is not closed by EventRouter.Close().
	log.Debug().Msg("Closing router")
	err = e.router.Close()
	if err != nil {
		log.Error().Err(err).Msg("Failed to close router")
		// not returning just yet
	}
	log.Debug().Msg("Router closed")

	return nil
}

// RegisterChatEventHandler sets up event publishing for a chat step and registers a handler
// function that dispatches events to the provided ChatEventHandler.
// This introduces a dependency from the events package to the chat package.
// RegisterChatEventHandler removed: use sinks with engines and manual AddHandler

// createChatDispatchHandler creates a Watermill handler function that parses chat events
// and dispatches them to the appropriate method of the provided ChatEventHandler.
// Moved from pinocchio/cmd/experiments/web-ui/client/client.go
// nolint:unused // kept for legacy compatibility
func createChatDispatchHandler(handler ChatEventHandler) message.NoPublishHandlerFunc {
	return func(msg *message.Message) error {
		logFields := watermill.LogFields{"message_id": msg.UUID}
		log.Debug().Interface("logFields", logFields).Msg("Received message for chat handler")

		// Parse the generic chat event
		e, err := NewEventFromJson(msg.Payload)
		if err != nil {
			logFields["payload"] = string(msg.Payload) // Add payload for context
			log.Error().Interface("logFields", logFields).Err(err).Msg("Failed to parse chat event from message payload")
			// Don't kill the handler for one bad message, just log and continue
			return nil // Or return err depending on desired resilience
		}

		logFields["event_type"] = string(e.Type())
		log.Debug().Interface("logFields", logFields).Msg("Parsed chat event")

		// Dispatch to the specific handler method based on event type
		// Pass the message context down to the handler
		msgCtx := msg.Context()
		var handlerErr error
		switch ev := e.(type) {
		case *EventPartialCompletion:
			handlerErr = handler.HandlePartialCompletion(msgCtx, ev)
		case *EventText:
			handlerErr = handler.HandleText(msgCtx, ev)
		case *EventFinal:
			handlerErr = handler.HandleFinal(msgCtx, ev)
		case *EventError:
			handlerErr = handler.HandleError(msgCtx, ev)
		case *EventInterrupt:
			handlerErr = handler.HandleInterrupt(msgCtx, ev)
		default:
			log.Warn().Interface("logFields", logFields).Msg("Unhandled chat event type")
			// Decide if unknown types should be an error or ignored
		}

		if handlerErr != nil {
			log.Error().Interface("logFields", logFields).Err(handlerErr).Msg("Error processing chat event")
			// Return the error to potentially signal Watermill handler failure
			return handlerErr
		}

		return nil
	}
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
		if step, ok := s["step"].(map[string]interface{}); ok {
			s["step_type"] = step["type"]
		}
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

func (e *EventRouter) IsRunning() bool {
	return e.router.IsRunning()
}

func (e *EventRouter) Run(ctx context.Context) error {
	return e.router.Run(ctx)
}

func (e *EventRouter) RunHandlers(ctx context.Context) error {
	return e.router.RunHandlers(ctx)
}

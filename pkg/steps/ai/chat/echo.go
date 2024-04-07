package chat

import (
	"context"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/go-go-golems/bobatea/pkg/conversation"
	"github.com/go-go-golems/geppetto/pkg/events"
	"github.com/go-go-golems/geppetto/pkg/helpers"
	"github.com/go-go-golems/geppetto/pkg/steps"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
	"time"
)

type EchoStep struct {
	TimePerCharacter    time.Duration
	cancel              context.CancelFunc
	eg                  *errgroup.Group
	subscriptionManager *events.PublisherManager
}

func NewEchoStep() *EchoStep {
	return &EchoStep{
		TimePerCharacter:    100 * time.Millisecond,
		subscriptionManager: events.NewPublisherManager(),
	}
}

func (e *EchoStep) Interrupt() {
	if e.cancel != nil {
		e.cancel()
	}
}

func (e *EchoStep) AddPublishedTopic(publisher message.Publisher, topic string) error {
	e.subscriptionManager.SubscribePublisher(topic, publisher)
	return nil
}

func (e *EchoStep) Start(ctx context.Context, input conversation.Conversation) (steps.StepResult[string], error) {
	if len(input) == 0 {
		return nil, errors.New("no input")
	}

	eg, ctx := errgroup.WithContext(ctx)
	ctx, cancel := context.WithCancel(ctx)
	e.cancel = cancel

	c := make(chan helpers.Result[string], 1)
	res := steps.NewStepResult(c)

	// TODO(manuel, 2024-01-12) We need to add conversational metadata here
	eg.Go(func() error {
		defer close(c)
		msg, ok := input[len(input)-1].Content.(*conversation.ChatMessageContent)
		if !ok {
			c <- helpers.NewErrorResult[string](errors.New("invalid input"))
			return errors.New("invalid input")
		}

		for idx, c_ := range msg.Text {
			select {
			case <-ctx.Done():
				e.subscriptionManager.PublishBlind(&EventText{
					Event: Event{
						Type: EventTypeInterrupt,
					},
					Text: msg.Text,
				})
				c <- helpers.NewErrorResult[string](ctx.Err())
				return ctx.Err()
			case <-time.After(e.TimePerCharacter):
				e.subscriptionManager.PublishBlind(&EventPartialCompletion{
					Event: Event{
						Type: EventTypePartial,
					},
					Delta:      string(c_),
					Completion: msg.Text[:idx+1],
				})
			}
		}
		e.subscriptionManager.PublishBlind(&EventText{
			Event: Event{
				Type: EventTypeFinal,
			},
			Text: msg.Text,
		})
		c <- helpers.NewValueResult[string](msg.Text)
		return nil
	})
	e.eg = eg

	return res, nil
}

var _ Step = (*EchoStep)(nil)

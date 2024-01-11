package chat

import (
	"context"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/go-go-golems/bobatea/pkg/chat/conversation"
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
	subscriptionManager *helpers.SubscriptionManager
}

func NewEchoStep() *EchoStep {
	return &EchoStep{
		TimePerCharacter:    100 * time.Millisecond,
		subscriptionManager: helpers.NewSubscriptionManager(),
	}
}

func (e *EchoStep) Interrupt() {
	if e.cancel != nil {
		e.cancel()
	}
}

func (e *EchoStep) Publish(publisher message.Publisher, topic string) error {
	e.subscriptionManager.AddSubscription(topic, publisher)
	return nil
}

func (e *EchoStep) Start(ctx context.Context, input []*conversation.Message) (steps.StepResult[string], error) {
	if len(input) == 0 {
		return nil, errors.New("no input")
	}

	eg, ctx := errgroup.WithContext(ctx)
	ctx, cancel := context.WithCancel(ctx)
	e.cancel = cancel

	c := make(chan helpers.Result[string], 1)
	res := steps.NewStepResult(c)

	eg.Go(func() error {
		defer close(c)
		msg := input[len(input)-1]
		for _, c_ := range msg.Text {
			select {
			case <-ctx.Done():
				e.subscriptionManager.PublishBlind(&Event{
					Type: EventTypeInterrupt,
				})
				c <- helpers.NewErrorResult[string](ctx.Err())
				return ctx.Err()
			case <-time.After(e.TimePerCharacter):
				e.subscriptionManager.PublishBlind(&Event{
					Type: EventTypePartial,
					Text: string(c_),
				})
			}
		}
		e.subscriptionManager.PublishBlind(&Event{
			Type: EventTypeFinal,
			Text: msg.Text,
		})
		c <- helpers.NewValueResult[string](msg.Text)
		return nil
	})
	e.eg = eg

	return res, nil
}

var _ Step = (*EchoStep)(nil)

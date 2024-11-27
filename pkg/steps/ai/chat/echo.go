package chat

import (
	"context"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/go-go-golems/geppetto/pkg/conversation"
	"github.com/go-go-golems/geppetto/pkg/events"
	"github.com/go-go-golems/geppetto/pkg/helpers"
	"github.com/go-go-golems/geppetto/pkg/steps"
	"github.com/google/uuid"
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
	e.subscriptionManager.RegisterPublisher(topic, publisher)
	return nil
}

func (e *EchoStep) Start(ctx context.Context, input conversation.Conversation) (steps.StepResult[string], error) {
	if len(input) == 0 {
		return nil, errors.New("no input")
	}

	eg, ctx := errgroup.WithContext(ctx)
	ctx, cancel := context.WithCancel(ctx)
	e.cancel = cancel

	parentID := conversation.NullNode
	if len(input) > 1 {
		parentID = input[0].ID
	}

	metadata := EventMetadata{
		ID:       conversation.NewNodeID(),
		ParentID: parentID,
	}

	stepMetadata := &steps.StepMetadata{
		StepID:     uuid.New(),
		Type:       "echo-completion",
		InputType:  "conversation.Conversation",
		OutputType: "string",
		Metadata: map[string]interface{}{
			"timePerCharacter": e.TimePerCharacter,
		},
	}

	c := make(chan helpers.Result[string], 1)
	res := steps.NewStepResult(c)

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
				e.subscriptionManager.PublishBlind(NewInterruptEvent(metadata, stepMetadata, msg.Text))
				c <- helpers.NewErrorResult[string](ctx.Err())
				return ctx.Err()
			case <-time.After(e.TimePerCharacter):
				e.subscriptionManager.PublishBlind(NewPartialCompletionEvent(metadata, stepMetadata, string(c_), msg.Text[:idx+1]))
			}
		}
		e.subscriptionManager.PublishBlind(NewFinalEvent(metadata, stepMetadata, msg.Text))
		c <- helpers.NewValueResult[string](msg.Text)
		return nil
	})
	e.eg = eg

	return res, nil
}

var _ Step = (*EchoStep)(nil)

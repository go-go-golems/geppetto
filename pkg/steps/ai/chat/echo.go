package chat

import (
	"context"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/go-go-golems/geppetto/pkg/conversation"
	"github.com/go-go-golems/geppetto/pkg/events"
	"github.com/go-go-golems/geppetto/pkg/helpers"
	"github.com/go-go-golems/geppetto/pkg/steps"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"

	"github.com/rs/zerolog/log"
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

func (e *EchoStep) Start(ctx context.Context, input conversation.Conversation) (steps.StepResult[*conversation.Message], error) {
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

	c := make(chan helpers.Result[*conversation.Message], 1)

	eg.Go(func() error {
		defer close(c)
		msg := input[len(input)-1]
		msgContent, ok := msg.Content.(*conversation.ChatMessageContent)
		if !ok {
			c <- helpers.NewErrorResult[*conversation.Message](errors.New("invalid input"))
			return errors.New("invalid input")
		}

		for idx, c_ := range msgContent.Text {
			select {
			case <-ctx.Done():
				log.Debug().Msg("Interrupting step")
				e.subscriptionManager.PublishBlind(NewInterruptEvent(metadata, stepMetadata, msgContent.Text))
				c <- helpers.NewErrorResult[*conversation.Message](ctx.Err())
				return ctx.Err()
			case <-time.After(e.TimePerCharacter):
				log.Debug().Msg("Publishing partial completion event")
				e.subscriptionManager.PublishBlind(NewPartialCompletionEvent(metadata, stepMetadata, string(c_), msgContent.Text[:idx+1]))
			}
		}
		log.Debug().Msg("Publishing final event")
		e.subscriptionManager.PublishBlind(NewFinalEvent(metadata, stepMetadata, msgContent.Text))
		c <- helpers.NewValueResult[*conversation.Message](conversation.NewChatMessage(conversation.RoleAssistant, msgContent.Text))
		return nil
	})
	e.eg = eg

	ret := steps.NewStepResult[*conversation.Message](
		c,
		steps.WithMetadata[*conversation.Message](stepMetadata),
	)
	return ret, nil
}

var _ Step = (*EchoStep)(nil)

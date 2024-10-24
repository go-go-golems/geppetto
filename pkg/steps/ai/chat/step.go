package chat

import (
	"context"
	"github.com/ThreeDotsLabs/watermill/message"
	geppetto_context "github.com/go-go-golems/geppetto/pkg/context"
	"github.com/go-go-golems/geppetto/pkg/conversation"
	"github.com/go-go-golems/geppetto/pkg/steps"
)

type Step steps.Step[conversation.Conversation, string]

type StepOption func(Step) error

func WithPublishedTopic(publisher message.Publisher, topic string) StepOption {
	return func(step Step) error {
		err := step.AddPublishedTopic(publisher, topic)
		if err != nil {
			return err
		}

		return nil
	}
}

type AddToHistoryStep struct {
	manager conversation.Manager
	role    string
}

var _ steps.Step[string, string] = &AddToHistoryStep{}

func (a *AddToHistoryStep) Start(ctx context.Context, input string) (steps.StepResult[string], error) {
	a.manager.AppendMessages(conversation.NewChatMessage(conversation.Role(a.role), input))

	return steps.Resolve(input), nil
}

func (a *AddToHistoryStep) AddPublishedTopic(publisher message.Publisher, topic string) error {
	return nil
}

type RunnableStep struct {
	c       geppetto_context.GeppettoRunnable
	manager conversation.Manager
}

var _ steps.Step[interface{}, string] = &RunnableStep{}

func (r *RunnableStep) Start(ctx context.Context, input interface{}) (steps.StepResult[string], error) {
	return r.c.RunWithManager(ctx, r.manager)
}

func (r *RunnableStep) AddPublishedTopic(publisher message.Publisher, topic string) error {
	return nil
}

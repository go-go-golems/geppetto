package steps

import (
	"context"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/go-go-golems/geppetto/pkg/conversation"
	"github.com/go-go-golems/geppetto/pkg/steps"
)

type AddToHistoryStep struct {
	manager conversation.Manager
	role    string
}

var _ steps.Step[string, string] = &AddToHistoryStep{}

func (a *AddToHistoryStep) Start(ctx context.Context, input string) (steps.StepResult[string], error) {
	if err := a.manager.AppendMessages(conversation.NewChatMessage(conversation.Role(a.role), input)); err != nil {
		return steps.Reject[string](err), nil
	}

	return steps.Resolve(input), nil
}

func (a *AddToHistoryStep) AddPublishedTopic(publisher message.Publisher, topic string) error {
	return nil
}

package steps

import (
	context2 "context"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/go-go-golems/geppetto/pkg/context"
	"github.com/go-go-golems/geppetto/pkg/conversation"
	"github.com/go-go-golems/geppetto/pkg/steps"
)

type RunnableStep struct {
	c       context.GeppettoRunnable
	manager conversation.Manager
}

var _ steps.Step[interface{}, *conversation.Message] = &RunnableStep{}

func (r *RunnableStep) Start(ctx context2.Context, input interface{}) (steps.StepResult[*conversation.Message], error) {
	return r.c.RunWithManager(ctx, r.manager)
}

func (r *RunnableStep) AddPublishedTopic(publisher message.Publisher, topic string) error {
	return nil
}

package chat

import (
	"context"
	"github.com/ThreeDotsLabs/watermill/message"
	geppetto_context "github.com/go-go-golems/geppetto/pkg/context"
	"github.com/go-go-golems/geppetto/pkg/steps"
	"time"
)

type EventType string

type Step interface {
	steps.Step[[]*geppetto_context.Message, string]
	Interrupt()
	Publish(publisher message.Publisher, topic string) error
}

const (
	EventTypePartial   EventType = "partial"
	EventTypeFinal     EventType = "final"
	EventTypeError     EventType = "error"
	EventTypeInterrupt EventType = "interrupt"
)

type Event struct {
	Type  EventType `json:"type"`
	Text  string    `json:"text,omitempty"`
	Error error     `json:"error,omitempty"`
}

type StepOption func(Step) error

func WithSubscription(publisher message.Publisher, topic string) StepOption {
	return func(step Step) error {
		err := step.Publish(publisher, topic)
		if err != nil {
			return err
		}

		return nil
	}
}

type AddToHistoryStep struct {
	manager *geppetto_context.Manager
	role    string
}

var _ steps.Step[string, string] = &AddToHistoryStep{}

func (a *AddToHistoryStep) Start(ctx context.Context, input string) (steps.StepResult[string], error) {
	a.manager.AddMessages(&geppetto_context.Message{
		Text: input,
		Time: time.Time{},
		Role: a.role,
	})

	return steps.Resolve(input), nil
}

type RunnableStep struct {
	c       geppetto_context.GeppettoRunnable
	manager *geppetto_context.Manager
}

var _ steps.Step[interface{}, string] = &RunnableStep{}

func (r *RunnableStep) Start(ctx context.Context, input interface{}) (steps.StepResult[string], error) {
	return r.c.RunWithManager(ctx, r.manager)
}

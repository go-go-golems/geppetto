package chat

import (
	"context"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/go-go-golems/bobatea/pkg/chat/conversation"
	geppetto_context "github.com/go-go-golems/geppetto/pkg/context"
	"github.com/go-go-golems/geppetto/pkg/steps"
	"github.com/google/uuid"
)

type EventType string

type Step interface {
	steps.Step[[]*conversation.Message, string]
	Interrupt()
	Publish(publisher message.Publisher, topic string) error
}

const (
	EventTypeStart     EventType = "start"
	EventTypeStatus    EventType = "status"
	EventTypePartial   EventType = "partial"
	EventTypeFinal     EventType = "final"
	EventTypeError     EventType = "error"
	EventTypeInterrupt EventType = "interrupt"
)

type Event struct {
	Type     EventType     `json:"type"`
	Text     string        `json:"text,omitempty"`
	Error    error         `json:"error,omitempty"`
	Metadata EventMetadata `json:"meta,omitempty"`
}

// EventMetadata contains all the information that is passed along with watermill message,
// specific to chat steps.
type EventMetadata struct {
	ID             uuid.UUID `json:"message_id"`
	ParentID       uuid.UUID `json:"parent_id"`
	ConversationID uuid.UUID `json:"conversation_id"`
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
	a.manager.AddMessages(conversation.NewMessage(input, a.role))

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

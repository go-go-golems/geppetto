package chat

import (
	"context"
	"encoding/json"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/go-go-golems/bobatea/pkg/chat/conversation"
	geppetto_context "github.com/go-go-golems/geppetto/pkg/context"
	"github.com/go-go-golems/geppetto/pkg/steps"
	"github.com/google/uuid"
)

type EventType string

type Step steps.Step[[]*conversation.Message, string]

const (
	EventTypeStart     EventType = "start"
	EventTypeStatus    EventType = "status"
	EventTypePartial   EventType = "partial"
	EventTypeFinal     EventType = "final"
	EventTypeError     EventType = "error"
	EventTypeInterrupt EventType = "interrupt"
)

type Event struct {
	Type     EventType           `json:"type"`
	Error    error               `json:"error,omitempty"`
	Metadata EventMetadata       `json:"meta,omitempty"`
	Step     *steps.StepMetadata `json:"step,omitempty"`
	payload  []byte
}

type EventText struct {
	Event
	Text string `json:"text"`
}

type EventPartialCompletion struct {
	Event
	Delta      string `json:"delta"`
	Completion string `json:"completion"`
}

// EventMetadata contains all the information that is passed along with watermill message,
// specific to chat steps.
type EventMetadata struct {
	ID             uuid.UUID `json:"message_id"`
	ParentID       uuid.UUID `json:"parent_id"`
	ConversationID uuid.UUID `json:"conversation_id"`
}

func NewEventFromJson(b []byte) (Event, error) {
	var e Event
	err := json.Unmarshal(b, &e)
	if err != nil {
		return Event{}, err
	}

	e.payload = b

	return e, nil
}

func (e Event) ToText() (EventText, bool) {
	var ret EventText
	err := json.Unmarshal(e.payload, &ret)
	if err != nil {
		return EventText{}, false
	}

	return ret, true
}

func (e Event) ToPartialCompletion() (EventPartialCompletion, bool) {
	var ret EventPartialCompletion
	err := json.Unmarshal(e.payload, &ret)
	if err != nil {
		return EventPartialCompletion{}, false
	}

	return ret, true
}

type StepOption func(Step) error

func WithSubscription(publisher message.Publisher, topic string) StepOption {
	return func(step Step) error {
		err := step.AddPublishedTopic(publisher, topic)
		if err != nil {
			return err
		}

		return nil
	}
}

type AddToHistoryStep struct {
	manager *conversation.Manager
	role    string
}

var _ steps.Step[string, string] = &AddToHistoryStep{}

func (a *AddToHistoryStep) Start(ctx context.Context, input string) (steps.StepResult[string], error) {
	a.manager.AddMessages(conversation.NewMessage(input, a.role))

	return steps.Resolve(input), nil
}

func (a *AddToHistoryStep) AddPublishedTopic(publisher message.Publisher, topic string) error {
	return nil
}

type RunnableStep struct {
	c       geppetto_context.GeppettoRunnable
	manager *conversation.Manager
}

var _ steps.Step[interface{}, string] = &RunnableStep{}

func (r *RunnableStep) Start(ctx context.Context, input interface{}) (steps.StepResult[string], error) {
	return r.c.RunWithManager(ctx, r.manager)
}

func (r *RunnableStep) AddPublishedTopic(publisher message.Publisher, topic string) error {
	return nil
}

package chat

import (
	"context"
	"encoding/json"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/go-go-golems/bobatea/pkg/conversation"
	geppetto_context "github.com/go-go-golems/geppetto/pkg/context"
	"github.com/go-go-golems/geppetto/pkg/steps"
)

type EventType string

type Step steps.Step[conversation.Conversation, string]

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

	// store payload if the event was deserialized from JSON (see NewEventFromJson), not further used
	payload []byte
}

type EventText struct {
	Event
	Text string `json:"text"`
	// TODO(manuel, 2024-06-04) Add ToolCall information here, and potentially multiple responses (see the claude API that allows multiple content blocks)
	// This is currently stored in the metadata uder the MetadataToolCallsSlug (see chat-with-tools-step.go in openai)
}

type EventPartialCompletion struct {
	Event
	Delta string `json:"delta"`
	// This is the complete completion string so far (when using openai, this is currently also the toolcall json)
	Completion string `json:"completion"`

	// TODO(manuel, 2024-06-04) This might need partial tool completion if it is of interest, this is less important than adding tool call information to the result above
}

type ToolCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// MetadataToolCallsSlug is the slug used to store ToolCall metadata as returned by the openai API
const MetadataToolCallsSlug = "tool-calls"

// EventMetadata contains all the information that is passed along with watermill message,
// specific to chat steps.
type EventMetadata struct {
	ID       conversation.NodeID `json:"message_id"`
	ParentID conversation.NodeID `json:"parent_id"`
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

func ToTypedEvent[T any](e Event) (*T, bool) {
	var ret *T
	err := json.Unmarshal(e.payload, &ret)
	if err != nil {
		return nil, false
	}

	return ret, true
}

func (e Event) ToText() (EventText, bool) {
	ret, ok := ToTypedEvent[EventText](e)
	if !ok || ret == nil {
		return EventText{}, false
	}
	return *ret, true
}

func (e Event) ToPartialCompletion() (EventPartialCompletion, bool) {
	ret, ok := ToTypedEvent[EventPartialCompletion](e)
	if !ok || ret == nil {
		return EventPartialCompletion{}, false
	}
	return *ret, true
}

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

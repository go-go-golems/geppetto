At the core:

geppetto uses "steps" internally to chain async operations that transform T into []U.
Besides streaming these results (using the StepResult monad), steps can also publish information to a watermill pubsub topic.

These watermill events can be handled to for example update the UI (in pinocchio), or output events to stdout (in GeppettoCommand's RunIntoWriter).

## Watermill concepts

- message.Publisher
- message.Message

## Usage in geppetto / pinocchio

- send information about streaming completion

- used by @pinocchio/pkg/ui/backend.go to convert into bubbletea messages
    - the bobachat backend returns a func that takes a list of message.Message (watermill) and injects them into the bubbletea program
    - the func is registered to the event router

## PublisherManager

@geppetto/pkg/events/publish.go

- what is a publisher manager
- what publishers get registered?

- who publishes to it?

## Router

@geppetto/pkg/events/event-router.go

- who creates it
- uses go channel as a pubsub
- creates a watermill router
- adding a handler just registers a NoPublisherHandler (watermill)

## Messages

- metadata: sequence_number
- payloads: 
  - who creates them?
    - used for example in RunIntoWriter for geppetto commands (see @pinocchio/pkgs/cmds/cmd.go)
  - who decodes them?
    - NewEventFromJson in @geppetto/pkg/steps/ai/chat/events.go
    - StepPrinterFunc in @pinocchio/pkg/steps/ai/chat/conversation.go, which outputs it to stdout
    - StepForwardFunc in @pinocchio/pkg/ui/backend.go, which injects it into the bubbletea program

### Events / Message schema

See @geppetto/pkg/steps/ai/chat/events.go

Seems to have a set schema:

(from event-router.go):

```go
		s["id"] = s["meta"].(map[string]interface{})["message_id"]
		s["step_type"] = s["step"].(map[string]interface{})["type"]
```

The actual implementation is EventImpl and it encodes to JSON:

```json
{
    "type": "...",
    "meta": {
        "message_id": "...",
        "parent_id": "..."
    },
    "step": { ... },
    ...
}
```

Types of events:

- EventTypeStart
- EventTypeFinal
- EventTypePartialCompletion
- EventTypeStatus
- EventTypeToolCall
- EventTypeToolResult
- EventTypeError
- EventTypeInterrupt

```go
type EventType string

const (
	// EventTypeStart to EventTypeFinal are for text completion, actually
	EventTypeStart             EventType = "start"
	EventTypeFinal             EventType = "final"
	EventTypePartialCompletion EventType = "partial"

	// TODO(manuel, 2024-07-04) I'm not sure if this is needed
	EventTypeStatus EventType = "status"

	// TODO(manuel, 2024-07-04) Should potentially have a EventTypeText for a block stop here
	EventTypeToolCall   EventType = "tool-call"
	EventTypeToolResult EventType = "tool-result"
	EventTypeError      EventType = "error"
	EventTypeInterrupt  EventType = "interrupt"
)

type Event interface {
	Type() EventType
	Metadata() EventMetadata
	StepMetadata() *steps.StepMetadata
	Payload() []byte
}
```

#### Event metadata

From geppetto/pkg/steps/ai/chat/events.go:

```go
type EventMetadata struct {
	ID       conversation.NodeID `json:"message_id"`
	ParentID conversation.NodeID `json:"parent_id"`
}
```

Also encodes the StepMetadata (see @geppetto/pkg/steps/step.go):

```go
type StepMetadata struct {
	StepID     uuid.UUID `json:"step_id"`
	Type       string    `json:"type"`
	InputType  string    `json:"input_type"`
	OutputType string    `json:"output_type"`

	Metadata map[string]interface{} `json:"meta"`
}
```
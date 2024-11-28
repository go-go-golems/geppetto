# Steps

- What is a step? See @geppetto/pkg/steps/step.go
    - asynchronous transformation T -> []U
    - can be started, then result is handled in the StepResult monad
    - StepResult[T] - monad for results of a step
    - can publish additional events to a watermill topic
    - StepFactory is a thunk to create a new Step

A Step is a piece of a pipeline, generically typed T -> StepResult[U], that

- can be started, with an input
- can publish information to a topic using watermill message.Publisher
    - usually steps do this by using a PublisherManager (see @geppetto/pkg/events/publish.go) and registering it to the given topic (see @geppetto/pkg/steps/ai/claude/chat-step.go for an example)
    - these steps then publish events while streaming, along with outputting results to the StepResult channel

LLM chat events that get published are (see @geppetto/pkg/steps/ai/chat/events.go)

```go
type Event interface {
	Type() EventType
	Metadata() EventMetadata
	StepMetadata() *steps.StepMetadata
	Payload() []byte
}

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
```

This is the part that takes the streaming results from claude from eventCh, and forwards it to the output channel, but also
publishes chat events to the publisher manager, and thus to the published topic for the step:

```go
			case event, ok := <-eventCh:
				if !ok {
					// TODO(manuel, 2024-07-04) Probably not necessary, the completionMerger probably took care of it
					response := completionMerger.Response()
					if response == nil {
						csf.subscriptionManager.PublishBlind(chat.NewErrorEvent(metadata, stepMetadata, "no response"))
						c <- helpers2.NewErrorResult[string](errors.New("no response"))
						return
					}
					c <- helpers2.NewValueResult[string](response.FullText())
					return
				}
```

- What is StepResult?
    - a monad combining:
      - errors (through helpers.Result)
      - streaming (through a channel)
      - list (either through streaming or returning multiple values)
      - cancellation
      - metadata
    - step metadata:
      - step id
      - type
      - input type
      - output type
      - freeform keyvalue metadata

See @geppetto/pkg/steps/step.go


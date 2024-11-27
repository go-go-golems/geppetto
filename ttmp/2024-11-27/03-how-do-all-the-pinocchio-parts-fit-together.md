# How do you set up a streaming completion in pinocchio

## Look at RunIntoWriter in @pinocchio/pkg/cmds/cmd.go

- create stepSettings
- create a standardStepFactory
- create an EventRouter
- add a handler to the router for the chat topic

- create a conversation manager
- create a chatstep using the stepfactory
    - pass in a published topic:
        - passed as constructor, but calls step.AddPublishedTopic


## Step

From @geppetto/pkg/steps/step.go:

```go
// Step is the generalization of a lambda function, with cancellation and closing
// to allow it to own resources.
type Step[T any, U any] interface {
	// Start gets called multiple times for the same Step, once per incoming value,
	// since StepResult is also the list monad (ie., supports multiple values)
	Start(ctx context.Context, input T) (StepResult[U], error)
	AddPublishedTopic(publisher message.Publisher, topic string) error
}
```

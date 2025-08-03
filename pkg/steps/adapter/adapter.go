package adapter

import (
	"context"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/go-go-golems/geppetto/pkg/conversation"
	"github.com/go-go-golems/geppetto/pkg/events"
	"github.com/go-go-golems/geppetto/pkg/helpers"
	"github.com/go-go-golems/geppetto/pkg/inference"
	"github.com/go-go-golems/geppetto/pkg/steps"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/chat"
	"github.com/pkg/errors"
)

var _ chat.Step = &StepAdapter{}

// StepAdapter provides backwards compatibility for the existing Step API
// while using the new inference engines under the hood.
type StepAdapter struct {
	engine           inference.Engine
	publisherManager *events.PublisherManager
	metadata         *steps.StepMetadata
}

// NewStepAdapter creates a new StepAdapter that wraps an inference engine
// and provides the existing Step interface.
func NewStepAdapter(engine inference.Engine, metadata *steps.StepMetadata) *StepAdapter {
	return &StepAdapter{
		engine:           engine,
		publisherManager: events.NewPublisherManager(),
		metadata:         metadata,
	}
}

// AddPublishedTopic implements the Step interface by registering a publisher
// for event publishing. This preserves the existing watermill-based event system.
func (a *StepAdapter) AddPublishedTopic(publisher message.Publisher, topic string) error {
	a.publisherManager.RegisterPublisher(topic, publisher)
	return nil
}

// Start implements the Step interface by wrapping the inference engine's
// RunInference method and converting it to the channel-based StepResult system.
func (a *StepAdapter) Start(
	ctx context.Context,
	messages conversation.Conversation,
) (steps.StepResult[*conversation.Message], error) {
	// Check if the engine supports streaming by checking if it implements SimpleChatStep
	if simpleChatStep, ok := a.engine.(chat.SimpleChatStep); ok {
		// Use the SimpleChatStep interface which might have streaming capabilities
		return a.startWithSimpleChatStep(ctx, messages, simpleChatStep)
	}

	// Fall back to basic engine interface
	return a.startWithEngine(ctx, messages)
}

// startWithEngine handles the case where we only have the basic Engine interface
func (a *StepAdapter) startWithEngine(
	ctx context.Context,
	messages conversation.Conversation,
) (steps.StepResult[*conversation.Message], error) {
	// For basic engines, always use non-streaming approach
	message, err := a.engine.RunInference(ctx, messages)
	if err != nil {
		return steps.Reject[*conversation.Message](err), nil
	}
	return steps.Resolve(message, steps.WithMetadata[*conversation.Message](a.metadata)), nil
}

// startWithSimpleChatStep handles engines that implement SimpleChatStep,
// which may have their own event publishing and streaming logic
func (a *StepAdapter) startWithSimpleChatStep(
	ctx context.Context,
	messages conversation.Conversation,
	simpleChatStep chat.SimpleChatStep,
) (steps.StepResult[*conversation.Message], error) {
	// Check if this is a streaming step by looking for specific implementations
	// that have their own Start method (like OpenAI ChatStep)
	if existingStep, ok := simpleChatStep.(chat.Step); ok {
		// This is already a complete Step implementation, delegate to it
		// but first copy our publisher configuration
		if a.publisherManager != nil {
			// Copy all registered publishers from our manager to the existing step
			for topic, publishers := range a.publisherManager.Publishers {
				for _, publisher := range publishers {
					err := existingStep.AddPublishedTopic(publisher, topic)
					if err != nil {
						return steps.Reject[*conversation.Message](err), nil
					}
				}
			}
		}
		return existingStep.Start(ctx, messages)
	}

	// For SimpleChatStep implementations without their own Start method,
	// wrap the RunInference call in our own channel-based system
	var cancel context.CancelFunc
	cancellableCtx, cancel := context.WithCancel(ctx)

	c := make(chan helpers.Result[*conversation.Message])
	ret := steps.NewStepResult[*conversation.Message](
		c,
		steps.WithCancel[*conversation.Message](cancel),
		steps.WithMetadata[*conversation.Message](a.metadata),
	)

	go func() {
		defer close(c)
		defer cancel()

		// Setup event metadata
		var parentMessage *conversation.Message
		parentID := conversation.NullNode

		if len(messages) > 0 {
			parentMessage = messages[len(messages)-1]
			parentID = parentMessage.ID
		}

		eventMetadata := events.EventMetadata{
			ID:       conversation.NewNodeID(),
			ParentID: parentID,
		}

		// Publish start event
		a.publisherManager.PublishBlind(events.NewStartEvent(eventMetadata, a.metadata))

		// Check for cancellation before starting
		select {
		case <-cancellableCtx.Done():
			c <- helpers.NewErrorResult[*conversation.Message](context.Canceled)
			return
		default:
		}

		// Use RunInference from the SimpleChatStep
		message, err := simpleChatStep.RunInference(cancellableCtx, messages)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				// Publish interrupt event
				a.publisherManager.PublishBlind(events.NewInterruptEvent(eventMetadata, a.metadata, ""))
				c <- helpers.NewErrorResult[*conversation.Message](context.Canceled)
			} else {
				// Publish error event
				a.publisherManager.PublishBlind(events.NewErrorEvent(eventMetadata, a.metadata, err))
				c <- helpers.NewErrorResult[*conversation.Message](err)
			}
			return
		}

		// Publish final event
		content := ""
		if message != nil && message.Content != nil {
			if chatContent, ok := message.Content.(*conversation.ChatMessageContent); ok {
				content = chatContent.Text
			}
		}
		a.publisherManager.PublishBlind(events.NewFinalEvent(eventMetadata, a.metadata, content))

		// Send the result
		c <- helpers.NewValueResult[*conversation.Message](message)
	}()

	return ret, nil
}

// StepAdapterOption is a functional option for configuring StepAdapter
type StepAdapterOption func(*StepAdapter) error

// WithPublisherManager sets a specific publisher manager for the adapter
func WithPublisherManager(publisherManager *events.PublisherManager) StepAdapterOption {
	return func(adapter *StepAdapter) error {
		adapter.publisherManager = publisherManager
		return nil
	}
}

// ApplyOptions applies a set of options to the adapter
func (a *StepAdapter) ApplyOptions(options ...StepAdapterOption) error {
	for _, option := range options {
		if err := option(a); err != nil {
			return err
		}
	}
	return nil
}

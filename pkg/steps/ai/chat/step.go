package chat

import (
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/go-go-golems/geppetto/pkg/conversation"
	"github.com/go-go-golems/geppetto/pkg/steps"
)

// Deprecated: The Steps API is deprecated and will be removed in a future version.
// Use the new inference.Engine interface instead, which provides a simpler and more
// powerful way to handle AI inference operations.
//
// Migration guide:
// - Replace chat.Step with inference.Engine
// - Replace ai.StandardStepFactory with inference.StandardEngineFactory
// - Use Engine.RunInference() instead of steps.Resolve() and steps.Bind()
// - Use inference.WatermillSink for event publishing instead of WithPublishedTopic
//
// For more information, see the Engine-first architecture documentation.
type Step steps.Step[conversation.Conversation, *conversation.Message]

// Deprecated: Use inference.Option with inference.WithSink instead.
// The WithPublishedTopic option is replaced by creating an inference.WatermillSink
// and passing it to the engine factory using inference.WithSink(sink).
type StepOption func(Step) error

// Deprecated: Use inference.WatermillSink with the inference.Engine interface instead.
// This function is part of the deprecated Steps API.
//
// Migration:
//
//	sink := inference.NewWatermillSink(publisher, topic)
//	engine, err := factory.CreateEngine(settings, inference.WithSink(sink))
func WithPublishedTopic(publisher message.Publisher, topic string) StepOption {
	return func(step Step) error {
		err := step.AddPublishedTopic(publisher, topic)
		if err != nil {
			return err
		}

		return nil
	}
}

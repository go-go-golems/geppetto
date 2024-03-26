package ollama

import (
	"context"
	"github.com/go-go-golems/bobatea/pkg/chat/conversation"
	"github.com/go-go-golems/geppetto/pkg/events"
	"github.com/go-go-golems/geppetto/pkg/helpers"
	"github.com/go-go-golems/geppetto/pkg/steps"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/chat"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/go-go-golems/glazed/pkg/helpers/maps"
	"github.com/google/uuid"
	"github.com/jmorganca/ollama/api"
)

type ChatCompletionStep struct {
	Client              *api.Client
	Settings            *settings.StepSettings
	subscriptionManager *events.PublisherManager
}

func NewChatCompletionStep(client *api.Client, settings *settings.StepSettings) *ChatCompletionStep {
	return &ChatCompletionStep{
		Client:              client,
		Settings:            settings,
		subscriptionManager: events.NewPublisherManager(),
	}
}

func (ccs *ChatCompletionStep) Start(
	ctx context.Context,
	messages []*conversation.Message,
) (steps.StepResult[string], error) {
	ollamaMessages := []api.Message{}
	for _, msg := range messages {
		switch content := msg.Content.(type) {
		case *conversation.ChatMessageContent:
			ollamaMessages = append(ollamaMessages, api.Message{
				Content: content.Text,
				Role:    string(content.Role),
			})
		}
	}
	var parentMessage *conversation.Message
	parentID := conversation.NullNode

	if len(messages) > 0 {
		parentMessage = messages[len(messages)-1]
		parentID = parentMessage.ID
	}

	metadata := chat.EventMetadata{
		ID:       conversation.NewNodeID(),
		ParentID: parentID,
	}
	stepMetadata := &steps.StepMetadata{
		StepID:     uuid.New(),
		Type:       "openai-chat",
		InputType:  "conversation.Conversation",
		OutputType: "string",
		Metadata: map[string]interface{}{
			steps.MetadataSettingsSlug: ccs.Settings.GetMetadata(),
		},
	}

	stream := ccs.Settings.Chat.Stream

	yaml, err := maps.StructToMapThroughYAML(ccs.Settings.Ollama)
	if err != nil {
		return nil, err
	}

	req := &api.ChatRequest{
		Model:    *ccs.Settings.Chat.Engine,
		Messages: ollamaMessages,
		Stream:   &stream,
		Format:   "json", // Assuming a text format
		Options:  yaml,
	}

	cancellableCtx, cancel := context.WithCancel(ctx)
	go func() {
		<-ctx.Done()
		cancel()
	}()

	c := make(chan helpers.Result[string])
	ret := steps.NewStepResult[string](c, steps.WithMetadata[string](stepMetadata), steps.WithCancel[string](cancel))

	go func() {
		defer close(c)

		message := ""

		err := ccs.Client.Chat(cancellableCtx, req, func(resp api.ChatResponse) error {
			// TODO(manuel, 2024-01-13) Handle metrics

			if resp.Done {
				ccs.subscriptionManager.PublishBlind(&chat.EventText{
					Event: chat.Event{
						Type:     chat.EventTypeFinal,
						Metadata: metadata,
						Step:     ret.GetMetadata(),
					},
					Text: message,
				})
				c <- helpers.NewValueResult[string](resp.Message.Content)
				return nil
			}

			message += resp.Message.Content

			ccs.subscriptionManager.PublishBlind(&chat.EventPartialCompletion{
				Event: chat.Event{
					Type:     chat.EventTypePartial,
					Metadata: metadata,
					Step:     ret.GetMetadata(),
				},
				Delta:      resp.Message.Content,
				Completion: message,
			})

			return nil
		})

		if err != nil {
			ccs.subscriptionManager.PublishBlind(&chat.EventText{
				Event: chat.Event{
					Type:     chat.EventTypeError,
					Error:    err,
					Metadata: metadata,
					Step:     ret.GetMetadata(),
				},
				Text: message,
			})
			c <- helpers.NewErrorResult[string](err)
		}
	}()

	return ret, nil
}

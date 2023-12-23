package ollama

import (
	"context"
	"errors"
	"time"

	geppetto_context "github.com/go-go-golems/geppetto/pkg/context"
	"github.com/go-go-golems/geppetto/pkg/helpers"
	"github.com/go-go-golems/geppetto/pkg/steps"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/chat"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/jmorganca/ollama/api"
)

type ChatCompletionStep struct {
	Client              *api.Client
	Settings            *settings.StepSettings
	subscriptionManager *helpers.SubscriptionManager
	cancel              context.CancelFunc
}

func NewChatCompletionStep(client *api.Client, settings *settings.StepSettings) *ChatCompletionStep {
	return &ChatCompletionStep{
		Client:              client,
		Settings:            settings,
		subscriptionManager: helpers.NewSubscriptionManager(),
	}
}

func ConvertMessage(ollamaMsg *api.Message) *geppetto_context.Message {

	gepMsg := &geppetto_context.Message{
		Text: ollamaMsg.Content,
		Role: ollamaMsg.Role,
		Time: time.Now(),
	}

	return gepMsg
}

func (ccs *ChatCompletionStep) Start(
	ctx context.Context,
	messages []*geppetto_context.Message,
) (steps.StepResult[string], error) {
	if ccs.cancel != nil {
		return nil, errors.New("step already started")
	}

	var cancel context.CancelFunc
	ctx, cancel = context.WithCancel(ctx)
	ccs.cancel = cancel

	ollamaMessages := []api.Message{}
	for _, msg := range messages {
		ollamaMessages = append(ollamaMessages, api.Message{
			Content: msg.Text,
			Role:    msg.Role,
		})
	}

	stream := ccs.Settings.Chat.Stream

	req := &api.ChatRequest{
		Model:    "gpt-4", // Or any other desired model
		Messages: ollamaMessages,
		Stream:   &stream,
		Format:   "text", // Assuming a text format
		Options:  make(map[string]interface{}),
	}

	if stream {
		return ccs.handleStreaming(ctx, req)
	} else {
		return ccs.handleNonStreaming(ctx, req)
	}
}

func (ccs *ChatCompletionStep) handleStreaming(ctx context.Context, req *api.ChatRequest) (steps.StepResult[string], error) {
	c := make(chan helpers.Result[string])
	ret := steps.NewStepResult[string](c)

	go func() {
		defer close(c)
		defer func() {
			ccs.cancel = nil
		}()

		err := ccs.Client.Chat(ctx, req, func(resp api.ChatResponse) error {
			if resp.Done {
				ccs.subscriptionManager.PublishBlind(&chat.Event{
					Type: chat.EventTypeFinal,
					Text: resp.Message.Content,
				})
				c <- helpers.NewValueResult[string](resp.Message.Content)
				return nil
			}

			ccs.subscriptionManager.PublishBlind(&chat.Event{
				Type: chat.EventTypePartial,
				Text: resp.Message.Content,
			})

			return nil
		})

		if err != nil {
			ccs.subscriptionManager.PublishBlind(&chat.Event{
				Type:  chat.EventTypeError,
				Error: err,
			})
			c <- helpers.NewErrorResult[string](err)
		}
	}()

	return ret, nil
}

func (ccs *ChatCompletionStep) handleNonStreaming(ctx context.Context, req *api.ChatRequest) (steps.StepResult[string], error) {
	c := make(chan helpers.Result[string])
	ret := steps.NewStepResult[string](c)

	msg := ""
	err := ccs.Client.Chat(ctx, req, func(resp api.ChatResponse) error {
		msg += resp.Message.Content

		if resp.Done {
			ccs.subscriptionManager.PublishBlind(&chat.Event{
				Type: chat.EventTypePartial,
				Text: resp.Message.Content,
			})

			c <- helpers.NewValueResult[string](msg)
			ccs.subscriptionManager.PublishBlind(&chat.Event{
				Type: chat.EventTypeFinal,
				Text: msg,
			})

			close(c)
			return nil
		}

		ccs.subscriptionManager.PublishBlind(&chat.Event{
			Type: chat.EventTypePartial,
			Text: resp.Message.Content,
		})

		return nil
	})
	if err != nil {
		ccs.subscriptionManager.PublishBlind(&chat.Event{
			Type:  chat.EventTypeError,
			Error: err,
		})
		return steps.Reject[string](err), nil
	}

	return ret, nil
}

func (ccs *ChatCompletionStep) Interrupt() {
	if ccs.cancel != nil {
		ccs.cancel()
	}
}

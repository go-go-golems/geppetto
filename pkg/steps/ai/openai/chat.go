package openai

import (
	"context"
	"github.com/ThreeDotsLabs/watermill/message"
	geppetto_context "github.com/go-go-golems/geppetto/pkg/context"
	"github.com/go-go-golems/geppetto/pkg/helpers"
	"github.com/go-go-golems/geppetto/pkg/steps"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/chat"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"io"
)

var _ steps.Step[[]*geppetto_context.Message, string] = &Step{}

type Step struct {
	Settings            *settings.StepSettings
	cancel              context.CancelFunc
	subscriptionManager *helpers.SubscriptionManager
}

func (csf *Step) Publish(publisher message.Publisher, topic string) error {
	csf.subscriptionManager.AddSubscription(topic, publisher)
	return nil
}

func NewStep(settings *settings.StepSettings) *Step {
	return &Step{
		Settings:            settings,
		subscriptionManager: helpers.NewSubscriptionManager(),
	}
}

func (csf *Step) Interrupt() {
	if csf.cancel != nil {
		csf.cancel()
	}
}

func (csf *Step) Start(
	ctx context.Context,
	messages []*geppetto_context.Message,
) (steps.StepResult[string], error) {
	if csf.cancel != nil {
		return nil, errors.New("step already started")
	}

	var cancel context.CancelFunc
	ctx, cancel = context.WithCancel(ctx)
	csf.cancel = cancel

	client := makeClient(csf.Settings.OpenAI)

	req, err := makeCompletionRequest(csf.Settings, messages)
	if err != nil {
		return nil, err
	}

	stream := csf.Settings.Chat.Stream

	if stream {
		stream, err := client.CreateChatCompletionStream(ctx, *req)
		if err != nil {
			return steps.Reject[string](err), nil
		}
		c := make(chan helpers.Result[string])
		ret := steps.NewStepResult[string](c)

		message := ""

		// TODO(manuel, 2023-11-28) We need to collect this goroutine in Close(), or at least I think so?
		go func() {
			defer close(c)
			defer stream.Close()

			for {
				select {
				case <-ctx.Done():
					log.Warn().Msg("context cancelled")
					csf.subscriptionManager.PublishBlind(&chat.Event{
						Type: chat.EventTypeInterrupt,
					})
					log.Warn().Msg("return error")
					c <- helpers.NewErrorResult[string](ctx.Err())
					return
				default:
					response, err := stream.Recv()
					if errors.Is(err, io.EOF) {
						csf.subscriptionManager.PublishBlind(&chat.Event{
							Type: chat.EventTypeFinal,
							Text: message,
						})
						c <- helpers.NewValueResult[string](message)

						return
					}
					if err != nil {
						csf.subscriptionManager.PublishBlind(&chat.Event{
							Type:  chat.EventTypeError,
							Error: err,
						})
						c <- helpers.NewErrorResult[string](err)
						return
					}

					csf.subscriptionManager.PublishBlind(&chat.Event{
						Type: chat.EventTypePartial,
						Text: response.Choices[0].Delta.Content,
					})

					message += response.Choices[0].Delta.Content
				}
			}
		}()

		return ret, nil
	} else {
		resp, err := client.CreateChatCompletion(ctx, *req)
		if errors.Is(err, context.Canceled) {
			csf.subscriptionManager.PublishBlind(&chat.Event{
				Type: chat.EventTypeInterrupt,
			})
			return steps.Reject[string](err), nil
		}

		if err != nil {
			csf.subscriptionManager.PublishBlind(&chat.Event{
				Type:  chat.EventTypeError,
				Error: err,
			})
			return steps.Reject[string](err), nil
		}

		if err != nil {
			csf.subscriptionManager.PublishBlind(&chat.Event{
				Type:  chat.EventTypeError,
				Error: err,
			})
			return steps.Reject[string](err), nil
		}

		csf.subscriptionManager.PublishBlind(&chat.Event{
			Type: chat.EventTypeFinal,
			Text: resp.Choices[0].Message.Content,
		})
		return steps.Resolve(resp.Choices[0].Message.Content), nil
	}
}

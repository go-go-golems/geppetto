package openai

import (
	"context"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/go-go-golems/bobatea/pkg/chat/conversation"
	"github.com/go-go-golems/geppetto/pkg/helpers"
	"github.com/go-go-golems/geppetto/pkg/steps"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/chat"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"io"
)

var _ steps.Step[[]*conversation.Message, string] = &Step{}

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
	messages []*conversation.Message,
) (steps.StepResult[string], error) {
	if csf.cancel != nil {
		return nil, errors.New("step already started")
	}

	var cancel context.CancelFunc
	cancellableCtx, cancel := context.WithCancel(ctx)
	csf.cancel = cancel

	client := makeClient(csf.Settings.OpenAI)

	req, err := makeCompletionRequest(csf.Settings, messages)
	if err != nil {
		return nil, err
	}

	var parentMessage *conversation.Message
	parentID := uuid.Nil
	conversationID := uuid.New()

	if len(messages) > 0 {
		parentMessage = messages[len(messages)-1]
		parentID = parentMessage.ID
		conversationID = parentMessage.ConversationID
	}

	metadata := chat.EventMetadata{
		ID:             uuid.New(),
		ParentID:       parentID,
		ConversationID: conversationID,
	}

	stream := csf.Settings.Chat.Stream

	csf.subscriptionManager.PublishBlind(&chat.Event{
		Type:     chat.EventTypeStart,
		Metadata: metadata,
	})

	if stream {
		stream, err := client.CreateChatCompletionStream(cancellableCtx, *req)
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
			defer func() {
				csf.cancel = nil
			}()

			for {
				select {
				case <-cancellableCtx.Done():
					csf.subscriptionManager.PublishBlind(&chat.Event{
						Type: chat.EventTypeInterrupt,
						Text: message,
					})
					c <- helpers.NewErrorResult[string](cancellableCtx.Err())
					return

				default:
					response, err := stream.Recv()

					if errors.Is(err, io.EOF) {
						csf.subscriptionManager.PublishBlind(&chat.Event{
							Type:     chat.EventTypeFinal,
							Text:     message,
							Metadata: metadata,
						})
						c <- helpers.NewValueResult[string](message)

						return
					}
					if err != nil {
						if errors.Is(err, context.Canceled) {
							csf.subscriptionManager.PublishBlind(&chat.Event{
								Type:     chat.EventTypeInterrupt,
								Text:     message,
								Metadata: metadata,
							})
							c <- helpers.NewErrorResult[string](err)
							return
						}

						csf.subscriptionManager.PublishBlind(&chat.Event{
							Type:     chat.EventTypeError,
							Error:    err,
							Metadata: metadata,
						})
						c <- helpers.NewErrorResult[string](err)
						return
					}

					csf.subscriptionManager.PublishBlind(&chat.Event{
						Type:     chat.EventTypePartial,
						Text:     response.Choices[0].Delta.Content,
						Metadata: metadata,
					})

					message += response.Choices[0].Delta.Content
				}
			}
		}()

		return ret, nil
	} else {
		resp, err := client.CreateChatCompletion(cancellableCtx, *req)
		if errors.Is(err, context.Canceled) {
			csf.subscriptionManager.PublishBlind(&chat.Event{
				Type:     chat.EventTypeInterrupt,
				Metadata: metadata,
			})
			return steps.Reject[string](err), nil
		}

		if err != nil {
			csf.subscriptionManager.PublishBlind(&chat.Event{
				Type:     chat.EventTypeError,
				Error:    err,
				Metadata: metadata,
			})
			return steps.Reject[string](err), nil
		}

		if err != nil {
			csf.subscriptionManager.PublishBlind(&chat.Event{
				Type:     chat.EventTypeError,
				Error:    err,
				Metadata: metadata,
			})
			return steps.Reject[string](err), nil
		}

		csf.subscriptionManager.PublishBlind(&chat.Event{
			Type:     chat.EventTypeFinal,
			Text:     resp.Choices[0].Message.Content,
			Metadata: metadata,
		})
		return steps.Resolve(resp.Choices[0].Message.Content), nil
	}
}

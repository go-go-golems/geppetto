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

type StepOption func(*Step) error

func WithSubscriptionManager(subscriptionManager *helpers.SubscriptionManager) StepOption {
	return func(step *Step) error {
		step.subscriptionManager = subscriptionManager
		return nil
	}
}

func NewStep(settings *settings.StepSettings, options ...StepOption) (*Step, error) {
	ret := &Step{
		Settings:            settings,
		subscriptionManager: helpers.NewSubscriptionManager(),
	}

	for _, option := range options {
		err := option(ret)
		if err != nil {
			return nil, err
		}
	}

	return ret, nil
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

		// TODO(manuel, 2023-11-28) We need to collect this goroutine in Close(), or at least I think so?
		go func() {
			defer close(c)
			defer stream.Close()
			defer func() {
				csf.cancel = nil
			}()

			message := ""

			for {
				select {
				case <-cancellableCtx.Done():
					csf.subscriptionManager.PublishBlind(&chat.EventText{
						Event: chat.Event{
							Type:     chat.EventTypeInterrupt,
							Metadata: metadata,
						},
						Text: message,
					})
					c <- helpers.NewErrorResult[string](cancellableCtx.Err())
					return

				default:
					response, err := stream.Recv()

					if errors.Is(err, io.EOF) {
						csf.subscriptionManager.PublishBlind(&chat.EventText{
							Event: chat.Event{
								Type:     chat.EventTypeFinal,
								Metadata: metadata,
							},
							Text: message,
						})
						c <- helpers.NewValueResult[string](message)

						return
					}
					if err != nil {
						if errors.Is(err, context.Canceled) {
							csf.subscriptionManager.PublishBlind(&chat.EventText{
								Event: chat.Event{
									Type:     chat.EventTypeInterrupt,
									Metadata: metadata,
								},
								Text: message,
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

					message += response.Choices[0].Delta.Content

					csf.subscriptionManager.PublishBlind(&chat.EventPartialCompletion{
						Event: chat.Event{
							Type:     chat.EventTypePartial,
							Metadata: metadata,
						},
						Delta:      response.Choices[0].Delta.Content,
						Completion: message,
					})
				}
			}
		}()

		return ret, nil
	} else {
		resp, err := client.CreateChatCompletion(cancellableCtx, *req)
		if errors.Is(err, context.Canceled) {
			csf.subscriptionManager.PublishBlind(&chat.EventText{
				Event: chat.Event{
					Type:     chat.EventTypeInterrupt,
					Metadata: metadata,
				},
				Text: "",
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

		csf.subscriptionManager.PublishBlind(&chat.EventText{
			Event: chat.Event{
				Type:     chat.EventTypeFinal,
				Metadata: metadata,
			},
			Text: resp.Choices[0].Message.Content,
		})
		return steps.Resolve(resp.Choices[0].Message.Content), nil
	}
}

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

var _ steps.Step[conversation.Conversation, string] = &Step{}

type Step struct {
	Settings            *settings.StepSettings
	subscriptionManager *helpers.SubscriptionManager
}

func (csf *Step) AddPublishedTopic(publisher message.Publisher, topic string) error {
	csf.subscriptionManager.AddPublishedTopic(topic, publisher)
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

func (csf *Step) Start(
	ctx context.Context,
	messages conversation.Conversation,
) (steps.StepResult[string], error) {
	var cancel context.CancelFunc
	cancellableCtx, cancel := context.WithCancel(ctx)
	go func() {
		<-ctx.Done()
		cancel()
	}()

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
	stepMetadata := &steps.StepMetadata{
		StepID:     uuid.New(),
		Type:       "openai-chat",
		InputType:  "conversation.Conversation",
		OutputType: "string",
		Metadata:   csf.Settings.GetMetadata(),
	}

	stream := csf.Settings.Chat.Stream

	csf.subscriptionManager.PublishBlind(&chat.Event{
		Type:     chat.EventTypeStart,
		Metadata: metadata,
		Step:     stepMetadata,
	})

	if stream {
		stream, err := client.CreateChatCompletionStream(cancellableCtx, *req)
		if err != nil {
			return steps.Reject[string](err), nil
		}
		c := make(chan helpers.Result[string])
		ret := steps.NewStepResult[string](
			c,
			steps.WithCancel[string](cancel),
			steps.WithMetadata[string](
				stepMetadata,
			),
		)

		// TODO(manuel, 2023-11-28) We need to collect this goroutine in Close(), or at least I think so?
		go func() {
			defer close(c)
			defer stream.Close()

			message := ""

			for {
				select {
				case <-cancellableCtx.Done():
					csf.subscriptionManager.PublishBlind(&chat.EventText{
						Event: chat.Event{
							Type:     chat.EventTypeInterrupt,
							Metadata: metadata,
							Step:     ret.GetMetadata(),
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
								Step:     ret.GetMetadata(),
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
									Step:     ret.GetMetadata(),
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
							Step:     ret.GetMetadata(),
						})
						c <- helpers.NewErrorResult[string](err)
						return
					}

					message += response.Choices[0].Delta.Content

					csf.subscriptionManager.PublishBlind(&chat.EventPartialCompletion{
						Event: chat.Event{
							Type:     chat.EventTypePartial,
							Metadata: metadata,
							Step:     ret.GetMetadata(),
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
					Step:     stepMetadata,
				},
				Text: "",
			})
			return steps.Reject[string](err, steps.WithMetadata[string](stepMetadata)), nil
		}

		if err != nil {
			csf.subscriptionManager.PublishBlind(&chat.Event{
				Type:     chat.EventTypeError,
				Error:    err,
				Metadata: metadata,
				Step:     stepMetadata,
			})
			return steps.Reject[string](err, steps.WithMetadata[string](stepMetadata)), nil
		}

		csf.subscriptionManager.PublishBlind(&chat.EventText{
			Event: chat.Event{
				Type:     chat.EventTypeFinal,
				Metadata: metadata,
				Step:     stepMetadata,
			},
			Text: resp.Choices[0].Message.Content,
		})
		return steps.Resolve(resp.Choices[0].Message.Content, steps.WithMetadata[string](stepMetadata)), nil
	}
}

package openai

import (
	"context"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/go-go-golems/bobatea/pkg/conversation"
	"github.com/go-go-golems/geppetto/pkg/events"
	"github.com/go-go-golems/geppetto/pkg/helpers"
	"github.com/go-go-golems/geppetto/pkg/steps"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/chat"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"io"
)

var _ steps.Step[conversation.Conversation, string] = &ChatStep{}

type ChatStep struct {
	Settings         *settings.StepSettings
	publisherManager *events.PublisherManager
}

func (csf *ChatStep) AddPublishedTopic(publisher message.Publisher, topic string) error {
	csf.publisherManager.SubscribePublisher(topic, publisher)
	return nil
}

type StepOption func(*ChatStep) error

func WithSubscriptionManager(subscriptionManager *events.PublisherManager) StepOption {
	return func(step *ChatStep) error {
		step.publisherManager = subscriptionManager
		return nil
	}
}

func NewStep(settings *settings.StepSettings, options ...StepOption) (*ChatStep, error) {
	ret := &ChatStep{
		Settings:         settings,
		publisherManager: events.NewPublisherManager(),
	}

	for _, option := range options {
		err := option(ret)
		if err != nil {
			return nil, err
		}
	}

	return ret, nil
}

func (csf *ChatStep) Start(
	ctx context.Context,
	messages conversation.Conversation,
) (steps.StepResult[string], error) {
	var cancel context.CancelFunc
	cancellableCtx, cancel := context.WithCancel(ctx)
	go func() {
		<-ctx.Done()
		cancel()
	}()

	if csf.Settings.Chat.ApiType == nil {
		return steps.Reject[string](errors.New("no chat engine specified")), nil
	}

	client, err := makeClient(csf.Settings.API, *csf.Settings.Chat.ApiType)
	if err != nil {
		return nil, err
	}

	req, err := makeCompletionRequest(csf.Settings, messages)
	if err != nil {
		return nil, err
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
			steps.MetadataSettingsSlug: csf.Settings.GetMetadata(),
		},
	}

	stream := csf.Settings.Chat.Stream

	csf.publisherManager.PublishBlind(chat.NewStartEvent(metadata, stepMetadata))

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
			defer func() {
				close(c)
			}()
			defer stream.Close()

			message := ""

			for {
				select {
				case <-cancellableCtx.Done():
					csf.publisherManager.PublishBlind(chat.NewInterruptEvent(metadata, ret.GetMetadata(), message))
					c <- helpers.NewErrorResult[string](cancellableCtx.Err())
					return

				default:
					response, err := stream.Recv()

					if errors.Is(err, io.EOF) {
						csf.publisherManager.PublishBlind(chat.NewFinalEvent(metadata, ret.GetMetadata(), message))
						c <- helpers.NewValueResult[string](message)

						return
					}
					if err != nil {
						if errors.Is(err, context.Canceled) {
							csf.publisherManager.PublishBlind(chat.NewInterruptEvent(metadata, ret.GetMetadata(), message))
							c <- helpers.NewErrorResult[string](err)
							return
						}

						csf.publisherManager.PublishBlind(chat.NewErrorEvent(metadata, ret.GetMetadata(), err.Error()))
						c <- helpers.NewErrorResult[string](err)
						return
					}

					message += response.Choices[0].Delta.Content

					csf.publisherManager.PublishBlind(chat.NewPartialCompletionEvent(metadata, ret.GetMetadata(), response.Choices[0].Delta.Content, message))
				}
			}
		}()

		return ret, nil
	} else {
		resp, err := client.CreateChatCompletion(cancellableCtx, *req)
		if errors.Is(err, context.Canceled) {
			csf.publisherManager.PublishBlind(chat.NewInterruptEvent(metadata, stepMetadata, ""))
			return steps.Reject[string](err, steps.WithMetadata[string](stepMetadata)), nil
		}

		if err != nil {
			csf.publisherManager.PublishBlind(chat.NewErrorEvent(metadata, stepMetadata, err.Error()))
			return steps.Reject[string](err, steps.WithMetadata[string](stepMetadata)), nil
		}

		csf.publisherManager.PublishBlind(chat.NewFinalEvent(metadata, stepMetadata, resp.Choices[0].Message.Content))
		return steps.Resolve(resp.Choices[0].Message.Content, steps.WithMetadata[string](stepMetadata)), nil
	}
}

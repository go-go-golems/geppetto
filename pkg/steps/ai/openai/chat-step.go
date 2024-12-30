package openai

import (
	"context"
	"io"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/go-go-golems/geppetto/pkg/conversation"
	"github.com/go-go-golems/geppetto/pkg/events"
	"github.com/go-go-golems/geppetto/pkg/helpers"
	"github.com/go-go-golems/geppetto/pkg/steps"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/chat"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

var _ steps.Step[conversation.Conversation, string] = &ChatStep{}

type ChatStep struct {
	Settings         *settings.StepSettings
	publisherManager *events.PublisherManager
}

func (csf *ChatStep) AddPublishedTopic(publisher message.Publisher, topic string) error {
	csf.publisherManager.RegisterPublisher(topic, publisher)
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
		Engine:   string(*csf.Settings.Chat.ApiType),
	}
	if csf.Settings.Chat.Temperature != nil {
		metadata.Temperature = *csf.Settings.Chat.Temperature
	}
	if csf.Settings.Chat.TopP != nil {
		metadata.TopP = *csf.Settings.Chat.TopP
	}
	if csf.Settings.Chat.MaxResponseTokens != nil {
		metadata.MaxTokens = *csf.Settings.Chat.MaxResponseTokens
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
						// Update both step metadata and event metadata with usage information
						if openaiMetadata, ok := stepMetadata.Metadata["openai-metadata"].(map[string]interface{}); ok {
							if usage, ok := openaiMetadata["usage"].(map[string]interface{}); ok {
								stepMetadata.Metadata["usage"] = map[string]interface{}{
									"input_tokens":  usage["prompt_tokens"],
									"output_tokens": usage["completion_tokens"],
								}
								metadata.Usage = &chat.Usage{
									InputTokens:  int(usage["prompt_tokens"].(float64)),
									OutputTokens: int(usage["completion_tokens"].(float64)),
								}
							}
							if finishReason, ok := openaiMetadata["finish_reason"].(string); ok {
								metadata.StopReason = finishReason
							}
						}
						csf.publisherManager.PublishBlind(chat.NewFinalEvent(
							metadata,
							stepMetadata,
							message,
						))
						c <- helpers.NewValueResult[string](message)

						return
					}
					if err != nil {
						if errors.Is(err, context.Canceled) {
							csf.publisherManager.PublishBlind(chat.NewInterruptEvent(metadata, stepMetadata, message))
							c <- helpers.NewErrorResult[string](err)
							return
						}

						csf.publisherManager.PublishBlind(chat.NewErrorEvent(metadata, stepMetadata, err.Error()))
						c <- helpers.NewErrorResult[string](err)
						return
					}
					delta := ""
					if len(response.Choices) > 0 {
						delta = response.Choices[0].Delta.Content
						message += delta
					}

					// Extract metadata from OpenAI chat response and update both step and event metadata
					if responseMetadata, err := ExtractChatCompletionMetadata(&response); err == nil && responseMetadata != nil {
						stepMetadata.Metadata["openai-metadata"] = responseMetadata
						if usage, ok := responseMetadata["usage"].(map[string]interface{}); ok {
							metadata.Usage = &chat.Usage{
								InputTokens:  int(usage["prompt_tokens"].(float64)),
								OutputTokens: int(usage["completion_tokens"].(float64)),
							}
						}
						if finishReason, ok := responseMetadata["finish_reason"].(string); ok {
							metadata.StopReason = finishReason
						}
					}

					csf.publisherManager.PublishBlind(
						chat.NewPartialCompletionEvent(
							metadata,
							stepMetadata,
							delta, message),
					)
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

		// Extract metadata from non-streaming response
		if usage := resp.Usage; usage.PromptTokens > 0 || usage.CompletionTokens > 0 {
			metadata.Usage = &chat.Usage{
				InputTokens:  usage.PromptTokens,
				OutputTokens: usage.CompletionTokens,
			}
			stepMetadata.Metadata["usage"] = map[string]interface{}{
				"input_tokens":  usage.PromptTokens,
				"output_tokens": usage.CompletionTokens,
			}
		}
		if len(resp.Choices) > 0 && resp.Choices[0].FinishReason != "" {
			metadata.StopReason = string(resp.Choices[0].FinishReason)
		}

		csf.publisherManager.PublishBlind(chat.NewFinalEvent(metadata, stepMetadata, resp.Choices[0].Message.Content))
		return steps.Resolve(resp.Choices[0].Message.Content, steps.WithMetadata[string](stepMetadata)), nil
	}
}

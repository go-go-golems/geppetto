package openai

import (
	"context"
	"io"
	stdlog "log"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/go-go-golems/geppetto/pkg/conversation"
	"github.com/go-go-golems/geppetto/pkg/events"
	"github.com/go-go-golems/geppetto/pkg/helpers"
	"github.com/go-go-golems/geppetto/pkg/steps"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/chat"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/go-go-golems/glazed/pkg/helpers/cast"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

var _ chat.Step = &ChatStep{}
var _ chat.SimpleChatStep = &ChatStep{}

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

func (csf *ChatStep) RunInference(
	ctx context.Context,
	messages conversation.Conversation,
) (*conversation.Message, error) {
	log.Debug().Int("num_messages", len(messages)).Bool("stream", csf.Settings.Chat.Stream).Msg("OpenAI RunInference started")
	if csf.Settings.Chat.ApiType == nil {
		return nil, errors.New("no chat engine specified")
	}

	client, err := MakeClient(csf.Settings.API, *csf.Settings.Chat.ApiType)
	if err != nil {
		return nil, err
	}

	req, err := MakeCompletionRequest(csf.Settings, messages)
	if err != nil {
		return nil, err
	}

	// Setup metadata and event publishing
	var parentMessage *conversation.Message
	parentID := conversation.NullNode

	if len(messages) > 0 {
		parentMessage = messages[len(messages)-1]
		parentID = parentMessage.ID
	}

	metadata := events.EventMetadata{
		ID:       conversation.NewNodeID(),
		ParentID: parentID,
		LLMMessageMetadata: conversation.LLMMessageMetadata{
			Engine:      string(*csf.Settings.Chat.Engine),
			Temperature: csf.Settings.Chat.Temperature,
			TopP:        csf.Settings.Chat.TopP,
			MaxTokens:   csf.Settings.Chat.MaxResponseTokens,
		},
	}
	if csf.Settings.Chat.Temperature != nil {
		metadata.Temperature = csf.Settings.Chat.Temperature
	}
	if csf.Settings.Chat.TopP != nil {
		metadata.TopP = csf.Settings.Chat.TopP
	}
	if csf.Settings.Chat.MaxResponseTokens != nil {
		metadata.MaxTokens = csf.Settings.Chat.MaxResponseTokens
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

	// Publish start event
	log.Debug().Str("event_id", metadata.ID.String()).Msg("OpenAI publishing start event")
	csf.publisherManager.PublishBlind(events.NewStartEvent(metadata, stepMetadata))

	if csf.Settings.Chat.Stream {
		// For streaming, collect all chunks and return the final message
		log.Debug().Msg("OpenAI using streaming mode")
		stream, err := client.CreateChatCompletionStream(ctx, *req)
		if err != nil {
			log.Error().Err(err).Msg("OpenAI streaming request failed")
			csf.publisherManager.PublishBlind(events.NewErrorEvent(metadata, stepMetadata, err))
			return nil, err
		}
		defer func() {
			if err := stream.Close(); err != nil {
				stdlog.Printf("Failed to close stream: %v", err)
			}
		}()

		message := ""
		var usage *conversation.Usage
		var stopReason *string

		log.Debug().Msg("OpenAI starting streaming loop")
		chunkCount := 0
		for {
			select {
			case <-ctx.Done():
				log.Debug().Msg("OpenAI streaming cancelled by context")
				// Publish interrupt event with current partial text
				csf.publisherManager.PublishBlind(events.NewInterruptEvent(metadata, stepMetadata, message))
				return nil, ctx.Err()

			default:
				response, err := stream.Recv()
				if errors.Is(err, io.EOF) {
					log.Debug().Int("chunks_received", chunkCount).Msg("OpenAI stream completed")
					goto streamingComplete
				}
				if err != nil {
					log.Error().Err(err).Int("chunks_received", chunkCount).Msg("OpenAI stream receive failed")
					csf.publisherManager.PublishBlind(events.NewErrorEvent(metadata, stepMetadata, err))
					return nil, err
				}
				chunkCount++

				delta := ""
				if len(response.Choices) > 0 {
					delta = response.Choices[0].Delta.Content
					message += delta
					log.Debug().Int("chunk", chunkCount).Str("delta", delta).Int("total_length", len(message)).Msg("OpenAI received chunk")
				}

				// Extract metadata from OpenAI chat response
				if responseMetadata, err := ExtractChatCompletionMetadata(&response); err == nil && responseMetadata != nil {
					if usageData, ok := responseMetadata["usage"].(map[string]interface{}); ok {
						inputTokens, _ := cast.CastNumberInterfaceToInt[int](usageData["prompt_tokens"])
						outputTokens, _ := cast.CastNumberInterfaceToInt[int](usageData["completion_tokens"])
						usage = &conversation.Usage{
							InputTokens:  inputTokens,
							OutputTokens: outputTokens,
						}
					}
					if finishReason, ok := responseMetadata["finish_reason"].(string); ok {
						stopReason = &finishReason
					}
				}

				// Publish intermediate streaming event (this was missing!)
				log.Debug().Int("chunk", chunkCount).Str("delta", delta).Msg("OpenAI publishing partial completion event")
				csf.publisherManager.PublishBlind(
					events.NewPartialCompletionEvent(
						metadata,
						stepMetadata,
						delta, message),
				)
			}
		}

	streamingComplete:

		// Update event metadata with usage information
		if usage != nil {
			metadata.Usage = usage
		}
		if stopReason != nil {
			metadata.StopReason = stopReason
		}

		llmMetadata := &conversation.LLMMessageMetadata{
			Engine: string(*csf.Settings.Chat.Engine),
		}
		if csf.Settings.Chat.Temperature != nil {
			llmMetadata.Temperature = csf.Settings.Chat.Temperature
		}
		if csf.Settings.Chat.TopP != nil {
			llmMetadata.TopP = csf.Settings.Chat.TopP
		}
		if csf.Settings.Chat.MaxResponseTokens != nil {
			llmMetadata.MaxTokens = csf.Settings.Chat.MaxResponseTokens
		}
		if usage != nil {
			llmMetadata.Usage = usage
		}
		if stopReason != nil {
			llmMetadata.StopReason = stopReason
		}

		messageContent := conversation.NewChatMessageContent(conversation.RoleAssistant, message, nil)
		finalMessage := conversation.NewMessage(messageContent, conversation.WithLLMMessageMetadata(llmMetadata))

		// Publish final event for streaming
		log.Debug().Str("event_id", metadata.ID.String()).Int("final_length", len(message)).Msg("OpenAI publishing final event (streaming)")
		csf.publisherManager.PublishBlind(events.NewFinalEvent(metadata, stepMetadata, message))

		log.Debug().Msg("OpenAI RunInference completed (streaming)")
		return finalMessage, nil
	} else {
		log.Debug().Msg("OpenAI using non-streaming mode")
		resp, err := client.CreateChatCompletion(ctx, *req)
		if err != nil {
			log.Error().Err(err).Msg("OpenAI non-streaming request failed")
			csf.publisherManager.PublishBlind(events.NewErrorEvent(metadata, stepMetadata, err))
			return nil, err
		}

		llmMetadata := &conversation.LLMMessageMetadata{
			Engine: string(*csf.Settings.Chat.Engine),
		}
		if csf.Settings.Chat.Temperature != nil {
			llmMetadata.Temperature = csf.Settings.Chat.Temperature
		}
		if csf.Settings.Chat.TopP != nil {
			llmMetadata.TopP = csf.Settings.Chat.TopP
		}
		if csf.Settings.Chat.MaxResponseTokens != nil {
			llmMetadata.MaxTokens = csf.Settings.Chat.MaxResponseTokens
		}

		// Extract metadata from non-streaming response and update event metadata
		if usage := resp.Usage; usage.PromptTokens > 0 || usage.CompletionTokens > 0 {
			usageData := &conversation.Usage{
				InputTokens:  usage.PromptTokens,
				OutputTokens: usage.CompletionTokens,
			}
			llmMetadata.Usage = usageData
			metadata.Usage = usageData
		}
		if len(resp.Choices) > 0 && resp.Choices[0].FinishReason != "" {
			finishReason := string(resp.Choices[0].FinishReason)
			llmMetadata.StopReason = &finishReason
			metadata.StopReason = &finishReason
		}

		finalMessage := conversation.NewMessage(
			conversation.NewChatMessageContent(conversation.RoleAssistant, resp.Choices[0].Message.Content, nil),
			conversation.WithLLMMessageMetadata(llmMetadata),
		)

		// Publish final event for non-streaming
		log.Debug().Str("event_id", metadata.ID.String()).Msg("OpenAI publishing final event (non-streaming)")
		csf.publisherManager.PublishBlind(events.NewFinalEvent(metadata, stepMetadata, resp.Choices[0].Message.Content))

		log.Debug().Msg("OpenAI RunInference completed (non-streaming)")
		return finalMessage, nil
	}
}

func (csf *ChatStep) Start(
	ctx context.Context,
	messages conversation.Conversation,
) (steps.StepResult[*conversation.Message], error) {
	log.Debug().Bool("stream", csf.Settings.Chat.Stream).Msg("OpenAI Start called")
	// For non-streaming, use the simplified RunInference method
	if !csf.Settings.Chat.Stream {
		log.Debug().Msg("OpenAI Start using non-streaming path")
		message, err := csf.RunInference(ctx, messages)
		if err != nil {
			return steps.Reject[*conversation.Message](err), nil
		}
		return steps.Resolve(message), nil
	}

	// For streaming, use RunInference in a goroutine to handle cancellation
	log.Debug().Msg("OpenAI Start using streaming path with goroutine")
	var cancel context.CancelFunc
	cancellableCtx, cancel := context.WithCancel(ctx)

	c := make(chan helpers.Result[*conversation.Message])
	ret := steps.NewStepResult[*conversation.Message](
		c,
		steps.WithCancel[*conversation.Message](cancel),
		steps.WithMetadata[*conversation.Message](&steps.StepMetadata{
			StepID:     uuid.New(),
			Type:       "openai-chat",
			InputType:  "conversation.Conversation",
			OutputType: "*conversation.Message",
			Metadata: map[string]interface{}{
				steps.MetadataSettingsSlug: csf.Settings.GetMetadata(),
			},
		}),
	)

	go func() {
		defer close(c)
		defer cancel()
		log.Debug().Msg("OpenAI streaming goroutine started")

		// Check for cancellation before starting
		select {
		case <-cancellableCtx.Done():
			log.Debug().Msg("OpenAI context cancelled before starting")
			c <- helpers.NewErrorResult[*conversation.Message](context.Canceled)
			return
		default:
		}

		// Use RunInference which now handles all the streaming logic
		log.Debug().Msg("OpenAI calling RunInference from goroutine")
		message, err := csf.RunInference(cancellableCtx, messages)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				// Context was cancelled during RunInference
				log.Debug().Msg("OpenAI RunInference cancelled")
				c <- helpers.NewErrorResult[*conversation.Message](context.Canceled)
			} else {
				log.Error().Err(err).Msg("OpenAI RunInference failed")
				c <- helpers.NewErrorResult[*conversation.Message](err)
			}
			return
		}
		log.Debug().Msg("OpenAI RunInference succeeded, sending result")
		result := helpers.NewValueResult[*conversation.Message](message)
		log.Debug().Msg("OpenAI about to send result to channel")
		c <- result
		log.Debug().Msg("OpenAI result sent to channel successfully")
	}()

	return ret, nil
}

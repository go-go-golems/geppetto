package openai

import (
	"context"
	"github.com/go-go-golems/geppetto/pkg/inference/engine"
	"io"
	stdlog "log"

	"github.com/go-go-golems/geppetto/pkg/conversation"
	"github.com/go-go-golems/geppetto/pkg/events"
	"github.com/go-go-golems/geppetto/pkg/steps"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/go-go-golems/glazed/pkg/helpers/cast"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

// OpenAIEngine implements the Engine interface for OpenAI API calls.
// It wraps the existing OpenAI logic from geppetto's ChatStep implementation.
type OpenAIEngine struct {
	settings *settings.StepSettings
	config   *engine.Config
}

// NewOpenAIEngine creates a new OpenAI inference engine with the given settings and options.
func NewOpenAIEngine(settings *settings.StepSettings, options ...engine.Option) (*OpenAIEngine, error) {
	config := engine.NewConfig()
	if err := engine.ApplyOptions(config, options...); err != nil {
		return nil, err
	}

	return &OpenAIEngine{
		settings: settings,
		config:   config,
	}, nil
}

// RunInference processes a conversation using OpenAI API and returns the generated message.
// This implementation is extracted from the existing OpenAI ChatStep RunInference method.
func (e *OpenAIEngine) RunInference(
	ctx context.Context,
	messages conversation.Conversation,
) (*conversation.Message, error) {
	log.Debug().Int("num_messages", len(messages)).Bool("stream", e.settings.Chat.Stream).Msg("OpenAI RunInference started")
	if e.settings.Chat.ApiType == nil {
		return nil, errors.New("no chat engine specified")
	}

	client, err := MakeClient(e.settings.API, *e.settings.Chat.ApiType)
	if err != nil {
		return nil, err
	}

	req, err := MakeCompletionRequest(e.settings, messages)
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
			Engine:      string(*e.settings.Chat.Engine),
			Temperature: e.settings.Chat.Temperature,
			TopP:        e.settings.Chat.TopP,
			MaxTokens:   e.settings.Chat.MaxResponseTokens,
		},
	}
	if e.settings.Chat.Temperature != nil {
		metadata.Temperature = e.settings.Chat.Temperature
	}
	if e.settings.Chat.TopP != nil {
		metadata.TopP = e.settings.Chat.TopP
	}
	if e.settings.Chat.MaxResponseTokens != nil {
		metadata.MaxTokens = e.settings.Chat.MaxResponseTokens
	}
	stepMetadata := &steps.StepMetadata{
		StepID:     uuid.New(),
		Type:       "openai-chat",
		InputType:  "conversation.Conversation",
		OutputType: "string",
		Metadata: map[string]interface{}{
			steps.MetadataSettingsSlug: e.settings.GetMetadata(),
		},
	}

	// Publish start event
	log.Debug().Str("event_id", metadata.ID.String()).Msg("OpenAI publishing start event")
	e.publishEvent(events.NewStartEvent(metadata, stepMetadata))

	if e.settings.Chat.Stream {
		// For streaming, collect all chunks and return the final message
		log.Debug().Msg("OpenAI using streaming mode")
		stream, err := client.CreateChatCompletionStream(ctx, *req)
		if err != nil {
			log.Error().Err(err).Msg("OpenAI streaming request failed")
			e.publishEvent(events.NewErrorEvent(metadata, stepMetadata, err))
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
				e.publishEvent(events.NewInterruptEvent(metadata, stepMetadata, message))
				return nil, ctx.Err()

			default:
				response, err := stream.Recv()
				if errors.Is(err, io.EOF) {
					log.Debug().Int("chunks_received", chunkCount).Msg("OpenAI stream completed")
					goto streamingComplete
				}
				if err != nil {
					log.Error().Err(err).Int("chunks_received", chunkCount).Msg("OpenAI stream receive failed")
					e.publishEvent(events.NewErrorEvent(metadata, stepMetadata, err))
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

				// Publish intermediate streaming event
				log.Debug().Int("chunk", chunkCount).Str("delta", delta).Msg("OpenAI publishing partial completion event")
				e.publishEvent(
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
			Engine: string(*e.settings.Chat.Engine),
		}
		if e.settings.Chat.Temperature != nil {
			llmMetadata.Temperature = e.settings.Chat.Temperature
		}
		if e.settings.Chat.TopP != nil {
			llmMetadata.TopP = e.settings.Chat.TopP
		}
		if e.settings.Chat.MaxResponseTokens != nil {
			llmMetadata.MaxTokens = e.settings.Chat.MaxResponseTokens
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
		e.publishEvent(events.NewFinalEvent(metadata, stepMetadata, message))

		log.Debug().Msg("OpenAI RunInference completed (streaming)")
		return finalMessage, nil
	} else {
		log.Debug().Msg("OpenAI using non-streaming mode")
		resp, err := client.CreateChatCompletion(ctx, *req)
		if err != nil {
			log.Error().Err(err).Msg("OpenAI non-streaming request failed")
			e.publishEvent(events.NewErrorEvent(metadata, stepMetadata, err))
			return nil, err
		}

		llmMetadata := &conversation.LLMMessageMetadata{
			Engine: string(*e.settings.Chat.Engine),
		}
		if e.settings.Chat.Temperature != nil {
			llmMetadata.Temperature = e.settings.Chat.Temperature
		}
		if e.settings.Chat.TopP != nil {
			llmMetadata.TopP = e.settings.Chat.TopP
		}
		if e.settings.Chat.MaxResponseTokens != nil {
			llmMetadata.MaxTokens = e.settings.Chat.MaxResponseTokens
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
		e.publishEvent(events.NewFinalEvent(metadata, stepMetadata, resp.Choices[0].Message.Content))

		log.Debug().Msg("OpenAI RunInference completed (non-streaming)")
		return finalMessage, nil
	}
}

// publishEvent publishes an event to all configured sinks.
func (e *OpenAIEngine) publishEvent(event events.Event) {
	for _, sink := range e.config.EventSinks {
		if err := sink.PublishEvent(event); err != nil {
			log.Warn().Err(err).Str("event_type", string(event.Type())).Msg("Failed to publish event to sink")
		}
	}
}

var _ engine.Engine = (*OpenAIEngine)(nil)

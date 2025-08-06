package openai

import (
	"context"
	"github.com/go-go-golems/geppetto/pkg/inference/engine"
	"github.com/go-go-golems/geppetto/pkg/inference/tools"
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
	settings    *settings.StepSettings
	config      *engine.Config
	toolAdapter *tools.OpenAIToolAdapter
}

// NewOpenAIEngine creates a new OpenAI inference engine with the given settings and options.
func NewOpenAIEngine(settings *settings.StepSettings, options ...engine.Option) (*OpenAIEngine, error) {
	config := engine.NewConfig()
	if err := engine.ApplyOptions(config, options...); err != nil {
		return nil, err
	}

	return &OpenAIEngine{
		settings:    settings,
		config:      config,
		toolAdapter: tools.NewOpenAIToolAdapter(),
	}, nil
}

// RunInference processes a conversation using OpenAI API and returns the full updated conversation.
// This implementation is extracted from the existing OpenAI ChatStep RunInference method.
func (e *OpenAIEngine) RunInference(
	ctx context.Context,
	messages conversation.Conversation,
) (conversation.Conversation, error) {
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

		// Clone the input conversation and append the new message
		result := append(conversation.Conversation(nil), messages...)
		result = append(result, finalMessage)

		// Publish final event for streaming
		log.Debug().Str("event_id", metadata.ID.String()).Int("final_length", len(message)).Msg("OpenAI publishing final event (streaming)")
		e.publishEvent(events.NewFinalEvent(metadata, stepMetadata, message))

		log.Debug().Msg("OpenAI RunInference completed (streaming)")
		return result, nil
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

		// Clone the input conversation and append the new message
		result := append(conversation.Conversation(nil), messages...)
		result = append(result, finalMessage)

		// Publish final event for non-streaming
		log.Debug().Str("event_id", metadata.ID.String()).Msg("OpenAI publishing final event (non-streaming)")
		e.publishEvent(events.NewFinalEvent(metadata, stepMetadata, resp.Choices[0].Message.Content))

		log.Debug().Msg("OpenAI RunInference completed (non-streaming)")
		return result, nil
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

// GetSupportedToolFeatures returns the tool features supported by OpenAI
func (e *OpenAIEngine) GetSupportedToolFeatures() engine.ToolFeatures {
	limits := e.toolAdapter.GetProviderLimits()
	return engine.ToolFeatures{
		SupportsParallelCalls: true,
		SupportsToolChoice:    true,
		SupportsSystemTools:   false,
		SupportsStreaming:     true,
		Limits: engine.ProviderLimits{
			MaxToolsPerRequest:      limits.MaxToolsPerRequest,
			MaxToolNameLength:       limits.MaxToolNameLength,
			MaxTotalSizeBytes:       limits.MaxTotalSizeBytes,
			SupportedParameterTypes: limits.SupportedParameterTypes,
		},
		SupportedChoiceTypes: []engine.ToolChoice{
			engine.ToolChoiceAuto,
			engine.ToolChoiceNone,
			engine.ToolChoiceRequired,
		},
	}
}

// PrepareToolsForRequest converts tools to OpenAI-specific format
func (e *OpenAIEngine) PrepareToolsForRequest(toolDefs []engine.ToolDefinition, config engine.ToolConfig) (interface{}, error) {
	if !config.Enabled {
		return nil, nil
	}

	// Convert our ToolDefinition to tools.ToolDefinition
	var convertedTools []tools.ToolDefinition
	for _, td := range toolDefs {
		converted := tools.ToolDefinition{
			Name:        td.Name,
			Description: td.Description,
			Parameters:  td.Parameters,
			Function:    tools.ToolFunc{}, // Function not needed for preparation
			Examples:    convertToolExamples(td.Examples),
			Tags:        td.Tags,
			Version:     td.Version,
		}
		convertedTools = append(convertedTools, converted)
	}

	// Filter tools based on config
	toolConfig := tools.ToolConfig{
		Enabled:             config.Enabled,
		ToolChoice:          tools.ToolChoice(config.ToolChoice),
		MaxIterations:       config.MaxIterations,
		ExecutionTimeout:    config.ExecutionTimeout,
		MaxParallelTools:    config.MaxParallelTools,
		AllowedTools:        config.AllowedTools,
		ToolErrorHandling:   tools.ToolErrorHandling(config.ToolErrorHandling),
		RetryConfig:         tools.RetryConfig(config.RetryConfig),
	}
	filteredTools := toolConfig.FilterTools(convertedTools)

	// Convert to OpenAI format
	var openaiTools []interface{}
	for _, tool := range filteredTools {
		converted, err := e.toolAdapter.ConvertToProviderFormat(tool)
		if err != nil {
			return nil, err
		}
		openaiTools = append(openaiTools, converted)
	}

	return openaiTools, nil
}

// RunInferenceStream implements streaming inference for OpenAI with tool support
func (e *OpenAIEngine) RunInferenceStream(ctx context.Context, messages conversation.Conversation, chunkHandler engine.StreamChunkHandler) error {
	// For now, delegate to the existing streaming logic in RunInference
	// A full implementation would properly handle streaming tool calls
	
	result, err := e.RunInference(ctx, messages)
	if err != nil {
		return err
	}

	// Extract the last message and send as chunks
	if len(result) > len(messages) {
		lastMessage := result[len(result)-1]
		
		if chatContent, ok := lastMessage.Content.(*conversation.ChatMessageContent); ok {
			chunk := engine.StreamChunk{
				Type:       engine.ChunkTypeContent,
				Content:    chatContent.Text,
				IsComplete: true,
			}
			if err := chunkHandler(chunk); err != nil {
				return err
			}
		}
		
		// Handle tool use content as well
		if toolUse, ok := lastMessage.Content.(*conversation.ToolUseContent); ok {
			chunk := engine.StreamChunk{
				Type: engine.ChunkTypeToolCall,
				ToolCall: &engine.PartialToolCall{
					ID:        toolUse.ToolID,
					Name:      toolUse.Name,
					Arguments: string(toolUse.Input),
				},
				IsComplete: true,
			}
			if err := chunkHandler(chunk); err != nil {
				return err
			}
		}
	}

	// Send completion chunk
	return chunkHandler(engine.StreamChunk{
		Type:       engine.ChunkTypeComplete,
		IsComplete: true,
	})
}

// Helper function to convert tool examples
func convertToolExamples(examples []engine.ToolExample) []tools.ToolExample {
	var converted []tools.ToolExample
	for _, ex := range examples {
		converted = append(converted, tools.ToolExample{
			Input:       ex.Input,
			Output:      ex.Output,
			Description: ex.Description,
		})
	}
	return converted
}

var _ engine.Engine = (*OpenAIEngine)(nil)
var _ engine.EngineWithTools = (*OpenAIEngine)(nil)
var _ engine.StreamingEngine = (*OpenAIEngine)(nil)

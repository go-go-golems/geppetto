package inference

import (
	"context"

	"github.com/go-go-golems/geppetto/pkg/conversation"
	"github.com/go-go-golems/geppetto/pkg/events"
	"github.com/go-go-golems/geppetto/pkg/steps"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/claude"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/claude/api"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/go-go-golems/glazed/pkg/helpers/cast"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

// ClaudeEngine implements the Engine interface for Claude (Anthropic) API calls.
// It wraps the existing Claude logic from geppetto's ChatStep implementation.
type ClaudeEngine struct {
	settings *settings.StepSettings
	config   *Config
	tools    []api.Tool
}

// NewClaudeEngine creates a new Claude inference engine with the given settings and options.
func NewClaudeEngine(settings *settings.StepSettings, tools []api.Tool, options ...Option) (*ClaudeEngine, error) {
	config := NewConfig()
	if err := ApplyOptions(config, options...); err != nil {
		return nil, err
	}

	return &ClaudeEngine{
		settings: settings,
		config:   config,
		tools:    tools,
	}, nil
}

// RunInference processes a conversation using Claude API and returns the generated message.
// This implementation is extracted from the existing Claude ChatStep RunInference method.
func (e *ClaudeEngine) RunInference(
	ctx context.Context,
	messages conversation.Conversation,
) (*conversation.Message, error) {
	log.Debug().Int("num_messages", len(messages)).Bool("stream", e.settings.Chat.Stream).Msg("Claude RunInference started")
	clientSettings := e.settings.Client
	if clientSettings == nil {
		return nil, steps.ErrMissingClientSettings
	}
	anthropicSettings := e.settings.Claude
	if anthropicSettings == nil {
		return nil, errors.New("no claude settings")
	}

	apiType_ := e.settings.Chat.ApiType
	if apiType_ == nil {
		return nil, errors.New("no chat engine specified")
	}
	apiType := *apiType_
	apiSettings := e.settings.API

	apiKey, ok := apiSettings.APIKeys[string(apiType)+"-api-key"]
	if !ok {
		return nil, errors.Errorf("no API key for %s", apiType)
	}
	baseURL, ok := apiSettings.BaseUrls[string(apiType)+"-base-url"]
	if !ok {
		return nil, errors.Errorf("no base URL for %s", apiType)
	}

	client := api.NewClient(apiKey, baseURL)

	req, err := claude.MakeMessageRequest(e.settings, messages)
	if err != nil {
		return nil, err
	}

	req.Tools = e.tools
	// Safely handle Temperature and TopP settings with default fallback
	if req.Temperature == nil {
		defaultTemp := float64(1.0)
		req.Temperature = &defaultTemp
	}
	if req.TopP == nil {
		defaultTopP := float64(1.0)
		req.TopP = &defaultTopP
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
			Engine:      req.Model,
			Usage:       nil,
			StopReason:  nil,
			Temperature: req.Temperature,
			TopP:        req.TopP,
			MaxTokens:   cast.WrapAddr[int](req.MaxTokens),
		},
	}
	stepMetadata := &steps.StepMetadata{
		StepID:     uuid.New(),
		Type:       "claude-chat",
		InputType:  "conversation.Conversation",
		OutputType: "string",
		Metadata: map[string]interface{}{
			steps.MetadataSettingsSlug: e.settings.GetMetadata(),
		},
	}

	if !e.settings.Chat.Stream {
		log.Debug().Msg("Claude using non-streaming mode")
		response, err := client.SendMessage(ctx, req)
		if err != nil {
			log.Error().Err(err).Msg("Claude non-streaming request failed")
			e.publishEvent(events.NewErrorEvent(metadata, stepMetadata, err))
			return nil, err
		}

		// Update metadata with response data
		metadata.Usage = &conversation.Usage{
			InputTokens:  response.Usage.InputTokens,
			OutputTokens: response.Usage.OutputTokens,
		}
		if response.StopReason != "" {
			metadata.StopReason = &response.StopReason
		}

		llmMessageMetadata := &conversation.LLMMessageMetadata{
			Engine:    req.Model,
			MaxTokens: cast.WrapAddr[int](req.MaxTokens),
			Usage: &conversation.Usage{
				InputTokens:  response.Usage.InputTokens,
				OutputTokens: response.Usage.OutputTokens,
			},
		}
		if response.StopReason != "" {
			llmMessageMetadata.StopReason = &response.StopReason
		}

		message := conversation.NewChatMessage(
			conversation.RoleAssistant, response.FullText(),
			conversation.WithLLMMessageMetadata(llmMessageMetadata),
		)

		// Publish final event
		log.Debug().Str("event_id", metadata.ID.String()).Msg("Claude publishing final event (non-streaming)")
		e.publishEvent(events.NewFinalEvent(metadata, stepMetadata, response.FullText()))

		log.Debug().Msg("Claude RunInference completed (non-streaming)")
		return message, nil
	}

	// For streaming, we need to collect all events and return the final message
	log.Debug().Msg("Claude starting streaming mode")
	eventCh, err := client.StreamMessage(ctx, req)
	if err != nil {
		log.Error().Err(err).Msg("Claude streaming request failed")
		e.publishEvent(events.NewErrorEvent(metadata, stepMetadata, err))
		return nil, err
	}

	log.Debug().Msg("Claude creating ContentBlockMerger")
	completionMerger := claude.NewContentBlockMerger(metadata, stepMetadata)

	log.Debug().Msg("Claude starting streaming event loop")
	eventCount := 0
	for {
		select {
		case <-ctx.Done():
			log.Debug().Msg("Claude streaming cancelled by context")
			// Publish interrupt event with current partial text
			e.publishEvent(events.NewInterruptEvent(metadata, stepMetadata, completionMerger.Text()))
			return nil, ctx.Err()

		case event, ok := <-eventCh:
			if !ok {
				log.Debug().Int("total_events", eventCount).Msg("Claude streaming channel closed, loop completed")
				goto streamingComplete
			}

			eventCount++
			log.Debug().Int("event_count", eventCount).Interface("event", event).Msg("Claude processing streaming event")

			events_, err := completionMerger.Add(event)
			if err != nil {
				log.Error().Err(err).Int("event_count", eventCount).Msg("Claude ContentBlockMerger.Add failed")
				e.publishEvent(events.NewErrorEvent(metadata, stepMetadata, err))
				return nil, err
			}
			// Publish intermediate events generated by the ContentBlockMerger
			log.Debug().Int("num_events", len(events_)).Msg("Claude publishing intermediate events")
			for _, event_ := range events_ {
				e.publishEvent(event_)
			}
		}
	}

streamingComplete:

	log.Debug().Msg("Claude getting final response from ContentBlockMerger")
	response := completionMerger.Response()
	if response == nil {
		err := errors.New("no response")
		log.Error().Err(err).Msg("Claude ContentBlockMerger returned nil response")
		e.publishEvent(events.NewErrorEvent(metadata, stepMetadata, err))
		return nil, err
	}

	log.Debug().Str("response_text", response.FullText()).Msg("Claude creating final message")
	// Update metadata with final response data
	metadata.Usage = &conversation.Usage{
		InputTokens:  response.Usage.InputTokens,
		OutputTokens: response.Usage.OutputTokens,
	}
	if response.StopReason != "" {
		metadata.StopReason = &response.StopReason
	}

	msg := conversation.NewChatMessage(
		conversation.RoleAssistant, response.FullText(),
		conversation.WithLLMMessageMetadata(&conversation.LLMMessageMetadata{
			Engine: req.Model,
			Usage: &conversation.Usage{
				InputTokens:  response.Usage.InputTokens,
				OutputTokens: response.Usage.OutputTokens,
			},
			StopReason:  &response.StopReason,
			Temperature: req.Temperature,
			TopP:        req.TopP,
			MaxTokens:   &req.MaxTokens,
		}),
	)

	// NOTE: Final event is already published by ContentBlockMerger during event processing
	// Do not publish duplicate final event here
	log.Debug().Msg("Claude RunInference completed (streaming)")
	return msg, nil
}

// publishEvent publishes an event to all configured sinks.
func (e *ClaudeEngine) publishEvent(event events.Event) {
	for _, sink := range e.config.EventSinks {
		if err := sink.PublishEvent(event); err != nil {
			log.Warn().Err(err).Str("event_type", string(event.Type())).Msg("Failed to publish event to sink")
		}
	}
}

var _ Engine = (*ClaudeEngine)(nil)

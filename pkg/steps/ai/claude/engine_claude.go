package claude

import (
	"context"
	"github.com/go-go-golems/geppetto/pkg/conversation"
	"github.com/go-go-golems/geppetto/pkg/events"
	"github.com/go-go-golems/geppetto/pkg/inference/engine"
	"github.com/go-go-golems/geppetto/pkg/inference/tools"
	"github.com/go-go-golems/geppetto/pkg/steps"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/claude/api"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/go-go-golems/glazed/pkg/helpers/cast"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

// ClaudeEngine implements the Engine interface for Claude (Anthropic) API calls.
// It wraps the existing Claude logic from geppetto's ChatStep implementation.
type ClaudeEngine struct {
	settings     *settings.StepSettings
	config       *engine.Config
	toolAdapter  *tools.ClaudeToolAdapter
	toolsEnabled bool
	tools        []engine.ToolDefinition
	toolConfig   engine.ToolConfig
}

// NewClaudeEngine creates a new Claude inference engine with the given settings and options.
func NewClaudeEngine(settings *settings.StepSettings, options ...engine.Option) (*ClaudeEngine, error) {
	config := engine.NewConfig()
	if err := engine.ApplyOptions(config, options...); err != nil {
		return nil, err
	}

	return &ClaudeEngine{
		settings:     settings,
		config:       config,
		toolAdapter:  tools.NewClaudeToolAdapter(),
		toolsEnabled: false,
		tools:        nil,
		toolConfig:   engine.ToolConfig{},
	}, nil
}

// ConfigureTools configures the engine to use tools
func (e *ClaudeEngine) ConfigureTools(tools []engine.ToolDefinition, config engine.ToolConfig) {
	e.toolsEnabled = config.Enabled
	e.tools = tools
	e.toolConfig = config
	log.Debug().
		Bool("enabled", e.toolsEnabled).
		Int("tool_count", len(e.tools)).
		Str("tool_choice", string(config.ToolChoice)).
		Msg("Claude engine tools configured")
}

// RunInference processes a conversation using Claude API and returns the full updated conversation.
// This implementation is extracted from the existing Claude ChatStep RunInference method.
func (e *ClaudeEngine) RunInference(
	ctx context.Context,
	messages conversation.Conversation,
) (conversation.Conversation, error) {
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

	req, err := MakeMessageRequest(e.settings, messages)
	if err != nil {
		return nil, err
	}

	// Add tools to the request if enabled
	if e.toolsEnabled && len(e.tools) > 0 {
		log.Debug().Int("tool_count", len(e.tools)).Msg("Adding tools to Claude request")

		// Convert our tools to api.Tool format
		var claudeTools []api.Tool
		for _, tool := range e.tools {
			claudeTool := api.Tool{
				Name:        tool.Name,
				Description: tool.Description,
				InputSchema: tool.Parameters,
			}
			claudeTools = append(claudeTools, claudeTool)
			log.Debug().
				Str("tool_name", claudeTool.Name).
				Str("tool_description", claudeTool.Description).
				Interface("tool_input_schema", claudeTool.InputSchema).
				Msg("Converted tool to Claude format")
		}

		// Set tools in request
		req.Tools = claudeTools

		log.Debug().
			Int("claude_tool_count", len(claudeTools)).
			Interface("claude_tools", claudeTools).
			Msg("Tools added to Claude request")
	}
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
	stepMetadata := &events.StepMetadata{
		StepID:     conversation.NewNodeID(),
		Type:       "claude-chat",
		InputType:  "conversation.Conversation",
		OutputType: "string",
		Metadata: map[string]interface{}{
			events.MetadataSettingsSlug: e.settings.GetMetadata(),
		},
	}

	if !e.settings.Chat.Stream {
		log.Debug().Msg("Claude using non-streaming mode")
        response, err := client.SendMessage(ctx, req)
		if err != nil {
			log.Error().Err(err).Msg("Claude non-streaming request failed")
            e.publishEvent(ctx, events.NewErrorEvent(metadata, stepMetadata, err))
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

		// Clone the input conversation
		result := append(conversation.Conversation(nil), messages...)

		// Check if there's text content - if so, create one message with the original content
		// If not, create separate messages for each content block
		textContent := ""
		hasText := false
		for _, content := range response.Content {
			if tc, ok := content.(api.TextContent); ok {
				textContent += tc.Text
				hasText = true
			}
		}

		if hasText {
			// Create one message with original Claude content (text + tool_use combined)
			textMsg := conversation.NewChatMessage(
				conversation.RoleAssistant, textContent,
				conversation.WithLLMMessageMetadata(llmMessageMetadata),
			)
			// Store the original Claude response for proper reconstruction
			if textMsg.Metadata == nil {
				textMsg.Metadata = make(map[string]interface{})
			}
			textMsg.Metadata["claude_original_content"] = response.Content
			result = append(result, textMsg)
		} else {
			// No text content, create separate ToolUseContent messages
			for _, content := range response.Content {
				if toolUseContent, ok := content.(api.ToolUseContent); ok {
					toolUseMsg := conversation.NewMessage(
						&conversation.ToolUseContent{
							ToolID: toolUseContent.ID,
							Name:   toolUseContent.Name,
							Input:  toolUseContent.Input,
							Type:   "function",
						},
						conversation.WithLLMMessageMetadata(llmMessageMetadata),
					)
					result = append(result, toolUseMsg)
				}
			}
		}

        // Publish final event
		log.Debug().Str("event_id", metadata.ID.String()).Msg("Claude publishing final event (non-streaming)")
        e.publishEvent(ctx, events.NewFinalEvent(metadata, stepMetadata, response.FullText()))

		log.Debug().Msg("Claude RunInference completed (non-streaming)")
		return result, nil
	}

	// For streaming, we need to collect all events and return the final message
	log.Debug().Msg("Claude starting streaming mode")
    eventCh, err := client.StreamMessage(ctx, req)
	if err != nil {
		log.Error().Err(err).Msg("Claude streaming request failed")
        e.publishEvent(ctx, events.NewErrorEvent(metadata, stepMetadata, err))
		return nil, err
	}

	log.Debug().Msg("Claude creating ContentBlockMerger")
	completionMerger := NewContentBlockMerger(metadata, stepMetadata)

	log.Debug().Msg("Claude starting streaming event loop")
	eventCount := 0
	for {
		select {
		case <-ctx.Done():
			log.Debug().Msg("Claude streaming cancelled by context")
            // Publish interrupt event with current partial text
            e.publishEvent(ctx, events.NewInterruptEvent(metadata, stepMetadata, completionMerger.Text()))
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
                e.publishEvent(ctx, events.NewErrorEvent(metadata, stepMetadata, err))
				return nil, err
			}
			// Publish intermediate events generated by the ContentBlockMerger
			log.Debug().Int("num_events", len(events_)).Msg("Claude publishing intermediate events")
			for _, event_ := range events_ {
                e.publishEvent(ctx, event_)
			}
		}
	}

streamingComplete:

	log.Debug().Msg("Claude getting final response from ContentBlockMerger")
    response := completionMerger.Response()
	if response == nil {
		err := errors.New("no response")
		log.Error().Err(err).Msg("Claude ContentBlockMerger returned nil response")
        e.publishEvent(ctx, events.NewErrorEvent(metadata, stepMetadata, err))
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

	// Create separate messages for each content block so tool extraction can work
	llmMetadata := &conversation.LLMMessageMetadata{
		Engine: req.Model,
		Usage: &conversation.Usage{
			InputTokens:  response.Usage.InputTokens,
			OutputTokens: response.Usage.OutputTokens,
		},
		StopReason:  &response.StopReason,
		Temperature: req.Temperature,
		TopP:        req.TopP,
		MaxTokens:   &req.MaxTokens,
	}

	// Clone the input conversation
	result := append(conversation.Conversation(nil), messages...)

	// Check if there's text content - if so, create one message with the original content
	// If not, create separate messages for each content block
	textContent := ""
	hasText := false
	for _, content := range response.Content {
		if tc, ok := content.(api.TextContent); ok {
			textContent += tc.Text
			hasText = true
		}
	}

	if hasText {
		// Create one message with original Claude content (text + tool_use combined)
		textMsg := conversation.NewChatMessage(
			conversation.RoleAssistant, textContent,
			conversation.WithLLMMessageMetadata(llmMetadata),
		)
		// Store the original Claude response for proper reconstruction
		if textMsg.Metadata == nil {
			textMsg.Metadata = make(map[string]interface{})
		}
		textMsg.Metadata["claude_original_content"] = response.Content
		result = append(result, textMsg)
	} else {
		// No text content, create separate ToolUseContent messages
		for _, content := range response.Content {
			if toolUseContent, ok := content.(api.ToolUseContent); ok {
				toolUseMsg := conversation.NewMessage(
					&conversation.ToolUseContent{
						ToolID: toolUseContent.ID,
						Name:   toolUseContent.Name,
						Input:  toolUseContent.Input,
						Type:   "function",
					},
					conversation.WithLLMMessageMetadata(llmMetadata),
				)
				result = append(result, toolUseMsg)
			}
		}
	}

	// NOTE: Final event is already published by ContentBlockMerger during event processing
	// Do not publish duplicate final event here
	log.Debug().Msg("Claude RunInference completed (streaming)")
	return result, nil
}

// publishEvent publishes an event to all configured sinks and any sinks carried in context.
func (e *ClaudeEngine) publishEvent(ctx context.Context, event events.Event) {
    for _, sink := range e.config.EventSinks {
        if err := sink.PublishEvent(event); err != nil {
            log.Warn().Err(err).Str("event_type", string(event.Type())).Msg("Failed to publish event to sink")
        }
    }
    // Best-effort publish to context sinks
    events.PublishEventToContext(ctx, event)
}

// GetSupportedToolFeatures returns the tool features supported by Claude
func (e *ClaudeEngine) GetSupportedToolFeatures() engine.ToolFeatures {
	limits := e.toolAdapter.GetProviderLimits()
	return engine.ToolFeatures{
		SupportsParallelCalls: false, // Claude currently doesn't support parallel tool calls
		SupportsToolChoice:    false, // Claude doesn't have explicit tool choice like OpenAI
		SupportsSystemTools:   false,
		SupportsStreaming:     true,
		Limits: engine.ProviderLimits{
			MaxToolsPerRequest:      limits.MaxToolsPerRequest,
			MaxToolNameLength:       limits.MaxToolNameLength,
			MaxTotalSizeBytes:       limits.MaxTotalSizeBytes,
			SupportedParameterTypes: limits.SupportedParameterTypes,
		},
		SupportedChoiceTypes: []engine.ToolChoice{
			engine.ToolChoiceAuto, // Claude automatically decides when to use tools
		},
	}
}

// PrepareToolsForRequest converts tools to Claude-specific format
func (e *ClaudeEngine) PrepareToolsForRequest(toolDefs []engine.ToolDefinition, config engine.ToolConfig) (interface{}, error) {
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
		Enabled:           config.Enabled,
		ToolChoice:        tools.ToolChoice(config.ToolChoice),
		MaxIterations:     config.MaxIterations,
		ExecutionTimeout:  config.ExecutionTimeout,
		MaxParallelTools:  config.MaxParallelTools,
		AllowedTools:      config.AllowedTools,
		ToolErrorHandling: tools.ToolErrorHandling(config.ToolErrorHandling),
		RetryConfig:       tools.RetryConfig(config.RetryConfig),
	}
	filteredTools := toolConfig.FilterTools(convertedTools)

	// Convert to Claude format
	var claudeTools []interface{}
	for _, tool := range filteredTools {
		converted, err := e.toolAdapter.ConvertToProviderFormat(tool)
		if err != nil {
			return nil, err
		}
		claudeTools = append(claudeTools, converted)
	}

	return claudeTools, nil
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

var _ engine.Engine = (*ClaudeEngine)(nil)

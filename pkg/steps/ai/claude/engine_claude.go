package claude

import (
	"context"
	"encoding/json"
	"github.com/go-go-golems/geppetto/pkg/conversation"
	"github.com/go-go-golems/geppetto/pkg/events"
	"github.com/go-go-golems/geppetto/pkg/inference/engine"
	"github.com/go-go-golems/geppetto/pkg/inference/tools"
	"github.com/go-go-golems/geppetto/pkg/steps"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/claude/api"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/go-go-golems/geppetto/pkg/turns"
	"github.com/go-go-golems/glazed/pkg/helpers/cast"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

// ClaudeEngine implements the Engine interface for Claude (Anthropic) API calls.
// It wraps the existing Claude logic from geppetto's ChatStep implementation.
type ClaudeEngine struct {
	settings    *settings.StepSettings
	config      *engine.Config
	toolAdapter *tools.ClaudeToolAdapter
}

// NewClaudeEngine creates a new Claude inference engine with the given settings and options.
func NewClaudeEngine(settings *settings.StepSettings, options ...engine.Option) (*ClaudeEngine, error) {
	config := engine.NewConfig()
	if err := engine.ApplyOptions(config, options...); err != nil {
		return nil, err
	}

	return &ClaudeEngine{
		settings:    settings,
		config:      config,
		toolAdapter: tools.NewClaudeToolAdapter(),
	}, nil
}

// Tool configuration now read from Turn.Data; no ConfigureTools method

// RunInference processes a conversation using Claude API and returns the full updated conversation.
// This implementation is extracted from the existing Claude ChatStep RunInference method.
func (e *ClaudeEngine) RunInference(
	ctx context.Context,
	t *turns.Turn,
) (*turns.Turn, error) {
	// Build request messages directly from Turn blocks (no conversation dependency)
	log.Debug().Int("num_blocks", len(t.Blocks)).Bool("stream", e.settings.Chat.Stream).Msg("Claude RunInference started")
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

	req, err := MakeMessageRequestFromTurn(e.settings, t)
	if err != nil {
		return nil, err
	}

	// Add tools from Turn.Data if present
	if t != nil && t.Data != nil {
		if regAny, ok := t.Data[turns.DataKeyToolRegistry]; ok && regAny != nil {
			if reg, ok := regAny.(tools.ToolRegistry); ok && reg != nil {
				var claudeTools []api.Tool
				for _, tool := range reg.ListTools() {
					claudeTool := api.Tool{
						Name:        tool.Name,
						Description: tool.Description,
						InputSchema: tool.Parameters,
					}
					claudeTools = append(claudeTools, claudeTool)
					log.Trace().
						Str("tool_name", claudeTool.Name).
						Str("tool_description", claudeTool.Description).
						Interface("tool_input_schema", claudeTool.InputSchema).
						Msg("Converted tool to Claude format")
				}
				req.Tools = claudeTools
				log.Debug().
					Int("claude_tool_count", len(claudeTools)).
					Msg("Tools added to Claude request from Turn.Data")
			}
		}
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
	metadata := events.EventMetadata{
		ID: conversation.NewNodeID(),
		LLMMessageMetadata: conversation.LLMMessageMetadata{
			Engine:      req.Model,
			Usage:       nil,
			StopReason:  nil,
			Temperature: req.Temperature,
			TopP:        req.TopP,
			MaxTokens:   cast.WrapAddr[int](req.MaxTokens),
		},
	}
	if t != nil {
		metadata.RunID = t.RunID
		metadata.TurnID = t.ID
	}
	if metadata.Extra == nil {
		metadata.Extra = map[string]interface{}{}
	}
	metadata.Extra[events.MetadataSettingsSlug] = e.settings.GetMetadata()

	// Non-streaming mode removed. We always use streaming.

	// For streaming, we need to collect all events and return the final message
	log.Debug().Msg("Claude starting streaming mode")
	eventCh, err := client.StreamMessage(ctx, req)
	if err != nil {
		log.Error().Err(err).Msg("Claude streaming request failed")
		e.publishEvent(ctx, events.NewErrorEvent(metadata, err))
		return nil, err
	}

	log.Debug().Msg("Claude creating ContentBlockMerger")
	completionMerger := NewContentBlockMerger(metadata)

	log.Debug().Msg("Claude starting streaming event loop")
	eventCount := 0
	for {
		select {
		case <-ctx.Done():
			log.Debug().Msg("Claude streaming cancelled by context")
			// Publish interrupt event with current partial text
			e.publishEvent(ctx, events.NewInterruptEvent(metadata, completionMerger.Text()))
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
				e.publishEvent(ctx, events.NewErrorEvent(metadata, err))
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
		e.publishEvent(ctx, events.NewErrorEvent(metadata, err))
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

	// Create blocks from content blocks: text -> llm_text, tool_use -> tool_call
	for _, c := range response.Content {
		switch v := c.(type) {
		case api.TextContent:
			if s := v.Text; s != "" {
				turns.AppendBlock(t, turns.NewAssistantTextBlock(s))
			}
		case api.ToolUseContent:
			var args any
			_ = json.Unmarshal(v.Input, &args)
			turns.AppendBlock(t, turns.NewToolCallBlock(v.ID, v.Name, args))
		}
	}

	// NOTE: Final event is already published by ContentBlockMerger during event processing
	// Do not publish duplicate final event here
	log.Debug().Msg("Claude RunInference completed (streaming)")
	return t, nil
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

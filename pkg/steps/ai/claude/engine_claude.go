package claude

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/go-go-golems/geppetto/pkg/events"
	"github.com/go-go-golems/geppetto/pkg/inference/engine"
	gepsession "github.com/go-go-golems/geppetto/pkg/inference/session"
	"github.com/go-go-golems/geppetto/pkg/inference/tools"
	geppettoobs "github.com/go-go-golems/geppetto/pkg/observability"
	"github.com/go-go-golems/geppetto/pkg/steps"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/claude/api"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/runtimeattrib"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/go-go-golems/geppetto/pkg/turns"
	"github.com/go-go-golems/geppetto/pkg/turns/toolblocks"
	"github.com/go-go-golems/glazed/pkg/helpers/cast"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

// ClaudeEngine implements the Engine interface for Claude (Anthropic) API calls.
// It wraps the existing Claude logic from geppetto's ChatStep implementation.
type ClaudeEngine struct {
	settings            *settings.InferenceSettings
	toolAdapter         *tools.ClaudeToolAdapter
	observer            geppettoobs.Observer
	observabilityConfig geppettoobs.Config
}

// NewClaudeEngine creates a new Claude inference engine with the given settings and options.
func NewClaudeEngine(settings *settings.InferenceSettings, opts ...EngineOption) (*ClaudeEngine, error) {
	e := &ClaudeEngine{
		settings:            settings,
		toolAdapter:         tools.NewClaudeToolAdapter(),
		observabilityConfig: geppettoobs.DefaultConfig(),
	}
	for _, opt := range opts {
		if opt != nil {
			opt(e)
		}
	}
	e.observabilityConfig = e.observabilityConfig.Normalized()
	return e, nil
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
	httpClient, err := settings.EnsureHTTPClient(clientSettings)
	if err != nil {
		return nil, err
	}
	client.SetHTTPClient(httpClient)

	req, err := e.MakeMessageRequestFromTurn(t)
	if err != nil {
		return nil, err
	}

	// Add tools from context if present (no Turn.Data registry).
	if reg, ok := tools.RegistryFrom(ctx); ok && reg != nil {
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
			Msg("Tools added to Claude request from context")
	}
	// Do not force defaults for Temperature/TopP; omit when at API defaults (1.0)

	// Setup metadata and event publishing
	metadata := events.EventMetadata{
		ID: uuid.New(),
		LLMInferenceData: events.LLMInferenceData{
			Model:       req.Model,
			Usage:       nil,
			StopReason:  nil,
			Temperature: req.Temperature,
			TopP:        req.TopP,
			MaxTokens:   cast.WrapAddr[int](req.MaxTokens),
		},
	}
	log.Debug().
		Str("event_id", metadata.ID.String()).
		Str("model", metadata.Model).
		Interface("temperature", metadata.Temperature).
		Interface("top_p", metadata.TopP).
		Interface("max_tokens", metadata.MaxTokens).
		Msg("LLMInferenceData initialized (Claude)")
	if t != nil {
		if sid, ok, err := turns.KeyTurnMetaSessionID.Get(t.Metadata); err == nil && ok {
			metadata.SessionID = sid
		}
		if iid, ok, err := turns.KeyTurnMetaInferenceID.Get(t.Metadata); err == nil && ok {
			metadata.InferenceID = iid
		}
		metadata.TurnID = t.ID
	}
	if metadata.Extra == nil {
		metadata.Extra = map[string]interface{}{}
	}
	metadata.Extra[events.MetadataSettingsSlug] = e.settings.GetMetadata()
	runtimeattrib.AddRuntimeAttributionToExtra(metadata.Extra, t)

	// Non-streaming mode removed. We always use streaming.

	// For streaming, we need to collect all events and return the final message
	eventCh, err := client.StreamMessage(ctx, req)
	if err != nil {
		log.Error().Err(err).Msg("Claude streaming request failed")
		e.publishEvent(ctx, events.NewErrorEvent(metadata, err))
		return nil, err
	}

	completionMerger := NewContentBlockMerger(metadata)
	if idx, ok := gepsession.ProviderCallIndexFromContext(ctx); ok {
		completionMerger.providerCallIndex = idx
	}

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
			e.observeProviderEvent(ctx, metadata, req.Model, event)

			events_, err := completionMerger.Add(event)
			if err != nil {
				log.Error().Err(err).Int("event_count", eventCount).Msg("Claude ContentBlockMerger.Add failed")
				e.publishEvent(ctx, events.NewErrorEvent(metadata, err))
				return nil, err
			}
			// Keep local metadata in sync even when Add returns no publishable
			// event. Claude message_delta/message_stop can carry stop_reason,
			// usage, and duration metadata while producing no transcript event.
			syncClaudeEventMetadata(&metadata, completionMerger.Metadata())
			// Publish intermediate events generated by the ContentBlockMerger.
			for _, event_ := range events_ {
				syncClaudeEventMetadata(&metadata, event_.Metadata())
				e.publishEvent(ctx, event_)
			}
		}
	}

streamingComplete:

	response := completionMerger.Response()
	if response == nil {
		err := errors.New("no response")
		log.Error().Err(err).Msg("Claude ContentBlockMerger returned nil response")
		e.publishEvent(ctx, events.NewErrorEvent(metadata, err))
		return nil, err
	}

	log.Trace().Str("response_text", response.FullText()).Msg("Claude creating final message")
	// Update metadata with final response data without overwriting streamed
	// message_delta metadata with a zero-value message_start response.
	syncClaudeEventMetadata(&metadata, completionMerger.Metadata())
	if usage := claudeResponseUsageToEventUsage(response.Usage); usage != nil {
		metadata.Usage = usage
	}
	stopReason := ""
	if strings.TrimSpace(response.StopReason) != "" {
		sr := strings.TrimSpace(response.StopReason)
		metadata.StopReason = &sr
	}
	if metadata.StopReason != nil {
		stopReason = *metadata.StopReason
	}
	log.Trace().
		Interface("usage", metadata.Usage).
		Str("stop_reason", stopReason).
		Msg("Claude metadata finalized")

	// Create blocks from content blocks: text -> llm_text, tool_use -> tool_call
	hasToolCalls := false
	for i, c := range response.Content {
		switch v := c.(type) {
		case api.TextContent:
			if s := v.Text; s != "" {
				turns.AppendBlock(t, turns.NewAssistantTextBlock(s))
			}
		case api.ToolUseContent:
			hasToolCalls = true
			var args any
			_ = json.Unmarshal(v.Input, &args)
			corr := completionMerger.contentBlockCorrelation(i, events.SegmentTypeTool)
			corr.ToolCallID = v.ID
			turns.AppendBlock(t, toolblocks.NewToolCallBlockWithCorrelation(v.ID, v.Name, args, corr))
		}
	}

	result := engine.BuildInferenceResultFromEventMetadata(metadata, "claude", hasToolCalls)
	settings.ApplyModelInfoCost(&result, e.settings.ModelInfo)
	if err := engine.PersistInferenceResult(t, result); err != nil {
		log.Warn().Err(err).Msg("Claude: failed to persist canonical inference_result")
	}

	// NOTE: Final event is already published by ContentBlockMerger during event processing
	log.Trace().Msg("Claude RunInference completed (streaming)")
	return t, nil
}

func syncClaudeEventMetadata(dst *events.EventMetadata, src events.EventMetadata) {
	if dst == nil {
		return
	}
	if src.StopReason != nil && strings.TrimSpace(*src.StopReason) != "" {
		sr := strings.TrimSpace(*src.StopReason)
		dst.StopReason = &sr
	}
	if src.Usage != nil {
		dst.Usage = src.Usage
	}
	if src.DurationMs != nil {
		dst.DurationMs = src.DurationMs
	}
	if len(src.Extra) > 0 {
		if dst.Extra == nil {
			dst.Extra = map[string]interface{}{}
		}
		for k, v := range src.Extra {
			dst.Extra[k] = v
		}
	}
}

func claudeResponseUsageToEventUsage(u api.Usage) *events.Usage {
	if u.InputTokens == 0 && u.OutputTokens == 0 && u.CacheCreationInputTokens == 0 && u.CacheReadInputTokens == 0 {
		return nil
	}
	return &events.Usage{
		InputTokens:              u.InputTokens,
		OutputTokens:             u.OutputTokens,
		CacheCreationInputTokens: u.CacheCreationInputTokens,
		CacheReadInputTokens:     u.CacheReadInputTokens,
	}
}

// publishEvent publishes an event to all configured sinks and any sinks carried in context.
func (e *ClaudeEngine) publishEvent(ctx context.Context, event events.Event) {
	e.observePublishStarted(ctx, event)
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

	// Convert to Claude format
	var claudeTools []interface{}
	for _, tool := range convertedTools {
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

package openai

import (
	"context"
	"encoding/json"
	"io"
	stdlog "log"
	"strings"
	"time"

	"github.com/go-go-golems/geppetto/pkg/inference/engine"
	"github.com/go-go-golems/geppetto/pkg/inference/tools"
	"github.com/go-go-golems/geppetto/pkg/turns"

	"github.com/go-go-golems/geppetto/pkg/events"
	"github.com/google/uuid"

	"github.com/go-go-golems/geppetto/pkg/steps/ai/runtimeattrib"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/streamhelpers"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

// OpenAIEngine implements the Engine interface for OpenAI API calls.
// It wraps the existing OpenAI logic from geppetto's ChatStep implementation.
type OpenAIEngine struct {
	settings    *settings.InferenceSettings
	toolAdapter *tools.OpenAIToolAdapter
}

// NewOpenAIEngine creates a new OpenAI inference engine with the given settings and options.
func NewOpenAIEngine(settings *settings.InferenceSettings) (*OpenAIEngine, error) {
	return &OpenAIEngine{
		settings:    settings,
		toolAdapter: tools.NewOpenAIToolAdapter(),
	}, nil
}

// ConfigureTools configures the engine to use tools

// RunInference processes a Turn using OpenAI API and appends result blocks.
func (e *OpenAIEngine) RunInference(
	ctx context.Context,
	t *turns.Turn,
) (*turns.Turn, error) {
	// Build request messages directly from Turn blocks (no conversation dependency)
	log.Debug().Int("num_blocks", len(t.Blocks)).Bool("stream", true).Msg("OpenAI RunInference started")
	startTime := time.Now()
	if e.settings.Chat.ApiType == nil {
		return nil, errors.New("no chat engine specified")
	}

	// Chat engine no longer routes to Responses; factory selects the correct engine

	streamCfg, err := resolveChatStreamConfig(e.settings.API, e.settings.Client, *e.settings.Chat.ApiType)
	if err != nil {
		return nil, err
	}

	req, err := e.MakeCompletionRequestFromTurn(t)
	if err != nil {
		return nil, err
	}
	// RunInference always executes through the streaming path, regardless of the
	// profile's chat.stream default. The SSE decoder below requires an actual
	// streaming response body, so force the request shape here.
	req.Stream = true
	if req.StreamOptions == nil && !strings.Contains(strings.ToLower(req.Model), "mistral") {
		req.StreamOptions = &ChatStreamOptions{IncludeUsage: true}
	}

	// Debug: confirm adjacency constraints before sending
	if req != nil {
		// Check that any assistant message with tool_calls is followed by tool messages
		for i, m := range req.Messages {
			if len(m.ToolCalls) > 0 {
				missing := []string{}
				// Collect tool_call ids in this assistant message
				idset := map[string]bool{}
				for _, tc := range m.ToolCalls {
					if tc.ID != "" {
						idset[tc.ID] = false
					}
				}
				// Look ahead until next non-tool message
				for j := i + 1; j < len(req.Messages); j++ {
					nm := req.Messages[j]
					if nm.Role != "tool" {
						break
					}
					if nm.ToolCallID != "" {
						if _, ok := idset[nm.ToolCallID]; ok {
							idset[nm.ToolCallID] = true
						}
					}
				}
				for id, ok := range idset {
					if !ok {
						missing = append(missing, id)
					}
				}
				if len(missing) > 0 {
					log.Warn().
						Int("assistant_idx", i).
						Strs("missing_tool_result_ids", missing).
						Msg("OpenAI request: assistant tool_calls missing immediate tool results in following messages")
				}
			}
		}
	}

	// Add tools to the request if present in context (no Turn.Data registry).
	engineTools := tools.AdvertisedToolDefinitionsFromContext(ctx)

	var toolCfg engine.ToolConfig
	if t != nil {
		if cfg, ok, err := engine.KeyToolConfig.Get(t.Data); err != nil {
			return nil, errors.Wrap(err, "get tool config")
		} else if ok {
			toolCfg = cfg
		}
	}

	if len(engineTools) > 0 {
		log.Debug().Int("tool_count", len(engineTools)).Msg("Adding tools to OpenAI request")

		// Convert our tools to chat request tool format
		var openaiTools []ChatCompletionTool
		for _, tool := range engineTools {
			openaiTool := ChatCompletionTool{
				Type: chatToolTypeFunction,
				Function: &ChatFunctionDefinition{
					Name:        tool.Name,
					Description: tool.Description,
					Parameters:  tool.Parameters,
				},
			}
			openaiTools = append(openaiTools, openaiTool)
		}

		// Set tools in request
		req.Tools = openaiTools

		// Set tool choice if specified
		switch toolCfg.ToolChoice {
		case engine.ToolChoiceNone:
			req.ToolChoice = "none"
		case engine.ToolChoiceRequired:
			req.ToolChoice = "required"
		case engine.ToolChoiceAuto:
			req.ToolChoice = "auto"
		default:
			req.ToolChoice = "auto"
		}

		// Set parallel tool calls preference
		if toolCfg.MaxParallelTools > 1 {
			req.ParallelToolCalls = boolRef(true)
		} else if toolCfg.MaxParallelTools == 1 {
			req.ParallelToolCalls = boolRef(false)
		}

		log.Debug().
			Int("openai_tool_count", len(openaiTools)).
			Interface("tool_choice", req.ToolChoice).
			Interface("parallel_tool_calls", req.ParallelToolCalls).
			Msg("Tools added to OpenAI request")
	}

	// Setup metadata and event publishing
	metadata := events.EventMetadata{
		ID: uuid.New(),
		LLMInferenceData: events.LLMInferenceData{
			Model:       req.Model,
			Usage:       nil,
			StopReason:  nil,
			Temperature: e.settings.Chat.Temperature,
			TopP:        e.settings.Chat.TopP,
			MaxTokens:   e.settings.Chat.MaxResponseTokens,
		},
	}
	log.Debug().
		Str("event_id", metadata.ID.String()).
		Str("model", metadata.Model).
		Interface("temperature", metadata.Temperature).
		Interface("top_p", metadata.TopP).
		Interface("max_tokens", metadata.MaxTokens).
		Msg("LLMInferenceData initialized")
	// Propagate Turn correlation identifiers when present
	if t != nil {
		if sid, ok, err := turns.KeyTurnMetaSessionID.Get(t.Metadata); err == nil && ok {
			metadata.SessionID = sid
		}
		if iid, ok, err := turns.KeyTurnMetaInferenceID.Get(t.Metadata); err == nil && ok {
			metadata.InferenceID = iid
		}
		metadata.TurnID = t.ID
	}
	// Step metadata removed; settings metadata moved to EventMetadata.Extra
	if metadata.Extra == nil {
		metadata.Extra = map[string]interface{}{}
	}
	metadata.Extra[events.MetadataSettingsSlug] = e.settings.GetMetadata()
	runtimeattrib.AddRuntimeAttributionToExtra(metadata.Extra, t)

	// Publish start event
	log.Debug().Str("event_id", metadata.ID.String()).Msg("OpenAI publishing start event")
	startEvent := events.NewStartEvent(metadata)
	e.publishEvent(ctx, startEvent)

	// Always use streaming mode
	log.Debug().Msg("OpenAI using streaming mode")
	stream, err := openChatCompletionStream(ctx, streamCfg, req)
	if err != nil {
		log.Error().Err(err).Msg("OpenAI streaming request failed")
		// set duration up to error
		d := time.Since(startTime).Milliseconds()
		dm := int64(d)
		metadata.DurationMs = &dm
		e.publishEvent(ctx, events.NewErrorEvent(metadata, err))
		return nil, err
	}
	defer func() {
		if err := stream.Close(); err != nil {
			stdlog.Printf("Failed to close stream: %v", err)
		}
	}()

	message := ""
	var thinkingBuf strings.Builder
	// Collect streamed tool calls so we can preserve them in the conversation
	toolCallMerger := NewToolCallMerger()
	var usageInputTokens, usageOutputTokens, cachedTokens, reasoningTokens int
	var stopReason *string
	thinkingStarted := false

	log.Debug().Msg("OpenAI starting streaming loop")
	chunkCount := 0
	for {
		select {
		case <-ctx.Done():
			log.Debug().Msg("OpenAI streaming cancelled by context")
			// Publish interrupt event with current partial text
			d := time.Since(startTime).Milliseconds()
			dm := int64(d)
			metadata.DurationMs = &dm
			interruptEvent := events.NewInterruptEvent(metadata, message)
			e.publishEvent(ctx, interruptEvent)
			return nil, ctx.Err()

		default:
			response, err := stream.Recv()
			if errors.Is(err, io.EOF) {
				log.Debug().Int("chunks_received", chunkCount).Msg("OpenAI stream completed")
				goto streamingComplete
			}
			if err != nil {
				log.Error().Err(err).Int("chunks_received", chunkCount).Msg("OpenAI stream receive failed")
				d := time.Since(startTime).Milliseconds()
				dm := int64(d)
				metadata.DurationMs = &dm
				errEvent := events.NewErrorEvent(metadata, err)
				e.publishEvent(ctx, errEvent)
				return nil, err
			}
			chunkCount++

			delta := response.DeltaText
			if delta != "" {
				message += delta
			}
			if response.DeltaReasoning != "" {
				if !thinkingStarted {
					thinkingStarted = true
					e.publishEvent(ctx, events.NewInfoEvent(metadata, "thinking-started", nil))
				}
				thinkingBuf.WriteString(streamhelpers.NormalizeReasoningDelta(thinkingBuf.String(), response.DeltaReasoning))
				e.publishEvent(ctx, events.NewReasoningTextDelta(metadata, response.DeltaReasoning))
				e.publishEvent(ctx, events.NewThinkingPartialEvent(metadata, response.DeltaReasoning, thinkingBuf.String()))
			}

			// Tool call deltas
			if len(response.ToolCalls) > 0 {
				toolCallMerger.AddToolCalls(response.ToolCalls)
			}

			// Extract usage and finish reason from normalized stream response
			if response.Usage != nil {
				usageInputTokens = response.Usage.promptTokens
				usageOutputTokens = response.Usage.completionTokens
				cachedTokens = response.Usage.cachedTokens
				reasoningTokens = response.Usage.reasoningTokens
			}
			if response.FinishReason != nil && *response.FinishReason != "" {
				stopReason = response.FinishReason
			}

			// Publish intermediate streaming event only if we have a non-empty delta
			if delta != "" {
				partialEvent := events.NewPartialCompletionEvent(
					metadata,
					delta, message,
				)
				e.publishEvent(ctx, partialEvent)
			}
		}
	}

streamingComplete:

	// Update event metadata with usage information
	if usageInputTokens > 0 || usageOutputTokens > 0 || cachedTokens > 0 {
		if metadata.Usage == nil {
			metadata.Usage = &events.Usage{}
		}
		metadata.Usage.InputTokens = usageInputTokens
		metadata.Usage.OutputTokens = usageOutputTokens
		metadata.Usage.CachedTokens = cachedTokens
	}
	if metadata.Extra == nil {
		metadata.Extra = map[string]any{}
	}
	metadata.Extra["thinking_text"] = thinkingBuf.String()
	metadata.Extra["saying_text"] = message
	if reasoningTokens > 0 {
		metadata.Extra["reasoning_tokens"] = reasoningTokens
	}
	metadata.StopReason = stopReason
	// set duration for successful completion
	d := time.Since(startTime).Milliseconds()
	dm := int64(d)
	metadata.DurationMs = &dm

	log.Debug().
		Int("input_tokens", usageInputTokens).
		Int("output_tokens", usageOutputTokens).
		Str("stop_reason", func() string {
			if stopReason != nil {
				return *stopReason
			}
			return ""
		}()).
		Msg("OpenAI metadata finalized")

	mergedToolCalls := toolCallMerger.GetToolCalls()
	log.Debug().Int("final_text_length", len(message)).Int("tool_call_count", len(mergedToolCalls)).Msg("OpenAI streaming complete, preparing messages")

	if thinkingBuf.Len() > 0 {
		e.publishEvent(ctx, events.NewReasoningTextDone(metadata, thinkingBuf.String()))
		if thinkingStarted {
			e.publishEvent(ctx, events.NewInfoEvent(metadata, "thinking-ended", nil))
		}
	}

	// If we have tool calls, publish ToolCall events now
	if len(mergedToolCalls) > 0 {
		for _, tc := range mergedToolCalls {
			inputStr := tc.Function.Arguments
			toolCallEvent := events.NewToolCallEvent(
				metadata,
				events.ToolCall{ID: tc.ID, Name: tc.Function.Name, Input: inputStr},
			)
			e.publishEvent(ctx, toolCallEvent)
		}
	}

	// Append messages in order that keeps last message as tool-use when present
	if thinkingBuf.Len() > 0 {
		turns.AppendBlock(t, turns.Block{
			ID:   uuid.NewString(),
			Kind: turns.BlockKindReasoning,
			Payload: map[string]any{
				turns.PayloadKeyText: thinkingBuf.String(),
			},
		})
	}
	if len(message) > 0 {
		turns.AppendBlock(t, turns.NewAssistantTextBlock(message))
	}
	for _, tc := range mergedToolCalls {
		var args any
		_ = json.Unmarshal([]byte(tc.Function.Arguments), &args)
		turns.AppendBlock(t, turns.NewToolCallBlock(tc.ID, tc.Function.Name, args))
	}
	result := engine.BuildInferenceResultFromEventMetadata(metadata, "openai", len(mergedToolCalls) > 0)
	if err := engine.PersistInferenceResult(t, result); err != nil {
		log.Warn().Err(err).Msg("OpenAI: failed to persist canonical inference_result")
	}

	// Publish final event for streaming
	log.Debug().
		Str("event_id", metadata.ID.String()).
		Str("model", metadata.Model).
		Interface("temperature", metadata.Temperature).
		Interface("top_p", metadata.TopP).
		Interface("max_tokens", metadata.MaxTokens).
		Interface("usage", metadata.Usage).
		Str("stop_reason", func() string {
			if stopReason != nil {
				return *stopReason
			}
			return ""
		}()).
		Msg("OpenAI publishing final event (streaming)")
	finalEvent := events.NewFinalEvent(metadata, message)
	e.publishEvent(ctx, finalEvent)

	log.Debug().Msg("OpenAI RunInference completed (streaming)")
	return t, nil
}

// publishEvent publishes an event to all configured sinks and any sinks carried in context.
func (e *OpenAIEngine) publishEvent(ctx context.Context, event events.Event) {
	events.PublishEventToContext(ctx, event)
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

	// Convert to OpenAI format
	var openaiTools []interface{}
	for _, tool := range convertedTools {
		converted, err := e.toolAdapter.ConvertToProviderFormat(tool)
		if err != nil {
			return nil, err
		}
		openaiTools = append(openaiTools, converted)
	}

	return openaiTools, nil
}

// NOTE: RunInferenceStream has been removed in the simplified tool calling architecture.
// Streaming is now handled internally by engines when event sinks are configured.
// As noted in the design: "if you don't pass an event sink, then you won't notice it anyway"

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

// var _ engine.EngineWithTools = (*OpenAIEngine)(nil)  // Commented out - simplified approach
// var _ engine.StreamingEngine = (*OpenAIEngine)(nil)  // Commented out - simplified approach

package openai

import (
	"context"
	"encoding/json"
	"io"
	stdlog "log"

	"github.com/go-go-golems/geppetto/pkg/inference/engine"
	"github.com/go-go-golems/geppetto/pkg/inference/tools"
	"github.com/go-go-golems/geppetto/pkg/turns"

	"github.com/go-go-golems/geppetto/pkg/events"
	"github.com/google/uuid"

	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/go-go-golems/glazed/pkg/helpers/cast"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	go_openai "github.com/sashabaranov/go-openai"
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

// ConfigureTools configures the engine to use tools

// RunInference processes a Turn using OpenAI API and appends result blocks.
func (e *OpenAIEngine) RunInference(
	ctx context.Context,
	t *turns.Turn,
) (*turns.Turn, error) {
	// Build request messages directly from Turn blocks (no conversation dependency)
	log.Debug().Int("num_blocks", len(t.Blocks)).Bool("stream", true).Msg("OpenAI RunInference started")
	if e.settings.Chat.ApiType == nil {
		return nil, errors.New("no chat engine specified")
	}

	client, err := MakeClient(e.settings.API, *e.settings.Chat.ApiType)
	if err != nil {
		return nil, err
	}

	req, err := MakeCompletionRequestFromTurn(e.settings, t)
	if err != nil {
		return nil, err
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

	// Add tools to the request if present on the Turn Data
	if t != nil && t.Data != nil {
		var engineTools []engine.ToolDefinition
		if regAny, ok := t.Data[turns.DataKeyToolRegistry]; ok && regAny != nil {
			if reg, ok := regAny.(tools.ToolRegistry); ok && reg != nil {
				for _, td := range reg.ListTools() {
					engineTools = append(engineTools, engine.ToolDefinition{
						Name:        td.Name,
						Description: td.Description,
						Parameters:  td.Parameters,
						Examples:    []engine.ToolExample{},
						Tags:        td.Tags,
						Version:     td.Version,
					})
				}
			}
		}
		var toolCfg engine.ToolConfig
		if cfgAny, ok := t.Data[turns.DataKeyToolConfig]; ok && cfgAny != nil {
			if cfg, ok := cfgAny.(engine.ToolConfig); ok {
				toolCfg = cfg
			}
		}
		if len(engineTools) > 0 {
			log.Debug().Int("tool_count", len(engineTools)).Msg("Adding tools to OpenAI request")

			// Convert our tools to go_openai.Tool format
			var openaiTools []go_openai.Tool
			for _, tool := range engineTools {
				openaiTool := go_openai.Tool{
					Type: go_openai.ToolTypeFunction,
					Function: &go_openai.FunctionDefinition{
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
				req.ParallelToolCalls = true
			} else if toolCfg.MaxParallelTools == 1 {
				req.ParallelToolCalls = false
			}

			log.Debug().
				Int("openai_tool_count", len(openaiTools)).
				Interface("tool_choice", req.ToolChoice).
				Interface("parallel_tool_calls", req.ParallelToolCalls).
				Msg("Tools added to OpenAI request")
		}
	}

	// Setup metadata and event publishing
	metadata := events.EventMetadata{
		ID: uuid.New(),
		LLMInferenceData: events.LLMInferenceData{
			Engine:      req.Model,
			Model:       req.Model,
			Usage:       nil,
			StopReason:  nil,
			Temperature: e.settings.Chat.Temperature,
			TopP:        e.settings.Chat.TopP,
			MaxTokens:   e.settings.Chat.MaxResponseTokens,
		},
	}
	// Propagate Turn correlation identifiers when present
	if t != nil {
		metadata.RunID = t.RunID
		metadata.TurnID = t.ID
	}
	// Step metadata removed; settings metadata moved to EventMetadata.Extra
	if metadata.Extra == nil {
		metadata.Extra = map[string]interface{}{}
	}
	metadata.Extra[events.MetadataSettingsSlug] = e.settings.GetMetadata()

	// Publish start event
	log.Debug().Str("event_id", metadata.ID.String()).Msg("OpenAI publishing start event")
	startEvent := events.NewStartEvent(metadata)
	e.publishEvent(ctx, startEvent)

	// Always use streaming mode
	log.Debug().Msg("OpenAI using streaming mode")
	stream, err := client.CreateChatCompletionStream(ctx, *req)
	if err != nil {
		log.Error().Err(err).Msg("OpenAI streaming request failed")
		e.publishEvent(ctx, events.NewErrorEvent(metadata, err))
		return nil, err
	}
	defer func() {
		if err := stream.Close(); err != nil {
			stdlog.Printf("Failed to close stream: %v", err)
		}
	}()

	message := ""
	// Collect streamed tool calls so we can preserve them in the conversation
	toolCallMerger := NewToolCallMerger()
	var usageInputTokens, usageOutputTokens int
	var stopReason *string

	log.Debug().Msg("OpenAI starting streaming loop")
	chunkCount := 0
	for {
		select {
		case <-ctx.Done():
			log.Debug().Msg("OpenAI streaming cancelled by context")
			// Publish interrupt event with current partial text
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
				errEvent := events.NewErrorEvent(metadata, err)
				e.publishEvent(ctx, errEvent)
				return nil, err
			}
			chunkCount++

			delta := ""
			if len(response.Choices) > 0 {
				choice := response.Choices[0]
				// Text delta
				delta = choice.Delta.Content
				// Only accumulate and publish when there is a non-empty text delta
				if delta != "" {
					message += delta
					log.Debug().Int("chunk", chunkCount).Str("delta", delta).Int("total_length", len(message)).Msg("OpenAI received chunk")
				}

				// Tool call deltas
				if len(choice.Delta.ToolCalls) > 0 {
					toolCallMerger.AddToolCalls(choice.Delta.ToolCalls)
					for _, tc := range choice.Delta.ToolCalls {
						// Safe logging of arguments size to avoid very long logs
						argPreview := tc.Function.Arguments
						if len(argPreview) > 200 {
							argPreview = argPreview[:200] + "…"
						}
						log.Debug().
							Int("chunk", chunkCount).
							Str("tool_id", tc.ID).
							Str("name", tc.Function.Name).
							Str("arguments_delta", argPreview).
							Msg("OpenAI received tool_call delta")
					}
				}
			}

			// Extract metadata from OpenAI chat response
			if responseMetadata, err := ExtractChatCompletionMetadata(&response); err == nil && responseMetadata != nil {
				if usageData, ok := responseMetadata["usage"].(map[string]interface{}); ok {
					usageInputTokens, _ = cast.CastNumberInterfaceToInt[int](usageData["prompt_tokens"])
					usageOutputTokens, _ = cast.CastNumberInterfaceToInt[int](usageData["completion_tokens"])
				}
				if finishReason, ok := responseMetadata["finish_reason"].(string); ok {
					stopReason = &finishReason
				}
			}

			// Publish intermediate streaming event only if we have a non-empty delta
			if delta != "" {
				log.Debug().Int("chunk", chunkCount).Str("delta", delta).Msg("OpenAI publishing partial completion event")
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
	if usageInputTokens > 0 || usageOutputTokens > 0 {
		metadata.Usage = &events.Usage{InputTokens: usageInputTokens, OutputTokens: usageOutputTokens}
	}
	metadata.StopReason = stopReason

	// Provider metadata carried in events only for now

	mergedToolCalls := toolCallMerger.GetToolCalls()
	log.Debug().Int("final_text_length", len(message)).Int("tool_call_count", len(mergedToolCalls)).Msg("OpenAI streaming complete, preparing messages")

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
	// Convert conversation delta back to Turn blocks and append
	// First, reconstruct a temporary conversation with new messages based on streamed results
	// Append assistant text and tool calls as blocks on the Turn
	if len(message) > 0 {
		// simple assistant message
		turns.AppendBlock(t, turns.NewAssistantTextBlock(message))
	}
	// append tool calls
	for _, tc := range mergedToolCalls {
		var args any
		_ = json.Unmarshal([]byte(tc.Function.Arguments), &args)
		turns.AppendBlock(t, turns.NewToolCallBlock(tc.ID, tc.Function.Name, args))
	}

	// Publish final event for streaming
	log.Debug().Str("event_id", metadata.ID.String()).Int("final_length", len(message)).Int("tool_call_count", len(mergedToolCalls)).Msg("OpenAI publishing final event (streaming)")
	finalEvent := events.NewFinalEvent(metadata, message)
	e.publishEvent(ctx, finalEvent)

	log.Debug().Msg("OpenAI RunInference completed (streaming)")
	return t, nil
}

// publishEvent publishes an event to all configured sinks and any sinks carried in context.
func (e *OpenAIEngine) publishEvent(ctx context.Context, event events.Event) {
	for _, sink := range e.config.EventSinks {
		if err := sink.PublishEvent(event); err != nil {
			log.Warn().Err(err).Str("event_type", string(event.Type())).Msg("Failed to publish event to sink")
		}
	}
	// Best-effort publish to context sinks
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

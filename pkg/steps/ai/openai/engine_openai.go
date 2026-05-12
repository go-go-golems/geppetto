package openai

import (
	"context"
	"encoding/json"
	"io"
	stdlog "log"
	"strings"
	"time"

	"github.com/go-go-golems/geppetto/pkg/inference/engine"
	gepsession "github.com/go-go-golems/geppetto/pkg/inference/session"
	"github.com/go-go-golems/geppetto/pkg/inference/tools"
	"github.com/go-go-golems/geppetto/pkg/turns"
	"github.com/go-go-golems/geppetto/pkg/turns/toolblocks"

	"github.com/go-go-golems/geppetto/pkg/events"
	"github.com/google/uuid"

	geppettoobs "github.com/go-go-golems/geppetto/pkg/observability"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/runtimeattrib"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

// OpenAIEngine implements the Engine interface for OpenAI API calls.
// It wraps the existing OpenAI logic from geppetto's ChatStep implementation.
type OpenAIEngine struct {
	settings            *settings.InferenceSettings
	toolAdapter         *tools.OpenAIToolAdapter
	observer            geppettoobs.Observer
	observabilityConfig geppettoobs.Config
}

// NewOpenAIEngine creates a new OpenAI inference engine with the given settings and options.
func NewOpenAIEngine(settings *settings.InferenceSettings, opts ...EngineOption) (*OpenAIEngine, error) {
	e := &OpenAIEngine{
		settings:            settings,
		toolAdapter:         tools.NewOpenAIToolAdapter(),
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

	// Publish provider-call start event.
	log.Debug().Str("event_id", metadata.ID.String()).Msg("OpenAI publishing provider call start event")
	inferenceScopeID := metadata.InferenceID
	if inferenceScopeID == "" {
		inferenceScopeID = metadata.ID.String()
	}
	providerCallIndex := 0
	if idx, ok := gepsession.ProviderCallIndexFromContext(ctx); ok {
		providerCallIndex = idx
	}
	providerCallCorr := events.BuildProviderCallCorrelation(e.inferenceProvider(), inferenceScopeID, "", providerCallIndex, "")
	providerCallCorr.SessionID = metadata.SessionID
	providerCallCorr.TurnID = metadata.TurnID
	e.publishEvent(ctx, events.NewProviderCallStartedEvent(metadata, providerCallCorr))

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

	state := newOpenAIChatStreamState(metadata, e.inferenceProvider(), req.Model, providerCallCorr, providerCallIndex)
	state, terminal, runErr := e.consumeOpenAIChatStream(ctx, stream, state, metadata, req.Model)
	state, metadata = e.completeOpenAIChatStream(ctx, t, state, metadata, req.Model, startTime, terminal)

	log.Debug().
		Str("event_id", metadata.ID.String()).
		Str("model", metadata.Model).
		Interface("temperature", metadata.Temperature).
		Interface("top_p", metadata.TopP).
		Interface("max_tokens", metadata.MaxTokens).
		Interface("usage", metadata.Usage).
		Str("stop_reason", stopReasonString(state.StopReason)).
		Msg("OpenAI publishing final event (streaming)")

	log.Debug().Err(runErr).Msg("OpenAI RunInference completed (streaming)")
	return t, runErr
}

func (e *OpenAIEngine) consumeOpenAIChatStream(
	ctx context.Context,
	stream *chatCompletionStream,
	state openAIChatStreamState,
	metadata events.EventMetadata,
	model string,
) (openAIChatStreamState, openAIChatTerminal, error) {
	log.Debug().Msg("OpenAI starting streaming loop")
	for {
		select {
		case <-ctx.Done():
			log.Debug().Msg("OpenAI streaming cancelled by context")
			return state, openAIChatTerminal{Kind: openAIChatTerminalCancelled, Err: ctx.Err()}, ctx.Err()
		default:
			response, err := stream.Recv()
			if errors.Is(err, io.EOF) {
				log.Debug().Int("chunks_received", state.ChunkCount).Msg("OpenAI stream completed")
				return state, openAIChatTerminal{Kind: openAIChatTerminalEOF}, nil
			}
			if err != nil {
				log.Error().Err(err).Int("chunks_received", state.ChunkCount).Msg("OpenAI stream receive failed")
				return state, openAIChatTerminal{Kind: openAIChatTerminalError, Err: err}, err
			}

			var effects []openAIChatStreamEffect
			state, effects = reduceOpenAIChatStream(state, openAIChatStreamInput{
				Kind:  openAIChatStreamInputChunk,
				Chunk: response,
			})
			e.applyOpenAIChatStreamEffects(ctx, metadata, model, effects)
		}
	}
}

func (e *OpenAIEngine) completeOpenAIChatStream(
	ctx context.Context,
	t *turns.Turn,
	state openAIChatStreamState,
	metadata events.EventMetadata,
	model string,
	startTime time.Time,
	terminal openAIChatTerminal,
) (openAIChatStreamState, events.EventMetadata) {
	state = state.withTerminalStopReason(terminal)
	metadata = finalizeOpenAIChatMetadata(metadata, state, startTime)
	state.Metadata = metadata

	log.Debug().
		Int("input_tokens", state.UsageInputTokens).
		Int("output_tokens", state.UsageOutputTokens).
		Str("stop_reason", stopReasonString(state.StopReason)).
		Msg("OpenAI metadata finalized")

	var effects []openAIChatStreamEffect
	state, effects = reduceOpenAIChatStream(state, openAIChatStreamInput{
		Kind:     openAIChatStreamInputTerminal,
		Terminal: terminal,
	})
	e.applyOpenAIChatStreamEffects(ctx, metadata, model, effects)

	includeToolCalls := terminal.Kind == openAIChatTerminalEOF
	toolCallCount := appendOpenAIChatTurnBlocks(t, state, includeToolCalls)
	log.Debug().
		Int("final_text_length", len(state.Message)).
		Int("tool_call_count", toolCallCount).
		Str("terminal", string(terminal.Kind)).
		Msg("OpenAI streaming complete, preparing messages")

	result := engine.BuildInferenceResultFromEventMetadata(metadata, "openai", includeToolCalls && toolCallCount > 0)
	settings.ApplyModelInfoCost(&result, e.settings.ModelInfo)
	if err := engine.PersistInferenceResult(t, result); err != nil {
		log.Warn().Err(err).Msg("OpenAI: failed to persist canonical inference_result")
	}

	return state, metadata
}

func appendOpenAIChatTurnBlocks(t *turns.Turn, state openAIChatStreamState, includeToolCalls bool) int {
	if state.Reasoning != "" {
		turns.AppendBlock(t, turns.Block{
			ID:   uuid.NewString(),
			Kind: turns.BlockKindReasoning,
			Payload: map[string]any{
				turns.PayloadKeyText: state.Reasoning,
			},
		})
	}
	if state.Message != "" {
		turns.AppendBlock(t, turns.NewAssistantTextBlock(state.Message))
	}
	if !includeToolCalls {
		return 0
	}

	mergedToolCalls := state.mergedToolCalls()
	for _, tc := range mergedToolCalls {
		var args any
		_ = json.Unmarshal([]byte(tc.Function.Arguments), &args)
		corr := state.chatCorrelation(state.CurrentChoiceIndex, events.StreamKindToolCall, tc.ID, tc.Index)
		turns.AppendBlock(t, toolblocks.NewToolCallBlockWithCorrelation(tc.ID, tc.Function.Name, args, corr))
	}
	return len(mergedToolCalls)
}

func (state openAIChatStreamState) withTerminalStopReason(terminal openAIChatTerminal) openAIChatStreamState {
	reason, ok := openAIChatTerminalStopReason(terminal.Kind)
	if !ok {
		return state
	}
	state.StopReason = &reason
	return state
}

func openAIChatTerminalStopReason(kind openAIChatTerminalKind) (string, bool) {
	switch kind {
	case openAIChatTerminalEOF:
		return "", false
	case openAIChatTerminalCancelled:
		return "cancelled", true
	case openAIChatTerminalError:
		return "error", true
	}
	return "", false
}

func stopReasonString(stopReason *string) string {
	if stopReason == nil {
		return ""
	}
	return *stopReason
}

func providerCallCorrWithResponse(corr events.Correlation, _ string) events.Correlation {
	return corr
}

func openAIChatMetadataWithDuration(metadata events.EventMetadata, startTime time.Time) events.EventMetadata {
	d := time.Since(startTime).Milliseconds()
	dm := int64(d)
	metadata.DurationMs = &dm
	return metadata
}

func finalizeOpenAIChatMetadata(metadata events.EventMetadata, state openAIChatStreamState, startTime time.Time) events.EventMetadata {
	if state.UsageInputTokens > 0 || state.UsageOutputTokens > 0 || state.CachedTokens > 0 {
		if metadata.Usage == nil {
			metadata.Usage = &events.Usage{}
		}
		metadata.Usage.InputTokens = state.UsageInputTokens
		metadata.Usage.OutputTokens = state.UsageOutputTokens
		metadata.Usage.CachedTokens = state.CachedTokens
	}
	if metadata.Extra == nil {
		metadata.Extra = map[string]any{}
	}
	metadata.Extra["thinking_text"] = state.Reasoning
	metadata.Extra["saying_text"] = state.Message
	if state.ReasoningTokens > 0 {
		metadata.Extra["reasoning_tokens"] = state.ReasoningTokens
	}
	metadata.StopReason = state.StopReason
	return openAIChatMetadataWithDuration(metadata, startTime)
}

func (e *OpenAIEngine) applyOpenAIChatStreamEffects(ctx context.Context, metadata events.EventMetadata, model string, effects []openAIChatStreamEffect) {
	for _, effect := range effects {
		if effect.ObserveProviderEvent != nil {
			e.observeProviderEvent(ctx, metadata, model, *effect.ObserveProviderEvent)
		}
		if effect.ObserveNormalizedReason != nil {
			observation := effect.ObserveNormalizedReason
			e.observeProviderNormalizeDelta(ctx, metadata, model, observation.Chunk, observation.RawLength, observation.NormalizedLength, observation.TotalLength)
		}
		if effect.Event != nil {
			e.publishEvent(ctx, effect.Event)
		}
	}
}

// publishEvent publishes an event to all configured sinks and any sinks carried in context.
func (e *OpenAIEngine) publishEvent(ctx context.Context, event events.Event) {
	e.observePublishStarted(ctx, event)
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

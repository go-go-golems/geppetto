package openai

import (
	"strconv"
	"strings"

	"github.com/go-go-golems/geppetto/pkg/events"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/streamhelpers"
)

type openAIChatStreamInputKind string

const (
	openAIChatStreamInputChunk    openAIChatStreamInputKind = "chunk"
	openAIChatStreamInputTerminal openAIChatStreamInputKind = "terminal"
)

type openAIChatTerminalKind string

const (
	openAIChatTerminalEOF       openAIChatTerminalKind = "eof"
	openAIChatTerminalCancelled openAIChatTerminalKind = "cancelled"
	openAIChatTerminalError     openAIChatTerminalKind = "error"
)

type openAIChatStreamInput struct {
	Kind     openAIChatStreamInputKind
	Chunk    chatStreamEvent
	Terminal openAIChatTerminal
}

type openAIChatTerminal struct {
	Kind openAIChatTerminalKind
	Err  error
}

type openAIChatStreamEffect struct {
	Event events.Event

	ObserveProviderEvent    *chatStreamEvent
	ObserveNormalizedReason *openAIReasoningNormalizeObservation
}

type openAIReasoningNormalizeObservation struct {
	Chunk            chatStreamEvent
	RawLength        int
	NormalizedLength int
	TotalLength      int
}

type openAIChatStreamState struct {
	Metadata events.EventMetadata

	Provider          string
	Model             string
	TurnID            string
	ProviderCallIndex int32
	ProviderCallCorr  events.Correlation

	CurrentResponseID  string
	CurrentChoiceIndex *int

	Message   string
	Reasoning string

	TextSegmentStarted  bool
	TextSegmentFinished bool

	ReasoningSegmentStarted  bool
	ReasoningSegmentFinished bool

	ToolCallMerger *ToolCallMerger

	StartedToolStreams map[string]bool
	ToolArgBuffers     map[string]string
	ToolArgSequences   map[string]int64
	ToolCallIDTracker  chatToolCallIDTracker

	UsageInputTokens  int
	UsageOutputTokens int
	CachedTokens      int
	ReasoningTokens   int

	StopReason *string

	ChunkCount int
	Done       bool
	Failed     bool
}

func newOpenAIChatStreamState(
	metadata events.EventMetadata,
	provider string,
	model string,
	providerCallCorr events.Correlation,
	providerCallIndex int,
) openAIChatStreamState {
	return openAIChatStreamState{
		Metadata:           metadata,
		Provider:           provider,
		Model:              model,
		TurnID:             metadata.TurnID,
		ProviderCallIndex:  int32(providerCallIndex), // #nosec G115 -- tool-loop indexes are small, local ordinals.
		ProviderCallCorr:   providerCallCorr,
		ToolCallMerger:     NewToolCallMerger(),
		StartedToolStreams: map[string]bool{},
		ToolArgBuffers:     map[string]string{},
		ToolArgSequences:   map[string]int64{},
		ToolCallIDTracker:  chatToolCallIDTracker{},
	}
}

func reduceOpenAIChatStream(
	state openAIChatStreamState,
	input openAIChatStreamInput,
) (openAIChatStreamState, []openAIChatStreamEffect) {
	switch input.Kind {
	case openAIChatStreamInputChunk:
		return reduceOpenAIChatChunk(state, input.Chunk)
	case openAIChatStreamInputTerminal:
		return reduceOpenAIChatTerminal(state, input.Terminal)
	default:
		return state, nil
	}
}

func reduceOpenAIChatChunk(
	state openAIChatStreamState,
	chunk chatStreamEvent,
) (openAIChatStreamState, []openAIChatStreamEffect) {
	state.ChunkCount++

	if id := stringFromRawMap(chunk.RawPayload, "id"); id != "" {
		state.CurrentResponseID = id
	}
	if chunk.ChoiceIndex != nil {
		state.CurrentChoiceIndex = cloneIntPtr(chunk.ChoiceIndex)
	}

	chunk = state.ToolCallIDTracker.Enrich(chunk)
	effects := []openAIChatStreamEffect{{ObserveProviderEvent: &chunk}}

	state, effects = reduceOpenAIChatReasoningDelta(state, chunk, effects)
	state, effects = reduceOpenAIChatToolDeltas(state, chunk, effects)
	state, effects = reduceOpenAIChatUsageAndFinish(state, chunk, effects)
	state, effects = reduceOpenAIChatTextDelta(state, chunk, effects)

	return state, effects
}

func reduceOpenAIChatTextDelta(
	state openAIChatStreamState,
	chunk chatStreamEvent,
	effects []openAIChatStreamEffect,
) (openAIChatStreamState, []openAIChatStreamEffect) {
	if chunk.DeltaText == "" {
		return state, effects
	}

	state.Message += chunk.DeltaText
	corr := state.chatCorrelation(chunk.ChoiceIndex, events.StreamKindContent, "", nil)

	if !state.TextSegmentStarted {
		state.TextSegmentStarted = true
		effects = appendOpenAIChatEvent(effects, events.NewTextSegmentStartedEvent(state.Metadata, corr, "assistant"))
	}

	effects = appendOpenAIChatEvent(effects, events.NewTextDeltaEvent(state.Metadata, corr, chunk.DeltaText, state.Message, 0))
	return state, effects
}

func reduceOpenAIChatReasoningDelta(
	state openAIChatStreamState,
	chunk chatStreamEvent,
	effects []openAIChatStreamEffect,
) (openAIChatStreamState, []openAIChatStreamEffect) {
	if chunk.DeltaReasoning == "" {
		return state, effects
	}

	corr := state.chatCorrelation(chunk.ChoiceIndex, events.StreamKindReasoning, "", nil)
	if !state.ReasoningSegmentStarted {
		state.ReasoningSegmentStarted = true
		effects = appendOpenAIChatEvent(effects, events.NewReasoningSegmentStartedEvent(state.Metadata, corr, "provider"))
	}

	before := len(state.Reasoning)
	normalized := streamhelpers.NormalizeReasoningDelta(state.Reasoning, chunk.DeltaReasoning)
	state.Reasoning += normalized
	effects = append(effects, openAIChatStreamEffect{ObserveNormalizedReason: &openAIReasoningNormalizeObservation{
		Chunk:            chunk,
		RawLength:        len(chunk.DeltaReasoning),
		NormalizedLength: len(normalized),
		TotalLength:      before + len(normalized),
	}})
	effects = appendOpenAIChatEvent(effects, events.NewReasoningDeltaEvent(state.Metadata, corr, chunk.DeltaReasoning, state.Reasoning, 0))
	return state, effects
}

func reduceOpenAIChatToolDeltas(
	state openAIChatStreamState,
	chunk chatStreamEvent,
	effects []openAIChatStreamEffect,
) (openAIChatStreamState, []openAIChatStreamEffect) {
	if len(chunk.ToolCalls) == 0 {
		return state, effects
	}

	for _, tc := range chunk.ToolCalls {
		corr := state.chatCorrelation(chunk.ChoiceIndex, events.StreamKindToolCall, tc.ID, tc.Index)
		key := openAIChatToolStreamKey(corr, tc)
		if !state.StartedToolStreams[key] {
			state.StartedToolStreams[key] = true
			effects = appendOpenAIChatEvent(effects, events.NewToolCallStartedEvent(state.Metadata, corr, corr.ToolCallID, tc.Function.Name))
		}
		if tc.Function.Arguments != "" {
			state.ToolArgBuffers[key] += tc.Function.Arguments
			state.ToolArgSequences[key]++
			effects = appendOpenAIChatEvent(effects, events.NewToolCallArgumentsDeltaEvent(state.Metadata, corr, corr.ToolCallID, tc.Function.Arguments, state.ToolArgBuffers[key], state.ToolArgSequences[key]))
		}
	}

	state.ToolCallMerger.AddToolCalls(chunk.ToolCalls)
	return state, effects
}

func reduceOpenAIChatUsageAndFinish(
	state openAIChatStreamState,
	chunk chatStreamEvent,
	effects []openAIChatStreamEffect,
) (openAIChatStreamState, []openAIChatStreamEffect) {
	if chunk.Usage != nil {
		state.UsageInputTokens = chunk.Usage.promptTokens
		state.UsageOutputTokens = chunk.Usage.completionTokens
		state.CachedTokens = chunk.Usage.cachedTokens
		state.ReasoningTokens = chunk.Usage.reasoningTokens
	}
	if chunk.FinishReason != nil && *chunk.FinishReason != "" {
		state.StopReason = chunk.FinishReason
	}
	if chunk.Usage == nil && stopReasonString(chunk.FinishReason) == "" {
		return state, effects
	}

	var usage *events.Usage
	if chunk.Usage != nil {
		usage = &events.Usage{
			InputTokens:  chunk.Usage.promptTokens,
			OutputTokens: chunk.Usage.completionTokens,
			CachedTokens: chunk.Usage.cachedTokens,
		}
	}

	effects = appendOpenAIChatEvent(effects, events.NewProviderCallMetadataUpdatedEvent(
		state.Metadata,
		providerCallCorrWithResponse(state.ProviderCallCorr, state.CurrentResponseID),
		stopReasonString(chunk.FinishReason),
		"",
		usage,
	))
	return state, effects
}

func reduceOpenAIChatTerminal(
	state openAIChatStreamState,
	terminal openAIChatTerminal,
) (openAIChatStreamState, []openAIChatStreamEffect) {
	state.Done = true
	var effects []openAIChatStreamEffect

	finishReason := stopReasonString(state.StopReason)
	finishClass := "completed"
	hasToolCalls := len(state.mergedToolCalls()) > 0
	emitToolRequests := false

	switch terminal.Kind {
	case openAIChatTerminalEOF:
		emitToolRequests = true
		if hasToolCalls {
			finishClass = "tool_calls_pending"
		}
	case openAIChatTerminalCancelled:
		finishReason = "cancelled"
		finishClass = "cancelled"
	case openAIChatTerminalError:
		finishReason = "error"
		finishClass = "failed"
		state.Failed = true
	}

	state, effects = finishOpenAIChatSegments(state, finishReason, effects)

	if emitToolRequests {
		for _, tc := range state.mergedToolCalls() {
			effects = appendOpenAIChatEvent(effects, events.NewToolCallRequestedEvent(
				state.Metadata,
				state.chatCorrelation(state.CurrentChoiceIndex, events.StreamKindToolCall, tc.ID, tc.Index),
				tc.ID,
				tc.Function.Name,
				tc.Function.Arguments,
			))
		}
	}

	switch terminal.Kind {
	case openAIChatTerminalEOF:
		// Normal completion needs no extra terminal event beyond provider-call finish.
	case openAIChatTerminalCancelled:
		effects = appendOpenAIChatEvent(effects, events.NewInterruptEvent(state.Metadata, state.Message))
	case openAIChatTerminalError:
		effects = appendOpenAIChatEvent(effects, events.NewErrorEvent(state.Metadata, terminal.Err))
	}

	effects = appendOpenAIChatEvent(effects, events.NewProviderCallFinishedEvent(
		state.Metadata,
		providerCallCorrWithResponse(state.ProviderCallCorr, state.CurrentResponseID),
		finishReason,
		finishClass,
		finalOpenAIChatUsage(state),
		state.Metadata.DurationMs,
		emitToolRequests && hasToolCalls,
	))

	return state, effects
}

func finishOpenAIChatSegments(
	state openAIChatStreamState,
	finishReason string,
	effects []openAIChatStreamEffect,
) (openAIChatStreamState, []openAIChatStreamEffect) {
	if state.ReasoningSegmentStarted && !state.ReasoningSegmentFinished {
		state.ReasoningSegmentFinished = true
		effects = appendOpenAIChatEvent(effects, events.NewReasoningSegmentFinishedEvent(
			state.Metadata,
			state.chatCorrelation(state.CurrentChoiceIndex, events.StreamKindReasoning, "", nil),
			state.Reasoning,
			finishReason,
		))
	}

	if state.TextSegmentStarted && !state.TextSegmentFinished {
		state.TextSegmentFinished = true
		effects = appendOpenAIChatEvent(effects, events.NewTextSegmentFinishedEvent(
			state.Metadata,
			state.chatCorrelation(state.CurrentChoiceIndex, events.StreamKindContent, "", nil),
			state.Message,
			finishReason,
		))
	}

	return state, effects
}

func (state openAIChatStreamState) chatCorrelation(
	choiceIndex *int,
	streamKind string,
	toolCallID string,
	toolCallIndex *int,
) events.Correlation {
	corr := events.BuildChatCompletionsCorrelation(state.Provider, state.CurrentResponseID, choiceIndex, streamKind, toolCallID, toolCallIndex)
	corr.SessionID = state.ProviderCallCorr.SessionID
	corr.RunID = state.ProviderCallCorr.RunID
	corr.TurnID = state.TurnID
	corr.ProviderCallID = state.ProviderCallCorr.ProviderCallID
	return corr
}

func (state openAIChatStreamState) mergedToolCalls() []ChatToolCall {
	if state.ToolCallMerger == nil {
		return nil
	}
	return state.ToolCallMerger.GetToolCalls()
}

func finalOpenAIChatUsage(state openAIChatStreamState) *events.Usage {
	if state.UsageInputTokens == 0 && state.UsageOutputTokens == 0 && state.CachedTokens == 0 {
		return state.Metadata.Usage
	}
	return &events.Usage{
		InputTokens:  state.UsageInputTokens,
		OutputTokens: state.UsageOutputTokens,
		CachedTokens: state.CachedTokens,
	}
}

func appendOpenAIChatEvent(effects []openAIChatStreamEffect, event events.Event) []openAIChatStreamEffect {
	return append(effects, openAIChatStreamEffect{Event: event})
}

func openAIChatToolStreamKey(corr events.Correlation, tc ChatToolCall) string {
	if corr.ToolCallID != "" {
		return corr.ToolCallID
	}
	if corr.SegmentID != "" {
		return corr.SegmentID
	}
	if strings.TrimSpace(tc.ID) != "" {
		return tc.ID
	}
	if tc.Index != nil {
		return "tool-index-" + strconv.Itoa(*tc.Index)
	}
	return "tool-unknown"
}

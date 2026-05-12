package openai_responses

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/go-go-golems/geppetto/pkg/events"
	"github.com/go-go-golems/geppetto/pkg/inference/engine"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/go-go-golems/geppetto/pkg/turns"
	"github.com/go-go-golems/geppetto/pkg/turns/toolblocks"
	"github.com/rs/zerolog/log"
)

type responsesStreamTerminalKind string

const (
	responsesStreamTerminalEOF       responsesStreamTerminalKind = "eof"
	responsesStreamTerminalCancelled responsesStreamTerminalKind = "cancelled"
	responsesStreamTerminalError     responsesStreamTerminalKind = "error"
)

type responsesStreamTerminal struct {
	Kind responsesStreamTerminalKind
	Err  error
}

type responsesPendingCall struct {
	callID, name, itemID string
	outputIndex          *int
	status               string
	args                 strings.Builder
}

type responsesStreamState struct {
	reqBody          responsesRequest
	providerCallCorr events.Correlation
	tap              engine.DebugTap

	message         string
	inputTokens     int
	outputTokens    int
	cachedTokens    int
	reasoningTokens int
	stopReason      *string

	responseCompleted bool
	streamErr         error

	thinkBuf                strings.Builder
	sayBuf                  strings.Builder
	currentReasoningText    strings.Builder
	currentReasoningSummary strings.Builder
	summaryBuf              strings.Builder

	currentReasoningItemID           string
	lastReasoningItemID              string
	assistantByItem                  map[string]string
	currentResponseID                string
	callsByItem                      map[string]*responsesPendingCall
	finalCalls                       []responsesPendingCall
	currentReasoningEncryptedContent string
	currentReasoningOutputIndex      *int
	lastReasoningOutputIndex         *int
	currentReasoningSummaryIndex     *int
	lastReasoningSummaryIndex        *int
	currentReasoningStatus           string
	latestMessageItemID              string
	latestMessageOutputIndex         *int
	latestMessageStatus              string
}

func newResponsesStreamState(reqBody responsesRequest, providerCallCorr events.Correlation, tap engine.DebugTap) *responsesStreamState {
	return &responsesStreamState{
		reqBody:          reqBody,
		providerCallCorr: providerCallCorr,
		tap:              tap,
		assistantByItem:  map[string]string{},
		callsByItem:      map[string]*responsesPendingCall{},
	}
}

func (s *responsesStreamState) providerCallCorrelation() events.Correlation {
	return s.providerCallCorr
}

func (s *responsesStreamState) segmentCorrelation(itemID string, outputIndex, summaryIndex *int, _ string) events.Correlation {
	corr := events.BuildResponsesCorrelation("openai_responses", s.currentResponseID, itemID, outputIndex, summaryIndex)
	corr.SessionID = s.providerCallCorr.SessionID
	corr.RunID = s.providerCallCorr.RunID
	corr.TurnID = s.providerCallCorr.TurnID
	corr.ProviderCallID = s.providerCallCorr.ProviderCallID
	return corr
}

func (s *responsesStreamState) toolCorrelation(itemID, callID string, outputIndex *int) events.Correlation {
	corr := s.segmentCorrelation(itemID, outputIndex, nil, events.SegmentTypeTool)
	if callID != "" {
		corr.ToolCallID = callID
	} else {
		corr.ToolCallID = itemID
	}
	return corr
}

func finalizeResponsesStreamMetadata(metadata events.EventMetadata, state *responsesStreamState, startTime time.Time, terminal responsesStreamTerminal) events.EventMetadata {
	state.applyTerminalStopReason(terminal)
	if state.inputTokens > 0 || state.outputTokens > 0 || state.cachedTokens > 0 {
		if metadata.Usage == nil {
			metadata.Usage = &events.Usage{}
		}
		metadata.Usage.InputTokens = state.inputTokens
		metadata.Usage.OutputTokens = state.outputTokens
		metadata.Usage.CachedTokens = state.cachedTokens
	}
	if metadata.Extra == nil {
		metadata.Extra = map[string]any{}
	}
	if state.reasoningTokens > 0 {
		metadata.Extra["reasoning_tokens"] = state.reasoningTokens
	}
	metadata.Extra["thinking_text"] = state.thinkBuf.String()
	metadata.Extra["saying_text"] = state.sayBuf.String()
	if state.summaryBuf.Len() > 0 {
		metadata.Extra["reasoning_summary_text"] = state.summaryBuf.String()
	}
	if state.stopReason != nil {
		metadata.StopReason = state.stopReason
	}
	d := time.Since(startTime).Milliseconds()
	dm := int64(d)
	metadata.DurationMs = &dm
	return metadata
}

func (s *responsesStreamState) applyTerminalStopReason(terminal responsesStreamTerminal) {
	reason, ok := responsesTerminalStopReason(terminal.Kind)
	if !ok {
		return
	}
	s.stopReason = &reason
}

func responsesTerminalStopReason(kind responsesStreamTerminalKind) (string, bool) {
	switch kind {
	case responsesStreamTerminalEOF:
		return "", false
	case responsesStreamTerminalCancelled:
		return "cancelled", true
	case responsesStreamTerminalError:
		return "error", true
	}
	return "", false
}

func appendResponsesFinalTurnBlocks(t *turns.Turn, state *responsesStreamState, includeToolCalls bool) int {
	if strings.TrimSpace(state.message) != "" {
		ab := turns.NewAssistantTextBlock(state.message)
		if state.latestMessageItemID != "" {
			if ab.Payload == nil {
				ab.Payload = map[string]any{}
			}
			ab.Payload[turns.PayloadKeyItemID] = state.latestMessageItemID
		}
		setOpenAIResponsesBlockMetadata(&ab, state.currentResponseID, state.latestMessageOutputIndex, "message", state.latestMessageStatus)
		turns.AppendBlock(t, ab)
	}
	if !includeToolCalls {
		return 0
	}
	for _, pc := range state.finalCalls {
		var args any
		if err := json.Unmarshal([]byte(pc.args.String()), &args); err != nil {
			args = map[string]any{}
		}
		b := toolblocks.NewToolCallBlockWithCorrelation(pc.callID, pc.name, args, state.toolCorrelation(pc.itemID, pc.callID, pc.outputIndex))
		if b.Payload == nil {
			b.Payload = map[string]any{}
		}
		if pc.itemID != "" {
			b.Payload[turns.PayloadKeyItemID] = pc.itemID
		}
		setOpenAIResponsesBlockMetadata(&b, state.currentResponseID, pc.outputIndex, "function_call", pc.status)
		turns.AppendBlock(t, b)
	}
	return len(state.finalCalls)
}

func responsesFinishClass(state *responsesStreamState, terminal responsesStreamTerminal, toolCallCount int) string {
	switch terminal.Kind {
	case responsesStreamTerminalEOF:
		finishClass := "completed"
		if !state.responseCompleted {
			finishClass = "stream_closed"
		}
		if toolCallCount > 0 {
			finishClass = "tool_calls_pending"
		}
		return finishClass
	case responsesStreamTerminalCancelled:
		return "cancelled"
	case responsesStreamTerminalError:
		return "failed"
	}
	return "stream_closed"
}

func persistResponsesInferenceResult(t *turns.Turn, metadata events.EventMetadata, provider string, hasToolCalls bool, modelInfo *settings.ModelInfo) {
	result := engine.BuildInferenceResultFromEventMetadata(metadata, provider, hasToolCalls)
	settings.ApplyModelInfoCost(&result, modelInfo)
	if err := engine.PersistInferenceResult(t, result); err != nil {
		log.Warn().Err(err).Msg("Responses: failed to persist canonical inference_result")
	}
}

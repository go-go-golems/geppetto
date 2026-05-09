package gemini

import (
	"errors"
	"io"
	"strings"
	"time"

	"github.com/go-go-golems/geppetto/pkg/events"
	"github.com/go-go-golems/geppetto/pkg/inference/engine"
	"github.com/go-go-golems/geppetto/pkg/turns"
	genai "github.com/google/generative-ai-go/genai"
	"google.golang.org/api/iterator"
)

type geminiStreamIterator interface {
	Next() (*genai.GenerateContentResponse, error)
}

func consumeGeminiStream(
	iter geminiStreamIterator,
	metadata events.EventMetadata,
	state *geminiStreamState,
	publishEvent func(events.Event),
	publishProviderRecord func(string, any),
) error {
	for {
		resp, err := iter.Next()
		if err == iterator.Done || errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		if publishProviderRecord != nil {
			publishProviderRecord("gemini.stream.chunk", resp)
		}
		for _, event := range reduceGeminiStreamResponse(metadata, state, resp) {
			if publishEvent != nil {
				publishEvent(event)
			}
		}
	}
}

func completeGeminiStream(
	t *turns.Turn,
	metadata *events.EventMetadata,
	state *geminiStreamState,
	startedAt time.Time,
	terminalErr error,
) (engine.InferenceResult, []events.Event) {
	if metadata == nil {
		return engine.InferenceResult{}, nil
	}
	if state == nil {
		state = newGeminiStreamState(events.Correlation{})
	}
	if terminalErr != nil && strings.TrimSpace(state.finalStopReason) == "" {
		state.finalStopReason = "error"
	}

	out := make([]events.Event, 0, 3)
	if state.message != "" && state.textSegmentStarted {
		out = append(out, events.NewTextSegmentFinishedEvent(*metadata, state.textCorr, state.message, state.finalStopReason))
	}

	appendGeminiFinalTurnBlocks(t, state)

	durationMs := time.Since(startedAt).Milliseconds()
	metadata.DurationMs = &durationMs
	if strings.TrimSpace(state.finalStopReason) != "" {
		metadata.StopReason = &state.finalStopReason
	}
	if state.finalUsage != nil {
		metadata.Usage = state.finalUsage
	}

	hasToolCalls := len(state.pendingCalls) > 0
	result := engine.BuildInferenceResultFromEventMetadata(*metadata, "gemini", hasToolCalls)
	if terminalErr != nil {
		result.FinishClass = engine.InferenceFinishClassError
	}

	if terminalErr != nil {
		out = append(out, events.NewErrorEvent(*metadata, terminalErr))
	}
	out = append(out, events.NewProviderCallFinishedEvent(
		*metadata,
		state.providerCallCorr,
		state.finalStopReason,
		string(result.FinishClass),
		metadata.Usage,
		metadata.DurationMs,
		hasToolCalls,
	))
	return result, out
}

func appendGeminiFinalTurnBlocks(t *turns.Turn, state *geminiStreamState) {
	if t == nil || state == nil {
		return
	}
	if state.message != "" {
		turns.AppendBlock(t, turns.NewAssistantTextBlock(state.message))
	}
	for _, c := range state.pendingCalls {
		turns.AppendBlock(t, turns.NewToolCallBlock(c.id, c.name, c.args))
	}
}

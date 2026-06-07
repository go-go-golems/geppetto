package gemini

import (
	"encoding/json"

	"github.com/go-go-golems/geppetto/pkg/events"
	genai "github.com/google/generative-ai-go/genai"
	"github.com/google/uuid"
)

type geminiPendingCall struct {
	id, name         string
	args             map[string]any
	thoughtSignature []byte
}

type geminiStreamState struct {
	providerCallCorr events.Correlation
	message          string
	chunkCount       int
	finalStopReason  string
	finalUsage       *events.Usage

	textSegmentStarted bool
	textSequence       int64
	textCorr           events.Correlation

	toolCallIndex int
	pendingCalls  []geminiPendingCall
}

func newGeminiStreamState(providerCallCorr events.Correlation) *geminiStreamState {
	return &geminiStreamState{providerCallCorr: providerCallCorr}
}

func reduceGeminiStreamResponse(metadata events.EventMetadata, state *geminiStreamState, resp *genai.GenerateContentResponse) []events.Event {
	if state == nil {
		return nil
	}
	state.chunkCount++

	var out []events.Event
	var chunkUsage *events.Usage
	if resp != nil {
		if u, ok := extractGeminiUsage(resp); ok {
			state.finalUsage = u
			chunkUsage = u
		}
	}

	delta := ""
	chunkStopReason := ""
	if resp != nil && len(resp.Candidates) > 0 {
		for _, cand := range resp.Candidates {
			if fr, ok := extractGeminiFinishReason(cand); ok {
				state.finalStopReason = fr
				chunkStopReason = fr
			}
			if cand.Content == nil {
				continue
			}
			for _, p := range cand.Content.Parts {
				switch v := p.(type) {
				case genai.Text:
					delta += string(v)
				case genai.FunctionCall:
					args := v.Args
					if args == nil {
						args = map[string]any{}
					}
					id := uuid.NewString()
					state.pendingCalls = append(state.pendingCalls, geminiPendingCall{id: id, name: v.Name, args: args})
					inputBytes, _ := json.Marshal(args)
					toolCorr := geminiToolCorrelation(state.providerCallCorr, id, state.toolCallIndex)
					state.toolCallIndex++
					out = append(out, events.NewToolCallStartedEvent(metadata, toolCorr, id, v.Name))
					out = append(out, events.NewToolCallRequestedEvent(metadata, toolCorr, id, v.Name, string(inputBytes)))
				}
			}
		}
	}

	if chunkUsage != nil || chunkStopReason != "" {
		out = append(out, events.NewProviderCallMetadataUpdatedEvent(metadata, state.providerCallCorr, state.finalStopReason, "", state.finalUsage))
	}
	if delta != "" {
		if !state.textSegmentStarted {
			state.textSegmentStarted = true
			state.textCorr = geminiSegmentCorrelation(state.providerCallCorr, "", 0, events.SegmentTypeText)
			out = append(out, events.NewTextSegmentStartedEvent(metadata, state.textCorr, "assistant"))
		}
		state.message += delta
		state.textSequence++
		out = append(out, events.NewTextDeltaEvent(metadata, state.textCorr, delta, state.message, state.textSequence))
	}

	return out
}

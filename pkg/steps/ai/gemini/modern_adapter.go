package gemini

import (
	"encoding/base64"
	"encoding/json"
	"strings"

	"github.com/go-go-golems/geppetto/pkg/events"
	"github.com/go-go-golems/geppetto/pkg/turns"
	"github.com/google/uuid"
	moderngenai "google.golang.org/genai"
)

type geminiThoughtPart struct {
	Text      string
	Signature []byte
}

type modernGeminiStreamState struct {
	providerCallCorr events.Correlation
	responseID       string

	message            string
	textSegmentStarted bool
	textSequence       int64
	textCorr           events.Correlation

	reasoning         string
	reasoningParts    []geminiThoughtPart
	reasoningStarted  bool
	reasoningSequence int64
	reasoningCorr     events.Correlation

	toolCallIndex int
	pendingCalls  []geminiPendingCall

	finalStopReason string
	finalUsage      *events.Usage
	finalUsageExtra map[string]any
}

func newModernGeminiStreamState(providerCallCorr events.Correlation) *modernGeminiStreamState {
	return &modernGeminiStreamState{providerCallCorr: providerCallCorr}
}

func reduceModernGeminiResponse(metadata events.EventMetadata, state *modernGeminiStreamState, resp *moderngenai.GenerateContentResponse) []events.Event {
	if state == nil || resp == nil {
		return nil
	}
	if strings.TrimSpace(resp.ResponseID) != "" {
		state.responseID = resp.ResponseID
	}

	var out []events.Event
	var chunkStopReason string
	if usage, extra, ok := extractModernGeminiUsage(resp); ok {
		state.finalUsage = usage
		state.finalUsageExtra = extra
		out = append(out, events.NewProviderCallMetadataUpdatedEvent(metadata, state.providerCallCorr, state.finalStopReason, "", state.finalUsage))
	}

	for _, cand := range resp.Candidates {
		if cand == nil {
			continue
		}
		if s := strings.TrimSpace(string(cand.FinishReason)); s != "" && s != "FINISH_REASON_UNSPECIFIED" {
			state.finalStopReason = s
			chunkStopReason = s
		}
		if cand.Content == nil {
			continue
		}
		for partIndex, part := range cand.Content.Parts {
			if part == nil {
				continue
			}
			if part.Thought {
				out = append(out, reduceModernGeminiThoughtPart(metadata, state, part, partIndex)...)
				continue
			}
			if part.Text != "" {
				out = append(out, reduceModernGeminiTextPart(metadata, state, part.Text)...)
			}
			if part.FunctionCall != nil {
				out = append(out, reduceModernGeminiFunctionCall(metadata, state, part.FunctionCall)...)
			}
		}
	}
	if chunkStopReason != "" {
		out = append(out, events.NewProviderCallMetadataUpdatedEvent(metadata, state.providerCallCorr, state.finalStopReason, "", state.finalUsage))
	}
	return out
}

func reduceModernGeminiThoughtPart(metadata events.EventMetadata, state *modernGeminiStreamState, part *moderngenai.Part, partIndex int) []events.Event {
	if part == nil {
		return nil
	}
	if !state.reasoningStarted {
		state.reasoningStarted = true
		state.reasoningCorr = geminiSegmentCorrelation(state.providerCallCorr, state.responseID, partIndex, events.SegmentTypeReasoning)
		return append([]events.Event{events.NewReasoningSegmentStartedEvent(metadata, state.reasoningCorr, "provider")}, reduceModernGeminiThoughtDelta(metadata, state, part)...)
	}
	return reduceModernGeminiThoughtDelta(metadata, state, part)
}

func reduceModernGeminiThoughtDelta(metadata events.EventMetadata, state *modernGeminiStreamState, part *moderngenai.Part) []events.Event {
	if part == nil {
		return nil
	}
	if len(part.ThoughtSignature) > 0 || part.Text != "" {
		state.reasoningParts = append(state.reasoningParts, geminiThoughtPart{Text: part.Text, Signature: append([]byte(nil), part.ThoughtSignature...)})
	}
	if part.Text == "" {
		return nil
	}
	state.reasoning += part.Text
	state.reasoningSequence++
	return []events.Event{events.NewReasoningDeltaEventWithSource(metadata, state.reasoningCorr, "provider", part.Text, state.reasoning, state.reasoningSequence)}
}

func reduceModernGeminiTextPart(metadata events.EventMetadata, state *modernGeminiStreamState, text string) []events.Event {
	if text == "" {
		return nil
	}
	var out []events.Event
	if !state.textSegmentStarted {
		state.textSegmentStarted = true
		state.textCorr = geminiSegmentCorrelation(state.providerCallCorr, state.responseID, 0, events.SegmentTypeText)
		out = append(out, events.NewTextSegmentStartedEvent(metadata, state.textCorr, "assistant"))
	}
	state.message += text
	state.textSequence++
	out = append(out, events.NewTextDeltaEvent(metadata, state.textCorr, text, state.message, state.textSequence))
	return out
}

func reduceModernGeminiFunctionCall(metadata events.EventMetadata, state *modernGeminiStreamState, call *moderngenai.FunctionCall) []events.Event {
	if call == nil {
		return nil
	}
	args := call.Args
	if args == nil {
		args = map[string]any{}
	}
	id := strings.TrimSpace(call.ID)
	if id == "" {
		id = uuid.NewString()
	}
	state.pendingCalls = append(state.pendingCalls, geminiPendingCall{id: id, name: call.Name, args: args})
	inputBytes, _ := json.Marshal(args)
	toolCorr := geminiToolCorrelation(state.providerCallCorr, id, state.toolCallIndex)
	state.toolCallIndex++
	return []events.Event{
		events.NewToolCallStartedEvent(metadata, toolCorr, id, call.Name),
		events.NewToolCallRequestedEvent(metadata, toolCorr, id, call.Name, string(inputBytes)),
	}
}

func extractModernGeminiUsage(resp *moderngenai.GenerateContentResponse) (*events.Usage, map[string]any, bool) {
	if resp == nil || resp.UsageMetadata == nil {
		return nil, nil, false
	}
	u := resp.UsageMetadata
	if u.PromptTokenCount == 0 && u.CandidatesTokenCount == 0 && u.CachedContentTokenCount == 0 && u.ThoughtsTokenCount == 0 && u.ToolUsePromptTokenCount == 0 && u.TotalTokenCount == 0 {
		return nil, nil, false
	}
	extra := map[string]any{}
	if u.ThoughtsTokenCount > 0 {
		extra["thoughts_token_count"] = int(u.ThoughtsTokenCount)
	}
	if u.ToolUsePromptTokenCount > 0 {
		extra["tool_use_prompt_token_count"] = int(u.ToolUsePromptTokenCount)
	}
	if u.TotalTokenCount > 0 {
		extra["total_token_count"] = int(u.TotalTokenCount)
	}
	return &events.Usage{
		InputTokens:  int(u.PromptTokenCount),
		OutputTokens: int(u.CandidatesTokenCount),
		CachedTokens: int(u.CachedContentTokenCount),
	}, extra, true
}

func appendModernGeminiStateBlocks(t *turns.Turn, state *modernGeminiStreamState) error {
	if t == nil || state == nil {
		return nil
	}
	if state.reasoning != "" || len(state.reasoningParts) > 0 {
		b := turns.Block{Kind: turns.BlockKindReasoning, Role: turns.RoleAssistant, Payload: map[string]any{}}
		if state.reasoning != "" {
			b.Payload[turns.PayloadKeyText] = state.reasoning
		}
		if len(state.reasoningParts) > 0 {
			last := state.reasoningParts[len(state.reasoningParts)-1]
			if len(last.Signature) > 0 {
				if err := keyBlockMetaGeminiThoughtSignature.Set(&b.Metadata, base64.StdEncoding.EncodeToString(last.Signature)); err != nil {
					return err
				}
			}
			if err := keyBlockMetaGeminiThought.Set(&b.Metadata, true); err != nil {
				return err
			}
		}
		turns.AppendBlock(t, b)
	}
	if state.message != "" {
		turns.AppendBlock(t, turns.NewAssistantTextBlock(state.message))
	}
	for _, call := range state.pendingCalls {
		turns.AppendBlock(t, turns.NewToolCallBlock(call.id, call.name, call.args))
	}
	return nil
}

func buildModernGeminiContentsFromTurn(t *turns.Turn) ([]*moderngenai.Content, error) {
	if t == nil || len(t.Blocks) == 0 {
		return nil, nil
	}
	idToName := map[string]string{}
	for _, b := range t.Blocks {
		if b.Kind == turns.BlockKindToolCall {
			id, _ := b.Payload[turns.PayloadKeyID].(string)
			name, _ := b.Payload[turns.PayloadKeyName].(string)
			if id != "" && name != "" {
				idToName[id] = name
			}
		}
	}
	contents := make([]*moderngenai.Content, 0, len(t.Blocks))
	for _, b := range t.Blocks {
		content := &moderngenai.Content{}
		switch b.Kind {
		case turns.BlockKindUser, turns.BlockKindSystem, turns.BlockKindOther:
			content.Role = string(moderngenai.RoleUser)
			if txt, ok := blockText(b); ok {
				content.Parts = append(content.Parts, moderngenai.NewPartFromText(txt))
			}
		case turns.BlockKindLLMText:
			content.Role = string(moderngenai.RoleModel)
			if txt, ok := blockText(b); ok {
				content.Parts = append(content.Parts, moderngenai.NewPartFromText(txt))
			}
		case turns.BlockKindReasoning:
			content.Role = string(moderngenai.RoleModel)
			part := &moderngenai.Part{Thought: true}
			if txt, ok := blockText(b); ok {
				part.Text = txt
			}
			if sig64, ok, err := keyBlockMetaGeminiThoughtSignature.Get(b.Metadata); err != nil {
				return nil, err
			} else if ok && strings.TrimSpace(sig64) != "" {
				sig, err := base64.StdEncoding.DecodeString(sig64)
				if err != nil {
					return nil, err
				}
				part.ThoughtSignature = sig
			}
			content.Parts = append(content.Parts, part)
		case turns.BlockKindToolCall:
			content.Role = string(moderngenai.RoleModel)
			id, _ := b.Payload[turns.PayloadKeyID].(string)
			name, _ := b.Payload[turns.PayloadKeyName].(string)
			args := toolCallArgsMap(b.Payload[turns.PayloadKeyArgs])
			content.Parts = append(content.Parts, &moderngenai.Part{FunctionCall: &moderngenai.FunctionCall{ID: id, Name: name, Args: args}})
		case turns.BlockKindToolUse:
			content.Role = string(moderngenai.RoleUser)
			id, _ := b.Payload[turns.PayloadKeyID].(string)
			name := idToName[id]
			if name == "" {
				name = "result"
			}
			content.Parts = append(content.Parts, &moderngenai.Part{FunctionResponse: &moderngenai.FunctionResponse{ID: id, Name: name, Response: toolUseResponseMap(b)}})
		}
		if len(content.Parts) > 0 {
			contents = append(contents, content)
		}
	}
	return contents, nil
}

func blockText(b turns.Block) (string, bool) {
	if b.Payload == nil {
		return "", false
	}
	switch v := b.Payload[turns.PayloadKeyText].(type) {
	case string:
		return v, v != ""
	case []byte:
		return string(v), len(v) > 0
	default:
		return "", false
	}
}

func toolCallArgsMap(raw any) map[string]any {
	switch v := raw.(type) {
	case map[string]any:
		return v
	case string:
		var obj map[string]any
		if json.Unmarshal([]byte(v), &obj) == nil && obj != nil {
			return obj
		}
	case json.RawMessage:
		var obj map[string]any
		if json.Unmarshal(v, &obj) == nil && obj != nil {
			return obj
		}
	}
	return map[string]any{}
}

func toolUseResponseMap(b turns.Block) map[string]any {
	res := b.Payload[turns.PayloadKeyResult]
	errStr, _ := b.Payload[turns.PayloadKeyError].(string)
	var response map[string]any
	switch rv := res.(type) {
	case map[string]any:
		response = rv
	case string:
		if json.Unmarshal([]byte(rv), &response) != nil {
			response = map[string]any{"result": rv}
		}
	default:
		bts, _ := json.Marshal(rv)
		if json.Unmarshal(bts, &response) != nil {
			response = map[string]any{"result": rv}
		}
	}
	if response == nil {
		response = map[string]any{}
	}
	if errStr != "" {
		return map[string]any{"error": errStr, "result": response}
	}
	return response
}

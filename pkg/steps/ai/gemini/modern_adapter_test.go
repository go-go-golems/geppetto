package gemini

import (
	"encoding/base64"
	"testing"

	"github.com/go-go-golems/geppetto/pkg/events"
	"github.com/go-go-golems/geppetto/pkg/turns"
	moderngenai "google.golang.org/genai"
)

func TestModernGeminiReducerMapsThoughtPartsToReasoningAndPreservesSignature(t *testing.T) {
	metadata := events.EventMetadata{SessionID: "session-1", InferenceID: "inference-1", TurnID: "turn-1"}
	providerCorr := geminiProviderCallCorrelation(metadata, metadata.InferenceID, "gemini-3", 0)
	state := newModernGeminiStreamState(providerCorr)

	got := reduceModernGeminiResponse(metadata, state, &moderngenai.GenerateContentResponse{
		ResponseID: "resp-1",
		Candidates: []*moderngenai.Candidate{{Content: &moderngenai.Content{Parts: []*moderngenai.Part{{
			Text:             "thinking privately",
			Thought:          true,
			ThoughtSignature: []byte("sig-1"),
		}}}}},
	})

	assertGeminiEventTypes(t, got, []events.EventType{
		events.EventTypeReasoningSegmentStarted,
		events.EventTypeReasoningDelta,
	})
	if state.message != "" {
		t.Fatalf("thought text leaked into assistant message: %q", state.message)
	}
	if state.reasoning != "thinking privately" {
		t.Fatalf("reasoning = %q, want thought text", state.reasoning)
	}
	if len(state.reasoningParts) != 1 || string(state.reasoningParts[0].Signature) != "sig-1" {
		t.Fatalf("reasoning signatures = %#v, want sig-1", state.reasoningParts)
	}

	turn := &turns.Turn{ID: "turn-1"}
	if err := appendModernGeminiStateBlocks(turn, state); err != nil {
		t.Fatalf("append blocks: %v", err)
	}
	if len(turn.Blocks) != 1 || turn.Blocks[0].Kind != turns.BlockKindReasoning {
		t.Fatalf("turn blocks = %#v, want one reasoning block", turn.Blocks)
	}
	if txt, _ := turn.Blocks[0].Payload[turns.PayloadKeyText].(string); txt != "thinking privately" {
		t.Fatalf("reasoning payload text = %q", txt)
	}
	sig64, ok, err := keyBlockMetaGeminiThoughtSignature.Get(turn.Blocks[0].Metadata)
	if err != nil || !ok {
		t.Fatalf("missing thought signature metadata: ok=%v err=%v", ok, err)
	}
	if gotSig, _ := base64.StdEncoding.DecodeString(sig64); string(gotSig) != "sig-1" {
		t.Fatalf("signature metadata = %q decoded=%q, want sig-1", sig64, string(gotSig))
	}
}

func TestModernGeminiReducerUsesProviderFunctionCallID(t *testing.T) {
	metadata := events.EventMetadata{SessionID: "session-1", InferenceID: "inference-1", TurnID: "turn-1"}
	providerCorr := geminiProviderCallCorrelation(metadata, metadata.InferenceID, "gemini-3", 0)
	state := newModernGeminiStreamState(providerCorr)

	got := reduceModernGeminiResponse(metadata, state, &moderngenai.GenerateContentResponse{Candidates: []*moderngenai.Candidate{{Content: &moderngenai.Content{Parts: []*moderngenai.Part{{
		FunctionCall: &moderngenai.FunctionCall{ID: "provider-call-1", Name: "lookup", Args: map[string]any{"q": "x"}},
	}}}}}})

	assertGeminiEventTypes(t, got, []events.EventType{
		events.EventTypeToolCallStarted,
		events.EventTypeToolCallRequested,
	})
	if len(state.pendingCalls) != 1 {
		t.Fatalf("pending calls = %#v, want one", state.pendingCalls)
	}
	if state.pendingCalls[0].id != "provider-call-1" {
		t.Fatalf("tool call id = %q, want provider id", state.pendingCalls[0].id)
	}
	requested, ok := got[1].(*events.EventToolCallRequested)
	if !ok {
		t.Fatalf("event[1] = %#v, want requested", got[1])
	}
	if requested.ToolCallID != "provider-call-1" || requested.ToolName != "lookup" || requested.Input != `{"q":"x"}` {
		t.Fatalf("requested = %#v, want provider id/name/json args", requested)
	}
}

func TestModernGeminiUsagePreservesThoughtsTokenCountInExtra(t *testing.T) {
	usage, extra, ok := extractModernGeminiUsage(&moderngenai.GenerateContentResponse{UsageMetadata: &moderngenai.GenerateContentResponseUsageMetadata{
		PromptTokenCount:        11,
		CandidatesTokenCount:    7,
		CachedContentTokenCount: 3,
		ThoughtsTokenCount:      5,
		ToolUsePromptTokenCount: 2,
		TotalTokenCount:         25,
	}})
	if !ok {
		t.Fatalf("expected usage")
	}
	if usage.InputTokens != 11 || usage.OutputTokens != 7 || usage.CachedTokens != 3 {
		t.Fatalf("usage = %#v, want prompt/response/cached counts", usage)
	}
	if extra["thoughts_token_count"] != 5 || extra["tool_use_prompt_token_count"] != 2 || extra["total_token_count"] != 25 {
		t.Fatalf("extra = %#v, want thoughts/tool/total token counts", extra)
	}
}

func TestModernGeminiContentsReplayThoughtSignatureAndToolIDs(t *testing.T) {
	turn := &turns.Turn{ID: "turn-1"}
	reasoning := turns.Block{Kind: turns.BlockKindReasoning, Role: turns.RoleAssistant, Payload: map[string]any{turns.PayloadKeyText: "private chain"}}
	if err := keyBlockMetaGeminiThoughtSignature.Set(&reasoning.Metadata, base64.StdEncoding.EncodeToString([]byte("sig-1"))); err != nil {
		t.Fatalf("set signature: %v", err)
	}
	turns.AppendBlock(turn, turns.NewUserTextBlock("question"))
	turns.AppendBlock(turn, reasoning)
	turns.AppendBlock(turn, turns.NewToolCallBlock("provider-call-1", "lookup", `{"q":"x"}`))
	turns.AppendBlock(turn, turns.NewToolUseBlock("provider-call-1", map[string]any{"answer": "y"}))

	contents, err := buildModernGeminiContentsFromTurn(turn)
	if err != nil {
		t.Fatalf("build contents: %v", err)
	}
	if len(contents) != 4 {
		t.Fatalf("contents len = %d, want 4: %#v", len(contents), contents)
	}
	thought := contents[1].Parts[0]
	if !thought.Thought || thought.Text != "private chain" || string(thought.ThoughtSignature) != "sig-1" {
		t.Fatalf("thought part = %#v, want thought text + signature", thought)
	}
	call := contents[2].Parts[0].FunctionCall
	if call == nil || call.ID != "provider-call-1" || call.Name != "lookup" || call.Args["q"] != "x" {
		t.Fatalf("function call = %#v, want provider id/name/args", call)
	}
	response := contents[3].Parts[0].FunctionResponse
	if response == nil || response.ID != "provider-call-1" || response.Name != "lookup" || response.Response["answer"] != "y" {
		t.Fatalf("function response = %#v, want matching provider id/name/result", response)
	}
}

func TestModernGeminiContentsMapsInlineImageData(t *testing.T) {
	turn := &turns.Turn{ID: "turn-image"}
	turns.AppendBlock(turn, turns.NewUserMultimodalBlock("describe", []map[string]any{{
		"content": "data:image/png;base64,UE5H",
	}}))

	contents, err := buildModernGeminiContentsFromTurn(turn)
	if err != nil {
		t.Fatalf("build contents: %v", err)
	}
	if len(contents) != 1 || len(contents[0].Parts) != 2 {
		t.Fatalf("contents = %#v", contents)
	}
	if contents[0].Parts[0].Text != "describe" {
		t.Fatalf("text part = %#v", contents[0].Parts[0])
	}
	inline := contents[0].Parts[1].InlineData
	if inline == nil || inline.MIMEType != "image/png" || string(inline.Data) != "PNG" {
		t.Fatalf("inline data = %#v", inline)
	}
}

func TestModernGeminiContentsMapsImageFileURI(t *testing.T) {
	turn := &turns.Turn{ID: "turn-image-uri"}
	turns.AppendBlock(turn, turns.NewUserMultimodalBlock("describe", []map[string]any{{
		"file_uri":   "gs://bucket/image.png",
		"media_type": "image/png",
	}}))

	contents, err := buildModernGeminiContentsFromTurn(turn)
	if err != nil {
		t.Fatalf("build contents: %v", err)
	}
	fileData := contents[0].Parts[1].FileData
	if fileData == nil || fileData.FileURI != "gs://bucket/image.png" || fileData.MIMEType != "image/png" {
		t.Fatalf("file data = %#v", fileData)
	}
}

func TestModernGeminiContentsRejectsGenericImageURL(t *testing.T) {
	turn := &turns.Turn{ID: "turn-image-url"}
	turns.AppendBlock(turn, turns.NewUserMultimodalBlock("describe", []map[string]any{{
		"url": "https://example.com/image.png",
	}}))

	_, err := buildModernGeminiContentsFromTurn(turn)
	if err == nil {
		t.Fatalf("expected generic URL error")
	}
}

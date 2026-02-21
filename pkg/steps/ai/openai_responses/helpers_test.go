package openai_responses

import (
	"testing"

	infengine "github.com/go-go-golems/geppetto/pkg/inference/engine"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/go-go-golems/geppetto/pkg/turns"
)

func typesOf(parts []responsesContentPart) []string {
	ts := make([]string, 0, len(parts))
	for _, p := range parts {
		ts = append(ts, p.Type)
	}
	return ts
}

func TestBuildInputItemsFromTurn_PlainChat(t *testing.T) {
	turn := &turns.Turn{Blocks: []turns.Block{
		turns.NewSystemTextBlock("You are a LLM."),
		turns.NewUserTextBlock("Hello"),
	}}

	got := buildInputItemsFromTurn(turn)
	if len(got) != 2 {
		t.Fatalf("expected 2 items, got %d", len(got))
	}
	if got[0].Type != "" || got[0].Role != "system" {
		t.Fatalf("first item must be role-based system, got type=%q role=%q", got[0].Type, got[0].Role)
	}
	if got[1].Type != "" || got[1].Role != "user" {
		t.Fatalf("second item must be role-based user, got type=%q role=%q", got[1].Type, got[1].Role)
	}
	if len(got[0].Content) != 1 || typesOf(got[0].Content)[0] != "input_text" {
		t.Fatalf("system content must have single input_text part")
	}
	if len(got[1].Content) != 1 || typesOf(got[1].Content)[0] != "input_text" {
		t.Fatalf("user content must have single input_text part")
	}
}

func TestBuildInputItemsFromTurn_ReasoningWithAssistantFollower(t *testing.T) {
	rs := turns.Block{Kind: turns.BlockKindReasoning, ID: "rs_1", Payload: map[string]any{
		turns.PayloadKeyEncryptedContent: "gAAAAA...",
	}}
	as := turns.NewAssistantTextBlock("Answer")
	turn := &turns.Turn{Blocks: []turns.Block{
		turns.NewSystemTextBlock("You are a LLM."),
		turns.NewUserTextBlock("Question"),
		rs,
		as,
	}}

	got := buildInputItemsFromTurn(turn)
	if len(got) != 4 {
		t.Fatalf("expected 4 items, got %d", len(got))
	}
	// Pre-context role-based items
	if got[0].Type != "" || got[0].Role != "system" {
		t.Fatalf("pre system wrong shape")
	}
	if got[1].Type != "" || got[1].Role != "user" {
		t.Fatalf("pre user wrong shape")
	}
	// Reasoning then item-based message follower
	if got[2].Type != "reasoning" {
		t.Fatalf("item 3 must be reasoning, got %q", got[2].Type)
	}
	if got[2].ID != "rs_1" {
		t.Fatalf("reasoning id mismatch: %q", got[2].ID)
	}
	if got[3].Type != "message" || got[3].Role != "assistant" {
		t.Fatalf("item 4 must be item-based assistant message")
	}
	if len(got[3].Content) != 1 || typesOf(got[3].Content)[0] != "output_text" {
		t.Fatalf("assistant item content must be output_text")
	}
}

func TestBuildInputItemsFromTurn_MultiTurnReasoningThenUser(t *testing.T) {
	rs := turns.Block{Kind: turns.BlockKindReasoning, ID: "rs_1", Payload: map[string]any{
		turns.PayloadKeyEncryptedContent: "gAAAAA...",
	}}
	turn := &turns.Turn{Blocks: []turns.Block{
		turns.NewSystemTextBlock("You are a LLM."),
		turns.NewUserTextBlock("Question"),
		rs,
		turns.NewAssistantTextBlock("Answer"),
		turns.NewUserTextBlock("Follow-up"),
	}}

	got := buildInputItemsFromTurn(turn)
	if len(got) != 5 {
		t.Fatalf("expected 5 items, got %d", len(got))
	}
	if got[0].Role != "system" || got[0].Type != "" {
		t.Fatalf("first item must be system role message")
	}
	if got[1].Role != "user" || got[1].Type != "" {
		t.Fatalf("second item must be user role message")
	}
	if got[2].Type != "reasoning" {
		t.Fatalf("third item must be reasoning")
	}
	if got[3].Type != "message" || got[3].Role != "assistant" {
		t.Fatalf("fourth item must be assistant item-based message")
	}
	if got[4].Role != "user" || got[4].Type != "" {
		t.Fatalf("fifth item must be follow-up user role message")
	}
}

func TestBuildInputItemsFromTurn_PreservesAssistantContextWithReasoningFollower(t *testing.T) {
	rs := turns.Block{Kind: turns.BlockKindReasoning, ID: "rs_latest", Payload: map[string]any{
		turns.PayloadKeyEncryptedContent: "enc_latest",
	}}
	turn := &turns.Turn{Blocks: []turns.Block{
		turns.NewSystemTextBlock("You are a LLM."),
		turns.NewUserTextBlock("Q1"),
		turns.NewAssistantTextBlock("A1 should remain"),
		turns.NewUserTextBlock("Q2"),
		turns.NewAssistantTextBlock("A2 should remain"),
		rs,
		turns.NewAssistantTextBlock("A3 follower"),
	}}

	got := buildInputItemsFromTurn(turn)

	var assistantRoleTexts []string
	for _, item := range got {
		if item.Type == "" && item.Role == "assistant" && len(item.Content) > 0 {
			assistantRoleTexts = append(assistantRoleTexts, item.Content[0].Text)
		}
	}
	if len(assistantRoleTexts) != 2 {
		t.Fatalf("expected two role-based assistant pre-context messages, got %d (%v)", len(assistantRoleTexts), assistantRoleTexts)
	}
	if assistantRoleTexts[0] != "A1 should remain" {
		t.Fatalf("expected preserved assistant context to be A1, got %q", assistantRoleTexts[0])
	}
	if assistantRoleTexts[1] != "A2 should remain" {
		t.Fatalf("expected preserved assistant context to include A2, got %q", assistantRoleTexts[1])
	}

	if len(got) == 0 {
		t.Fatalf("expected non-empty input items")
	}
	lastReasoningIdx := -1
	for i, item := range got {
		if item.Type == "reasoning" {
			lastReasoningIdx = i
		}
	}
	if lastReasoningIdx == -1 {
		t.Fatalf("expected a reasoning item in request input")
	}
	if lastReasoningIdx+1 >= len(got) {
		t.Fatalf("expected reasoning follower item")
	}
	follower := got[lastReasoningIdx+1]
	if follower.Type != "message" || follower.Role != "assistant" {
		t.Fatalf("expected reasoning follower to be assistant message, got type=%q role=%q", follower.Type, follower.Role)
	}
	if len(follower.Content) != 1 || follower.Content[0].Text != "A3 follower" {
		t.Fatalf("unexpected follower content: %#v", follower.Content)
	}
}

func TestBuildInputItemsFromTurn_PreservesReasoningForOlderFunctionCallChains(t *testing.T) {
	r1 := turns.Block{Kind: turns.BlockKindReasoning, ID: "rs_old", Payload: map[string]any{
		turns.PayloadKeyEncryptedContent: "enc_old",
	}}
	call := turns.Block{
		Kind: turns.BlockKindToolCall,
		Payload: map[string]any{
			turns.PayloadKeyID:     "call_1",
			turns.PayloadKeyItemID: "fc_1",
			turns.PayloadKeyName:   "inventory_report",
			turns.PayloadKeyArgs:   map[string]any{"threshold": 3},
		},
	}
	use := turns.Block{
		Kind: turns.BlockKindToolUse,
		Payload: map[string]any{
			turns.PayloadKeyID:     "call_1",
			turns.PayloadKeyResult: `{"ok":true}`,
			turns.PayloadKeyError:  "",
		},
	}
	r2 := turns.Block{Kind: turns.BlockKindReasoning, ID: "rs_new", Payload: map[string]any{
		turns.PayloadKeyEncryptedContent: "enc_new",
	}}
	turn := &turns.Turn{Blocks: []turns.Block{
		turns.NewSystemTextBlock("sys"),
		turns.NewUserTextBlock("u1"),
		r1,
		call,
		use,
		r2,
		turns.NewAssistantTextBlock("latest answer"),
		turns.NewUserTextBlock("follow-up"),
	}}

	got := buildInputItemsFromTurn(turn)

	oldCallIdx := -1
	for i, item := range got {
		if item.Type == "function_call" && item.ID == "fc_1" {
			oldCallIdx = i
			break
		}
	}
	if oldCallIdx <= 0 {
		t.Fatalf("expected older function_call with id fc_1, got %#v", got)
	}
	if got[oldCallIdx-1].Type != "reasoning" || got[oldCallIdx-1].ID != "rs_old" {
		t.Fatalf("expected reasoning rs_old immediately before function_call fc_1, got prev=%#v", got[oldCallIdx-1])
	}

	newReasoningIdx := -1
	for i, item := range got {
		if item.Type == "reasoning" && item.ID == "rs_new" {
			newReasoningIdx = i
			break
		}
	}
	if newReasoningIdx == -1 {
		t.Fatalf("expected reasoning rs_new in request input")
	}
	if newReasoningIdx+1 >= len(got) {
		t.Fatalf("expected item after reasoning rs_new")
	}
	if got[newReasoningIdx+1].Type != "message" || got[newReasoningIdx+1].Role != "assistant" {
		t.Fatalf("expected rs_new follower to be assistant message, got %#v", got[newReasoningIdx+1])
	}
}

func newTestEngine(ss *settings.StepSettings) *Engine {
	return &Engine{settings: ss}
}

func TestBuildResponsesRequestStructuredOutput(t *testing.T) {
	model := "gpt-4o-mini"
	strict := true
	ss := &settings.StepSettings{
		Chat: &settings.ChatSettings{
			Engine:                 &model,
			StructuredOutputMode:   settings.StructuredOutputModeJSONSchema,
			StructuredOutputName:   "person",
			StructuredOutputSchema: `{"type":"object","properties":{"name":{"type":"string"}}}`,
			StructuredOutputStrict: &strict,
			Stream:                 true,
		},
	}
	turn := &turns.Turn{Blocks: []turns.Block{
		turns.NewUserTextBlock("return JSON"),
	}}

	e := newTestEngine(ss)
	req, err := e.buildResponsesRequest(turn)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.Text == nil || req.Text.Format == nil {
		t.Fatalf("expected text.format to be populated for structured output")
	}
	if req.Text.Format.Type != "json_schema" {
		t.Fatalf("expected text.format.type=json_schema, got %q", req.Text.Format.Type)
	}
	if req.Text.Format.Name != "person" {
		t.Fatalf("expected schema name person, got %q", req.Text.Format.Name)
	}
}

func TestBuildResponsesRequestStructuredOutputInvalidSchemaRequireValid(t *testing.T) {
	model := "gpt-4o-mini"
	ss := &settings.StepSettings{
		Chat: &settings.ChatSettings{
			Engine:                       &model,
			StructuredOutputMode:         settings.StructuredOutputModeJSONSchema,
			StructuredOutputName:         "person",
			StructuredOutputSchema:       `{"type":"object",`,
			StructuredOutputRequireValid: true,
			Stream:                       true,
		},
	}
	turn := &turns.Turn{Blocks: []turns.Block{
		turns.NewUserTextBlock("return JSON"),
	}}

	e := newTestEngine(ss)
	if _, err := e.buildResponsesRequest(turn); err == nil {
		t.Fatalf("expected error when require_valid=true and schema JSON is invalid")
	}
}

func TestBuildResponsesRequestStructuredOutputInvalidSchemaIgnoredWhenNotRequired(t *testing.T) {
	model := "gpt-4o-mini"
	ss := &settings.StepSettings{
		Chat: &settings.ChatSettings{
			Engine:                       &model,
			StructuredOutputMode:         settings.StructuredOutputModeJSONSchema,
			StructuredOutputName:         "person",
			StructuredOutputSchema:       `{"type":"object",`,
			StructuredOutputRequireValid: false,
			Stream:                       true,
		},
	}
	turn := &turns.Turn{Blocks: []turns.Block{
		turns.NewUserTextBlock("return JSON"),
	}}

	e := newTestEngine(ss)
	req, err := e.buildResponsesRequest(turn)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.Text != nil {
		t.Fatalf("expected invalid schema to be ignored when require_valid=false")
	}
}

func TestBuildResponsesRequestInferenceEmptyStopClearsChatStop(t *testing.T) {
	model := "gpt-4o-mini"
	ss := &settings.StepSettings{
		Chat: &settings.ChatSettings{
			Engine: &model,
			Stop:   []string{"<END>"},
		},
	}
	turn := &turns.Turn{Blocks: []turns.Block{
		turns.NewUserTextBlock("hello"),
	}}
	if err := infengine.KeyInferenceConfig.Set(&turn.Data, infengine.InferenceConfig{Stop: []string{}}); err != nil {
		t.Fatalf("failed to set inference config: %v", err)
	}

	e := newTestEngine(ss)
	req, err := e.buildResponsesRequest(turn)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.StopSequences == nil {
		t.Fatalf("expected explicit empty stop override to produce non-nil empty stop list")
	}
	if len(req.StopSequences) != 0 {
		t.Fatalf("expected stop override to clear chat stop, got %v", req.StopSequences)
	}
}

package openai

import (
	"testing"

	infengine "github.com/go-go-golems/geppetto/pkg/inference/engine"
	aisettings "github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	aisettingsopenai "github.com/go-go-golems/geppetto/pkg/steps/ai/settings/openai"
	"github.com/go-go-golems/geppetto/pkg/turns"
)

func boolPtr(v bool) *bool { return &v }

func newTestEngine(st *aisettings.InferenceSettings) *OpenAIEngine {
	return &OpenAIEngine{settings: st}
}

func TestMakeCompletionRequestFromTurnStructuredOutput(t *testing.T) {
	engine := "gpt-4o-mini"
	st := &aisettings.InferenceSettings{
		Client: &aisettings.ClientSettings{},
		OpenAI: &aisettingsopenai.Settings{},
		Chat: &aisettings.ChatSettings{
			Engine:                 &engine,
			StructuredOutputMode:   aisettings.StructuredOutputModeJSONSchema,
			StructuredOutputName:   "person",
			StructuredOutputSchema: `{"type":"object","properties":{"name":{"type":"string"}}}`,
			StructuredOutputStrict: boolPtr(true),
		},
	}
	tu := &turns.Turn{Blocks: []turns.Block{
		turns.NewUserTextBlock("return JSON"),
	}}

	e := newTestEngine(st)
	req, err := e.MakeCompletionRequestFromTurn(tu)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.ResponseFormat == nil || req.ResponseFormat.JSONSchema == nil {
		t.Fatalf("expected structured response_format to be set")
	}
	if req.ResponseFormat.Type != "json_schema" {
		t.Fatalf("expected response format type json_schema, got %q", req.ResponseFormat.Type)
	}
	if req.ResponseFormat.JSONSchema.Name != "person" {
		t.Fatalf("expected schema name person, got %q", req.ResponseFormat.JSONSchema.Name)
	}
}

func TestMakeCompletionRequestFromTurnStructuredOutputInvalidSchemaIgnoredWhenNotRequired(t *testing.T) {
	engine := "gpt-4o-mini"
	st := &aisettings.InferenceSettings{
		Client: &aisettings.ClientSettings{},
		OpenAI: &aisettingsopenai.Settings{},
		Chat: &aisettings.ChatSettings{
			Engine:                       &engine,
			StructuredOutputMode:         aisettings.StructuredOutputModeJSONSchema,
			StructuredOutputName:         "person",
			StructuredOutputSchema:       `{"type":"object",`,
			StructuredOutputRequireValid: false,
		},
	}
	tu := &turns.Turn{Blocks: []turns.Block{
		turns.NewUserTextBlock("return JSON"),
	}}

	e := newTestEngine(st)
	req, err := e.MakeCompletionRequestFromTurn(tu)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.ResponseFormat != nil {
		t.Fatalf("expected invalid schema to be ignored when require_valid=false")
	}
}

func TestMakeCompletionRequestFromTurnReasoningModelSanitizesPenalties(t *testing.T) {
	engine := "o3-mini"
	pp := 0.5
	fp := 0.3
	st := &aisettings.InferenceSettings{
		Client: &aisettings.ClientSettings{},
		OpenAI: &aisettingsopenai.Settings{
			PresencePenalty:  &pp,
			FrequencyPenalty: &fp,
		},
		Chat: &aisettings.ChatSettings{
			Engine: &engine,
		},
	}
	tu := &turns.Turn{Blocks: []turns.Block{
		turns.NewUserTextBlock("hello"),
	}}

	e := newTestEngine(st)
	req, err := e.MakeCompletionRequestFromTurn(tu)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Reasoning models should have penalties zeroed
	if req.PresencePenalty != 0 {
		t.Errorf("expected PresencePenalty=0 for reasoning model, got %v", req.PresencePenalty)
	}
	if req.FrequencyPenalty != 0 {
		t.Errorf("expected FrequencyPenalty=0 for reasoning model, got %v", req.FrequencyPenalty)
	}
	if req.Temperature != 0 {
		t.Errorf("expected Temperature=0 for reasoning model, got %v", req.Temperature)
	}
}

func TestMakeCompletionRequestFromTurnAddsChatThinkingControls(t *testing.T) {
	engine := "DeepSeek-V4-Pro"
	thinkingType := "enabled"
	effort := "max"
	st := &aisettings.InferenceSettings{
		Client: &aisettings.ClientSettings{},
		OpenAI: &aisettingsopenai.Settings{
			ThinkingType:        &thinkingType,
			ChatReasoningEffort: &effort,
		},
		Chat: &aisettings.ChatSettings{
			Engine: &engine,
		},
	}
	tu := &turns.Turn{Blocks: []turns.Block{
		turns.NewUserTextBlock("hello"),
	}}

	e := newTestEngine(st)
	req, err := e.MakeCompletionRequestFromTurn(tu)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.Thinking == nil || req.Thinking.Type != "enabled" {
		t.Fatalf("expected thinking enabled, got %#v", req.Thinking)
	}
	if req.ReasoningEffort != "max" {
		t.Fatalf("expected reasoning_effort=max, got %q", req.ReasoningEffort)
	}
}

func TestMakeCompletionRequestFromTurnThinkingDisabled(t *testing.T) {
	engine := "DeepSeek-V4-Pro"
	thinkingType := "disabled"
	st := &aisettings.InferenceSettings{
		Client: &aisettings.ClientSettings{},
		OpenAI: &aisettingsopenai.Settings{ThinkingType: &thinkingType},
		Chat:   &aisettings.ChatSettings{Engine: &engine},
	}
	tu := &turns.Turn{Blocks: []turns.Block{
		turns.NewUserTextBlock("hello"),
	}}

	e := newTestEngine(st)
	req, err := e.MakeCompletionRequestFromTurn(tu)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.Thinking == nil || req.Thinking.Type != "disabled" {
		t.Fatalf("expected thinking disabled, got %#v", req.Thinking)
	}
	if req.ReasoningEffort != "" {
		t.Fatalf("expected no reasoning effort, got %q", req.ReasoningEffort)
	}
}

func TestMakeCompletionRequestFromTurnOmitsChatThinkingByDefault(t *testing.T) {
	engine := "gpt-4o-mini"
	st := &aisettings.InferenceSettings{
		Client: &aisettings.ClientSettings{},
		OpenAI: &aisettingsopenai.Settings{},
		Chat:   &aisettings.ChatSettings{Engine: &engine},
	}
	tu := &turns.Turn{Blocks: []turns.Block{
		turns.NewUserTextBlock("hello"),
	}}

	e := newTestEngine(st)
	req, err := e.MakeCompletionRequestFromTurn(tu)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.Thinking != nil {
		t.Fatalf("expected thinking to be omitted, got %#v", req.Thinking)
	}
	if req.ReasoningEffort != "" {
		t.Fatalf("expected reasoning effort to be omitted, got %q", req.ReasoningEffort)
	}
}

func TestMakeCompletionRequestFromTurnInferenceOverridesChatThinkingControls(t *testing.T) {
	engine := "DeepSeek-V4-Pro"
	thinkingType := "enabled"
	effort := "high"
	turnThinkingType := "disabled"
	turnEffort := "xhigh"
	st := &aisettings.InferenceSettings{
		Client: &aisettings.ClientSettings{},
		OpenAI: &aisettingsopenai.Settings{
			ThinkingType:        &thinkingType,
			ChatReasoningEffort: &effort,
		},
		Chat: &aisettings.ChatSettings{Engine: &engine},
	}
	tu := &turns.Turn{Blocks: []turns.Block{
		turns.NewUserTextBlock("hello"),
	}}
	if err := infengine.KeyInferenceConfig.Set(&tu.Data, infengine.InferenceConfig{
		ThinkingType:    &turnThinkingType,
		ReasoningEffort: &turnEffort,
	}); err != nil {
		t.Fatalf("failed to set inference config: %v", err)
	}

	e := newTestEngine(st)
	req, err := e.MakeCompletionRequestFromTurn(tu)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.Thinking == nil || req.Thinking.Type != "disabled" {
		t.Fatalf("expected turn thinking disabled, got %#v", req.Thinking)
	}
	if req.ReasoningEffort != "max" {
		t.Fatalf("expected xhigh to normalize to max, got %q", req.ReasoningEffort)
	}
}

func TestMakeCompletionRequestFromTurnRejectsInvalidThinkingType(t *testing.T) {
	engine := "DeepSeek-V4-Pro"
	thinkingType := "sometimes"
	st := &aisettings.InferenceSettings{
		Client: &aisettings.ClientSettings{},
		OpenAI: &aisettingsopenai.Settings{ThinkingType: &thinkingType},
		Chat:   &aisettings.ChatSettings{Engine: &engine},
	}
	tu := &turns.Turn{Blocks: []turns.Block{
		turns.NewUserTextBlock("hello"),
	}}

	e := newTestEngine(st)
	_, err := e.MakeCompletionRequestFromTurn(tu)
	if err == nil {
		t.Fatal("expected invalid thinking type error")
	}
}

func TestMakeCompletionRequestFromTurnInferenceEmptyStopClearsChatStop(t *testing.T) {
	engine := "gpt-4o-mini"
	st := &aisettings.InferenceSettings{
		Client: &aisettings.ClientSettings{},
		OpenAI: &aisettingsopenai.Settings{},
		Chat: &aisettings.ChatSettings{
			Engine: &engine,
			Stop:   []string{"<END>"},
		},
	}
	tu := &turns.Turn{Blocks: []turns.Block{
		turns.NewUserTextBlock("hello"),
	}}
	if err := infengine.KeyInferenceConfig.Set(&tu.Data, infengine.InferenceConfig{Stop: []string{}}); err != nil {
		t.Fatalf("failed to set inference config: %v", err)
	}

	e := newTestEngine(st)
	req, err := e.MakeCompletionRequestFromTurn(tu)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.Stop == nil {
		t.Fatalf("expected explicit empty stop override to produce non-nil empty stop")
	}
	if len(req.Stop) != 0 {
		t.Fatalf("expected stop override to clear chat stop, got %v", req.Stop)
	}
}

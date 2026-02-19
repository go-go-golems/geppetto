package openai

import (
	"testing"

	aisettings "github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	aisettingsopenai "github.com/go-go-golems/geppetto/pkg/steps/ai/settings/openai"
	"github.com/go-go-golems/geppetto/pkg/turns"
)

func boolPtr(v bool) *bool { return &v }

func TestMakeCompletionRequestFromTurnStructuredOutput(t *testing.T) {
	engine := "gpt-4o-mini"
	st := &aisettings.StepSettings{
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

	req, err := MakeCompletionRequestFromTurn(st, tu)
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
	st := &aisettings.StepSettings{
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

	req, err := MakeCompletionRequestFromTurn(st, tu)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.ResponseFormat != nil {
		t.Fatalf("expected invalid schema to be ignored when require_valid=false")
	}
}

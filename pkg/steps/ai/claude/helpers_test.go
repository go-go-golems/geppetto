package claude

import (
	"testing"

	aisettings "github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	claudesettings "github.com/go-go-golems/geppetto/pkg/steps/ai/settings/claude"
	"github.com/go-go-golems/geppetto/pkg/turns"
)

func TestMakeMessageRequestFromTurnStructuredOutput(t *testing.T) {
	engine := "claude-sonnet-4-20250514"
	st := &aisettings.StepSettings{
		Client: &aisettings.ClientSettings{},
		Claude: &claudesettings.Settings{},
		Chat: &aisettings.ChatSettings{
			Engine:                 &engine,
			StructuredOutputMode:   aisettings.StructuredOutputModeJSONSchema,
			StructuredOutputName:   "person",
			StructuredOutputSchema: `{"type":"object","properties":{"name":{"type":"string"}}}`,
			Stream:                 true,
		},
	}
	tu := &turns.Turn{Blocks: []turns.Block{
		turns.NewUserTextBlock("return JSON"),
	}}

	req, err := MakeMessageRequestFromTurn(st, tu)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.OutputFormat == nil {
		t.Fatalf("expected output_format to be set")
	}
	if req.OutputFormat.Type != "json_schema" {
		t.Fatalf("expected output_format.type=json_schema, got %q", req.OutputFormat.Type)
	}
	if req.OutputFormat.Name != "person" {
		t.Fatalf("expected output_format.name=person, got %q", req.OutputFormat.Name)
	}
}

func TestMakeMessageRequestFromTurnStructuredOutputInvalidSchemaRequireValid(t *testing.T) {
	engine := "claude-sonnet-4-20250514"
	st := &aisettings.StepSettings{
		Client: &aisettings.ClientSettings{},
		Claude: &claudesettings.Settings{},
		Chat: &aisettings.ChatSettings{
			Engine:                       &engine,
			StructuredOutputMode:         aisettings.StructuredOutputModeJSONSchema,
			StructuredOutputName:         "person",
			StructuredOutputSchema:       `{"type":"object",`,
			StructuredOutputRequireValid: true,
			Stream:                       true,
		},
	}
	tu := &turns.Turn{Blocks: []turns.Block{
		turns.NewUserTextBlock("return JSON"),
	}}

	if _, err := MakeMessageRequestFromTurn(st, tu); err == nil {
		t.Fatalf("expected error when require_valid=true and schema JSON is invalid")
	}
}

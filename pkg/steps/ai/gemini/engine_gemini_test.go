package gemini

import (
	"testing"

	genai "github.com/google/generative-ai-go/genai"
	"github.com/invopop/jsonschema"
)

func TestConvertJSONSchemaToGenAI_ObjectType(t *testing.T) {
	s := &jsonschema.Schema{Type: "object"}
	gs := convertJSONSchemaToGenAI(s)
	if gs == nil {
		t.Fatalf("convertJSONSchemaToGenAI returned nil")
	}
	if gs.Type != genai.TypeObject {
		t.Fatalf("expected TypeObject, got %v", gs.Type)
	}
}

func TestConvertJSONSchemaToGenAI_ScalarTypes(t *testing.T) {
	cases := []struct {
		name     string
		inType   string
		expected genai.Type
	}{
		{"string", "string", genai.TypeString},
		{"number", "number", genai.TypeNumber},
		{"integer", "integer", genai.TypeInteger},
		{"boolean", "boolean", genai.TypeBoolean},
		{"array", "array", genai.TypeArray},
		{"object", "object", genai.TypeObject},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := &jsonschema.Schema{Type: tc.inType}
			gs := convertJSONSchemaToGenAI(s)
			if gs == nil {
				t.Fatalf("nil result for %s", tc.inType)
			}
			if gs.Type != tc.expected {
				t.Fatalf("expected %v, got %v", tc.expected, gs.Type)
			}
		})
	}
}

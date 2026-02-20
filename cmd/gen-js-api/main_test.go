package main

import (
	"strings"
	"testing"
)

func TestValidateSchemaOK(t *testing.T) {
	s := &schema{
		Enums: []enumSchema{
			{
				Name: "ToolChoice",
				Doc:  "How tools are selected",
				Values: []enumValueSchema{
					{JSKey: "AUTO", Value: "auto"},
					{JSKey: "NONE", Value: "none"},
				},
			},
		},
	}
	if err := validateSchema(s); err != nil {
		t.Fatalf("validateSchema() unexpected error: %v", err)
	}
}

func TestValidateSchemaRejectsDuplicateEnumNames(t *testing.T) {
	s := &schema{
		Enums: []enumSchema{
			{Name: "ToolChoice", Values: []enumValueSchema{{JSKey: "AUTO", Value: "auto"}}},
			{Name: "ToolChoice", Values: []enumValueSchema{{JSKey: "NONE", Value: "none"}}},
		},
	}
	if err := validateSchema(s); err == nil {
		t.Fatalf("expected duplicate enum name error")
	}
}

func TestValidateSchemaRejectsDuplicateJSKeyAndValue(t *testing.T) {
	t.Run("duplicate js_key", func(t *testing.T) {
		s := &schema{
			Enums: []enumSchema{
				{
					Name: "ToolChoice",
					Values: []enumValueSchema{
						{JSKey: "AUTO", Value: "auto"},
						{JSKey: "AUTO", Value: "none"},
					},
				},
			},
		}
		if err := validateSchema(s); err == nil {
			t.Fatalf("expected duplicate js_key error")
		}
	})

	t.Run("duplicate value", func(t *testing.T) {
		s := &schema{
			Enums: []enumSchema{
				{
					Name: "ToolChoice",
					Values: []enumValueSchema{
						{JSKey: "AUTO", Value: "auto"},
						{JSKey: "REQUIRED", Value: "auto"},
					},
				},
			},
		}
		if err := validateSchema(s); err == nil {
			t.Fatalf("expected duplicate value error")
		}
	})
}

func TestValidateSchemaRejectsInvalidIdentifiers(t *testing.T) {
	s := &schema{
		Enums: []enumSchema{
			{
				Name:   "Tool Choice",
				Values: []enumValueSchema{{JSKey: "AUTO", Value: "auto"}},
			},
		},
	}
	if err := validateSchema(s); err == nil {
		t.Fatalf("expected invalid identifier error")
	}
}

func TestRenderTemplateIncludesEnumValues(t *testing.T) {
	s := &schema{
		Enums: []enumSchema{
			{
				Name: "ToolChoice",
				Doc:  "How tools are selected",
				Values: []enumValueSchema{
					{JSKey: "AUTO", Value: "auto"},
					{JSKey: "NONE", Value: "none"},
				},
			},
		},
	}

	out, err := renderTemplate(goConstsTemplate, s)
	if err != nil {
		t.Fatalf("renderTemplate() error: %v", err)
	}
	src := string(out)
	if !strings.Contains(src, "func (m *moduleRuntime) installConsts") {
		t.Fatalf("expected installConsts in generated source")
	}
	if !strings.Contains(src, `m.mustSet(o, "AUTO", "auto")`) {
		t.Fatalf("expected enum value in generated source")
	}
	if !strings.Contains(src, `m.mustSet(constsObj, "ToolChoice", o)`) {
		t.Fatalf("expected enum object assignment in generated source")
	}
}

func TestImportedTurnsEnumsIncludesKeyGroups(t *testing.T) {
	ts := &turnsSchema{
		BlockKinds: []turnsBlockKindSchema{
			{Value: "user"},
			{Value: "llm_text"},
		},
		Keys: []turnsKeySchema{
			{Scope: "data", Value: "tool_config"},
			{Scope: "turn_meta", Value: "session_id"},
			{Scope: "block_meta", Value: "claude_original_content"},
		},
	}

	enums := importedTurnsEnums(ts)
	joined := &schema{Enums: enums}
	if err := validateSchema(joined); err != nil {
		t.Fatalf("imported turns enums should validate: %v", err)
	}

	src, err := renderTemplate(goConstsTemplate, joined)
	if err != nil {
		t.Fatalf("renderTemplate() error: %v", err)
	}
	out := string(src)
	if !strings.Contains(out, `m.mustSet(constsObj, "BlockMetadataKeys", o)`) {
		t.Fatalf("expected BlockMetadataKeys group in generated output")
	}
	if !strings.Contains(out, `m.mustSet(o, "CLAUDE_ORIGINAL_CONTENT", "claude_original_content")`) {
		t.Fatalf("expected block metadata key value in generated output")
	}
	if !strings.Contains(out, `m.mustSet(constsObj, "TurnDataKeys", o)`) {
		t.Fatalf("expected TurnDataKeys group in generated output")
	}
}

func TestMergeEnumsImportedOverridesBase(t *testing.T) {
	base := []enumSchema{
		{Name: "BlockKind", Values: []enumValueSchema{{JSKey: "USER", Value: "user"}}},
		{Name: "ToolChoice", Values: []enumValueSchema{{JSKey: "AUTO", Value: "auto"}}},
	}
	imported := []enumSchema{
		{Name: "BlockKind", Values: []enumValueSchema{{JSKey: "SYSTEM", Value: "system"}}},
	}
	merged := mergeEnums(base, imported)
	if len(merged) != 2 {
		t.Fatalf("unexpected merged length: %d", len(merged))
	}
	if merged[0].Name != "ToolChoice" {
		t.Fatalf("expected non-overridden base enum first, got %q", merged[0].Name)
	}
	if merged[1].Name != "BlockKind" || merged[1].Values[0].Value != "system" {
		t.Fatalf("expected imported BlockKind override, got %+v", merged[1])
	}
}

func TestValueToJSKey(t *testing.T) {
	cases := map[string]string{
		"session_id":              "SESSION_ID",
		"claude-original-content": "CLAUDE_ORIGINAL_CONTENT",
		"  tool_config  ":         "TOOL_CONFIG",
		"9lives":                  "K_9LIVES",
	}
	for in, want := range cases {
		if got := valueToJSKey(in); got != want {
			t.Fatalf("valueToJSKey(%q) = %q, want %q", in, got, want)
		}
	}
}

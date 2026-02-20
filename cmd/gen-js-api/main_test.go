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

package settings

import (
	"strings"
	"testing"
)

func TestChatSettingsIsStructuredOutputEnabled(t *testing.T) {
	s := &ChatSettings{StructuredOutputMode: StructuredOutputModeOff}
	if s.IsStructuredOutputEnabled() {
		t.Fatalf("expected mode off to be disabled")
	}
	s.StructuredOutputMode = StructuredOutputModeJSONSchema
	if !s.IsStructuredOutputEnabled() {
		t.Fatalf("expected json_schema mode to be enabled")
	}
}

func TestChatSettingsParseStructuredOutputSchema(t *testing.T) {
	s := &ChatSettings{
		StructuredOutputSchema: `{"type":"object","properties":{"name":{"type":"string"}}}`,
	}
	got, err := s.ParseStructuredOutputSchema()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == nil {
		t.Fatalf("expected parsed schema map")
	}
	if got["type"] != "object" {
		t.Fatalf("expected type=object, got %#v", got["type"])
	}
}

func TestChatSettingsParseStructuredOutputSchemaInvalid(t *testing.T) {
	s := &ChatSettings{
		StructuredOutputSchema: `{"type":"object",`,
	}
	if _, err := s.ParseStructuredOutputSchema(); err == nil {
		t.Fatalf("expected parse error for invalid schema JSON")
	}
}

func TestChatSettingsStructuredOutputConfig(t *testing.T) {
	s := &ChatSettings{
		StructuredOutputMode:   StructuredOutputModeJSONSchema,
		StructuredOutputName:   "person",
		StructuredOutputSchema: `{"type":"object","properties":{"name":{"type":"string"}}}`,
	}
	cfg, err := s.StructuredOutputConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg == nil || !cfg.IsEnabled() {
		t.Fatalf("expected enabled structured output config")
	}
	if cfg.Name != "person" {
		t.Fatalf("expected name=person, got %q", cfg.Name)
	}
	if !cfg.StrictOrDefault() {
		t.Fatalf("expected strict default true")
	}
}

func TestChatValueSection_HelpMentionsProfileFirstForEngineFlags(t *testing.T) {
	section, err := NewChatValueSection()
	if err != nil {
		t.Fatalf("NewChatValueSection: %v", err)
	}

	defs := section.GetDefinitions()
	engineDef, ok := defs.Get("ai-engine")
	if !ok {
		t.Fatalf("expected ai-engine definition")
	}
	if !strings.Contains(engineDef.Help, "Prefer profile selection") {
		t.Fatalf("expected ai-engine help to mention profile-first path, got %q", engineDef.Help)
	}

	apiTypeDef, ok := defs.Get("ai-api-type")
	if !ok {
		t.Fatalf("expected ai-api-type definition")
	}
	if !strings.Contains(apiTypeDef.Help, "Prefer profile selection") {
		t.Fatalf("expected ai-api-type help to mention profile-first path, got %q", apiTypeDef.Help)
	}
}

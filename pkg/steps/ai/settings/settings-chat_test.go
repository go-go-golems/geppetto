package settings

import "testing"

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

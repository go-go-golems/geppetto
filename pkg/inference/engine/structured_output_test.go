package engine

import "testing"

func TestStructuredOutputConfigValidate(t *testing.T) {
	cfg := StructuredOutputConfig{
		Mode:   StructuredOutputModeJSONSchema,
		Name:   "person",
		Schema: map[string]any{"type": "object"},
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}
}

func TestResolveStructuredOutputConfigPrefersTurnOverride(t *testing.T) {
	defaultCfg := &StructuredOutputConfig{
		Mode: StructuredOutputModeJSONSchema,
		Name: "default",
		Schema: map[string]any{
			"type": "object",
		},
	}
	overrideCfg := &StructuredOutputConfig{
		Mode: StructuredOutputModeJSONSchema,
		Name: "override",
		Schema: map[string]any{
			"type": "object",
		},
	}
	got := ResolveStructuredOutputConfig(defaultCfg, overrideCfg)
	if got != overrideCfg {
		t.Fatalf("expected turn override config to win")
	}
}

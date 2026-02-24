package middlewarecfg

import (
	"strings"
	"testing"

	"github.com/go-go-golems/glazed/pkg/cmds/fields"
)

func TestAdaptSchemaToGlazedSection_MapsRequiredDefaultEnumAndHelp(t *testing.T) {
	result, err := AdaptSchemaToGlazedSection(
		"middleware-agentmode",
		"Middleware Agent Mode",
		map[string]any{
			"type": "object",
			"required": []any{
				"mode",
			},
			"properties": map[string]any{
				"mode": map[string]any{
					"type":        "string",
					"description": "Operating mode for agent strategy.",
					"enum":        []any{"safe", "aggressive"},
					"default":     "safe",
				},
				"retries": map[string]any{
					"type":        "integer",
					"description": "Maximum retry count.",
					"default":     3,
				},
			},
		},
	)
	if err != nil {
		t.Fatalf("AdaptSchemaToGlazedSection returned error: %v", err)
	}
	if result == nil || result.Section == nil {
		t.Fatalf("expected non-nil adapter result with section")
	}
	if len(result.Limitations) != 0 {
		t.Fatalf("expected no limitations, got: %v", result.Limitations)
	}

	defs := result.Section.GetDefinitions()
	mode, ok := defs.Get("mode")
	if !ok {
		t.Fatalf("expected mode field")
	}
	if mode.Type != fields.TypeChoice {
		t.Fatalf("mode type mismatch: got=%q want=%q", mode.Type, fields.TypeChoice)
	}
	if !mode.Required {
		t.Fatalf("expected mode to be required")
	}
	if got, want := mode.Help, "Operating mode for agent strategy."; got != want {
		t.Fatalf("mode help mismatch: got=%q want=%q", got, want)
	}
	if mode.Default == nil {
		t.Fatalf("expected mode default")
	}
	if got, want := (*mode.Default).(string), "safe"; got != want {
		t.Fatalf("mode default mismatch: got=%q want=%q", got, want)
	}
	if len(mode.Choices) != 2 || mode.Choices[0] != "aggressive" || mode.Choices[1] != "safe" {
		t.Fatalf("mode choices mismatch: got=%v", mode.Choices)
	}

	retries, ok := defs.Get("retries")
	if !ok {
		t.Fatalf("expected retries field")
	}
	if retries.Type != fields.TypeInteger {
		t.Fatalf("retries type mismatch: got=%q want=%q", retries.Type, fields.TypeInteger)
	}
	if retries.Default == nil {
		t.Fatalf("expected retries default")
	}
	if got, want := (*retries.Default).(int64), int64(3); got != want {
		t.Fatalf("retries default mismatch: got=%d want=%d", got, want)
	}
}

func TestAdaptSchemaToGlazedSection_ReportsLimitationsForUnmappableConstructs(t *testing.T) {
	result, err := AdaptSchemaToGlazedSection(
		"middleware-nested",
		"Middleware Nested",
		map[string]any{
			"type":                 "object",
			"additionalProperties": true,
			"properties": map[string]any{
				"nested": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"inner": map[string]any{"type": "string"},
					},
				},
			},
		},
	)
	if err != nil {
		t.Fatalf("AdaptSchemaToGlazedSection returned error: %v", err)
	}
	if result == nil {
		t.Fatalf("expected adapter result")
	}
	if len(result.Limitations) == 0 {
		t.Fatalf("expected non-empty limitations")
	}
	joined := strings.Join(result.Limitations, " | ")
	if !strings.Contains(joined, "additionalProperties=true") {
		t.Fatalf("expected additionalProperties limitation, got: %v", result.Limitations)
	}
	if !strings.Contains(joined, "nested object fields") {
		t.Fatalf("expected nested-object limitation, got: %v", result.Limitations)
	}
}

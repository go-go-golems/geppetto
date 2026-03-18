package engineprofiles

import (
	"reflect"
	"testing"
)

func TestMergeEngineProfileStackLayers_RuntimeMergeRules(t *testing.T) {
	layers := []EngineProfileStackLayer{
		{
			RegistrySlug:      MustRegistrySlug("default"),
			EngineProfileSlug: MustEngineProfileSlug("base"),
			EngineProfile: &EngineProfile{
				Slug: MustEngineProfileSlug("base"),
				Runtime: RuntimeSpec{
					SystemPrompt: "base prompt",
					Tools:        []string{"calculator"},
					Middlewares: []MiddlewareUse{
						{Name: "agentmode", ID: "planner", Config: map[string]any{"mode": "plan"}},
						{Name: "logger", Config: map[string]any{"level": "info"}},
					},
				},
			},
		},
		{
			RegistrySlug:      MustRegistrySlug("default"),
			EngineProfileSlug: MustEngineProfileSlug("leaf"),
			EngineProfile: &EngineProfile{
				Slug: MustEngineProfileSlug("leaf"),
				Runtime: RuntimeSpec{
					Tools: []string{"search"},
					Middlewares: []MiddlewareUse{
						{Name: "agentmode", ID: "planner", Config: map[string]any{"mode": "act"}},
						{Name: "logger", Config: map[string]any{"level": "debug"}},
						{Name: "telemetry", ID: "trace"},
					},
				},
			},
		},
	}

	merged, err := MergeEngineProfileStackLayers(layers)
	if err != nil {
		t.Fatalf("MergeEngineProfileStackLayers failed: %v", err)
	}

	if got := merged.Runtime.SystemPrompt; got != "base prompt" {
		t.Fatalf("expected last non-empty system prompt, got %q", got)
	}
	if got := merged.Runtime.Tools; !reflect.DeepEqual(got, []string{"search"}) {
		t.Fatalf("expected tools replace-on-write, got %#v", got)
	}
	if got, want := len(merged.Runtime.Middlewares), 3; got != want {
		t.Fatalf("middleware length mismatch: got=%d want=%d", got, want)
	}
	if got := merged.Runtime.Middlewares[0].Config.(map[string]any)["mode"].(string); got != "act" {
		t.Fatalf("expected id-based middleware replacement, got %q", got)
	}
	if got := merged.Runtime.Middlewares[1].Config.(map[string]any)["level"].(string); got != "debug" {
		t.Fatalf("expected index-based middleware replacement, got %q", got)
	}
	if got := merged.Runtime.Middlewares[2].Name; got != "telemetry" {
		t.Fatalf("expected appended middleware, got %q", got)
	}
}

func TestMergeEngineProfileStackLayers_ExtensionMergeRules(t *testing.T) {
	layers := []EngineProfileStackLayer{
		{
			EngineProfile: &EngineProfile{
				Slug: MustEngineProfileSlug("base"),
				Extensions: map[string]any{
					"custom.config@v1": map[string]any{
						"scalar": "base",
						"nested": map[string]any{
							"foo": "base",
							"bar": "keep",
						},
						"items": []any{"a", "b"},
					},
					"custom.other@v1": "old",
				},
			},
		},
		{
			EngineProfile: &EngineProfile{
				Slug: MustEngineProfileSlug("leaf"),
				Extensions: map[string]any{
					"custom.config@v1": map[string]any{
						"nested": map[string]any{
							"foo": "leaf",
						},
						"items":  []any{"replaced"},
						"scalar": "leaf",
					},
					"custom.other@v1": []any{"new"},
				},
			},
		},
	}

	merged, err := MergeEngineProfileStackLayers(layers)
	if err != nil {
		t.Fatalf("MergeEngineProfileStackLayers failed: %v", err)
	}

	config := merged.Extensions["custom.config@v1"].(map[string]any)
	nested := config["nested"].(map[string]any)
	if got := nested["foo"].(string); got != "leaf" {
		t.Fatalf("expected nested foo override, got %q", got)
	}
	if got := nested["bar"].(string); got != "keep" {
		t.Fatalf("expected nested bar to persist, got %q", got)
	}
	if got := config["items"].([]any); !reflect.DeepEqual(got, []any{"replaced"}) {
		t.Fatalf("expected list replace behavior, got %#v", got)
	}
	if got := config["scalar"].(string); got != "leaf" {
		t.Fatalf("expected scalar replace behavior, got %q", got)
	}
	if got := merged.Extensions["custom.other@v1"].([]any); !reflect.DeepEqual(got, []any{"new"}) {
		t.Fatalf("expected scalar/list replacement for extension key, got %#v", got)
	}
}

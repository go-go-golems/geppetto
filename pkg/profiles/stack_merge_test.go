package profiles

import (
	"reflect"
	"testing"
)

func TestMergeProfileStackLayers_RuntimeMergeRules(t *testing.T) {
	layers := []ProfileStackLayer{
		{
			RegistrySlug: MustRegistrySlug("default"),
			ProfileSlug:  MustProfileSlug("base"),
			Profile: &Profile{
				Slug: MustProfileSlug("base"),
				Runtime: RuntimeSpec{
					StepSettingsPatch: map[string]any{
						"ai-chat": map[string]any{
							"ai-engine":   "gpt-4o-mini",
							"temperature": 0.2,
						},
					},
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
			RegistrySlug: MustRegistrySlug("default"),
			ProfileSlug:  MustProfileSlug("leaf"),
			Profile: &Profile{
				Slug: MustProfileSlug("leaf"),
				Runtime: RuntimeSpec{
					StepSettingsPatch: map[string]any{
						"ai-chat": map[string]any{
							"temperature": 0.8,
						},
						"summarize": map[string]any{
							"max-tokens": 256,
						},
					},
					SystemPrompt: "",
					Tools:        []string{"search"},
					Middlewares: []MiddlewareUse{
						{Name: "agentmode", ID: "planner", Config: map[string]any{"mode": "act"}},
						{Name: "logger", Config: map[string]any{"level": "debug"}},
						{Name: "telemetry", ID: "trace"},
					},
				},
			},
		},
	}

	merged, err := MergeProfileStackLayers(layers)
	if err != nil {
		t.Fatalf("MergeProfileStackLayers failed: %v", err)
	}

	aiChat := merged.Runtime.StepSettingsPatch["ai-chat"].(map[string]any)
	if got := aiChat["ai-engine"].(string); got != "gpt-4o-mini" {
		t.Fatalf("expected ai-engine to be preserved, got %q", got)
	}
	if got := aiChat["temperature"].(float64); got != 0.8 {
		t.Fatalf("expected temperature override, got %v", got)
	}
	if _, ok := merged.Runtime.StepSettingsPatch["summarize"]; !ok {
		t.Fatalf("expected summarize section from leaf layer")
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

func TestMergeProfileStackLayers_ExtensionMergeRules(t *testing.T) {
	layers := []ProfileStackLayer{
		{
			Profile: &Profile{
				Slug: MustProfileSlug("base"),
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
			Profile: &Profile{
				Slug: MustProfileSlug("leaf"),
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

	merged, err := MergeProfileStackLayers(layers)
	if err != nil {
		t.Fatalf("MergeProfileStackLayers failed: %v", err)
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

func TestMergeProfileStackLayers_PolicyRestrictiveRules(t *testing.T) {
	layers := []ProfileStackLayer{
		{
			Profile: &Profile{
				Slug: MustProfileSlug("base"),
				Policy: PolicySpec{
					AllowOverrides:      true,
					AllowedOverrideKeys: []string{"system_prompt", "tools"},
					DeniedOverrideKeys:  []string{"middlewares"},
				},
			},
		},
		{
			Profile: &Profile{
				Slug: MustProfileSlug("leaf"),
				Policy: PolicySpec{
					AllowOverrides:      true,
					AllowedOverrideKeys: []string{"tools", "step_settings_patch"},
					DeniedOverrideKeys:  []string{"tools"},
					ReadOnly:            true,
				},
			},
		},
	}

	merged, err := MergeProfileStackLayers(layers)
	if err != nil {
		t.Fatalf("MergeProfileStackLayers failed: %v", err)
	}

	if !merged.Policy.AllowOverrides {
		t.Fatalf("expected allow_overrides to remain true when all layers allow")
	}
	if !merged.Policy.ReadOnly {
		t.Fatalf("expected read_only to become true if any layer is read-only")
	}
	if got := merged.Policy.AllowedOverrideKeys; len(got) != 0 {
		t.Fatalf("expected allowed keys to be reduced by intersection+deny, got %#v", got)
	}
	if got, want := merged.Policy.DeniedOverrideKeys, []string{"middlewares", "tools"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("denied key union mismatch: got=%#v want=%#v", got, want)
	}
}

func TestMergeProfileStackLayers_PolicyAllowOverridesBecomesFalse(t *testing.T) {
	layers := []ProfileStackLayer{
		{Profile: &Profile{Slug: MustProfileSlug("base"), Policy: PolicySpec{AllowOverrides: true}}},
		{Profile: &Profile{Slug: MustProfileSlug("leaf"), Policy: PolicySpec{AllowOverrides: false}}},
	}

	merged, err := MergeProfileStackLayers(layers)
	if err != nil {
		t.Fatalf("MergeProfileStackLayers failed: %v", err)
	}
	if merged.Policy.AllowOverrides {
		t.Fatalf("expected allow_overrides to be false when any layer disables overrides")
	}
}

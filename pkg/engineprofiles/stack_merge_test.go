package engineprofiles

import (
	"reflect"
	"testing"

	aistepssettings "github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	aitypes "github.com/go-go-golems/geppetto/pkg/steps/ai/types"
)

func TestMergeEngineProfileStackLayers_InferenceSettingsMergeRules(t *testing.T) {
	layers := []EngineProfileStackLayer{
		{
			RegistrySlug:      MustRegistrySlug("default"),
			EngineProfileSlug: MustEngineProfileSlug("base"),
			EngineProfile: &EngineProfile{
				Slug:              MustEngineProfileSlug("base"),
				InferenceSettings: mustTestInferenceSettings(t, aitypes.ApiTypeOpenAI, "gpt-4o-mini"),
			},
		},
		{
			RegistrySlug:      MustRegistrySlug("default"),
			EngineProfileSlug: MustEngineProfileSlug("leaf"),
			EngineProfile: &EngineProfile{
				Slug: MustEngineProfileSlug("leaf"),
				InferenceSettings: func() *aistepssettings.InferenceSettings {
					ss := mustTestInferenceSettings(t, aitypes.ApiTypeOpenAI, "gpt-5-mini")
					temp := 0.2
					ss.Chat.Temperature = &temp
					return ss
				}(),
			},
		},
	}

	merged, err := MergeEngineProfileStackLayers(layers)
	if err != nil {
		t.Fatalf("MergeEngineProfileStackLayers failed: %v", err)
	}
	if merged.InferenceSettings == nil || merged.InferenceSettings.Chat == nil || merged.InferenceSettings.Chat.Engine == nil {
		t.Fatalf("expected merged inference settings")
	}
	if got := *merged.InferenceSettings.Chat.Engine; got != "gpt-5-mini" {
		t.Fatalf("expected leaf engine override, got %q", got)
	}
	if merged.InferenceSettings.Chat.Temperature == nil || *merged.InferenceSettings.Chat.Temperature != 0.2 {
		t.Fatalf("expected leaf temperature override")
	}
	if merged.InferenceSettings.Chat.ApiType == nil || *merged.InferenceSettings.Chat.ApiType != aitypes.ApiTypeOpenAI {
		t.Fatalf("expected base api type to persist")
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

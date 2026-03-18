package engineprofiles

import (
	"testing"

	aitypes "github.com/go-go-golems/geppetto/pkg/steps/ai/types"
)

func TestEngineProfileClone_DeepCopiesInferenceSettingsAndExtensions(t *testing.T) {
	original := &EngineProfile{
		Slug:              MustEngineProfileSlug("agent"),
		InferenceSettings: mustTestInferenceSettings(t, aitypes.ApiTypeOpenAI, "gpt-4o-mini"),
		Extensions: map[string]any{
			"webchat.starter_suggestions@v1": map[string]any{
				"items": []any{"a", "b"},
			},
		},
	}
	temperature := 0.3
	original.InferenceSettings.Chat.Temperature = &temperature

	cloned := original.Clone()
	if cloned == nil || cloned.InferenceSettings == nil || cloned.InferenceSettings.Chat == nil || cloned.InferenceSettings.Chat.Engine == nil {
		t.Fatalf("expected cloned inference settings")
	}

	*cloned.InferenceSettings.Chat.Engine = "gpt-5-mini"
	*cloned.InferenceSettings.Chat.Temperature = 0.9
	cloned.Extensions["webchat.starter_suggestions@v1"].(map[string]any)["items"] = []any{"new"}

	if got := *original.InferenceSettings.Chat.Engine; got != "gpt-4o-mini" {
		t.Fatalf("expected original engine unchanged, got %q", got)
	}
	if got := *original.InferenceSettings.Chat.Temperature; got != 0.3 {
		t.Fatalf("expected original temperature unchanged, got %v", got)
	}
	items := original.Extensions["webchat.starter_suggestions@v1"].(map[string]any)["items"].([]any)
	if len(items) != 2 {
		t.Fatalf("expected original extensions unchanged, got %#v", items)
	}
}

func TestRegistryClone_DeepCopiesProfiles(t *testing.T) {
	original := &EngineProfileRegistry{
		Slug:                     MustRegistrySlug("default"),
		DefaultEngineProfileSlug: MustEngineProfileSlug("agent"),
		Profiles: map[EngineProfileSlug]*EngineProfile{
			MustEngineProfileSlug("agent"): {
				Slug:              MustEngineProfileSlug("agent"),
				InferenceSettings: mustTestInferenceSettings(t, aitypes.ApiTypeOpenAI, "gpt-4o-mini"),
			},
		},
	}

	cloned := original.Clone()
	*cloned.Profiles[MustEngineProfileSlug("agent")].InferenceSettings.Chat.Engine = "gpt-5-mini"

	if got := *original.Profiles[MustEngineProfileSlug("agent")].InferenceSettings.Chat.Engine; got != "gpt-4o-mini" {
		t.Fatalf("expected original profile unchanged, got %q", got)
	}
}

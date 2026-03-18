package engineprofiles

import (
	"strings"
	"testing"

	aitypes "github.com/go-go-golems/geppetto/pkg/steps/ai/types"
)

func TestDecodeYAMLRegistries_RejectsLegacyProfilesMap(t *testing.T) {
	input := []byte(`default:
  inference_settings:
    chat:
      engine: gpt-4o-mini
`)

	_, err := DecodeYAMLRegistries(input, MustRegistrySlug("default"))
	if err == nil {
		t.Fatalf("expected legacy profile-map decode error")
	}
	if !strings.Contains(err.Error(), "legacy profile-map format is not supported") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDecodeYAMLRegistries_RejectsCanonicalBundle(t *testing.T) {
	input := []byte(`registries:
  default:
    slug: default
    profiles:
      default:
        slug: default
`)

	_, err := DecodeYAMLRegistries(input, MustRegistrySlug("default"))
	if err == nil {
		t.Fatalf("expected canonical bundle decode error")
	}
	if !strings.Contains(err.Error(), "top-level registries is not supported") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDecodeYAMLRegistries_EngineProfileSingleRegistry(t *testing.T) {
	input := []byte(`slug: default
profiles:
  default:
    slug: default
    inference_settings:
      chat:
        api_type: openai
        engine: gpt-4o-mini
`)

	regs, err := DecodeYAMLRegistries(input, MustRegistrySlug("unused"))
	if err != nil {
		t.Fatalf("DecodeYAMLRegistries failed: %v", err)
	}
	if len(regs) != 1 {
		t.Fatalf("expected 1 registry, got %d", len(regs))
	}
	profile := regs[0].Profiles[MustEngineProfileSlug("default")]
	if profile == nil || profile.InferenceSettings == nil || profile.InferenceSettings.Chat == nil || profile.InferenceSettings.Chat.Engine == nil {
		t.Fatalf("expected decoded inference settings")
	}
	if got := *profile.InferenceSettings.Chat.Engine; got != "gpt-4o-mini" {
		t.Fatalf("engine mismatch: got=%q", got)
	}
}

func TestEncodeDecodeYAMLRoundTrip_SingleRegistry(t *testing.T) {
	in := &EngineProfileRegistry{
		Slug:                     MustRegistrySlug("default"),
		DefaultEngineProfileSlug: MustEngineProfileSlug("default"),
		Profiles: map[EngineProfileSlug]*EngineProfile{
			MustEngineProfileSlug("default"): {
				Slug:              MustEngineProfileSlug("default"),
				InferenceSettings: mustTestInferenceSettings(t, aitypes.ApiTypeOpenAI, "gpt-4o-mini"),
				Extensions: map[string]any{
					"app.custom@v1": map[string]any{
						"items": []any{
							map[string]any{"enabled": true},
							"note",
						},
					},
				},
			},
		},
	}

	b, err := EncodeYAMLRegistries([]*EngineProfileRegistry{in})
	if err != nil {
		t.Fatalf("EncodeYAMLRegistries failed: %v", err)
	}
	if strings.Contains(string(b), "default_profile_slug:") {
		t.Fatalf("engine profile YAML should not serialize default_profile_slug")
	}
	out, err := DecodeYAMLRegistries(b, MustRegistrySlug("unused"))
	if err != nil {
		t.Fatalf("DecodeYAMLRegistries failed: %v", err)
	}
	profile := out[0].Profiles[MustEngineProfileSlug("default")]
	if profile == nil || profile.InferenceSettings == nil || profile.InferenceSettings.Chat == nil || profile.InferenceSettings.Chat.Engine == nil {
		t.Fatalf("missing inference settings after roundtrip")
	}
	if got := *profile.InferenceSettings.Chat.Engine; got != "gpt-4o-mini" {
		t.Fatalf("engine mismatch after roundtrip: got=%q", got)
	}
	if got := profile.Extensions["app.custom@v1"]; got == nil {
		t.Fatalf("expected extension payload after roundtrip")
	}
}

func TestEncodeYAMLRegistries_RejectsMultipleRegistries(t *testing.T) {
	_, err := EncodeYAMLRegistries([]*EngineProfileRegistry{
		{Slug: MustRegistrySlug("a")},
		{Slug: MustRegistrySlug("b")},
	})
	if err == nil {
		t.Fatalf("expected multiple registries error")
	}
}

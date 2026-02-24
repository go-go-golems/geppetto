package profiles

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDecodeYAMLRegistries_LegacyProfilesMap(t *testing.T) {
	path := filepath.Join("..", "..", "misc", "profiles.yaml")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read legacy fixture: %v", err)
	}

	regs, err := DecodeYAMLRegistries(b, MustRegistrySlug("default"))
	if err != nil {
		t.Fatalf("DecodeYAMLRegistries failed: %v", err)
	}
	if len(regs) != 1 {
		t.Fatalf("expected 1 registry, got %d", len(regs))
	}
	reg := regs[0]
	if reg.Slug != MustRegistrySlug("default") {
		t.Fatalf("unexpected registry slug: %q", reg.Slug)
	}
	if len(reg.Profiles) == 0 {
		t.Fatalf("expected converted profiles from legacy map")
	}
	if reg.Profiles[MustProfileSlug("gemini-2.5-pro")] == nil {
		t.Fatalf("expected gemini-2.5-pro profile")
	}
}

func TestDecodeYAMLRegistries_CanonicalFormat(t *testing.T) {
	input := []byte(`registries:
  default:
    slug: default
    default_profile_slug: default
    profiles:
      default:
        slug: default
        runtime:
          step_settings_patch:
            ai-chat:
              ai-engine: gpt-4o-mini
`)

	regs, err := DecodeYAMLRegistries(input, MustRegistrySlug("unused"))
	if err != nil {
		t.Fatalf("DecodeYAMLRegistries failed: %v", err)
	}
	if len(regs) != 1 {
		t.Fatalf("expected 1 registry, got %d", len(regs))
	}
	if regs[0].DefaultProfileSlug != MustProfileSlug("default") {
		t.Fatalf("default profile mismatch: %q", regs[0].DefaultProfileSlug)
	}
}

func TestEncodeDecodeYAMLRoundTrip(t *testing.T) {
	in := []*ProfileRegistry{{
		Slug:               MustRegistrySlug("default"),
		DefaultProfileSlug: MustProfileSlug("default"),
		Profiles: map[ProfileSlug]*Profile{
			MustProfileSlug("default"): {
				Slug: MustProfileSlug("default"),
				Runtime: RuntimeSpec{StepSettingsPatch: map[string]any{
					"ai-chat": map[string]any{"ai-engine": "gpt-4o-mini"},
				}},
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
	}}

	b, err := EncodeYAMLRegistries(in)
	if err != nil {
		t.Fatalf("EncodeYAMLRegistries failed: %v", err)
	}
	out, err := DecodeYAMLRegistries(b, MustRegistrySlug("unused"))
	if err != nil {
		t.Fatalf("DecodeYAMLRegistries failed: %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("expected 1 registry, got %d", len(out))
	}
	profile := out[0].Profiles[MustProfileSlug("default")]
	if profile == nil {
		t.Fatalf("missing default profile after roundtrip")
	}
	ext, ok := profile.Extensions["app.custom@v1"]
	if !ok {
		t.Fatalf("expected extension payload after roundtrip")
	}
	items := ext.(map[string]any)["items"].([]any)
	if got, want := len(items), 2; got != want {
		t.Fatalf("extension items length mismatch: got=%d want=%d", got, want)
	}
}

func TestEncodeDecodeYAML_PreservesUnknownExtensionKeys(t *testing.T) {
	in := []*ProfileRegistry{{
		Slug:               MustRegistrySlug("default"),
		DefaultProfileSlug: MustProfileSlug("default"),
		Profiles: map[ProfileSlug]*Profile{
			MustProfileSlug("default"): {
				Slug: MustProfileSlug("default"),
				Extensions: map[string]any{
					"thirdparty.feature@v1": map[string]any{
						"config": map[string]any{"flag": true},
					},
					"vendor.extra@v2": []any{"a", "b"},
				},
			},
		},
	}}

	b, err := EncodeYAMLRegistries(in)
	if err != nil {
		t.Fatalf("EncodeYAMLRegistries failed: %v", err)
	}
	out, err := DecodeYAMLRegistries(b, MustRegistrySlug("unused"))
	if err != nil {
		t.Fatalf("DecodeYAMLRegistries failed: %v", err)
	}

	profile := out[0].Profiles[MustProfileSlug("default")]
	if profile == nil {
		t.Fatalf("missing default profile after roundtrip")
	}
	if _, ok := profile.Extensions["thirdparty.feature@v1"]; !ok {
		t.Fatalf("missing thirdparty extension key after roundtrip")
	}
	if _, ok := profile.Extensions["vendor.extra@v2"]; !ok {
		t.Fatalf("missing vendor extension key after roundtrip")
	}
}

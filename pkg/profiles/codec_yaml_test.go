package profiles

import (
	"strings"
	"testing"
)

func TestDecodeYAMLRegistries_RejectsLegacyProfilesMap(t *testing.T) {
	input := []byte(`default:
  ai-chat:
    ai-engine: gpt-4o-mini
agent:
  ai-chat:
    ai-engine: gpt-4.1
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

func TestDecodeYAMLRegistries_RuntimeSingleRegistry(t *testing.T) {
	input := []byte(`slug: default
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
	if got, want := regs[0].Slug, MustRegistrySlug("default"); got != want {
		t.Fatalf("registry slug mismatch: got=%q want=%q", got, want)
	}
	if got, want := regs[0].DefaultProfileSlug, MustProfileSlug("default"); got != want {
		t.Fatalf("default profile mismatch: got=%q want=%q", got, want)
	}
}

func TestEncodeDecodeYAMLRoundTrip_SingleRegistry(t *testing.T) {
	in := &ProfileRegistry{
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
	}

	b, err := EncodeYAMLRegistries([]*ProfileRegistry{in})
	if err != nil {
		t.Fatalf("EncodeYAMLRegistries failed: %v", err)
	}
	if strings.Contains(string(b), "default_profile_slug:") {
		t.Fatalf("runtime YAML should not serialize default_profile_slug")
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

func TestEncodeYAMLRegistries_RejectsMultipleRegistries(t *testing.T) {
	_, err := EncodeYAMLRegistries([]*ProfileRegistry{
		{
			Slug:               MustRegistrySlug("default"),
			DefaultProfileSlug: MustProfileSlug("default"),
			Profiles: map[ProfileSlug]*Profile{
				MustProfileSlug("default"): {Slug: MustProfileSlug("default")},
			},
		},
		{
			Slug:               MustRegistrySlug("shared"),
			DefaultProfileSlug: MustProfileSlug("default"),
			Profiles: map[ProfileSlug]*Profile{
				MustProfileSlug("default"): {Slug: MustProfileSlug("default")},
			},
		},
	})
	if err == nil {
		t.Fatalf("expected multi-registry encode error")
	}
	if !strings.Contains(err.Error(), "exactly one registry") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEncodeDecodeYAML_PreservesUnknownExtensionKeys(t *testing.T) {
	in := &ProfileRegistry{
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
	}

	b, err := EncodeYAMLRegistries([]*ProfileRegistry{in})
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

func TestEncodeDecodeYAML_PreservesStackRefs(t *testing.T) {
	in := &ProfileRegistry{
		Slug:               MustRegistrySlug("default"),
		DefaultProfileSlug: MustProfileSlug("agent"),
		Profiles: map[ProfileSlug]*Profile{
			MustProfileSlug("provider-openai"): {
				Slug: MustProfileSlug("provider-openai"),
			},
			MustProfileSlug("model-gpt4o"): {
				Slug: MustProfileSlug("model-gpt4o"),
				Stack: []ProfileRef{
					{ProfileSlug: MustProfileSlug("provider-openai")},
				},
			},
			MustProfileSlug("agent"): {
				Slug: MustProfileSlug("agent"),
				Stack: []ProfileRef{
					{ProfileSlug: MustProfileSlug("model-gpt4o")},
					{RegistrySlug: MustRegistrySlug("shared"), ProfileSlug: MustProfileSlug("mw-observability")},
				},
			},
		},
	}

	b, err := EncodeYAMLRegistries([]*ProfileRegistry{in})
	if err != nil {
		t.Fatalf("EncodeYAMLRegistries failed: %v", err)
	}
	out, err := DecodeYAMLRegistries(b, MustRegistrySlug("unused"))
	if err != nil {
		t.Fatalf("DecodeYAMLRegistries failed: %v", err)
	}

	reg := out[0]
	model := reg.Profiles[MustProfileSlug("model-gpt4o")]
	if model == nil {
		t.Fatalf("missing model-gpt4o profile after roundtrip")
	}
	if got, want := len(model.Stack), 1; got != want {
		t.Fatalf("model stack length mismatch: got=%d want=%d", got, want)
	}
	if got := model.Stack[0].ProfileSlug; got != MustProfileSlug("provider-openai") {
		t.Fatalf("model stack profile mismatch: got=%q", got)
	}
	if got := model.Stack[0].RegistrySlug; got != "" {
		t.Fatalf("model stack registry should be empty (same registry), got=%q", got)
	}

	agent := reg.Profiles[MustProfileSlug("agent")]
	if agent == nil {
		t.Fatalf("missing agent profile after roundtrip")
	}
	if got, want := len(agent.Stack), 2; got != want {
		t.Fatalf("agent stack length mismatch: got=%d want=%d", got, want)
	}
	if got := agent.Stack[0].ProfileSlug; got != MustProfileSlug("model-gpt4o") {
		t.Fatalf("agent stack first ref profile mismatch: got=%q", got)
	}
	if got := agent.Stack[1].RegistrySlug; got != MustRegistrySlug("shared") {
		t.Fatalf("agent stack second ref registry mismatch: got=%q", got)
	}
	if got := agent.Stack[1].ProfileSlug; got != MustProfileSlug("mw-observability") {
		t.Fatalf("agent stack second ref profile mismatch: got=%q", got)
	}
}

package engineprofiles

import (
	"strings"
	"testing"
)

func TestDecodeYAMLRegistries_RejectsLegacyProfilesMap(t *testing.T) {
	input := []byte(`default:
  runtime:
    system_prompt: hello
agent:
  runtime:
    system_prompt: world
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
      system_prompt: hello
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
	if got := regs[0].Profiles[MustProfileSlug("default")].Runtime.SystemPrompt; got != "hello" {
		t.Fatalf("system prompt mismatch: got=%q", got)
	}
}

func TestEncodeDecodeYAMLRoundTrip_SingleRegistry(t *testing.T) {
	in := &ProfileRegistry{
		Slug:               MustRegistrySlug("default"),
		DefaultProfileSlug: MustProfileSlug("default"),
		Profiles: map[ProfileSlug]*Profile{
			MustProfileSlug("default"): {
				Slug: MustProfileSlug("default"),
				Runtime: RuntimeSpec{
					SystemPrompt: "hello",
					Tools:        []string{"search"},
				},
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
	if got := profile.Runtime.SystemPrompt; got != "hello" {
		t.Fatalf("system prompt mismatch: got=%q", got)
	}
	if got := len(profile.Runtime.Tools); got != 1 || profile.Runtime.Tools[0] != "search" {
		t.Fatalf("tools mismatch: %#v", profile.Runtime.Tools)
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
		{Slug: MustRegistrySlug("a")},
		{Slug: MustRegistrySlug("b")},
	})
	if err == nil {
		t.Fatalf("expected multiple registries error")
	}
}

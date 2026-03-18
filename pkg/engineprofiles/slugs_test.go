package engineprofiles

import (
	"encoding/json"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestParseRegistrySlug_RejectsEmpty(t *testing.T) {
	if _, err := ParseRegistrySlug("  "); err == nil {
		t.Fatalf("expected empty registry slug error")
	}
}

func TestRegistryAndProfileSlugs_JSONRoundTrip(t *testing.T) {
	in := struct {
		Registry RegistrySlug      `json:"registry"`
		Profile  EngineProfileSlug `json:"profile"`
	}{
		Registry: MustRegistrySlug("workspace"),
		Profile:  MustEngineProfileSlug("assistant-v2"),
	}

	b, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("json marshal failed: %v", err)
	}

	var out struct {
		Registry RegistrySlug      `json:"registry"`
		Profile  EngineProfileSlug `json:"profile"`
	}
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("json unmarshal failed: %v", err)
	}
	if out.Registry != "workspace" {
		t.Fatalf("registry mismatch: %q", out.Registry)
	}
	if out.Profile != "assistant-v2" {
		t.Fatalf("profile mismatch: %q", out.Profile)
	}
}

func TestRegistryAndProfileSlugs_YAMLRoundTrip(t *testing.T) {
	in := struct {
		Registry RegistrySlug      `yaml:"registry"`
		Profile  EngineProfileSlug `yaml:"profile"`
	}{
		Registry: MustRegistrySlug("workspace"),
		Profile:  MustEngineProfileSlug("assistant-v2"),
	}

	b, err := yaml.Marshal(in)
	if err != nil {
		t.Fatalf("yaml marshal failed: %v", err)
	}

	var out struct {
		Registry RegistrySlug      `yaml:"registry"`
		Profile  EngineProfileSlug `yaml:"profile"`
	}
	if err := yaml.Unmarshal(b, &out); err != nil {
		t.Fatalf("yaml unmarshal failed: %v", err)
	}
	if out.Registry != "workspace" {
		t.Fatalf("registry mismatch: %q", out.Registry)
	}
	if out.Profile != "assistant-v2" {
		t.Fatalf("profile mismatch: %q", out.Profile)
	}
}

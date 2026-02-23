package profiles

import (
	"encoding/json"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestParseRegistrySlug_Normalizes(t *testing.T) {
	s, err := ParseRegistrySlug("  My-Registry_01  ")
	if err != nil {
		t.Fatalf("ParseRegistrySlug failed: %v", err)
	}
	if got, want := s.String(), "my-registry_01"; got != want {
		t.Fatalf("registry slug mismatch, got %q want %q", got, want)
	}
}

func TestParseProfileSlug_RejectsInvalid(t *testing.T) {
	_, err := ParseProfileSlug("bad slug")
	if err == nil {
		t.Fatalf("expected invalid slug error")
	}
}

func TestParseRuntimeKey_RejectsEmpty(t *testing.T) {
	_, err := ParseRuntimeKey("  ")
	if err == nil {
		t.Fatalf("expected empty runtime key error")
	}
}

func TestSlugJSONRoundTrip(t *testing.T) {
	type payload struct {
		Registry RegistrySlug `json:"registry"`
		Profile  ProfileSlug  `json:"profile"`
		Runtime  RuntimeKey   `json:"runtime"`
	}
	in := payload{
		Registry: MustRegistrySlug("Default"),
		Profile:  MustProfileSlug("Agent"),
		Runtime:  MustRuntimeKey("Agent"),
	}

	b, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("json marshal failed: %v", err)
	}

	var out payload
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("json unmarshal failed: %v", err)
	}

	if out.Registry != "default" {
		t.Fatalf("registry mismatch: %q", out.Registry)
	}
	if out.Profile != "agent" {
		t.Fatalf("profile mismatch: %q", out.Profile)
	}
	if out.Runtime != "agent" {
		t.Fatalf("runtime mismatch: %q", out.Runtime)
	}
}

func TestSlugYAMLRoundTrip(t *testing.T) {
	type payload struct {
		Registry RegistrySlug `yaml:"registry"`
		Profile  ProfileSlug  `yaml:"profile"`
		Runtime  RuntimeKey   `yaml:"runtime"`
	}
	in := payload{
		Registry: MustRegistrySlug("Work"),
		Profile:  MustProfileSlug("assistant-v2"),
		Runtime:  MustRuntimeKey("assistant-v2"),
	}

	b, err := yaml.Marshal(in)
	if err != nil {
		t.Fatalf("yaml marshal failed: %v", err)
	}

	var out payload
	if err := yaml.Unmarshal(b, &out); err != nil {
		t.Fatalf("yaml unmarshal failed: %v", err)
	}

	if out.Registry != "work" {
		t.Fatalf("registry mismatch: %q", out.Registry)
	}
	if out.Profile != "assistant-v2" {
		t.Fatalf("profile mismatch: %q", out.Profile)
	}
	if out.Runtime != "assistant-v2" {
		t.Fatalf("runtime mismatch: %q", out.Runtime)
	}
}

func TestSlugJSONRejectsInvalid(t *testing.T) {
	type payload struct {
		Profile ProfileSlug `json:"profile"`
	}

	var out payload
	err := json.Unmarshal([]byte(`{"profile":"bad slug"}`), &out)
	if err == nil {
		t.Fatalf("expected invalid slug json error")
	}
}

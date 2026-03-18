package engineprofiles

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

func TestParseEngineProfileSlug_RejectsInvalid(t *testing.T) {
	_, err := ParseEngineProfileSlug("bad slug")
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
		Registry      RegistrySlug      `json:"registry"`
		EngineProfile EngineProfileSlug `json:"profile"`
		Runtime       RuntimeKey        `json:"runtime"`
	}
	in := payload{
		Registry:      MustRegistrySlug("Default"),
		EngineProfile: MustEngineProfileSlug("Agent"),
		Runtime:       MustRuntimeKey("Agent"),
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
	if out.EngineProfile != "agent" {
		t.Fatalf("profile mismatch: %q", out.EngineProfile)
	}
	if out.Runtime != "agent" {
		t.Fatalf("runtime mismatch: %q", out.Runtime)
	}
}

func TestSlugYAMLRoundTrip(t *testing.T) {
	type payload struct {
		Registry      RegistrySlug      `yaml:"registry"`
		EngineProfile EngineProfileSlug `yaml:"profile"`
		Runtime       RuntimeKey        `yaml:"runtime"`
	}
	in := payload{
		Registry:      MustRegistrySlug("Work"),
		EngineProfile: MustEngineProfileSlug("assistant-v2"),
		Runtime:       MustRuntimeKey("assistant-v2"),
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
	if out.EngineProfile != "assistant-v2" {
		t.Fatalf("profile mismatch: %q", out.EngineProfile)
	}
	if out.Runtime != "assistant-v2" {
		t.Fatalf("runtime mismatch: %q", out.Runtime)
	}
}

func TestSlugJSONRejectsInvalid(t *testing.T) {
	type payload struct {
		EngineProfile EngineProfileSlug `json:"profile"`
	}

	var out payload
	err := json.Unmarshal([]byte(`{"profile":"bad slug"}`), &out)
	if err == nil {
		t.Fatalf("expected invalid slug json error")
	}
}

func TestSlugTextRoundTrip(t *testing.T) {
	registryIn := MustRegistrySlug("default")
	registryText, err := registryIn.MarshalText()
	if err != nil {
		t.Fatalf("registry MarshalText failed: %v", err)
	}
	var registryOut RegistrySlug
	if err := registryOut.UnmarshalText(registryText); err != nil {
		t.Fatalf("registry UnmarshalText failed: %v", err)
	}
	if registryOut != "default" {
		t.Fatalf("registry text round-trip mismatch: %q", registryOut)
	}

	profileIn := MustEngineProfileSlug("agent")
	profileText, err := profileIn.MarshalText()
	if err != nil {
		t.Fatalf("profile MarshalText failed: %v", err)
	}
	var profileOut EngineProfileSlug
	if err := profileOut.UnmarshalText(profileText); err != nil {
		t.Fatalf("profile UnmarshalText failed: %v", err)
	}
	if profileOut != "agent" {
		t.Fatalf("profile text round-trip mismatch: %q", profileOut)
	}

	runtimeIn := MustRuntimeKey("agent")
	runtimeText, err := runtimeIn.MarshalText()
	if err != nil {
		t.Fatalf("runtime MarshalText failed: %v", err)
	}
	var runtimeOut RuntimeKey
	if err := runtimeOut.UnmarshalText(runtimeText); err != nil {
		t.Fatalf("runtime UnmarshalText failed: %v", err)
	}
	if runtimeOut != "agent" {
		t.Fatalf("runtime text round-trip mismatch: %q", runtimeOut)
	}
}

func TestSlugTextRejectsInvalid(t *testing.T) {
	var profile EngineProfileSlug
	if err := profile.UnmarshalText([]byte("bad slug")); err == nil {
		t.Fatalf("expected invalid profile text error")
	}

	var runtime RuntimeKey
	if err := runtime.UnmarshalText([]byte(" ")); err == nil {
		t.Fatalf("expected invalid runtime key text error")
	}
}

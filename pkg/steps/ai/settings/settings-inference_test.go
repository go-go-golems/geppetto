package settings

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestInferenceSettingsUnmarshal_UsesAPIRoot(t *testing.T) {
	ss, err := NewInferenceSettings()
	if err != nil {
		t.Fatalf("NewInferenceSettings: %v", err)
	}

	input := `api:
  api_keys:
    gemini-api-key: test-gemini-key
  base_urls:
    gemini-base-url: https://example.test
chat:
  api_type: gemini
  engine: gemini-2.5-pro
`
	if err := yaml.NewDecoder(strings.NewReader(input)).Decode(ss); err != nil {
		t.Fatalf("Decode: %v", err)
	}

	if got := ss.API.APIKeys["gemini-api-key"]; got != "test-gemini-key" {
		t.Fatalf("expected gemini API key, got %q", got)
	}
	if got := ss.API.BaseUrls["gemini-base-url"]; got != "https://example.test" {
		t.Fatalf("expected gemini base URL, got %q", got)
	}
}

func TestInferenceSettingsUnmarshal_RejectsLegacyAPIKeysRoot(t *testing.T) {
	ss, err := NewInferenceSettings()
	if err != nil {
		t.Fatalf("NewInferenceSettings: %v", err)
	}

	input := `api_keys:
  api_keys:
    gemini-api-key: test-gemini-key
chat:
  api_type: gemini
  engine: gemini-2.5-pro
`
	err = yaml.NewDecoder(strings.NewReader(input)).Decode(ss)
	if err == nil {
		t.Fatal("expected legacy api_keys wrapper decode error")
	}
	if !strings.Contains(err.Error(), "rename it to inference_settings.api") {
		t.Fatalf("unexpected error: %v", err)
	}
}

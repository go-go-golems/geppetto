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

func TestInferenceSettingsUnmarshal_ModelInfo(t *testing.T) {
	ss, err := NewInferenceSettings()
	if err != nil {
		t.Fatalf("NewInferenceSettings: %v", err)
	}

	input := `chat:
  api_type: openai
  engine: gpt-4o-mini
model_info:
  id: gpt-4o-mini
  name: GPT-4o Mini
  reasoning: false
  input:
    - text
    - image
  context_window: 128000
  quality_high_watermark: 64000
  max_output_tokens: 16384
  cost:
    input: 0.15
    output: 0.60
    cache_read: 0.075
    cache_write: 0.30
  metadata:
    fine_tunable: true
`
	if err := yaml.NewDecoder(strings.NewReader(input)).Decode(ss); err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if ss.ModelInfo == nil {
		t.Fatal("expected model info")
	}
	if got := *ss.ModelInfo.ID; got != "gpt-4o-mini" {
		t.Fatalf("model id = %q", got)
	}
	if got := len(ss.ModelInfo.Input); got != 2 {
		t.Fatalf("input modalities = %d", got)
	}
	if got := ss.ModelInfo.EffectiveContextLimit(); got != 64000 {
		t.Fatalf("effective context = %d", got)
	}
	if ss.ModelInfo.Cost == nil || ss.ModelInfo.Cost.Output != 0.60 {
		t.Fatalf("cost = %#v", ss.ModelInfo.Cost)
	}

	clone := ss.Clone()
	*clone.ModelInfo.ID = "changed"
	if got := *ss.ModelInfo.ID; got != "gpt-4o-mini" {
		t.Fatalf("clone mutated original id: %q", got)
	}

	metadata := ss.GetMetadata()
	if got := metadata["ai-model-id"]; got != "gpt-4o-mini" {
		t.Fatalf("metadata ai-model-id = %v", got)
	}
	if got := metadata["ai-model-context-window"]; got != 128000 {
		t.Fatalf("metadata ai-model-context-window = %v", got)
	}
}

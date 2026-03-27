package openai_responses

import (
	"testing"

	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
)

func TestResponsesAPIKeyPrefersOpenResponsesKey(t *testing.T) {
	api := &settings.APISettings{
		APIKeys: map[string]string{
			"openai-api-key":           "openai-key",
			"openai-responses-api-key": "legacy-key",
			"open-responses-api-key":   "new-key",
		},
	}

	got := responsesAPIKey(api)
	if got != "new-key" {
		t.Fatalf("expected open-responses key, got %q", got)
	}
}

func TestResponsesAPIKeyFallsBackToLegacyAlias(t *testing.T) {
	api := &settings.APISettings{
		APIKeys: map[string]string{
			"openai-api-key":           "openai-key",
			"openai-responses-api-key": "legacy-key",
		},
	}

	got := responsesAPIKey(api)
	if got != "legacy-key" {
		t.Fatalf("expected legacy openai-responses key, got %q", got)
	}
}

func TestResponsesBaseURLPrefersOpenResponsesBaseURL(t *testing.T) {
	api := &settings.APISettings{
		BaseUrls: map[string]string{
			"openai-base-url":           "https://openai.example/v1",
			"openai-responses-base-url": "https://legacy.example/v1",
			"open-responses-base-url":   "https://new.example/v1",
		},
	}

	got := responsesBaseURL(api)
	if got != "https://new.example/v1" {
		t.Fatalf("expected open-responses base URL, got %q", got)
	}
}

func TestResponsesBaseURLFallsBackToOpenAIBaseURL(t *testing.T) {
	api := &settings.APISettings{
		BaseUrls: map[string]string{
			"openai-base-url": "https://openai.example/v1",
		},
	}

	got := responsesBaseURL(api)
	if got != "https://openai.example/v1" {
		t.Fatalf("expected openai fallback base URL, got %q", got)
	}
}

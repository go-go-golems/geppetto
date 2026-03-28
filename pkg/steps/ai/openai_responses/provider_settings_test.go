package openai_responses

import (
	"testing"

	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/types"
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

func TestResponsesEndpointBuildsProviderURL(t *testing.T) {
	api := &settings.APISettings{
		BaseUrls: map[string]string{
			"open-responses-base-url": "https://responses.example/v1/",
		},
	}

	got := responsesEndpoint(api, "responses/input_tokens")
	if got != "https://responses.example/v1/responses/input_tokens" {
		t.Fatalf("unexpected endpoint %q", got)
	}
}

func TestResponsesAPITypeNormalizesLegacyAliases(t *testing.T) {
	apiType := settingsWithAPIType("openai-responses")

	got := responsesAPIType(apiType)
	if got != types.ApiTypeOpenResponses {
		t.Fatalf("expected open-responses api type, got %q", got)
	}
}

func TestResponsesInferenceProviderUsesCanonicalUnderscoreName(t *testing.T) {
	apiType := settingsWithAPIType("openai")

	got := responsesInferenceProvider(apiType)
	if got != "open_responses" {
		t.Fatalf("expected open_responses provider, got %q", got)
	}
}

func settingsWithAPIType(apiType string) *settings.InferenceSettings {
	ret := &settings.InferenceSettings{
		Chat: &settings.ChatSettings{},
	}
	v := types.ApiType(apiType)
	ret.Chat.ApiType = &v
	return ret
}

package openai_responses

import (
	"strings"

	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/types"
)

func responsesAPIKey(api *settings.APISettings) string {
	if api == nil {
		return ""
	}
	for _, keyName := range []string{
		string(types.ApiTypeOpenResponses) + "-api-key",
		string(types.ApiTypeOpenAIResponses) + "-api-key",
		string(types.ApiTypeOpenAI) + "-api-key",
	} {
		if value, ok := api.APIKeys[keyName]; ok && strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func responsesBaseURL(api *settings.APISettings) string {
	baseURL := "https://api.openai.com/v1"
	if api == nil {
		return baseURL
	}
	for _, keyName := range []string{
		string(types.ApiTypeOpenResponses) + "-base-url",
		string(types.ApiTypeOpenAIResponses) + "-base-url",
		string(types.ApiTypeOpenAI) + "-base-url",
	} {
		if value, ok := api.BaseUrls[keyName]; ok && strings.TrimSpace(value) != "" {
			return value
		}
	}
	return baseURL
}

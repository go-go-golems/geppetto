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

func responsesEndpoint(api *settings.APISettings, path string) string {
	path = "/" + strings.TrimLeft(path, "/")
	return strings.TrimRight(responsesBaseURL(api), "/") + path
}

func responsesAPIType(s *settings.InferenceSettings) types.ApiType {
	if s == nil || s.Chat == nil || s.Chat.ApiType == nil {
		return types.ApiTypeOpenResponses
	}

	raw := strings.ToLower(strings.TrimSpace(string(*s.Chat.ApiType)))
	switch raw {
	case "", string(types.ApiTypeOpenAI), string(types.ApiTypeOpenAIResponses), string(types.ApiTypeOpenResponses):
		return types.ApiTypeOpenResponses
	default:
		return types.ApiType(raw)
	}
}

func responsesInferenceProvider(s *settings.InferenceSettings) string {
	return strings.ReplaceAll(string(responsesAPIType(s)), "-", "_")
}

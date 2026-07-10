package openai_responses

import (
	"fmt"
	"strings"

	"context"

	"github.com/go-go-golems/geppetto/pkg/security"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/credentials"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/types"
)

func resolveResponsesBearerToken(ctx context.Context, api *settings.APISettings, apiType types.ApiType, source credentials.BearerTokenSource) (string, error) {
	if source != nil {
		token, err := source.BearerToken(ctx, credentials.Request{Provider: string(apiType), BaseURL: responsesBaseURL(api)})
		if err != nil {
			return "", fmt.Errorf("resolve bearer credential for OpenAI Responses")
		}
		if strings.TrimSpace(token) == "" {
			return "", fmt.Errorf("bearer credential source returned an empty token for OpenAI Responses")
		}
		return token, nil
	}
	return responsesAPIKey(api), nil
}

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

func responsesOutboundURLOptions(api *settings.APISettings) security.OutboundURLOptions {
	return settings.OutboundURLOptionsForKeys(
		api,
		string(types.ApiTypeOpenResponses),
		string(types.ApiTypeOpenAIResponses),
		string(types.ApiTypeOpenAI),
	)
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

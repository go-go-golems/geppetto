package openai_responses

import (
	"strings"

	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/types"
)

func responsesAPIType(s *settings.InferenceSettings) types.ApiType {
	if s == nil || s.Chat == nil || s.Chat.ApiType == nil {
		return types.ApiTypeOpenResponses
	}

	raw := strings.ToLower(strings.TrimSpace(string(*s.Chat.ApiType)))
	switch raw {
	case "", string(types.ApiTypeOpenResponses):
		return types.ApiTypeOpenResponses
	default:
		return types.ApiType(raw)
	}
}

func responsesInferenceProvider(s *settings.InferenceSettings) string {
	return strings.ReplaceAll(string(responsesAPIType(s)), "-", "_")
}

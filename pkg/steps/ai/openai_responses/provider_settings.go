package openai_responses

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/go-go-golems/geppetto/pkg/security"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/credentials"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/types"
)

func resolveResponsesBearerToken(ctx context.Context, api *settings.APISettings, apiType types.ApiType, source credentials.BearerTokenSource) (string, error) {
	slot, err := resolveResponsesSlot(api, source != nil)
	if err != nil {
		return "", err
	}
	if source != nil {
		token, err := source.BearerToken(ctx, credentials.Request{Provider: string(apiType), BaseURL: slot.baseURL})
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return "", err
			}
			return "", fmt.Errorf("resolve bearer credential for OpenAI Responses")
		}
		if strings.TrimSpace(token) == "" {
			return "", fmt.Errorf("bearer credential source returned an empty token for OpenAI Responses")
		}
		return token, nil
	}
	return slot.key, nil
}

// responsesSlotChain is the precedence of the settings slots a Responses
// engine reads: open-responses (current name), openai-responses (legacy
// alias), openai (fallback).
var responsesSlotChain = []types.ApiType{types.ApiTypeOpenResponses, types.ApiTypeOpenAIResponses, types.ApiTypeOpenAI}

// openAIDefaultBaseURL is OpenAI's API root, the default only for the
// OpenAI-named slots.
const openAIDefaultBaseURL = "https://api.openai.com/v1"

// responsesSlot is the key and base URL of one settings slot, resolved as a
// pair so a key is never combined with another slot's URL.
type responsesSlot struct {
	apiType types.ApiType
	key     string
	baseURL string
}

// resolveResponsesSlot picks the slot first and then reads both values from
// it (GEPPETTO-CREDSCOPE-001):
//
//   - the slot is the first in responsesSlotChain with an API key; when a
//     token source supplies the credential (sourced) or no slot has a key,
//     the first with a base URL; otherwise openai;
//   - its base URL is its own; openai-responses, an alias of OpenAI's
//     API, may use openai-base-url; the OpenAI-named slots default to
//     api.openai.com;
//   - an open-responses key without open-responses-base-url is an error: it
//     may belong to any Open Responses provider, so no other slot's URL or
//     OpenAI's default may receive it.
func resolveResponsesSlot(api *settings.APISettings, sourced bool) (responsesSlot, error) {
	if api == nil {
		return responsesSlot{apiType: types.ApiTypeOpenAI, baseURL: openAIDefaultBaseURL}, nil
	}
	get := func(m map[string]string, t types.ApiType, suffix string) string {
		return strings.TrimSpace(m[string(t)+suffix])
	}
	first := func(m map[string]string, suffix string) (types.ApiType, bool) {
		for _, t := range responsesSlotChain {
			if get(m, t, suffix) != "" {
				return t, true
			}
		}
		return "", false
	}
	slot, ok := first(api.APIKeys, "-api-key")
	if sourced || !ok {
		// The credential is not a settings key (token source, or a transport
		// adapter such as Codex that supplies its own), so the slot is chosen
		// by its URL.
		if slot, ok = first(api.BaseUrls, "-base-url"); !ok {
			slot = types.ApiTypeOpenAI
		}
	}
	out := responsesSlot{apiType: slot, key: get(api.APIKeys, slot, "-api-key"), baseURL: get(api.BaseUrls, slot, "-base-url")}
	if out.baseURL == "" && slot == types.ApiTypeOpenAIResponses {
		out.baseURL = get(api.BaseUrls, types.ApiTypeOpenAI, "-base-url")
	}
	if out.baseURL == "" && slot != types.ApiTypeOpenResponses {
		out.baseURL = openAIDefaultBaseURL
	}
	if out.baseURL == "" {
		return out, fmt.Errorf("open-responses-api-key is set but open-responses-base-url is not; the key is not sent to another slot's URL or to OpenAI's default")
	}
	return out, nil
}

// responsesAPIKey is the key of the resolved slot ("" when unresolvable).
func responsesAPIKey(api *settings.APISettings) string {
	slot, err := resolveResponsesSlot(api, false)
	if err != nil {
		return ""
	}
	return slot.key
}

// responsesBaseURL is the base URL of the slot a static key would use
// ("" when unresolvable). Engines with a token source call
// resolveResponsesSlot(api, true) instead.
func responsesBaseURL(api *settings.APISettings) string {
	slot, err := resolveResponsesSlot(api, false)
	if err != nil {
		return ""
	}
	return slot.baseURL
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

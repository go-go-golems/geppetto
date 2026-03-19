package openai_responses

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/go-go-golems/geppetto/pkg/inference/tokencount"
	"github.com/go-go-golems/geppetto/pkg/security"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/types"
	"github.com/go-go-golems/geppetto/pkg/turns"
	"github.com/pkg/errors"
)

type TokenCounter struct {
	settings *settings.InferenceSettings
}

type inputTokensRequest struct {
	Model             string           `json:"model"`
	Input             []responsesInput `json:"input"`
	Text              *responsesText   `json:"text,omitempty"`
	Reasoning         *reasoningParam  `json:"reasoning,omitempty"`
	Tools             []any            `json:"tools,omitempty"`
	ToolChoice        any              `json:"tool_choice,omitempty"`
	ParallelToolCalls *bool            `json:"parallel_tool_calls,omitempty"`
}

func NewTokenCounter(s *settings.InferenceSettings) *TokenCounter {
	return &TokenCounter{settings: s}
}

func (tc *TokenCounter) CountTurn(ctx context.Context, t *turns.Turn) (*tokencount.Result, error) {
	if tc == nil || tc.settings == nil {
		return nil, errors.New("openai token count: settings are required")
	}

	engine := &Engine{settings: tc.settings}
	reqBody, err := engine.buildResponsesRequest(t)
	if err != nil {
		return nil, err
	}
	if err := engine.attachToolsToResponsesRequest(ctx, t, &reqBody); err != nil {
		return nil, err
	}

	countReq := inputTokensRequest{
		Model:             reqBody.Model,
		Input:             reqBody.Input,
		Text:              reqBody.Text,
		Reasoning:         reqBody.Reasoning,
		Tools:             reqBody.Tools,
		ToolChoice:        reqBody.ToolChoice,
		ParallelToolCalls: reqBody.ParallelToolCalls,
	}

	payload, err := json.Marshal(countReq)
	if err != nil {
		return nil, err
	}

	baseURL := "https://api.openai.com/v1"
	apiKey := ""
	if tc.settings.API != nil {
		if v, ok := tc.settings.API.BaseUrls["openai-base-url"]; ok && strings.TrimSpace(v) != "" {
			baseURL = v
		}
		if v, ok := tc.settings.API.APIKeys["openai-api-key"]; ok {
			apiKey = v
		}
		if v, ok := tc.settings.API.APIKeys[string(types.ApiTypeOpenAIResponses)+"-api-key"]; ok && strings.TrimSpace(v) != "" {
			apiKey = v
		}
	}
	if strings.TrimSpace(apiKey) == "" {
		return nil, errors.New("openai token count: no openai api key configured")
	}

	url := strings.TrimRight(baseURL, "/") + "/responses/input_tokens"
	if err := security.ValidateOutboundURL(url, security.OutboundURLOptions{AllowHTTP: false}); err != nil {
		return nil, errors.Wrap(err, "invalid openai token count URL")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	httpClient := http.DefaultClient
	if tc.settings.Client != nil && tc.settings.Client.HTTPClient != nil {
		httpClient = tc.settings.Client.HTTPClient
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var m map[string]any
		_ = json.Unmarshal(body, &m)
		return nil, fmt.Errorf("openai token count error: status=%d body=%v", resp.StatusCode, m)
	}

	inputTokens, err := parseOpenAIInputTokens(body)
	if err != nil {
		return nil, err
	}

	provider := string(types.ApiTypeOpenAI)
	if tc.settings.Chat != nil && tc.settings.Chat.ApiType != nil && strings.TrimSpace(string(*tc.settings.Chat.ApiType)) != "" {
		provider = string(*tc.settings.Chat.ApiType)
	}

	return &tokencount.Result{
		Provider:    provider,
		Model:       countReq.Model,
		InputTokens: inputTokens,
		Source:      tokencount.SourceProviderAPI,
		Endpoint:    url,
		RequestKind: "responses_input_tokens",
	}, nil
}

func parseOpenAIInputTokens(body []byte) (int, error) {
	type directResponse struct {
		InputTokens *int `json:"input_tokens"`
	}

	var direct directResponse
	if err := json.Unmarshal(body, &direct); err == nil {
		if direct.InputTokens != nil {
			return *direct.InputTokens, nil
		}
	}

	var raw map[string]any
	if err := json.Unmarshal(body, &raw); err != nil {
		return 0, err
	}
	if v, ok := toInt(raw["input_tokens"]); ok {
		return v, nil
	}
	if response, ok := raw["response"].(map[string]any); ok {
		if v, ok := toInt(response["input_tokens"]); ok {
			return v, nil
		}
		if inputTokens, ok := response["input_tokens"].(map[string]any); ok {
			if v, ok := toInt(inputTokens["total"]); ok {
				return v, nil
			}
			if v, ok := toInt(inputTokens["input_tokens"]); ok {
				return v, nil
			}
		}
	}
	return 0, errors.New("openai token count: input_tokens missing from response")
}

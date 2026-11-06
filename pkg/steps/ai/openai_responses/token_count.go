package openai_responses

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/go-go-golems/geppetto/pkg/inference/tokencount"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/connection"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/go-go-golems/geppetto/pkg/turns"
	"github.com/pkg/errors"
)

type TokenCounter struct {
	settings      *settings.InferenceSettings
	connection    *connection.Connection
	connectionErr error
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

// TokenCounterOption supplies the same bound authority used by an engine.
type TokenCounterOption func(*TokenCounter)

func WithTokenCountConnection(c *connection.Connection) TokenCounterOption {
	return func(counter *TokenCounter) { counter.connection = c }
}
func NewTokenCounter(s *settings.InferenceSettings, opts ...TokenCounterOption) *TokenCounter {
	counter := &TokenCounter{settings: settings.PayloadSettings(s)}
	for _, opt := range opts {
		if opt != nil {
			opt(counter)
		}
	}
	if counter.connection == nil {
		counter.connection, counter.connectionErr = resolveResponsesConnection(s, nil, nil)
	}
	return counter
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

	if tc.connectionErr != nil {
		return nil, tc.connectionErr
	}
	req, err := tc.connection.NewRequest(ctx, http.MethodPost, "input-tokens", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := tc.connection.Do(req)
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

	return &tokencount.Result{
		Provider:    string(responsesAPIType(tc.settings)),
		Model:       countReq.Model,
		InputTokens: inputTokens,
		Source:      tokencount.SourceProviderAPI,
		Endpoint:    req.URL.String(),
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

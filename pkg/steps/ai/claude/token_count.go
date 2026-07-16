package claude

import (
	"context"
	"strings"

	"github.com/go-go-golems/geppetto/pkg/inference/engine"
	"github.com/go-go-golems/geppetto/pkg/inference/tokencount"
	"github.com/go-go-golems/geppetto/pkg/inference/tools"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/claude/api"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/credentials"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/go-go-golems/geppetto/pkg/turns"
	"github.com/pkg/errors"
)

type TokenCounter struct {
	settings          *settings.InferenceSettings
	bearerTokenSource credentials.BearerTokenSource
}

// TokenCounterOption configures request-time authentication for token counting.
type TokenCounterOption func(*TokenCounter)

// WithTokenCountBearerTokenSource supplies a Go-only request-time credential source.
func WithTokenCountBearerTokenSource(source credentials.BearerTokenSource) TokenCounterOption {
	return func(counter *TokenCounter) { counter.bearerTokenSource = source }
}

func NewTokenCounter(s *settings.InferenceSettings, opts ...TokenCounterOption) *TokenCounter {
	counter := &TokenCounter{settings: s}
	for _, opt := range opts {
		if opt != nil {
			opt(counter)
		}
	}
	return counter
}

func (tc *TokenCounter) CountTurn(ctx context.Context, t *turns.Turn) (*tokencount.Result, error) {
	if tc == nil || tc.settings == nil {
		return nil, errors.New("claude token count: settings are required")
	}
	engine_ := &ClaudeEngine{settings: tc.settings}
	req, err := engine_.MakeCountTokensRequestFromTurn(ctx, t)
	if err != nil {
		return nil, err
	}

	apiType := "claude"
	if tc.settings.Chat != nil && tc.settings.Chat.ApiType != nil && strings.TrimSpace(string(*tc.settings.Chat.ApiType)) != "" {
		apiType = string(*tc.settings.Chat.ApiType)
	}

	if tc.settings.API == nil {
		return nil, errors.New("claude token count: api settings are required")
	}
	baseURL, ok := tc.settings.API.BaseUrls["claude-base-url"]
	if !ok || strings.TrimSpace(baseURL) == "" {
		baseURL = "https://api.anthropic.com"
	}
	apiKey := tc.settings.API.APIKeys["claude-api-key"]
	bearerAuthorization := ""
	if tc.bearerTokenSource != nil {
		request := credentials.Request{Provider: apiType, BaseURL: strings.TrimRight(baseURL, "/")}
		resolved, resolveErr := tc.bearerTokenSource.BearerToken(ctx, request)
		if resolveErr != nil {
			return nil, errors.New("claude token count: resolve Anthropic gateway credential")
		}
		apiKey = resolved
		bearerAuthorization = resolved
	}
	if strings.TrimSpace(apiKey) == "" {
		return nil, errors.New("claude token count: no claude api key configured")
	}

	client := api.NewClient(apiKey, baseURL)
	client.SetBearerAuthorization(bearerAuthorization)
	client.SetOutboundURLOptions(settings.OutboundURLOptions(tc.settings.API, "claude"))
	if tc.settings.Client != nil && tc.settings.Client.HTTPClient != nil {
		client.SetHTTPClient(tc.settings.Client.HTTPClient)
	}
	res, err := client.CountTokens(ctx, req)
	if err != nil {
		return nil, err
	}

	return &tokencount.Result{
		Provider:    apiType,
		Model:       req.Model,
		InputTokens: res.InputTokens,
		Source:      tokencount.SourceProviderAPI,
		Endpoint:    strings.TrimRight(baseURL, "/") + "/v1/messages/count_tokens",
		RequestKind: "messages_count_tokens",
	}, nil
}

func (e *ClaudeEngine) MakeCountTokensRequestFromTurn(ctx context.Context, t *turns.Turn) (*api.MessageCountTokensRequest, error) {
	if e == nil || e.settings == nil {
		return nil, errors.New("claude token count: settings are required")
	}
	projection, err := e.buildMessageProjectionFromTurn(t)
	if err != nil {
		return nil, err
	}

	chatSettings := e.settings.Chat
	model := ""
	if chatSettings != nil && chatSettings.Engine != nil {
		model = *chatSettings.Engine
	}
	if strings.TrimSpace(model) == "" {
		return nil, errors.New("no engine specified")
	}

	req := &api.MessageCountTokensRequest{
		Model:    model,
		System:   projection.System,
		Messages: projection.Messages,
	}

	infCfg := engine.ResolveInferenceConfig(t, e.settings.Inference)
	if infCfg != nil && infCfg.ThinkingBudget != nil && *infCfg.ThinkingBudget > 0 {
		req.Thinking = &api.ThinkingParam{
			Type:         "enabled",
			BudgetTokens: *infCfg.ThinkingBudget,
		}
	}

	if claudeCfg := engine.ResolveClaudeInferenceConfig(t); claudeCfg != nil {
		if claudeCfg.UserID != nil {
			req.Metadata = &api.Metadata{UserID: *claudeCfg.UserID}
		}
	}

	if reg, ok := tools.RegistryFrom(ctx); ok && reg != nil {
		var claudeTools []api.Tool
		for _, tool := range reg.ListTools() {
			claudeTools = append(claudeTools, api.Tool{
				Name:        tool.Name,
				Description: tool.Description,
				InputSchema: tool.Parameters,
			})
		}
		req.Tools = claudeTools
	}

	return req, nil
}

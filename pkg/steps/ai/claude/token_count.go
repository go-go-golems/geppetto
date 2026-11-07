package claude

import (
	"context"
	"strings"

	"github.com/go-go-golems/geppetto/pkg/inference/engine"
	"github.com/go-go-golems/geppetto/pkg/inference/tokencount"
	"github.com/go-go-golems/geppetto/pkg/inference/tools"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/claude/api"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/connection"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/credentials"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/go-go-golems/geppetto/pkg/turns"
	"github.com/pkg/errors"
)

type TokenCounter struct {
	settings          *settings.InferenceSettings
	bearerTokenSource credentials.BearerTokenSource
	oauthBearerMode   bool
	connection        *connection.Connection
}

// TokenCounterOption configures request-time authentication for token counting.
type TokenCounterOption func(*TokenCounter)

// WithTokenCountBearerTokenSource supplies a Go-only request-time credential source.
func WithTokenCountBearerTokenSource(source credentials.BearerTokenSource) TokenCounterOption {
	return func(counter *TokenCounter) { counter.bearerTokenSource = source }
}

// WithTokenCountOAuthBearerTokenSource selects Anthropic subscription OAuth
// header semantics for token counting.
func WithTokenCountOAuthBearerTokenSource(source credentials.BearerTokenSource) TokenCounterOption {
	return func(counter *TokenCounter) {
		counter.bearerTokenSource = source
		counter.oauthBearerMode = true
	}
}

func NewTokenCounter(s *settings.InferenceSettings, opts ...TokenCounterOption) (*TokenCounter, error) {
	if s == nil || s.Chat == nil {
		return nil, errors.New("token count model settings are required")
	}
	var err error
	counter := &TokenCounter{settings: s}
	for _, opt := range opts {
		if opt != nil {
			opt(counter)
		}
	}
	if counter.connection == nil {
		counter.connection, err = resolveClaudeConnection(s, counter.bearerTokenSource, counter.oauthBearerMode, "")
	} else if counter.bearerTokenSource != nil {
		return nil, errors.New("connection already owns authentication")
	}
	if err != nil {
		return nil, err
	}
	counter.bearerTokenSource = nil
	counter.settings = settings.PayloadSettings(s)
	return counter, nil
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

	client := api.NewClient(tc.connection)
	res, err := client.CountTokens(ctx, req)
	if err != nil {
		return nil, err
	}

	return &tokencount.Result{
		Provider:    apiType,
		Model:       req.Model,
		InputTokens: res.InputTokens,
		Source:      tokencount.SourceProviderAPI,
		Endpoint:    tc.connection.BaseURL() + "/v1/messages/count_tokens",
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
	if infCfg != nil {
		thinking, err := thinkingParam(infCfg)
		if err != nil {
			return nil, err
		}
		req.Thinking = thinking
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

func WithTokenCountConnection(c *connection.Connection) TokenCounterOption {
	return func(counter *TokenCounter) { counter.connection = c }
}

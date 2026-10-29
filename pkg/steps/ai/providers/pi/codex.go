package pi

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"strings"

	"github.com/go-go-golems/geppetto/pkg/steps/ai/credentials"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/credentials/oauth"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/providers/openaicodex"
	"github.com/pkg/errors"
)

// ProviderCodex is Pi's provider id for ChatGPT subscription (Codex) models.
const ProviderCodex = "openai-codex"

// Pi's Codex OAuth client (pi-ai/dist/auth/oauth/openai-codex.js). Its refresh
// tokens rotate: the previous one is invalid after a refresh, so the new
// credential is always written back to auth.json.
var codexOAuth = oauth.Config{
	AuthorizationURL: "https://auth.openai.com/oauth/authorize",
	TokenURL:         "https://auth.openai.com/oauth/token",
	ClientID:         "app_EMoamEEZ73f0CkXaXp7hrann",
	RedirectURL:      "http://localhost:1455/auth/callback",
	Scopes:           []string{"openid", "profile", "email", "offline_access"},
}

const codexClaim = "https://api.openai.com/auth"

// WithCodexTokenURL points the Codex refresher at another token endpoint (tests).
func WithCodexTokenURL(u string) Option {
	return func(o *options) {
		cfg := codexOAuth
		cfg.TokenURL = u
		o.codexOAuth = &cfg
	}
}

func (a *AgentDir) codexConfig() oauth.Config {
	if a.opts.codexOAuth != nil {
		return *a.opts.codexOAuth
	}
	return codexOAuth
}

// codexSource is the openaicodex.Source over auth.json: a renewable bearer
// token plus the account id claim of the current access token.
func (a *AgentDir) codexSource(provider string) (openaicodex.Source, error) {
	e, ok, err := a.Auth.Entry(provider)
	if err != nil {
		return nil, err
	}
	if !ok || e.Type != KindOAuth {
		return nil, &NoCredentialError{Provider: provider}
	}
	client, err := oauth.NewClient(a.codexConfig(), oauth.WithRefreshTokenPolicy(oauth.RequireReplacementRefreshToken))
	if err != nil {
		return nil, err
	}
	store := &providerStore{file: a.Auth, provider: provider, onSave: func(fields map[string]any, c credentials.Credential) {
		if id, err := accountID(c.AccessToken); err == nil {
			fields["accountId"] = id
		}
	}}
	opts := []credentials.RenewableOption{}
	if a.opts.now != nil {
		opts = append(opts, credentials.WithClock(a.opts.now))
	}
	src, err := credentials.NewRenewableBearerTokenSource(store, oauthRefresher{client}, opts...)
	if err != nil {
		return nil, err
	}
	return codexSource{src}, nil
}

type oauthRefresher struct{ c *oauth.Client }

func (r oauthRefresher) Refresh(ctx context.Context, _ credentials.Request, prev credentials.Credential) (credentials.Credential, error) {
	return r.c.Refresh(ctx, prev)
}

type codexSource struct {
	src *credentials.RenewableBearerTokenSource
}

var _ openaicodex.UnauthorizedSource = codexSource{}

func (s codexSource) Credential(ctx context.Context, r credentials.Request) (openaicodex.Credential, error) {
	tok, err := s.src.BearerToken(ctx, r)
	if err != nil {
		return openaicodex.Credential{}, err
	}
	return codexCredential(tok)
}

func (s codexSource) CredentialAfterUnauthorized(ctx context.Context, r credentials.Request, rejected openaicodex.Credential) (openaicodex.Credential, error) {
	tok, err := s.src.BearerTokenAfterUnauthorized(ctx, r, rejected.BearerToken)
	if err != nil {
		return openaicodex.Credential{}, err
	}
	return codexCredential(tok)
}

func codexCredential(tok string) (openaicodex.Credential, error) {
	id, err := accountID(tok)
	if err != nil {
		return openaicodex.Credential{}, err
	}
	return openaicodex.Credential{BearerToken: tok, AccountID: id}, nil
}

// accountID reads the ChatGPT account id claim from a Codex access token. The
// token is the provider's own; its signature is the provider's to check.
func accountID(token string) (string, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return "", errors.New("the Codex access token is not a JWT")
	}
	raw, err := base64.RawURLEncoding.DecodeString(strings.TrimRight(parts[1], "="))
	if err != nil {
		return "", errors.New("the Codex access token has a malformed payload")
	}
	var claims map[string]json.RawMessage
	if err := json.Unmarshal(raw, &claims); err != nil {
		return "", errors.New("the Codex access token has a malformed payload")
	}
	var auth struct {
		AccountID string `json:"chatgpt_account_id"`
	}
	if c, ok := claims[codexClaim]; ok {
		_ = json.Unmarshal(c, &auth)
	}
	if auth.AccountID == "" {
		return "", errors.New("the Codex access token has no account id")
	}
	return auth.AccountID, nil
}

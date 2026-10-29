package pi

import (
	"context"
	"net/http"
	"strings"

	"github.com/go-go-golems/geppetto/pkg/inference/engine"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/claude"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/credentials"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/gemini"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/openai"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/openai_responses"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/providers/openaicodex"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/types"
	"github.com/pkg/errors"
)

// ErrUnknownModel: the reference is not in the catalog.
var ErrUnknownModel = errors.New("unknown Pi model")

// UnsupportedAPIError: the model's wire API has no Geppetto engine
// (bedrock-converse-stream, google-vertex, mistral-conversations, …).
type UnsupportedAPIError struct{ Model, API string }

func (e *UnsupportedAPIError) Error() string {
	return "model " + e.Model + " uses " + e.API + ", which Geppetto does not support"
}

// NoCredentialError: neither auth.json nor models.json has a credential for
// the provider.
type NoCredentialError struct{ Provider string }

func (e *NoCredentialError) Error() string {
	return "no credential for " + e.Provider + " (log in with pi, or set apiKey in models.json)"
}

// ResolveOptions shapes one engine.
type ResolveOptions struct {
	// Thinking is a Pi thinking level: off, minimal, low, medium, high, xhigh.
	// "" uses settings.json's defaultThinkingLevel.
	Thinking string
	// MaxTokens bounds one response; 0 uses the model's maxTokens. A caller
	// can also set it per turn with engine.KeyInferenceConfig.
	MaxTokens int
}

// Resolved is a model ready to run: settings without secrets, plus the
// credential wiring the engine needs.
type Resolved struct {
	Model    Model
	Settings *settings.InferenceSettings
	build    func(*settings.InferenceSettings) (engine.Engine, error)
}

// Engine creates an engine. Engines are cheap; create one per call.
func (r *Resolved) Engine() (engine.Engine, error) { return r.build(r.Settings) }

// Resolve turns "provider/id" into a Resolved.
func (a *AgentDir) Resolve(ctx context.Context, ref string, o ResolveOptions) (*Resolved, error) {
	m, ok := a.Catalog.Lookup(ref)
	if !ok {
		return nil, errors.Wrap(ErrUnknownModel, ref)
	}
	if !m.Supported {
		return nil, &UnsupportedAPIError{Model: m.Ref(), API: m.API}
	}
	s, err := settings.NewInferenceSettings()
	if err != nil {
		return nil, err
	}
	s.Chat.Temperature, s.Chat.TopP = nil, nil // Pi sends neither
	model := m.ID
	s.Chat.Engine = &model
	level := o.Thinking
	if level == "" {
		level = a.Settings.DefaultThinkingLevel
	}
	inf := thinkingConfig(m, level)
	if limit := firstPositive(o.MaxTokens, m.MaxTokens); limit > 0 {
		inf.MaxResponseTokens = &limit
	}
	s.Inference = &inf
	reasoning := m.Reasoning
	if m.API == APIOpenAICompletions {
		// Geppetto sends max_completion_tokens for reasoning models and
		// max_tokens otherwise; Pi's compat.maxTokensField picks the field.
		reasoning = m.Compat.MaxTokensField != "max_tokens"
	}
	s.ModelInfo = &settings.ModelInfo{ID: &model, Reasoning: &reasoning}

	headers, err := a.headers(ctx, m)
	if err != nil {
		return nil, err
	}
	s.Client.HTTPClient = withHeaders(a.opts.httpClient, headers)

	r := &Resolved{Model: m, Settings: s}
	defer func() {
		if a.opts.insecure && s.Chat.ApiType != nil {
			s.API.AllowHTTP[string(*s.Chat.ApiType)] = true
			s.API.AllowLocalNetworks[string(*s.Chat.ApiType)] = true
		}
	}()
	switch m.API {
	case APIOpenAICodexResponses:
		src, err := a.codexSource(m.Provider)
		if err != nil {
			return nil, err
		}
		t := types.ApiTypeOpenResponses
		s.Chat.ApiType = &t
		s.API.BaseUrls[string(t)+"-base-url"] = first(m.BaseURL, "https://chatgpt.com/backend-api")
		rt, err := openaicodex.RequestTransport(src, openaicodex.Options{Originator: "pi"})
		if err != nil {
			return nil, err
		}
		r.build = func(s *settings.InferenceSettings) (engine.Engine, error) {
			return openai_responses.NewEngine(s, openai_responses.WithRequestTransport(rt))
		}
		return r, nil
	}

	src, key, err := a.credential(ctx, m.Provider)
	if err != nil {
		return nil, err
	}
	switch m.API {
	case APIOpenAICompletions, APIOpenAIResponses:
		t := types.ApiTypeOpenAI
		if m.API == APIOpenAIResponses {
			t = types.ApiTypeOpenAIResponses
		}
		s.Chat.ApiType = &t
		s.API.BaseUrls[string(t)+"-base-url"] = first(m.BaseURL, "https://api.openai.com/v1")
		r.build = func(s *settings.InferenceSettings) (engine.Engine, error) {
			if t == types.ApiTypeOpenAIResponses {
				return openai_responses.NewEngine(s, openai_responses.WithBearerTokenSource(src))
			}
			return openai.NewOpenAIEngine(s, openai.WithBearerTokenSource(src))
		}
	case APIAnthropicMessages:
		t := types.ApiTypeClaude
		s.Chat.ApiType = &t
		s.API.BaseUrls[string(t)+"-base-url"] = first(m.BaseURL, "https://api.anthropic.com")
		if key != "" {
			// API keys go in x-api-key, as Anthropic-compatible providers expect.
			// The settings object is private to this Resolved and never serialized.
			s.API.APIKeys[string(t)+"-api-key"] = key
			r.build = func(s *settings.InferenceSettings) (engine.Engine, error) { return claude.NewClaudeEngine(s) }
		} else {
			r.build = func(s *settings.InferenceSettings) (engine.Engine, error) {
				return claude.NewClaudeEngine(s, claude.WithOAuthBearerTokenSource(src))
			}
		}
	case APIGoogleGenerativeAI:
		if key == "" {
			return nil, errors.New("gemini needs an API key")
		}
		t := types.ApiTypeGemini
		s.Chat.ApiType = &t
		s.API.APIKeys[string(t)+"-api-key"] = key
		if m.BaseURL != "" {
			s.API.BaseUrls[string(t)+"-base-url"] = m.BaseURL
		}
		r.build = func(s *settings.InferenceSettings) (engine.Engine, error) { return gemini.NewGeminiEngine(s) }
	}
	return r, nil
}

// credential returns a bearer source for the provider, and the key itself
// when it is a static API key (some engines take keys only as settings).
// Precedence follows Pi: auth.json, then models.json's apiKey.
func (a *AgentDir) credential(ctx context.Context, provider string) (credentials.BearerTokenSource, string, error) {
	e, ok, err := a.Auth.Entry(provider)
	if err != nil {
		return nil, "", err
	}
	var key string
	switch {
	case ok && e.Type == KindAPIKey && e.Key != "":
		if key, err = a.resolveValue(ctx, e.Key, e.Env); err != nil {
			return nil, "", errors.Wrapf(err, "%s key", provider)
		}
	case ok && e.Type == KindOAuth:
		// Outside Codex (its own source), an OAuth entry's access token is
		// the key: lunaroute stores its gateway key this way, and its refresh
		// returns the same credential.
		if e.Access == "" {
			return nil, "", &NoCredentialError{Provider: provider}
		}
		key = e.Access
	default:
		pc, ok := a.providers[provider]
		if !ok || pc.APIKey == "" {
			return nil, "", &NoCredentialError{Provider: provider}
		}
		if key, err = a.resolveValue(ctx, pc.APIKey, nil); err != nil {
			return nil, "", errors.Wrapf(err, "%s apiKey", provider)
		}
	}
	src, err := credentials.NewStaticBearerTokenSource(key)
	if err != nil {
		return nil, "", &NoCredentialError{Provider: provider}
	}
	return src, key, nil
}

func (a *AgentDir) headers(ctx context.Context, m Model) (map[string]string, error) {
	out := map[string]string{}
	for k, v := range a.opts.headers[m.Provider] {
		out[k] = v
	}
	for k, v := range m.Headers {
		r, err := a.resolveValue(ctx, v, nil)
		if err != nil {
			return nil, errors.Wrapf(err, "header %s of %s", k, m.Ref())
		}
		out[k] = r
	}
	return out, nil
}

// withHeaders returns a client that adds headers to every request, without
// overriding ones the engine set (Authorization, Content-Type).
func withHeaders(base *http.Client, headers map[string]string) *http.Client {
	if base == nil {
		base = &http.Client{}
	}
	if len(headers) == 0 {
		return base
	}
	c := *base
	next := c.Transport
	if next == nil {
		next = http.DefaultTransport
	}
	c.Transport = headerTransport{next: next, headers: headers}
	return &c
}

type headerTransport struct {
	next    http.RoundTripper
	headers map[string]string
}

func (t headerTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	r = r.Clone(r.Context())
	for k, v := range t.headers {
		if r.Header.Get(k) == "" && !strings.EqualFold(k, "authorization") {
			r.Header.Set(k, v)
		}
	}
	return t.next.RoundTrip(r)
}

func firstPositive(vals ...int) int {
	for _, v := range vals {
		if v > 0 {
			return v
		}
	}
	return 0
}

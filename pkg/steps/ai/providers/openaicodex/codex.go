// Package openaicodex provides the trusted route and credential middleware used
// to run the OpenAI Responses core against the Codex-specific transport.
package openaicodex

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-go-golems/geppetto/pkg/steps/ai/credentials"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/openai_responses"
	aitransport "github.com/go-go-golems/geppetto/pkg/steps/ai/transport"
)

const (
	// Provider is the provider identity supplied to the shared Responses core.
	Provider = "openai-codex"
	// ResponsesPath is the Codex-specific Responses endpoint path.
	ResponsesPath     = "/codex/responses"
	defaultOriginator = "geppetto"
	responsesBeta     = "responses=experimental"
)

// Credential is private runtime authentication material for a Codex request.
// It must not be serialized into settings, emitted to JavaScript, or formatted
// into errors.
type Credential struct {
	BearerToken string
	AccountID   string
}

// Source returns a current Codex credential for an already validated target.
type Source interface {
	Credential(context.Context, credentials.Request) (Credential, error)
}

// UnauthorizedSource optionally supplies a replacement credential after a
// rejected pre-stream request. The shared Responses core permits one replay.
type UnauthorizedSource interface {
	Source
	CredentialAfterUnauthorized(context.Context, credentials.Request, Credential) (Credential, error)
}

// Options identifies the application to the provider. Values are Go-only
// engine configuration, never profile or JavaScript configuration.
type Options struct {
	Originator string
	UserAgent  string
}

func (o Options) normalized() Options {
	o.Originator = strings.TrimSpace(o.Originator)
	if o.Originator == "" {
		o.Originator = defaultOriginator
	}
	o.UserAgent = strings.TrimSpace(o.UserAgent)
	return o
}

// RequestTransport returns the Codex adapter for openai_responses.NewEngine.
// It disables the core's ordinary bearer middleware so Codex headers can only
// be supplied by a typed Codex source.
func RequestTransport(source Source, options Options) (openai_responses.RequestTransport, error) {
	if source == nil {
		return openai_responses.RequestTransport{}, errors.New("codex credential source is required")
	}
	options = options.normalized()
	middleware := &credentialMiddleware{
		source: source,
		request: credentials.Request{
			Provider: Provider,
			BaseURL:  "https://chatgpt.com/backend-api",
		},
		originator: options.Originator,
		userAgent:  options.UserAgent,
	}
	return openai_responses.RequestTransport{
		Provider:             Provider,
		RouteResolver:        Route{},
		DisableDefaultBearer: true,
		HeaderRules: []aitransport.HeaderRule{
			{Name: "Authorization", Sensitive: true},
			{Name: "chatgpt-account-id", Sensitive: true},
			{Name: "originator"},
			{Name: "OpenAI-Beta"},
			{Name: "User-Agent"},
		},
		Middlewares: []aitransport.Middleware{middleware},
	}, nil
}

// Route resolves the fixed Codex response path without accepting a
// profile-configured suffix.
type Route struct{}

func (Route) Resolve(request aitransport.RouteRequest) (*url.URL, error) {
	if request.Operation() != "responses" {
		return nil, fmt.Errorf("unsupported Codex operation %q", request.Operation())
	}
	target := request.BaseURL()
	target.Path = strings.TrimRight(target.Path, "/") + ResponsesPath
	target.RawPath = ""
	return &target, nil
}

type credentialMiddleware struct {
	source     Source
	request    credentials.Request
	originator string
	userAgent  string
}

type attempt struct {
	credential  Credential
	replacement *Credential
}

func (m *credentialMiddleware) BeforeRequest(ctx context.Context, _ aitransport.RequestContext, previous any, headers aitransport.HeaderWriter) (any, error) {
	credential, err := m.credentialForAttempt(ctx, previous)
	if err != nil {
		return nil, err
	}
	if err := headers.Set("Authorization", "Bearer "+credential.BearerToken); err != nil {
		return nil, err
	}
	if err := headers.Set("chatgpt-account-id", credential.AccountID); err != nil {
		return nil, err
	}
	if err := headers.Set("originator", m.originator); err != nil {
		return nil, err
	}
	if err := headers.Set("OpenAI-Beta", responsesBeta); err != nil {
		return nil, err
	}
	if m.userAgent != "" {
		if err := headers.Set("User-Agent", m.userAgent); err != nil {
			return nil, err
		}
	}
	return &attempt{credential: credential}, nil
}

func (m *credentialMiddleware) credentialForAttempt(ctx context.Context, previous any) (Credential, error) {
	if prior, ok := previous.(*attempt); ok && prior.replacement != nil {
		return *prior.replacement, nil
	}
	if previous != nil {
		return Credential{}, errors.New("codex middleware received incompatible prior attempt")
	}
	credential, err := m.source.Credential(ctx, m.request)
	if err != nil {
		return Credential{}, errors.New("codex credential unavailable")
	}
	if !credential.usable() {
		return Credential{}, errors.New("codex credential unavailable")
	}
	return credential, nil
}

func (m *credentialMiddleware) AfterResponse(ctx context.Context, _ aitransport.RequestContext, previous any, response aitransport.ResponseMetadata) (aitransport.ResponseDecision, error) {
	if response.StatusCode != http.StatusUnauthorized || response.StreamStarted || !response.RetryEligible {
		return aitransport.Continue, nil
	}
	source, ok := m.source.(UnauthorizedSource)
	if !ok {
		return aitransport.Continue, nil
	}
	prior, ok := previous.(*attempt)
	if !ok || !prior.credential.usable() {
		return aitransport.Continue, nil
	}
	replacement, err := source.CredentialAfterUnauthorized(ctx, m.request, prior.credential)
	if err != nil || !replacement.usable() {
		return aitransport.Continue, errors.New("refresh Codex credential after provider unauthorized")
	}
	prior.replacement = &replacement
	return aitransport.Retry, nil
}

func (c Credential) usable() bool {
	return strings.TrimSpace(c.BearerToken) != "" && strings.TrimSpace(c.AccountID) != ""
}

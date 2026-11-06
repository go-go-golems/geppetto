package connection

import (
	"context"
	"net/http"
	"strings"

	"github.com/go-go-golems/geppetto/pkg/security"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/credentials"
	aitransport "github.com/go-go-golems/geppetto/pkg/steps/ai/transport"
	"github.com/pkg/errors"
)

// ErrAuthentication classifies a credential failure without exposing source errors.
var ErrAuthentication = errors.New("API authentication failed")

// Authentication is an opaque recipe for request-time authentication. Its zero
// value permits anonymous requests. Token material is never exposed by a getter.
type Authentication struct {
	rules       []aitransport.HeaderRule
	middlewares []aitransport.Middleware
	err         error
}

func (Authentication) String() string     { return "<API authentication>" }
func (a Authentication) GoString() string { return a.String() }

// Middleware installs the existing restricted provider header/response hooks.
// Hooks are trusted host code; they receive no mutable URL, body or HTTP client.
func Middleware(rules []aitransport.HeaderRule, middlewares ...aitransport.Middleware) Authentication {
	return Authentication{
		rules:       append([]aitransport.HeaderRule(nil), rules...),
		middlewares: append([]aitransport.Middleware(nil), middlewares...),
	}
}

// TokenHeaders maps credential header names to prefixes (for example
// Authorization: "Bearer ", or x-api-key: ""). All are sensitive.
type TokenHeaders map[string]string

// Token binds a renewable source to one or more authentication header carriers.
// The provider and base URL supplied to the source come from the connection,
// never the protocol's route or a fallback applied at request time.
func Token(source credentials.BearerTokenSource, headers TokenHeaders) Authentication {
	if source == nil || len(headers) == 0 {
		return Authentication{err: errors.New("API authentication needs a source and header carrier")}
	}
	carriers := make(TokenHeaders, len(headers))
	rules := make([]aitransport.HeaderRule, 0, len(headers))
	for name, prefix := range headers {
		carriers[name] = prefix
		rules = append(rules, aitransport.HeaderRule{Name: name, Sensitive: true})
	}
	return Middleware(rules, &tokenMiddleware{source: source, headers: carriers})
}

// StaticToken declares a static credential at the configuration boundary.
func StaticToken(key string, headers TokenHeaders) Authentication {
	source, err := credentials.NewStaticBearerTokenSource(key)
	if err != nil {
		return Authentication{err: errors.New("API credential is empty")}
	}
	return Token(source, headers)
}

func Bearer(source credentials.BearerTokenSource) Authentication {
	return Token(source, TokenHeaders{"Authorization": "Bearer "})
}

func StaticBearer(key string) Authentication {
	return StaticToken(key, TokenHeaders{"Authorization": "Bearer "})
}

type tokenMiddleware struct {
	source  credentials.BearerTokenSource
	headers TokenHeaders
}

var _ aitransport.Middleware = (*tokenMiddleware)(nil)

type tokenAttempt struct{ token, replacement string }

func (m *tokenMiddleware) BeforeRequest(ctx context.Context, route aitransport.RequestContext, previous aitransport.Attempt, headers aitransport.HeaderWriter) (aitransport.Attempt, error) {
	var token string
	if prior, ok := previous.(*tokenAttempt); ok {
		token = prior.replacement
	} else if previous != nil {
		return nil, errors.New("incompatible API authentication attempt")
	} else {
		// The source sees the binding's base URL through a private context value.
		request, ok := ctx.Value(credentialContextKey{}).(credentials.Request)
		if !ok {
			return nil, errors.New("API authentication requires a bound connection")
		}
		var err error
		token, err = m.source.BearerToken(ctx, request)
		if err != nil {
			return nil, authError(ctx, err, "resolve API credential")
		}
	}
	if strings.TrimSpace(token) == "" {
		return nil, errors.Wrap(ErrAuthentication, "API credential source returned an empty token")
	}
	for name, prefix := range m.headers {
		if err := headers.Set(name, prefix+token); err != nil {
			return nil, err
		}
	}
	return &tokenAttempt{token: token}, nil
}

func (m *tokenMiddleware) AfterResponse(ctx context.Context, _ aitransport.RequestContext, attempt aitransport.Attempt, response aitransport.ResponseMetadata) (aitransport.ResponseDecision, error) {
	if response.StatusCode != http.StatusUnauthorized || response.StreamStarted || !response.RetryEligible {
		return aitransport.Continue, nil
	}
	source, ok := m.source.(credentials.UnauthorizedBearerTokenSource)
	if !ok {
		return aitransport.Continue, nil
	}
	prior, ok := attempt.(*tokenAttempt)
	if !ok {
		return aitransport.Continue, errors.New("invalid API authentication attempt")
	}
	request, _ := ctx.Value(credentialContextKey{}).(credentials.Request)
	token, err := source.BearerTokenAfterUnauthorized(ctx, request, prior.token)
	if err != nil {
		return aitransport.Continue, authError(ctx, err, "refresh API credential after unauthorized")
	}
	if strings.TrimSpace(token) == "" {
		return aitransport.Continue, errors.Wrap(ErrAuthentication, "refreshed API credential is empty")
	}
	prior.replacement = token
	return aitransport.Retry, nil
}

type credentialContextKey struct{}

func authError(ctx context.Context, err error, message string) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	// Preserve cancellation and scope refusal identities without source errors
	// that might include credential bytes.
	for _, sentinel := range []error{context.Canceled, context.DeadlineExceeded, security.ErrOriginNotAllowed} {
		if errors.Is(err, sentinel) {
			return errors.Wrap(sentinel, message)
		}
	}
	return errors.Wrap(ErrAuthentication, message)
}

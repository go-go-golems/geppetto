// Package transport provides restricted, Go-only extension points for provider
// request routing, credential header injection, and pre-stream response
// classification. It is intentionally separate from inference middleware: these
// hooks operate inside a provider engine after it has selected its protocol.
package transport

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// RouteRequest is the non-secret input supplied to a trusted provider route
// resolver. BaseURL returns a copy, so a resolver cannot mutate caller state.
type RouteRequest struct {
	baseURL   url.URL
	operation string
}

// BaseURL returns a copy of the configured provider base URL.
func (r RouteRequest) BaseURL() url.URL { return r.baseURL }

// Operation identifies the engine operation being routed, for example
// "responses" or "messages".
func (r RouteRequest) Operation() string { return r.operation }

// RouteResolver maps an engine operation to a provider endpoint. It is
// installed through trusted Go engine options, never profile settings or
// JavaScript values. The returned URL is validated by ResolveAndValidate before
// middleware can acquire credential material.
type RouteResolver interface {
	Resolve(RouteRequest) (*url.URL, error)
}

// URLValidator validates a fully resolved outbound URL.
type URLValidator func(*url.URL) error

// RequestContext is the non-secret, read-only context passed to request and
// response middleware. It is created only by ResolveAndValidate after the
// final URL has passed the engine's outbound URL policy.
type RequestContext struct {
	provider  string
	operation string
	url       url.URL
}

// Provider identifies the trusted provider adapter installed by the engine.
func (r RequestContext) Provider() string { return r.provider }

// Operation identifies the engine operation, for example "responses".
func (r RequestContext) Operation() string { return r.operation }

// URL returns a copy of the already validated final outbound URL.
func (r RequestContext) URL() url.URL { return r.url }

// ResolveAndValidate resolves a provider route and validates its final URL.
// It must run before an engine invokes Middleware.BeforeRequest.
func ResolveAndValidate(provider, operation string, baseURL *url.URL, resolver RouteResolver, validate URLValidator) (RequestContext, error) {
	if strings.TrimSpace(provider) == "" {
		return RequestContext{}, errors.New("transport provider is required")
	}
	if strings.TrimSpace(operation) == "" {
		return RequestContext{}, errors.New("transport operation is required")
	}
	if baseURL == nil {
		return RequestContext{}, errors.New("transport base URL is required")
	}
	if resolver == nil {
		return RequestContext{}, errors.New("transport route resolver is required")
	}
	if validate == nil {
		return RequestContext{}, errors.New("transport URL validator is required")
	}

	target, err := resolver.Resolve(RouteRequest{baseURL: *baseURL, operation: operation})
	if err != nil {
		return RequestContext{}, fmt.Errorf("resolve provider route: %w", err)
	}
	if target == nil {
		return RequestContext{}, errors.New("provider route resolver returned no URL")
	}
	resolved := *target
	if err := validate(&resolved); err != nil {
		return RequestContext{}, fmt.Errorf("validate provider route: %w", err)
	}
	return RequestContext{provider: provider, operation: operation, url: resolved}, nil
}

// HeaderRule declares one provider-approved request header. Sensitive header
// values are redacted from copies used for diagnostics.
type HeaderRule struct {
	Name      string
	Sensitive bool
}

var restrictedHeaders = map[string]struct{}{
	"Host":              {},
	"Content-Length":    {},
	"Connection":        {},
	"Transfer-Encoding": {},
	"Trailer":           {},
	"Te":                {},
	"Upgrade":           {},
	"Proxy-Connection":  {},
}

// HeaderWriter permits middleware to set only engine-declared request headers.
// It has no access to the URL, request body, host, or response stream.
type HeaderWriter interface {
	Set(name, value string) error
}

// HeaderSet owns one request's middleware-controlled headers. Engine code keeps
// the concrete value so it can obtain a redacted diagnostic copy after middleware
// applies its changes.
type HeaderSet struct {
	target    http.Header
	rules     map[string]HeaderRule
	written   map[string]string
	sensitive map[string]struct{}
}

// NewHeaderSet binds a non-nil target header map to an explicit provider header
// policy. Duplicate, empty, and framing/host header rules are rejected.
func NewHeaderSet(target http.Header, rules ...HeaderRule) (*HeaderSet, error) {
	if target == nil {
		return nil, errors.New("transport header target is required")
	}
	set := &HeaderSet{
		target:    target,
		rules:     make(map[string]HeaderRule, len(rules)),
		written:   make(map[string]string, len(rules)),
		sensitive: make(map[string]struct{}, len(rules)),
	}
	for _, rule := range rules {
		name := http.CanonicalHeaderKey(strings.TrimSpace(rule.Name))
		if name == "" {
			return nil, errors.New("transport header rule has no name")
		}
		if _, prohibited := restrictedHeaders[name]; prohibited {
			return nil, fmt.Errorf("transport header rule %q is prohibited", name)
		}
		if _, exists := set.rules[name]; exists {
			return nil, fmt.Errorf("transport header rule %q is duplicated", name)
		}
		rule.Name = name
		set.rules[name] = rule
		if rule.Sensitive {
			set.sensitive[name] = struct{}{}
		}
	}
	return set, nil
}

// Set writes one approved header. A second middleware cannot silently override
// a different value written by an earlier middleware. Values containing CR/LF
// are rejected without including the value in the error.
func (s *HeaderSet) Set(name, value string) error {
	if s == nil {
		return errors.New("nil transport header writer")
	}
	canonicalName := http.CanonicalHeaderKey(strings.TrimSpace(name))
	if _, prohibited := restrictedHeaders[canonicalName]; prohibited {
		return fmt.Errorf("transport header %q is prohibited", canonicalName)
	}
	if _, allowed := s.rules[canonicalName]; !allowed {
		return fmt.Errorf("transport header %q is not allowed", canonicalName)
	}
	if strings.ContainsAny(value, "\r\n") {
		return fmt.Errorf("transport header %q has an invalid value", canonicalName)
	}
	if previous, alreadyWritten := s.written[canonicalName]; alreadyWritten && previous != value {
		return fmt.Errorf("transport header %q was already set by middleware", canonicalName)
	}
	s.target.Set(canonicalName, value)
	s.written[canonicalName] = value
	return nil
}

// RedactedCopy returns a header copy suitable for diagnostics. It preserves
// non-sensitive engine headers and replaces sensitive middleware values.
func (s *HeaderSet) RedactedCopy() http.Header {
	if s == nil {
		return nil
	}
	out := s.target.Clone()
	for name := range s.sensitive {
		if _, present := out[name]; present {
			out.Set(name, "<redacted>")
		}
	}
	return out
}

// Attempt is provider-private state paired with one middleware invocation. The
// transport core stores it opaquely and must never log or serialize it.
type Attempt any

// AttemptState is an opaque collection of middleware attempt values.
type AttemptState struct {
	attempts []Attempt
}

// ResponseMetadata is the bounded, body-free response information available to
// middleware before the engine starts decoding output.
type ResponseMetadata struct {
	StatusCode    int
	StreamStarted bool
}

// ResponseDecision directs the engine's response handling. The engine, not
// middleware, enforces whether a retry is eligible and closes any response body.
type ResponseDecision uint8

const (
	// Continue tells the core to decode or report the current response.
	Continue ResponseDecision = iota
	// Retry requests one core-governed pre-stream replay.
	Retry
)

// Middleware is a trusted Go-only provider extension. It may inject declared
// headers and classify a body-free response; it cannot mutate the final URL or
// request body, consume a stream, or perform the retry itself.
type Middleware interface {
	BeforeRequest(context.Context, RequestContext, HeaderWriter) (Attempt, error)
	AfterResponse(context.Context, RequestContext, Attempt, ResponseMetadata) (ResponseDecision, error)
}

// Chain applies request middleware in registration order and response
// middleware in reverse order, like nested middleware. It leaves retry bounds
// to the engine core.
type Chain struct {
	middlewares []Middleware
}

// NewChain constructs a trusted middleware chain. Nil middleware is rejected
// so a partially configured engine fails at construction rather than request
// time.
func NewChain(middlewares ...Middleware) (*Chain, error) {
	chain := &Chain{middlewares: make([]Middleware, len(middlewares))}
	for i, middleware := range middlewares {
		if middleware == nil {
			return nil, fmt.Errorf("transport middleware %d is nil", i)
		}
		chain.middlewares[i] = middleware
	}
	return chain, nil
}

// BeforeRequest applies middleware in registration order and returns opaque
// values for the matching AfterResponse call.
func (c *Chain) BeforeRequest(ctx context.Context, request RequestContext, headers HeaderWriter) (AttemptState, error) {
	if c == nil {
		return AttemptState{}, errors.New("nil transport middleware chain")
	}
	if headers == nil {
		return AttemptState{}, errors.New("transport header writer is required")
	}
	attempts := make([]Attempt, 0, len(c.middlewares))
	for _, middleware := range c.middlewares {
		attempt, err := middleware.BeforeRequest(ctx, request, headers)
		if err != nil {
			return AttemptState{}, err
		}
		attempts = append(attempts, attempt)
	}
	return AttemptState{attempts: attempts}, nil
}

// AfterResponse applies middleware in reverse order. Any Retry request is
// returned to the core; the core decides if replay is still eligible.
func (c *Chain) AfterResponse(ctx context.Context, request RequestContext, attempts AttemptState, response ResponseMetadata) (ResponseDecision, error) {
	if c == nil {
		return Continue, errors.New("nil transport middleware chain")
	}
	if len(attempts.attempts) != len(c.middlewares) {
		return Continue, errors.New("transport middleware attempt state does not match chain")
	}
	decision := Continue
	for i := len(c.middlewares) - 1; i >= 0; i-- {
		result, err := c.middlewares[i].AfterResponse(ctx, request, attempts.attempts[i], response)
		if err != nil {
			return Continue, err
		}
		switch result {
		case Continue:
		case Retry:
			decision = Retry
		default:
			return Continue, fmt.Errorf("transport middleware returned unknown response decision %d", result)
		}
	}
	return decision, nil
}

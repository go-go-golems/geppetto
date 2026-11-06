// Package connection binds an API's endpoint, routes, authentication, headers,
// URL policy and HTTP client into one immutable request capability. Construct a
// connection at the configuration boundary and pass it to payload encoders.
package connection

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/go-go-golems/geppetto/pkg/security"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/credentials"
	aitransport "github.com/go-go-golems/geppetto/pkg/steps/ai/transport"
	"github.com/pkg/errors"
)

// SDKAPIKey is an inert constructor value for SDKs requiring a nonempty key.
// Real credentials are installed by the connection, never passed to an SDK.
const SDKAPIKey = "geppetto-connection-auth"

// Routes names the operations supported by a connection. Resolvers may select
// a path, but cannot grant permission to contact a different origin.
type Routes map[string]aitransport.RouteResolver

// Config declares a binding at a trusted configuration boundary. New snapshots
// its client, route table and headers. Credential sources and custom transports
// are trusted, concurrency-safe host code.
type Config struct {
	Provider    string
	BaseURL     string
	Routes      Routes
	Auth        Authentication
	Headers     http.Header
	HTTPClient  *http.Client
	OutboundURL security.OutboundURLOptions
	// RejectRedirects disables even same-origin redirects for strict API contracts.
	RejectRedirects bool
}

// Connection owns a fixed endpoint and its authority. It exposes no credential,
// client setter, transport getter or API for rebinding its origin.
type Connection struct {
	provider string
	baseURL  url.URL
	origin   security.Origin
	routes   Routes
	client   http.Client
	policy   security.OutboundURLOptions
	headers  http.Header
	rules    []aitransport.HeaderRule
	chain    *aitransport.Chain
}

func (*Connection) String() string     { return "<bound API connection>" }
func (c *Connection) GoString() string { return c.String() }

// New binds a declared endpoint to authentication before any protocol routing.
// It does not acquire a token or perform network IO.
func New(config Config) (*Connection, error) {
	if strings.TrimSpace(config.Provider) == "" {
		return nil, errors.New("connection provider is required")
	}
	base, err := url.Parse(strings.TrimRight(strings.TrimSpace(config.BaseURL), "/"))
	if err != nil {
		return nil, errors.New("invalid connection base URL")
	}
	if base.User != nil || base.RawQuery != "" || base.ForceQuery || base.Fragment != "" || base.Opaque != "" {
		return nil, errors.New("connection base URL cannot contain userinfo, query or fragment")
	}
	origin, err := security.OriginOf(base)
	if err != nil {
		return nil, errors.New("connection requires an absolute HTTP(S) base URL")
	}
	if err := security.ValidateOutboundURL(base.String(), config.OutboundURL); err != nil {
		return nil, errors.Wrap(err, "connection base URL policy")
	}
	if config.Auth.err != nil {
		return nil, config.Auth.err
	}
	chain, err := aitransport.NewChain(config.Auth.middlewares...)
	if err != nil {
		return nil, err
	}
	rules := append([]aitransport.HeaderRule(nil), config.Auth.rules...)
	if _, err := aitransport.NewHeaderSet(http.Header{}, rules...); err != nil {
		return nil, err
	}
	headers := http.Header{}
	for name, values := range config.Headers {
		canonical := http.CanonicalHeaderKey(strings.TrimSpace(name))
		if _, exists := headers[canonical]; exists {
			return nil, errors.New("duplicate API default header")
		}
		// Validate the name even for a header declared without values.
		if _, err := aitransport.NewHeaderSet(http.Header{}, aitransport.HeaderRule{Name: canonical, Sensitive: true}); err != nil {
			return nil, err
		}
		// Apply the same framing/host and value validation to default headers.
		for _, value := range values {
			set, err := aitransport.NewHeaderSet(http.Header{}, aitransport.HeaderRule{Name: canonical, Sensitive: true})
			if err != nil {
				return nil, err
			}
			if err := set.Set(canonical, value); err != nil {
				return nil, err
			}
		}
		headers[canonical] = append([]string(nil), values...)
	}
	c := &Connection{
		provider: strings.TrimSpace(config.Provider), baseURL: *base, origin: origin,
		policy: config.OutboundURL, headers: headers, rules: rules, chain: chain,
		routes: make(Routes, len(config.Routes)),
	}
	for name, route := range config.Routes {
		if name == "" || route == nil {
			return nil, errors.New("connection routes need a name and resolver")
		}
		c.routes[name] = route
	}
	if config.HTTPClient != nil {
		c.client = *config.HTTPClient
	}
	if config.RejectRedirects {
		c.client.CheckRedirect = func(*http.Request, []*http.Request) error {
			return errors.New("API connection rejects redirects")
		}
	}
	next := c.client.Transport
	if next == nil {
		next = http.DefaultTransport
	}
	c.client.Transport = guardedTransport{connection: c, next: next}
	return c, nil
}

// BaseURL returns non-secret endpoint metadata, never a mutable URL pointer.
func (c *Connection) BaseURL() string { return c.baseURL.String() }

// Resolve selects an operation's route and verifies it against the original
// binding before any middleware or credential source can run.
func (c *Connection) Resolve(operation string) (aitransport.RequestContext, error) {
	if c == nil {
		return aitransport.RequestContext{}, errors.New("nil API connection")
	}
	resolver, ok := c.routes[operation]
	if !ok {
		return aitransport.RequestContext{}, errors.Errorf("API connection has no route %q", operation)
	}
	return aitransport.ResolveAndValidate(c.provider, operation, &c.baseURL, resolver, c.checkURL)
}

type routeContextKey struct{}

// NewRequest creates a request for a registered operation. It contains no
// connection credentials; those are installed only inside Do.
func (c *Connection) NewRequest(ctx context.Context, method, operation string, body io.Reader) (*http.Request, error) {
	route, err := c.Resolve(operation)
	if err != nil {
		return nil, err
	}
	target := route.URL()
	ctx = context.WithValue(ctx, routeContextKey{}, route)
	return http.NewRequestWithContext(ctx, method, target.String(), body)
}

// Do sends an already encoded request through this binding. A foreign URL is
// refused before authentication. Middleware and unauthorized replay state are
// private to this invocation. At most one replay occurs, before decoding.
func (c *Connection) Do(request *http.Request) (*http.Response, error) {
	return c.DoWithObserver(request, nil)
}

// DoWithObserver supplies an authenticated but redacted copy for diagnostics.
// Observer code never receives real middleware or connection header values.
func (c *Connection) DoWithObserver(request *http.Request, observe func(*http.Request)) (*http.Response, error) {
	if c == nil || request == nil {
		return nil, errors.New("API connection and request are required")
	}
	if err := request.Context().Err(); err != nil {
		closeBody(request)
		return nil, err
	}
	if err := c.checkRequest(request); err != nil {
		closeBody(request)
		return nil, err
	}
	ctx := context.WithValue(request.Context(), credentialContextKey{}, credentials.Request{
		Provider: c.provider, BaseURL: c.baseURL.String(),
	})
	request = request.WithContext(ctx)
	operation := "sdk"
	if route, ok := request.Context().Value(routeContextKey{}).(aitransport.RequestContext); ok {
		operation = route.Operation()
	}
	// Construct from the actual URL, rather than trusting a caller's context.
	route, err := aitransport.ResolveAndValidate(c.provider, operation, request.URL, identityRoute{}, c.checkURL)
	if err != nil {
		closeBody(request)
		return nil, err
	}
	var previous aitransport.AttemptState
	for attempt := 0; ; attempt++ {
		req := request.Clone(request.Context())
		if req.Header == nil {
			req.Header = http.Header{}
		}
		if attempt > 0 && request.Body != nil && request.Body != http.NoBody {
			req.Body, err = request.GetBody()
			if err != nil {
				return nil, errors.New("recreate API request body")
			}
		}
		for name, values := range c.headers {
			if req.Header.Get(name) == "" {
				req.Header[name] = append([]string(nil), values...)
			}
		}
		headers, err := aitransport.NewHeaderSet(req.Header, c.rules...)
		if err != nil {
			closeBody(req)
			return nil, err
		}
		state, err := c.chain.BeforeRequest(req.Context(), route, previous, headers)
		if err != nil {
			closeBody(req)
			return nil, err
		}
		if observe != nil {
			redacted := req.Clone(req.Context())
			redacted.Header = headers.RedactedCopy()
			for name := range c.headers {
				redacted.Header.Set(name, "<redacted>")
			}
			observe(redacted)
		}
		resp, err := c.client.Do(req)
		if err != nil {
			return nil, err
		}
		replayable := request.Body == nil || request.Body == http.NoBody || request.GetBody != nil
		eligible := attempt == 0 && replayable
		decision, err := c.chain.AfterResponse(req.Context(), route, state, aitransport.ResponseMetadata{
			StatusCode: resp.StatusCode, RetryEligible: eligible,
		})
		if err != nil {
			_ = resp.Body.Close()
			return nil, err
		}
		if decision != aitransport.Retry || !eligible {
			if resp.Request != nil {
				resp.Request = c.redactedRequest(resp.Request)
			}
			return resp, nil
		}
		_ = resp.Body.Close()
		previous = state
	}
}

func (c *Connection) redactedRequest(req *http.Request) *http.Request {
	redacted := req.Clone(req.Context())
	for _, rule := range c.rules {
		if rule.Sensitive && redacted.Header.Get(rule.Name) != "" {
			redacted.Header.Set(rule.Name, "<redacted>")
		}
	}
	for name := range c.headers {
		if redacted.Header.Get(name) != "" {
			redacted.Header.Set(name, "<redacted>")
		}
	}
	return redacted
}

func (c *Connection) checkURL(target *url.URL) error {
	if target == nil {
		return errors.New("nil connection target")
	}
	origin, err := security.OriginOf(target)
	if err != nil || origin != c.origin {
		return errors.Wrap(security.ErrOriginNotAllowed, "destination is outside the API connection")
	}
	if target.User != nil || target.Opaque != "" || target.Fragment != "" {
		return errors.New("API target cannot contain userinfo, opaque URL or fragment")
	}
	return security.ValidateOutboundURL(target.String(), c.policy)
}

func (c *Connection) checkRequest(req *http.Request) error {
	if err := c.checkURL(req.URL); err != nil {
		return err
	}
	if req.Host != "" && !strings.EqualFold(req.Host, req.URL.Host) {
		return errors.New("API connection does not allow a Host override")
	}
	return nil
}

type guardedTransport struct {
	connection *Connection
	next       http.RoundTripper
}

var _ http.RoundTripper = guardedTransport{}

func (t guardedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if err := t.connection.checkRequest(req); err != nil {
		closeBody(req)
		return nil, err
	}
	return t.next.RoundTrip(req)
}

// SDKClient supplies an SDK-compatible facade without exposing the connection's
// network transport. The inner connection owns redirects, cookies and replay;
// the facade must not process redirects or cookies a second time.
func (c *Connection) SDKClient() *http.Client {
	return &http.Client{Transport: sdkTransport{c}, CheckRedirect: func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}}
}

type sdkTransport struct{ connection *Connection }

var _ http.RoundTripper = sdkTransport{}

func (t sdkTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return t.connection.Do(req)
}

func closeBody(req *http.Request) {
	if req.Body != nil {
		_ = req.Body.Close()
	}
}

type identityRoute struct{}

func (identityRoute) Resolve(r aitransport.RouteRequest) (*url.URL, error) {
	u := r.BaseURL()
	return &u, nil
}

var routeSegment = regexp.MustCompile(`^[A-Za-z0-9._~-]+$`)

type relativeRoute string

// Relative declares an operation path below the base URL. Absolute URLs,
// queries, fragments and dot-segment traversal are rejected when resolved.
func Relative(path string) aitransport.RouteResolver { return relativeRoute(path) }

func (path relativeRoute) Resolve(r aitransport.RouteRequest) (*url.URL, error) {
	rel, err := url.Parse(string(path))
	if err != nil || rel.IsAbs() || rel.Host != "" || rel.User != nil || rel.RawQuery != "" || rel.ForceQuery || rel.Fragment != "" || strings.HasPrefix(string(path), "/") || strings.Contains(string(path), "\\") {
		return nil, errors.New("API route must be a relative path")
	}
	for _, segment := range strings.Split(rel.Path, "/") {
		if segment == "." || segment == ".." || !routeSegment.MatchString(segment) {
			return nil, errors.New("API route requires plain relative path segments")
		}
	}
	target := r.BaseURL()
	target.Path = strings.TrimRight(target.Path, "/") + "/" + rel.Path
	target.RawPath = ""
	return &target, nil
}

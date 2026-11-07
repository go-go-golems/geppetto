package security

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"slices"
	"strings"
)

// ErrOriginNotAllowed reports that a request or credential lookup was refused
// because its URL's origin is outside the credential's scope.
var ErrOriginNotAllowed = errors.New("origin not allowed for this credential")

// Origin is the scheme, host and port of a URL: the unit a provider
// credential is scoped to. Scheme and Host are lower-case; Port is always
// explicit (443 for https, 80 for http when the URL omits it), so
// "https://api.openai.com" and "https://API.openai.com:443/v1" are equal.
type Origin struct {
	Scheme string
	Host   string
	Port   string
}

// ParseOrigin returns the origin of an absolute http or https URL. Empty,
// relative and other-scheme URLs are errors. User info is ignored: it is not
// part of the origin.
func ParseOrigin(raw string) (Origin, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return Origin{}, errors.New("origin: empty URL")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return Origin{}, fmt.Errorf("origin: invalid URL: %w", err)
	}
	return OriginOf(u)
}

// MustParseOrigin is ParseOrigin for constants; it panics on error.
func MustParseOrigin(raw string) Origin {
	o, err := ParseOrigin(raw)
	if err != nil {
		panic(err)
	}
	return o
}

// OriginOf returns the origin of u, which must be an absolute http or https
// URL with a host.
func OriginOf(u *url.URL) (Origin, error) {
	if u == nil {
		return Origin{}, errors.New("origin: nil URL")
	}
	scheme := strings.ToLower(u.Scheme)
	port := u.Port()
	switch scheme {
	case "https":
		if port == "" {
			port = "443"
		}
	case "http":
		if port == "" {
			port = "80"
		}
	default:
		return Origin{}, fmt.Errorf("origin: unsupported scheme %q", u.Scheme)
	}
	host := strings.ToLower(u.Hostname())
	if host == "" {
		return Origin{}, errors.New("origin: URL has no host")
	}
	return Origin{Scheme: scheme, Host: host, Port: port}, nil
}

// String renders the origin as scheme://host:port (IPv6 hosts bracketed).
func (o Origin) String() string {
	return o.Scheme + "://" + net.JoinHostPort(o.Host, o.Port)
}

// Allows reports whether u's origin is one of allowed.
func Allows(allowed []Origin, u *url.URL) bool {
	o, err := OriginOf(u)
	return err == nil && slices.Contains(allowed, o)
}

// PinnedTransport forwards only requests whose URL origin is in Allowed and
// refuses all others with ErrOriginNotAllowed, before anything is sent.
// http.Client calls RoundTrip once per redirect hop, so a redirect to another
// origin is refused too. An empty Allowed refuses everything.
//
// A nil Next means http.DefaultTransport, read when the request is made.
type PinnedTransport struct {
	Next    http.RoundTripper
	Allowed []Origin
}

// RoundTrip implements http.RoundTripper.
func (t *PinnedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if !Allows(t.Allowed, req.URL) {
		if req.Body != nil {
			_ = req.Body.Close() // RoundTripper contract, also on error
		}
		return nil, fmt.Errorf("request to %s refused (allowed: %s): %w", redactedTarget(req.URL), originList(t.Allowed), ErrOriginNotAllowed)
	}
	next := t.Next
	if next == nil {
		next = http.DefaultTransport
	}
	return next.RoundTrip(req)
}

// PinClient returns a shallow copy of base whose transport only reaches the
// allowed origins. A nil base means a new client. base is not modified, so
// pinning http.DefaultClient is safe.
func PinClient(base *http.Client, allowed ...Origin) *http.Client {
	var c http.Client
	if base != nil {
		c = *base
	}
	c.Transport = &PinnedTransport{Next: c.Transport, Allowed: slices.Clone(allowed)}
	return &c
}

// redactedTarget names a URL by scheme and host only: queries can carry keys
// (Gemini's ?key=), paths can carry tenant ids.
func redactedTarget(u *url.URL) string {
	if u == nil {
		return "<nil URL>"
	}
	return u.Scheme + "://" + u.Host
}

func originList(os []Origin) string {
	if len(os) == 0 {
		return "none"
	}
	s := make([]string, len(os))
	for i, o := range os {
		s[i] = o.String()
	}
	return strings.Join(s, ", ")
}

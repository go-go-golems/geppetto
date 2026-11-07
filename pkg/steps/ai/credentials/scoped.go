package credentials

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/go-go-golems/geppetto/pkg/security"
)

// Scoped wraps src so it only produces a token for requests whose BaseURL has
// one of the allowed origins. Any other Request, including one with an empty
// BaseURL, gets an error wrapping security.ErrOriginNotAllowed and src is not
// consulted. If src implements UnauthorizedBearerTokenSource, so does the
// result.
//
// Scoped is defense in depth: the HTTP client carrying the token should also
// be pinned (security.PinClient), which covers redirects and secrets that do
// not come from a token source.
func Scoped(src BearerTokenSource, allowed ...security.Origin) BearerTokenSource {
	s := scopedSource{src: src, allowed: slices.Clone(allowed)}
	if u, ok := src.(UnauthorizedBearerTokenSource); ok {
		return scopedUnauthorizedSource{scopedSource: s, unauthorized: u}
	}
	return s
}

type scopedSource struct {
	src     BearerTokenSource
	allowed []security.Origin
}

func (s scopedSource) check(r Request) error {
	o, err := security.ParseOrigin(r.BaseURL)
	if err != nil || !slices.Contains(s.allowed, o) {
		target := strings.TrimSpace(r.BaseURL)
		if err == nil {
			target = o.String()
		}
		return fmt.Errorf("bearer credential for provider %q refused for %q: %w", r.Provider, target, security.ErrOriginNotAllowed)
	}
	return nil
}

func (s scopedSource) BearerToken(ctx context.Context, r Request) (string, error) {
	if s.src == nil {
		return "", &ErrUnavailable{Provider: r.Provider, Operation: "scoped credential lookup"}
	}
	if err := s.check(r); err != nil {
		return "", err
	}
	return s.src.BearerToken(ctx, r)
}

type scopedUnauthorizedSource struct {
	scopedSource
	unauthorized UnauthorizedBearerTokenSource
}

func (s scopedUnauthorizedSource) BearerTokenAfterUnauthorized(ctx context.Context, r Request, rejected string) (string, error) {
	if err := s.check(r); err != nil {
		return "", err
	}
	return s.unauthorized.BearerTokenAfterUnauthorized(ctx, r, rejected)
}

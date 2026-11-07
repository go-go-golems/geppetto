package credentials

import (
	"context"
	"errors"
	"testing"

	"github.com/go-go-golems/geppetto/pkg/security"
)

type countingSource struct {
	calls, unauthorizedCalls int
}

func (c *countingSource) BearerToken(context.Context, Request) (string, error) {
	c.calls++
	return "TOKEN", nil
}

func (c *countingSource) BearerTokenAfterUnauthorized(context.Context, Request, string) (string, error) {
	c.unauthorizedCalls++
	return "TOKEN2", nil
}

func TestScopedRefusesForeignBaseURL(t *testing.T) {
	src := &countingSource{}
	s := Scoped(src, security.MustParseOrigin("https://gw.example/v1"))
	ctx := context.Background()

	tok, err := s.BearerToken(ctx, Request{Provider: "openai", BaseURL: "https://GW.example:443/v1/"})
	if err != nil || tok != "TOKEN" {
		t.Fatalf("allowed origin: %q %v", tok, err)
	}
	for _, u := range []string{"https://api.openai.com/v1", "", " ", "/", "http://gw.example/v1"} {
		_, err := s.BearerToken(ctx, Request{Provider: "openai", BaseURL: u})
		if !errors.Is(err, security.ErrOriginNotAllowed) {
			t.Fatalf("%q: want ErrOriginNotAllowed, got %v", u, err)
		}
	}
	if src.calls != 1 {
		t.Fatalf("source consulted %d times, want 1", src.calls)
	}

	u, ok := s.(UnauthorizedBearerTokenSource)
	if !ok {
		t.Fatal("Scoped must keep UnauthorizedBearerTokenSource")
	}
	if _, err := u.BearerTokenAfterUnauthorized(ctx, Request{Provider: "openai", BaseURL: "https://other.example"}, "TOKEN"); !errors.Is(err, security.ErrOriginNotAllowed) {
		t.Fatalf("unauthorized refresh for a foreign origin: %v", err)
	}
	if _, err := u.BearerTokenAfterUnauthorized(ctx, Request{Provider: "openai", BaseURL: "https://gw.example/v1"}, "TOKEN"); err != nil {
		t.Fatal(err)
	}
	if src.unauthorizedCalls != 1 {
		t.Fatalf("unauthorized refresh called %d times, want 1", src.unauthorizedCalls)
	}

	static, _ := NewStaticBearerTokenSource("K")
	if _, ok := Scoped(static, security.MustParseOrigin("https://gw.example")).(UnauthorizedBearerTokenSource); ok {
		t.Fatal("a static source must not gain BearerTokenAfterUnauthorized")
	}
}

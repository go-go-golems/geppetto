// Package credentials provides host-injected bearer credential sources for AI
// providers. It deliberately does not read profile YAML or own OAuth protocol
// endpoints: hosts retain control of credential storage and refresh behavior.
package credentials

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
)

// Request identifies the non-secret provider endpoint for which a bearer token
// is needed. A host chooses how Provider and BaseURL map to stored credentials.
type Request struct {
	Provider string
	BaseURL  string
}

func (r Request) key() (string, error) {
	provider := strings.ToLower(strings.TrimSpace(r.Provider))
	if provider == "" {
		return "", errors.New("bearer credential request has no provider")
	}
	return provider + "\x00" + strings.TrimSpace(r.BaseURL), nil
}

// Credential contains OAuth-like bearer material owned by a host credential
// store. It must never be serialized into inference settings or log output.
type Credential struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
}

// Usable reports whether the credential has a non-empty access token and has
// not reached its expiry after applying skew. A zero expiry means non-expiring.
func (c Credential) Usable(now time.Time, skew time.Duration) bool {
	if strings.TrimSpace(c.AccessToken) == "" {
		return false
	}
	return c.ExpiresAt.IsZero() || now.Add(skew).Before(c.ExpiresAt)
}

// Store loads and persists host-owned credentials. Save must atomically retain
// a rotated refresh token when a refresher returns one.
type Store interface {
	Load(context.Context, Request) (Credential, error)
	Save(context.Context, Request, Credential) error
}

// Refresher obtains a replacement credential. It receives the previous
// credential so provider adapters can use its refresh token; implementations
// must not include either token in returned errors.
type Refresher interface {
	Refresh(context.Context, Request, Credential) (Credential, error)
}

// BearerTokenSource returns a currently usable bearer token at outbound request
// time. It is intentionally smaller than Store/Refresher so applications may
// supply a custom implementation without adopting RenewableBearerTokenSource.
type BearerTokenSource interface {
	BearerToken(context.Context, Request) (string, error)
}

// UnauthorizedBearerTokenSource is an optional extension for a provider request
// that was rejected with HTTP 401 before it produced any response output. It
// returns a replacement token for exactly one caller-managed replay and must
// never expose token material through errors.
//
// Callers must not use it for static settings or retry a second 401 response.
type UnauthorizedBearerTokenSource interface {
	BearerTokenSource
	BearerTokenAfterUnauthorized(context.Context, Request, string) (string, error)
}

// ErrUnavailable is returned without token material when loading, refreshing,
// persisting, or validating a credential fails.
type ErrUnavailable struct {
	Provider  string
	Operation string
}

func (e *ErrUnavailable) Error() string {
	if e.Provider == "" {
		return "bearer credential unavailable during " + e.Operation
	}
	return fmt.Sprintf("bearer credential unavailable for provider %q during %s", e.Provider, e.Operation)
}

// RenewableOption configures a RenewableBearerTokenSource.
type RenewableOption func(*RenewableBearerTokenSource)

// WithRefreshSkew sets how soon before expiry a token is renewed. Negative
// values are normalized to zero.
func WithRefreshSkew(skew time.Duration) RenewableOption {
	return func(source *RenewableBearerTokenSource) {
		if skew > 0 {
			source.refreshSkew = skew
		} else {
			source.refreshSkew = 0
		}
	}
}

// WithClock supplies a clock for deterministic tests. Production callers
// normally use time.Now.
func WithClock(now func() time.Time) RenewableOption {
	return func(source *RenewableBearerTokenSource) {
		if now != nil {
			source.now = now
		}
	}
}

// RenewableBearerTokenSource caches host-owned credentials and refreshes them
// once per provider/base-URL pair when they are expired or inside refresh skew.
// It never writes credentials to settings, events, or logs.
type RenewableBearerTokenSource struct {
	store        Store
	refresher    Refresher
	refreshSkew  time.Duration
	now          func() time.Time
	mu           sync.Mutex
	cache        map[string]Credential
	refreshGroup singleflight.Group
}

var _ BearerTokenSource = (*RenewableBearerTokenSource)(nil)
var _ UnauthorizedBearerTokenSource = (*RenewableBearerTokenSource)(nil)

// NewRenewableBearerTokenSource constructs a source. Store and Refresher are
// required because a non-usable credential must be refreshed and safely saved.
func NewRenewableBearerTokenSource(store Store, refresher Refresher, opts ...RenewableOption) (*RenewableBearerTokenSource, error) {
	if store == nil {
		return nil, errors.New("bearer credential store is required")
	}
	if refresher == nil {
		return nil, errors.New("bearer credential refresher is required")
	}
	source := &RenewableBearerTokenSource{
		store:       store,
		refresher:   refresher,
		refreshSkew: 30 * time.Second,
		now:         time.Now,
		cache:       map[string]Credential{},
	}
	for _, option := range opts {
		if option != nil {
			option(source)
		}
	}
	return source, nil
}

// Invalidate drops an in-memory credential after an application explicitly
// replaces or revokes it in the host store. It never touches persistent state.
// BearerTokenAfterUnauthorized forces one refreshed credential when rejected
// matches the current credential. If another request already replaced it, the
// usable replacement is returned instead. Concurrent 401 handlers for the same
// rejected bearer share the forced refresh; distinct rejected bearers never do.
func (s *RenewableBearerTokenSource) BearerTokenAfterUnauthorized(ctx context.Context, request Request, rejected string) (string, error) {
	if s == nil {
		return "", &ErrUnavailable{Provider: request.Provider, Operation: "unauthorized source lookup"}
	}
	key, err := request.key()
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(rejected) == "" {
		return "", &ErrUnavailable{Provider: request.Provider, Operation: "unauthorized credential validation"}
	}

	result := s.refreshGroup.DoChan(unauthorizedRefreshKey(key, rejected), func() (any, error) {
		return s.refreshAfterUnauthorized(context.WithoutCancel(ctx), key, request, rejected)
	})
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case response := <-result:
		if response.Err != nil {
			return "", response.Err
		}
		credential, ok := response.Val.(Credential)
		if !ok || !credential.Usable(s.now(), 0) {
			return "", &ErrUnavailable{Provider: request.Provider, Operation: "unauthorized credential validation"}
		}
		return credential.AccessToken, nil
	}
}

func (s *RenewableBearerTokenSource) refreshAfterUnauthorized(ctx context.Context, key string, request Request, rejected string) (Credential, error) {
	credential, ok := s.cached(key)
	if ok && credential.Usable(s.now(), s.refreshSkew) && !tokensEqual(credential.AccessToken, rejected) {
		return credential, nil
	}
	if !ok {
		loaded, err := s.store.Load(ctx, request)
		if err != nil {
			return Credential{}, &ErrUnavailable{Provider: request.Provider, Operation: "credential load after unauthorized"}
		}
		credential = loaded
		if credential.Usable(s.now(), s.refreshSkew) && !tokensEqual(credential.AccessToken, rejected) {
			s.putCached(key, credential)
			return credential, nil
		}
	}

	refreshed, err := s.refresher.Refresh(ctx, request, credential)
	if err != nil {
		return Credential{}, &ErrUnavailable{Provider: request.Provider, Operation: "credential refresh after unauthorized"}
	}
	if !refreshed.Usable(s.now(), 0) {
		return Credential{}, &ErrUnavailable{Provider: request.Provider, Operation: "refreshed credential validation after unauthorized"}
	}
	if err := s.store.Save(ctx, request, refreshed); err != nil {
		return Credential{}, &ErrUnavailable{Provider: request.Provider, Operation: "credential save after unauthorized"}
	}
	s.putCached(key, refreshed)
	return refreshed, nil
}

func tokensEqual(left, right string) bool {
	return subtle.ConstantTimeCompare([]byte(left), []byte(right)) == 1
}

func unauthorizedRefreshKey(key, rejected string) string {
	fingerprint := sha256.Sum256([]byte(rejected))
	return key + "\x00unauthorized\x00" + string(fingerprint[:])
}

func (s *RenewableBearerTokenSource) Invalidate(request Request) error {
	if s == nil {
		return errors.New("nil renewable bearer token source")
	}
	key, err := request.key()
	if err != nil {
		return err
	}
	s.mu.Lock()
	delete(s.cache, key)
	s.mu.Unlock()
	return nil
}

// BearerToken returns a cache hit immediately when it remains usable. On an
// expired/missing credential it joins one refresh operation per request key.
// Shared load/refresh/save work ignores an initiating caller's cancellation;
// waiters can still abandon their own wait when their contexts are cancelled.
func (s *RenewableBearerTokenSource) BearerToken(ctx context.Context, request Request) (string, error) {
	if s == nil {
		return "", &ErrUnavailable{Provider: request.Provider, Operation: "source lookup"}
	}
	key, err := request.key()
	if err != nil {
		return "", err
	}
	if credential, ok := s.cached(key); ok && credential.Usable(s.now(), s.refreshSkew) {
		return credential.AccessToken, nil
	}

	result := s.refreshGroup.DoChan(key, func() (any, error) {
		return s.loadOrRefresh(context.WithoutCancel(ctx), key, request)
	})
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case response := <-result:
		if response.Err != nil {
			return "", response.Err
		}
		credential, ok := response.Val.(Credential)
		if !ok || !credential.Usable(s.now(), 0) {
			return "", &ErrUnavailable{Provider: request.Provider, Operation: "credential validation"}
		}
		return credential.AccessToken, nil
	}
}

func (s *RenewableBearerTokenSource) loadOrRefresh(ctx context.Context, key string, request Request) (Credential, error) {
	if credential, ok := s.cached(key); ok && credential.Usable(s.now(), s.refreshSkew) {
		return credential, nil
	}

	credential, ok := s.cached(key)
	if !ok {
		loaded, err := s.store.Load(ctx, request)
		if err != nil {
			return Credential{}, &ErrUnavailable{Provider: request.Provider, Operation: "credential load"}
		}
		credential = loaded
		s.putCached(key, credential)
	}
	if credential.Usable(s.now(), s.refreshSkew) {
		return credential, nil
	}

	refreshed, err := s.refresher.Refresh(ctx, request, credential)
	if err != nil {
		return Credential{}, &ErrUnavailable{Provider: request.Provider, Operation: "credential refresh"}
	}
	if !refreshed.Usable(s.now(), 0) {
		return Credential{}, &ErrUnavailable{Provider: request.Provider, Operation: "refreshed credential validation"}
	}
	if err := s.store.Save(ctx, request, refreshed); err != nil {
		return Credential{}, &ErrUnavailable{Provider: request.Provider, Operation: "credential save"}
	}
	s.putCached(key, refreshed)
	return refreshed, nil
}

func (s *RenewableBearerTokenSource) cached(key string) (Credential, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	credential, ok := s.cache[key]
	return credential, ok
}

func (s *RenewableBearerTokenSource) putCached(key string, credential Credential) {
	s.mu.Lock()
	s.cache[key] = credential
	s.mu.Unlock()
}

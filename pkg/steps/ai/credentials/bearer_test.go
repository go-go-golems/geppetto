package credentials_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-go-golems/geppetto/pkg/steps/ai/credentials"
)

type memoryStore struct {
	mu         sync.Mutex
	credential credentials.Credential
	loadCalls  int
	saveCalls  int
	loadErr    error
	saveErr    error
}

func (s *memoryStore) Load(context.Context, credentials.Request) (credentials.Credential, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.loadCalls++
	return s.credential, s.loadErr
}

func (s *memoryStore) Save(_ context.Context, _ credentials.Request, credential credentials.Credential) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.saveCalls++
	if s.saveErr != nil {
		return s.saveErr
	}
	s.credential = credential
	return nil
}

type refresher struct {
	mu              sync.Mutex
	calls           int
	result          credentials.Credential
	err             error
	started         chan struct{}
	continueRefresh chan struct{}
}

func (r *refresher) Refresh(ctx context.Context, _ credentials.Request, _ credentials.Credential) (credentials.Credential, error) {
	r.mu.Lock()
	r.calls++
	started, continueRefresh := r.started, r.continueRefresh
	r.mu.Unlock()
	if started != nil {
		close(started)
	}
	if continueRefresh != nil {
		select {
		case <-ctx.Done():
			return credentials.Credential{}, ctx.Err()
		case <-continueRefresh:
		}
	}
	return r.result, r.err
}

func (r *refresher) callCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.calls
}

type multiBlockingRefresher struct {
	mu      sync.Mutex
	calls   int
	started chan struct{}
	release <-chan struct{}
	result  credentials.Credential
}

func (r *multiBlockingRefresher) Refresh(context.Context, credentials.Request, credentials.Credential) (credentials.Credential, error) {
	r.mu.Lock()
	r.calls++
	r.mu.Unlock()
	r.started <- struct{}{}
	<-r.release
	return r.result, nil
}

func (r *multiBlockingRefresher) callCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.calls
}

func newSource(t *testing.T, store *memoryStore, refresher credentials.Refresher, now time.Time) *credentials.RenewableBearerTokenSource {
	t.Helper()
	source, err := credentials.NewRenewableBearerTokenSource(store, refresher,
		credentials.WithClock(func() time.Time { return now }),
		credentials.WithRefreshSkew(30*time.Second),
	)
	if err != nil {
		t.Fatal(err)
	}
	return source
}

func request() credentials.Request {
	return credentials.Request{Provider: "openai", BaseURL: "https://provider.example.test/v1"}
}

func TestRenewableBearerTokenSourceReturnsCachedUsableCredential(t *testing.T) {
	now := time.Date(2026, 7, 10, 20, 0, 0, 0, time.UTC)
	store := &memoryStore{credential: credentials.Credential{AccessToken: "access-secret", ExpiresAt: now.Add(time.Hour)}}
	refresher := &refresher{result: credentials.Credential{AccessToken: "unexpected"}}
	source := newSource(t, store, refresher, now)

	for range 2 {
		token, err := source.BearerToken(context.Background(), request())
		if err != nil || token != "access-secret" {
			t.Fatalf("BearerToken() = %q, %v", token, err)
		}
	}
	if store.loadCalls != 1 || refresher.callCount() != 0 || store.saveCalls != 0 {
		t.Fatalf("calls load=%d refresh=%d save=%d, want 1/0/0", store.loadCalls, refresher.callCount(), store.saveCalls)
	}
}

func TestRenewableBearerTokenSourceRefreshesAndPersistsRotatedCredential(t *testing.T) {
	now := time.Date(2026, 7, 10, 20, 0, 0, 0, time.UTC)
	store := &memoryStore{credential: credentials.Credential{AccessToken: "expired-access", RefreshToken: "old-refresh", ExpiresAt: now.Add(-time.Minute)}}
	refresher := &refresher{result: credentials.Credential{AccessToken: "fresh-access", RefreshToken: "rotated-refresh", ExpiresAt: now.Add(time.Hour)}}
	source := newSource(t, store, refresher, now)

	token, err := source.BearerToken(context.Background(), request())
	if err != nil || token != "fresh-access" {
		t.Fatalf("BearerToken() = %q, %v", token, err)
	}
	if refresher.callCount() != 1 || store.saveCalls != 1 || store.credential.RefreshToken != "rotated-refresh" {
		t.Fatalf("refresh=%d save=%d stored=%#v", refresher.callCount(), store.saveCalls, store.credential)
	}
}

func TestRenewableBearerTokenSourceRefreshesAfterUnauthorizedEvenWhenCredentialHasNotExpired(t *testing.T) {
	now := time.Date(2026, 7, 10, 20, 0, 0, 0, time.UTC)
	store := &memoryStore{credential: credentials.Credential{AccessToken: "rejected-access", RefreshToken: "old-refresh", ExpiresAt: now.Add(time.Hour)}}
	refresher := &refresher{result: credentials.Credential{AccessToken: "replacement-access", RefreshToken: "rotated-refresh", ExpiresAt: now.Add(time.Hour)}}
	source := newSource(t, store, refresher, now)

	first, err := source.BearerToken(context.Background(), request())
	if err != nil || first != "rejected-access" {
		t.Fatalf("initial BearerToken() = %q, %v", first, err)
	}
	replacement, err := source.BearerTokenAfterUnauthorized(context.Background(), request(), first)
	if err != nil || replacement != "replacement-access" {
		t.Fatalf("BearerTokenAfterUnauthorized() = %q, %v", replacement, err)
	}
	if refresher.callCount() != 1 || store.saveCalls != 1 || store.credential.RefreshToken != "rotated-refresh" {
		t.Fatalf("refresh=%d save=%d stored=%#v", refresher.callCount(), store.saveCalls, store.credential)
	}
}

func TestRenewableBearerTokenSourceCollapsesConcurrentRefreshesAndAllowsWaitingCancellation(t *testing.T) {
	now := time.Date(2026, 7, 10, 20, 0, 0, 0, time.UTC)
	store := &memoryStore{credential: credentials.Credential{AccessToken: "expired-access", RefreshToken: "refresh", ExpiresAt: now.Add(-time.Minute)}}
	refresher := &refresher{
		result:  credentials.Credential{AccessToken: "fresh-access", ExpiresAt: now.Add(time.Hour)},
		started: make(chan struct{}), continueRefresh: make(chan struct{}),
	}
	source := newSource(t, store, refresher, now)

	firstDone := make(chan error, 1)
	go func() {
		_, err := source.BearerToken(context.Background(), request())
		firstDone <- err
	}()
	<-refresher.started

	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := source.BearerToken(cancelled, request()); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled waiter error = %v, want context.Canceled", err)
	}

	const callers = 8
	var wg sync.WaitGroup
	errorsCh := make(chan error, callers)
	for range callers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			token, err := source.BearerToken(context.Background(), request())
			if err == nil && token != "fresh-access" {
				err = errors.New("unexpected refreshed token")
			}
			errorsCh <- err
		}()
	}
	close(refresher.continueRefresh)
	if err := <-firstDone; err != nil {
		t.Fatal(err)
	}
	wg.Wait()
	close(errorsCh)
	for err := range errorsCh {
		if err != nil {
			t.Fatal(err)
		}
	}
	if refresher.callCount() != 1 || store.saveCalls != 1 {
		t.Fatalf("refresh=%d save=%d, want one refresh/persist", refresher.callCount(), store.saveCalls)
	}
}

func TestRenewableBearerTokenSourceKeepsSharedRefreshAliveAfterInitiatorCancellation(t *testing.T) {
	now := time.Date(2026, 7, 10, 20, 0, 0, 0, time.UTC)
	store := &memoryStore{credential: credentials.Credential{AccessToken: "expired-access", RefreshToken: "refresh", ExpiresAt: now.Add(-time.Minute)}}
	refresher := &refresher{
		result:          credentials.Credential{AccessToken: "fresh-access", ExpiresAt: now.Add(time.Hour)},
		started:         make(chan struct{}),
		continueRefresh: make(chan struct{}),
	}
	source := newSource(t, store, refresher, now)

	initiatorCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	initiatorDone := make(chan error, 1)
	go func() {
		_, err := source.BearerToken(initiatorCtx, request())
		initiatorDone <- err
	}()
	<-refresher.started

	waiterDone := make(chan error, 1)
	go func() {
		token, err := source.BearerToken(context.Background(), request())
		if err == nil && token != "fresh-access" {
			err = errors.New("unexpected refreshed token")
		}
		waiterDone <- err
	}()

	cancel()
	if err := <-initiatorDone; !errors.Is(err, context.Canceled) {
		t.Fatalf("initiator error = %v, want context.Canceled", err)
	}
	close(refresher.continueRefresh)
	if err := <-waiterDone; err != nil {
		t.Fatal(err)
	}
	if refresher.callCount() != 1 || store.saveCalls != 1 {
		t.Fatalf("refresh=%d save=%d, want one refresh/persist", refresher.callCount(), store.saveCalls)
	}
}

func TestRenewableBearerTokenSourceSeparatesForcedRefreshesByRejectedBearer(t *testing.T) {
	now := time.Date(2026, 7, 10, 20, 0, 0, 0, time.UTC)
	store := &memoryStore{credential: credentials.Credential{AccessToken: "first-rejected", RefreshToken: "refresh", ExpiresAt: now.Add(time.Hour)}}
	release := make(chan struct{})
	refresher := &multiBlockingRefresher{
		started: make(chan struct{}, 2),
		release: release,
		result:  credentials.Credential{AccessToken: "replacement", ExpiresAt: now.Add(time.Hour)},
	}
	source := newSource(t, store, refresher, now)

	if _, err := source.BearerToken(context.Background(), request()); err != nil {
		t.Fatal(err)
	}
	firstDone := make(chan error, 1)
	go func() {
		_, err := source.BearerTokenAfterUnauthorized(context.Background(), request(), "first-rejected")
		firstDone <- err
	}()
	<-refresher.started

	store.mu.Lock()
	store.credential = credentials.Credential{AccessToken: "second-rejected", RefreshToken: "refresh", ExpiresAt: now.Add(time.Hour)}
	store.mu.Unlock()
	if err := source.Invalidate(request()); err != nil {
		t.Fatal(err)
	}
	secondDone := make(chan error, 1)
	go func() {
		_, err := source.BearerTokenAfterUnauthorized(context.Background(), request(), "second-rejected")
		secondDone <- err
	}()
	select {
	case <-refresher.started:
	case <-time.After(time.Second):
		t.Fatal("second rejected bearer joined the first forced refresh")
	}

	close(release)
	if err := <-firstDone; err != nil {
		t.Fatal(err)
	}
	if err := <-secondDone; err != nil {
		t.Fatal(err)
	}
	if refresher.callCount() != 2 {
		t.Fatalf("refresh=%d, want two independent forced refreshes", refresher.callCount())
	}
}

func TestRenewableBearerTokenSourceRedactsHostErrorsAndDoesNotCacheUnpersistedCredential(t *testing.T) {
	now := time.Date(2026, 7, 10, 20, 0, 0, 0, time.UTC)
	store := &memoryStore{credential: credentials.Credential{AccessToken: "expired-access", RefreshToken: "refresh-secret", ExpiresAt: now.Add(-time.Minute)}, saveErr: errors.New("save rotated-refresh failed")}
	refresher := &refresher{result: credentials.Credential{AccessToken: "fresh-access", RefreshToken: "rotated-refresh", ExpiresAt: now.Add(time.Hour)}}
	source := newSource(t, store, refresher, now)

	_, err := source.BearerToken(context.Background(), request())
	if err == nil {
		t.Fatal("expected error")
	}
	for _, secret := range []string{"expired-access", "refresh-secret", "fresh-access", "rotated-refresh"} {
		if strings.Contains(err.Error(), secret) {
			t.Fatalf("error leaked secret %q: %v", secret, err)
		}
	}
	if _, err := source.BearerToken(context.Background(), request()); err == nil {
		t.Fatal("second request unexpectedly succeeded with unpersisted credential")
	}
	if refresher.callCount() != 2 {
		t.Fatalf("refresh count=%d, want 2 after failed persist", refresher.callCount())
	}
}

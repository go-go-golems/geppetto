package credentials

import (
	"context"
	"errors"
	"strings"
	"time"
)

// Deleter is the optional local-delete capability a host store provides for
// logout. Geppetto never discovers a path or invokes provider revocation.
type Deleter interface {
	Delete(context.Context, Request) error
}

// State is the redacted lifecycle state of a credential.
type State string

const (
	StateMissing  State = "missing"
	StateReady    State = "ready"
	StateExpiring State = "expiring"
	StateExpired  State = "expired"
)

// Status contains no token material. A zero ExpiresAt means a non-expiring
// configured credential.
type Status struct {
	State     State
	ExpiresAt time.Time
	Renewable bool
}

// StatusOf loads a credential and derives a redacted lifecycle status. Store
// errors are normalized to ErrUnavailable so storage details do not enter CLI
// or engine diagnostics.
func StatusOf(ctx context.Context, store Store, request Request, now time.Time, refreshSkew time.Duration) (Status, error) {
	if store == nil {
		return Status{}, &ErrUnavailable{Provider: request.Provider, Operation: "credential status"}
	}
	if _, err := request.key(); err != nil {
		return Status{}, err
	}
	credential, err := store.Load(ctx, request)
	if err != nil {
		return Status{}, &ErrUnavailable{Provider: request.Provider, Operation: "credential status"}
	}
	status := Status{ExpiresAt: credential.ExpiresAt, Renewable: strings.TrimSpace(credential.RefreshToken) != ""}
	if strings.TrimSpace(credential.AccessToken) == "" {
		status.State = StateMissing
		return status, nil
	}
	if credential.ExpiresAt.IsZero() {
		status.State = StateReady
		return status, nil
	}
	if !now.Before(credential.ExpiresAt) {
		status.State = StateExpired
		return status, nil
	}
	if refreshSkew > 0 && !now.Add(refreshSkew).Before(credential.ExpiresAt) {
		status.State = StateExpiring
		return status, nil
	}
	status.State = StateReady
	return status, nil
}

// Logout removes a local credential tuple. It is intentionally separate from
// provider revocation, which requires provider-specific endpoint and client
// authentication review.
func Logout(ctx context.Context, store Store, request Request) error {
	if _, err := request.key(); err != nil {
		return err
	}
	deleter, ok := store.(Deleter)
	if !ok || deleter == nil {
		return errors.New("credential store does not support local logout")
	}
	if err := deleter.Delete(ctx, request); err != nil {
		return &ErrUnavailable{Provider: request.Provider, Operation: "credential logout"}
	}
	return nil
}

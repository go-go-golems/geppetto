package credentials_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/go-go-golems/geppetto/pkg/steps/ai/credentials"
)

type lifecycleStore struct {
	credential credentials.Credential
	deleteErr  error
	deleted    bool
}

func (s *lifecycleStore) Load(context.Context, credentials.Request) (credentials.Credential, error) {
	return s.credential, nil
}
func (*lifecycleStore) Save(context.Context, credentials.Request, credentials.Credential) error {
	return nil
}
func (s *lifecycleStore) Delete(context.Context, credentials.Request) error {
	if s.deleteErr != nil {
		return s.deleteErr
	}
	s.deleted = true
	s.credential = credentials.Credential{}
	return nil
}

func TestStatusOfRedactsCredentialMaterial(t *testing.T) {
	now := time.Date(2026, 7, 16, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name string
		cred credentials.Credential
		want credentials.State
	}{
		{"missing", credentials.Credential{}, credentials.StateMissing},
		{"ready", credentials.Credential{AccessToken: "test", ExpiresAt: now.Add(time.Hour)}, credentials.StateReady},
		{"expiring", credentials.Credential{AccessToken: "test", RefreshToken: "refresh", ExpiresAt: now.Add(time.Minute)}, credentials.StateExpiring},
		{"expired", credentials.Credential{AccessToken: "test", ExpiresAt: now.Add(-time.Minute)}, credentials.StateExpired},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			status, err := credentials.StatusOf(context.Background(), &lifecycleStore{credential: test.cred}, request(), now, 5*time.Minute)
			if err != nil || status.State != test.want {
				t.Fatalf("StatusOf = %#v, %v", status, err)
			}
		})
	}
}

func TestLogoutUsesHostDeleteAndRedactsFailure(t *testing.T) {
	store := &lifecycleStore{credential: credentials.Credential{AccessToken: "test"}}
	if err := credentials.Logout(context.Background(), store, request()); err != nil || !store.deleted {
		t.Fatalf("Logout = %v deleted=%v", err, store.deleted)
	}
	store = &lifecycleStore{deleteErr: errors.New("sensitive storage detail")}
	err := credentials.Logout(context.Background(), store, request())
	if err == nil || err.Error() == "sensitive storage detail" {
		t.Fatalf("Logout error = %v", err)
	}
}

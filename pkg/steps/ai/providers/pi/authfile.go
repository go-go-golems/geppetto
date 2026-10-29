package pi

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-go-golems/geppetto/pkg/steps/ai/credentials"
	"github.com/pkg/errors"
)

// Credential kinds in auth.json.
const (
	KindAPIKey = "api_key"
	KindOAuth  = "oauth"
)

// Entry is one provider's auth.json credential. Secret fields must never be
// logged or formatted into errors.
type Entry struct {
	Type    string
	Key     string            // api_key: a literal or a configured value ("$ENV", "!cmd")
	Env     map[string]string // api_key: provider-scoped values
	Access  string            // oauth
	Refresh string            // oauth
	Expires time.Time         // oauth; zero: no expiry
	// AccountID is the ChatGPT account of an openai-codex credential.
	AccountID string
}

// AuthFile reads and writes Pi's auth.json.
type AuthFile struct {
	path string
	lock fileLock
}

// NewAuthFile returns the store for path.
func NewAuthFile(path string) *AuthFile {
	return &AuthFile{path: path, lock: newFileLock(path)}
}

// Path is the file's path.
func (f *AuthFile) Path() string { return f.path }

// Entry returns provider's credential; ok is false when there is none.
func (f *AuthFile) Entry(provider string) (Entry, bool, error) {
	all, err := f.read()
	if err != nil {
		return Entry{}, false, err
	}
	raw, ok := all[provider]
	if !ok {
		return Entry{}, false, nil
	}
	var e struct {
		Type      string            `json:"type"`
		Key       string            `json:"key"`
		Env       map[string]string `json:"env"`
		Access    string            `json:"access"`
		Refresh   string            `json:"refresh"`
		Expires   float64           `json:"expires"`
		AccountID string            `json:"accountId"`
	}
	if err := json.Unmarshal(raw, &e); err != nil {
		return Entry{}, false, errors.Errorf("auth.json: the %s entry is malformed", provider)
	}
	out := Entry{Type: e.Type, Key: e.Key, Env: e.Env, Access: e.Access, Refresh: e.Refresh, AccountID: e.AccountID}
	if e.Expires > 0 {
		out.Expires = time.UnixMilli(int64(e.Expires))
	}
	if out.Type != KindAPIKey && out.Type != KindOAuth {
		return Entry{}, false, errors.Errorf("auth.json: the %s entry has unknown type %q", provider, e.Type)
	}
	return out, true, nil
}

func (f *AuthFile) read() (map[string]json.RawMessage, error) {
	raw, err := os.ReadFile(f.path)
	if errors.Is(err, os.ErrNotExist) {
		return map[string]json.RawMessage{}, nil
	}
	if err != nil {
		return nil, errors.Wrap(err, "read auth.json")
	}
	s := strings.TrimSpace(strings.TrimPrefix(string(raw), "\ufeff"))
	all := map[string]json.RawMessage{}
	if s == "" {
		return all, nil
	}
	if err := json.Unmarshal([]byte(s), &all); err != nil {
		return nil, errors.New("auth.json is not a JSON object")
	}
	return all, nil
}

// update rewrites provider's entry under Pi's lock. fn receives the entry's
// current fields (unknown ones included) and changes them in place.
func (f *AuthFile) update(ctx context.Context, provider string, fn func(fields map[string]any) error) error {
	h, err := f.lock.acquire(ctx)
	if err != nil {
		return err
	}
	defer h.release()
	all, err := f.read() // re-read under the lock: another process may have rotated a token
	if err != nil {
		return err
	}
	fields := map[string]any{}
	if raw, ok := all[provider]; ok {
		if err := json.Unmarshal(raw, &fields); err != nil {
			return errors.Errorf("auth.json: the %s entry is malformed", provider)
		}
	}
	if err := fn(fields); err != nil {
		return err
	}
	b, err := json.Marshal(fields)
	if err != nil {
		return err
	}
	all[provider] = b
	out, err := json.MarshalIndent(all, "", "  ")
	if err != nil {
		return err
	}
	if err := h.check(); err != nil {
		return err
	}
	return writeOwnerOnly(f.path, out)
}

func writeOwnerOnly(path string, data []byte) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".auth-*.json")
	if err != nil {
		return errors.Wrap(err, "write auth.json")
	}
	name := tmp.Name()
	defer func() { _ = os.Remove(name) }()
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return errors.Wrap(err, "write auth.json")
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return errors.Wrap(err, "write auth.json")
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return errors.Wrap(err, "write auth.json")
	}
	if err := tmp.Close(); err != nil {
		return errors.Wrap(err, "write auth.json")
	}
	if err := os.Rename(name, path); err != nil {
		return errors.Wrap(err, "write auth.json")
	}
	if d, err := os.Open(dir); err == nil { // #nosec G304 -- the auth file's directory
		_ = d.Sync()
		_ = d.Close()
	}
	return nil
}

// providerStore is a credentials.Store for one auth.json provider. Engines
// ask with their own API type as Request.Provider; the store ignores it.
type providerStore struct {
	file     *AuthFile
	provider string
	onSave   func(fields map[string]any, c credentials.Credential)
}

var _ credentials.Store = (*providerStore)(nil)

func (s *providerStore) Load(_ context.Context, _ credentials.Request) (credentials.Credential, error) {
	e, ok, err := s.file.Entry(s.provider)
	if err != nil {
		return credentials.Credential{}, err
	}
	if !ok || e.Type != KindOAuth {
		return credentials.Credential{}, &credentials.ErrUnavailable{Provider: s.provider, Operation: "load"}
	}
	return credentials.Credential{AccessToken: e.Access, RefreshToken: e.Refresh, ExpiresAt: e.Expires}, nil
}

func (s *providerStore) Save(ctx context.Context, _ credentials.Request, c credentials.Credential) error {
	return s.file.update(ctx, s.provider, func(fields map[string]any) error {
		fields["type"] = KindOAuth
		fields["access"] = c.AccessToken
		fields["refresh"] = c.RefreshToken
		if c.ExpiresAt.IsZero() {
			fields["expires"] = 0
		} else {
			fields["expires"] = c.ExpiresAt.UnixMilli()
		}
		if s.onSave != nil {
			s.onSave(fields, c)
		}
		return nil
	})
}

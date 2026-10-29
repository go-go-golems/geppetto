// Package pi reads a Pi coding-agent directory (~/.pi/agent) and turns a Pi
// model reference ("provider/model") into a Geppetto engine with the
// credentials Pi stores.
//
// It reads four files, as Pi 0.85 writes them:
//
//   - settings.json: defaultProvider, defaultModel, defaultThinkingLevel, enabledModels
//   - models-store.json: the catalog Pi refreshed from providers ({provider: {models: [...]}})
//   - models.json: user-defined providers and models ({providers: {id: {baseUrl, api, apiKey, headers, compat, models}}})
//   - auth.json: one credential per provider ({type: "api_key", key} or {type: "oauth", access, refresh, expires})
//
// Credential material never enters InferenceSettings: engines get it through
// credentials.BearerTokenSource at request time. auth.json is written only to
// persist a refreshed OAuth credential, under Pi's own lock (see lock.go).
package pi

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-go-golems/geppetto/pkg/steps/ai/credentials/oauth"
	"github.com/pkg/errors"
)

// EnvAgentDir names the environment variable Pi reads for its agent directory.
const EnvAgentDir = "PI_CODING_AGENT_DIR"

// DefaultDir returns Pi's agent directory: envValue (the value of
// $PI_CODING_AGENT_DIR, which the host reads) when set, else ~/.pi/agent.
func DefaultDir(envValue string) string {
	if d := envValue; d != "" {
		if strings.HasPrefix(d, "~/") {
			if home, err := os.UserHomeDir(); err == nil {
				return filepath.Join(home, d[2:])
			}
		}
		return d
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".pi", "agent")
	}
	return filepath.Join(home, ".pi", "agent")
}

// Settings is the part of settings.json this package uses.
type Settings struct {
	DefaultProvider      string   `json:"defaultProvider,omitempty"`
	DefaultModel         string   `json:"defaultModel,omitempty"`
	DefaultThinkingLevel string   `json:"defaultThinkingLevel,omitempty"`
	EnabledModels        []string `json:"enabledModels,omitempty"`
}

// AgentDir is a loaded Pi agent directory.
type AgentDir struct {
	Dir      string
	Settings Settings
	Catalog  *Catalog
	Auth     *AuthFile

	opts      options
	providers map[string]providerConfig // models.json, for configured API keys
}

type options struct {
	env           func(string) (string, bool)
	allowCommands bool
	headers       map[string]map[string]string
	httpClient    *http.Client
	now           func() time.Time
	codexOAuth    *oauth.Config
	insecure      bool
}

// Option configures LoadAgentDir.
type Option func(*options)

// WithEnv replaces os.LookupEnv for "$NAME" references in configured values.
func WithEnv(lookup func(string) (string, bool)) Option {
	return func(o *options) { o.env = lookup }
}

// WithCommandValues allows "!command" values in models.json (apiKey, headers),
// which Pi runs through the shell. Off by default: a library should not run
// commands from a config file unless the host asks for it.
func WithCommandValues() Option {
	return func(o *options) { o.allowCommands = true }
}

// WithProviderHeaders adds request headers for a provider's models, for
// providers Pi configures in code (an extension) rather than in models.json.
func WithProviderHeaders(provider string, headers map[string]string) Option {
	return func(o *options) {
		if o.headers == nil {
			o.headers = map[string]map[string]string{}
		}
		h := map[string]string{}
		for k, v := range o.headers[provider] {
			h[k] = v
		}
		for k, v := range headers {
			h[k] = v
		}
		o.headers[provider] = h
	}
}

// WithHTTPClient sets the base HTTP client for engines and token refreshes.
func WithHTTPClient(c *http.Client) Option {
	return func(o *options) { o.httpClient = c }
}

// WithLocalProviders allows plain-HTTP and loopback/private provider URLs
// (local gateways, tests). Off by default, as in Geppetto's settings.
func WithLocalProviders() Option {
	return func(o *options) { o.insecure = true }
}

// WithClock replaces time.Now (tests).
func WithClock(now func() time.Time) Option {
	return func(o *options) { o.now = now }
}

// LoadAgentDir reads dir. Missing files are empty; malformed files are errors.
func LoadAgentDir(dir string, opts ...Option) (*AgentDir, error) {
	o := options{env: os.LookupEnv, now: time.Now}
	for _, opt := range opts {
		opt(&o)
	}
	a := &AgentDir{Dir: dir, opts: o, Auth: NewAuthFile(filepath.Join(dir, "auth.json"))}
	if err := readJSON(filepath.Join(dir, "settings.json"), &a.Settings); err != nil {
		return nil, err
	}
	var store map[string]struct {
		Models []modelDef `json:"models"`
	}
	if err := readJSON(filepath.Join(dir, "models-store.json"), &store); err != nil {
		return nil, err
	}
	var cfg struct {
		Providers map[string]providerConfig `json:"providers"`
	}
	if err := readJSON(filepath.Join(dir, "models.json"), &cfg); err != nil {
		return nil, err
	}
	a.providers = cfg.Providers
	stored := map[string][]modelDef{}
	for p, s := range store {
		stored[p] = s.Models
	}
	a.Catalog = buildCatalog(stored, cfg.Providers)
	return a, nil
}

func readJSON(path string, v any) error {
	raw, err := os.ReadFile(path) // #nosec G304 -- a file of the agent directory the host names
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return errors.Wrapf(err, "read %s", filepath.Base(path))
	}
	raw = []byte(strings.TrimPrefix(string(raw), "\ufeff"))
	if len(strings.TrimSpace(string(raw))) == 0 {
		return nil
	}
	if err := json.Unmarshal(raw, v); err != nil {
		// The error text of encoding/json quotes no values, only offsets and types.
		return errors.Wrapf(err, "parse %s", filepath.Base(path))
	}
	return nil
}

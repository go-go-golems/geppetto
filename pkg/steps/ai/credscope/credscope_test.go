package credscope_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"

	enginefactory "github.com/go-go-golems/geppetto/pkg/inference/engine/factory"

	"github.com/go-go-golems/geppetto/pkg/decision"
	decisionconfig "github.com/go-go-golems/geppetto/pkg/decision/config"
	decisionfactory "github.com/go-go-golems/geppetto/pkg/decision/factory"
	"github.com/go-go-golems/geppetto/pkg/embeddings"
	"github.com/go-go-golems/geppetto/pkg/inference/engine"
	"github.com/go-go-golems/geppetto/pkg/rerank"
	rerankconfig "github.com/go-go-golems/geppetto/pkg/rerank/config"
	rerankfactory "github.com/go-go-golems/geppetto/pkg/rerank/factory"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/providers/pi"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/types"
	"github.com/go-go-golems/geppetto/pkg/turns"
)

// secret is the marker every row's credential carries.
const secret = "MATRIX-SECRET"

// recorder answers every request itself (nothing leaves the process) and
// records which hosts saw the marker in a header or the query. Hosts in
// redirect answer 307 with the mapped Location.
type recorder struct {
	mu       sync.Mutex
	contacts map[string]bool // host -> saw the marker
	redirect map[string]string
}

func newRecorder(redirect map[string]string) *recorder {
	return &recorder{contacts: map[string]bool{}, redirect: redirect}
}

func (r *recorder) RoundTrip(req *http.Request) (*http.Response, error) {
	var all strings.Builder
	for k, vs := range req.Header {
		all.WriteString(k + ":" + strings.Join(vs, ",") + "\n")
	}
	all.WriteString(req.URL.RawQuery)
	r.mu.Lock()
	r.contacts[req.URL.Host] = r.contacts[req.URL.Host] || strings.Contains(all.String(), secret)
	loc, redirect := r.redirect[req.URL.Host]
	r.mu.Unlock()
	if req.Body != nil {
		_, _ = io.Copy(io.Discard, req.Body)
		_ = req.Body.Close()
	}
	if redirect {
		return &http.Response{StatusCode: http.StatusTemporaryRedirect, Header: http.Header{"Location": {loc}},
			Body: io.NopCloser(strings.NewReader("")), Request: req}, nil
	}
	return &http.Response{StatusCode: http.StatusBadRequest, Header: http.Header{"Content-Type": {"application/json"}},
		Body: io.NopCloser(strings.NewReader(`{"error":{"message":"matrix","code":"matrix"}}`)), Request: req}, nil
}

func (r *recorder) hosts() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	var hs []string
	for h := range r.contacts {
		hs = append(hs, h)
	}
	sort.Strings(hs)
	return hs
}

// row is one credential path under one configuration.
type row struct {
	name string
	// run drives the path with the marker as its credential, sending
	// through client (nil for paths that build their own from Pi).
	run func(t *testing.T, client *http.Client) error
	// allowed are the only hosts that may be contacted; empty means none
	// (the configuration must be refused before sending).
	allowed []string
	// redirectFrom, when set, answers 307 to other.example from that host.
	redirectFrom string
}

func newSettings(t *testing.T, client *http.Client) *settings.InferenceSettings {
	t.Helper()
	s, err := settings.NewInferenceSettings()
	if err != nil {
		t.Fatal(err)
	}
	s.Client.HTTPClient = client
	model := "m"
	s.Chat.Engine = &model
	return s
}

func userTurn() *turns.Turn {
	tu := &turns.Turn{}
	turns.AppendBlock(tu, turns.NewUserTextBlock("hi"))
	return tu
}

func runEngine(e engine.Engine, err error) error {
	if err != nil {
		return err
	}
	_, err = e.RunInference(context.Background(), userTurn())
	return err
}

func chatRow(apiType types.ApiType, keySlot, urlSlot, baseURL string, build func(*settings.InferenceSettings) (engine.Engine, error)) func(*testing.T, *http.Client) error {
	return func(t *testing.T, c *http.Client) error {
		s := newSettings(t, c)
		at := apiType
		s.Chat.ApiType = &at
		s.API.APIKeys[keySlot] = secret
		s.API.BaseUrls[urlSlot] = baseURL
		return runEngine(build(s))
	}
}

func openaiEngine(s *settings.InferenceSettings) (engine.Engine, error) {
	return enginefactory.NewEngineFromSettings(s)
}
func claudeEngine(s *settings.InferenceSettings) (engine.Engine, error) {
	return enginefactory.NewEngineFromSettings(s)
}
func geminiEngine(s *settings.InferenceSettings) (engine.Engine, error) {
	return enginefactory.NewEngineFromSettings(s)
}
func responsesEngine(s *settings.InferenceSettings) (engine.Engine, error) {
	return enginefactory.NewEngineFromSettings(s)
}

func embedRow(typ, keySlot, urlSlot, baseURL string) func(*testing.T, *http.Client) error {
	return func(t *testing.T, c *http.Client) error {
		s := newSettings(t, c)
		s.API.APIKeys[keySlot] = secret
		if baseURL != "" {
			s.API.BaseUrls[urlSlot] = baseURL
		}
		s.Embeddings.Type, s.Embeddings.Engine, s.Embeddings.Dimensions = typ, "e", 8
		p, err := embeddings.NewSettingsFactoryFromInferenceSettings(s).NewProvider()
		if err != nil {
			return err
		}
		_, err = p.GenerateEmbedding(context.Background(), "hi")
		return err
	}
}

func rerankRow(baseURL string) func(*testing.T, *http.Client) error {
	return func(t *testing.T, c *http.Client) error {
		s := newSettings(t, c)
		s.API.APIKeys["cohere-api-key"] = secret
		s.API.BaseUrls["cohere-base-url"] = baseURL
		cfg, err := rerankconfig.NewRerankConfig()
		if err != nil {
			t.Fatal(err)
		}
		cfg.Type, cfg.Engine = "cohere", "rerank-v3.5"
		s.Rerank = cfg
		f, err := rerankfactory.NewSettingsFactoryFromInferenceSettings(s)
		if err != nil {
			return err
		}
		p, err := f.NewProvider()
		if err != nil {
			return err
		}
		_, err = p.Rerank(context.Background(), rerank.Request{Query: "q", Documents: []rerank.Document{{ID: "1", Text: "d"}}, TopN: 1})
		return err
	}
}

func decisionRow(baseURL string) func(*testing.T, *http.Client) error {
	return func(t *testing.T, c *http.Client) error {
		s := newSettings(t, c)
		s.API.APIKeys["systemone-api-key"] = secret
		s.API.BaseUrls["systemone-base-url"] = baseURL
		cfg, err := decisionconfig.NewDecisionConfig()
		if err != nil {
			t.Fatal(err)
		}
		cfg.Type, cfg.Engine = "systemone", "djev"
		s.Decision = cfg
		f, err := decisionfactory.NewSettingsFactoryFromInferenceSettings(s)
		if err != nil {
			return err
		}
		p, err := f.NewProvider()
		if err != nil {
			return err
		}
		_, err = p.Decide(context.Background(), decision.Request{State: "s",
			Questions: map[string]decision.Question{"q": {Type: decision.Noul, Instructions: "?"}}})
		return err
	}
}

// piRow resolves provider/m from a Pi agent dir whose only provider entry is
// given, and runs it (engine or decider).
func piRow(provider string, entry map[string]any, decider bool) func(*testing.T, *http.Client) error {
	return func(t *testing.T, c *http.Client) error {
		dir := t.TempDir()
		write := func(name string, v any) {
			b, _ := json.Marshal(v)
			if err := os.WriteFile(filepath.Join(dir, name), b, 0o600); err != nil {
				t.Fatal(err)
			}
		}
		write("settings.json", map[string]any{"defaultProvider": provider, "defaultModel": "m"})
		entry["apiKey"] = secret
		entry["models"] = []any{map[string]any{"id": "m"}}
		write("models.json", map[string]any{"providers": map[string]any{provider: entry}})
		a, err := pi.LoadAgentDir(dir, pi.WithHTTPClient(c))
		if err != nil {
			t.Fatal(err)
		}
		ctx := context.Background()
		if decider {
			d, err := a.Decider(ctx, provider+"/m", pi.DecisionOptions{})
			if err != nil {
				return err
			}
			_, err = d.Decide(ctx, decision.Request{State: "s", Questions: map[string]decision.Question{"q": {Type: decision.Noul, Instructions: "?"}}})
			return err
		}
		r, err := a.Resolve(ctx, provider+"/m", pi.ResolveOptions{})
		if err != nil {
			return err
		}
		return runEngine(r.Engine())
	}
}

const gw = "https://gateway.example/v1"

func rows() []row {
	var rs []row
	add := func(r row) { rs = append(rs, r) }
	// Each configured path: the marker may reach gateway.example only, and a
	// 307 from gateway.example to other.example must be refused.
	for _, r := range []row{
		{name: "openai chat", run: chatRow(types.ApiTypeOpenAI, "openai-api-key", "openai-base-url", gw, openaiEngine)},
		{name: "claude", run: chatRow(types.ApiTypeClaude, "claude-api-key", "claude-base-url", "https://gateway.example", claudeEngine)},
		{name: "gemini", run: chatRow(types.ApiTypeGemini, "gemini-api-key", "gemini-base-url", "https://gateway.example/", geminiEngine)},

		{name: "responses open-responses slot", run: chatRow(types.ApiTypeOpenResponses, "open-responses-api-key", "open-responses-base-url", gw, responsesEngine)},
		{name: "embeddings openai", run: embedRow("openai", "openai-api-key", "openai-base-url", gw)},
		{name: "embeddings cohere", run: embedRow("cohere", "cohere-api-key", "cohere-base-url", "https://gateway.example")},
		{name: "rerank cohere", run: rerankRow("https://gateway.example")},
		{name: "decision systemone", run: decisionRow(gw)},
		{name: "pi openai-completions", run: piRow("gateway", map[string]any{"api": "openai-completions", "baseUrl": gw,
			"headers": map[string]string{"x-team": secret}}, false)},
		{name: "pi anthropic-messages", run: piRow("gateway", map[string]any{"api": "anthropic-messages", "baseUrl": "https://gateway.example"}, false)},
		{name: "pi decider", run: piRow("gateway", map[string]any{"api": "openai-completions", "baseUrl": gw}, true)},
	} {
		r.allowed = []string{"gateway.example"}
		add(r)
		r.name += " + 307 to other.example"
		r.redirectFrom = "gateway.example"
		add(r)
	}
	add(row{name: "responses legacy protocol refused", run: chatRow(types.ApiType("openai-responses"), "openai-api-key", "openai-base-url", gw, responsesEngine)})
	add(row{name: "responses cannot inherit OpenAI credentials", run: chatRow(types.ApiTypeOpenResponses, "openai-api-key", "openai-base-url", gw, responsesEngine)})
	// Configurations that must be refused before anything is sent.
	for _, u := range []string{" ", "/"} {
		add(row{name: "openai chat base URL " + quote(u), run: chatRow(types.ApiTypeOpenAI, "openai-api-key", "openai-base-url", u, openaiEngine)})
		add(row{name: "claude base URL " + quote(u), run: chatRow(types.ApiTypeClaude, "claude-api-key", "claude-base-url", u, claudeEngine)})
		add(row{name: "pi foreign provider baseUrl " + quote(u), run: piRow("fireworks", map[string]any{"api": "openai-completions", "baseUrl": u}, false)})
		add(row{name: "pi decider foreign provider baseUrl " + quote(u), run: piRow("fireworks", map[string]any{"api": "openai-completions", "baseUrl": u}, true)})
	}
	add(row{name: "pi foreign provider without baseUrl (openai protocol)", run: piRow("fireworks", map[string]any{"api": "openai-completions"}, false)})
	add(row{name: "pi foreign provider without baseUrl (anthropic protocol)", run: piRow("zai", map[string]any{"api": "anthropic-messages"}, false)})
	add(row{name: "responses open-responses key with only openai-base-url",
		run: chatRow(types.ApiTypeOpenResponses, "open-responses-api-key", "openai-base-url", "https://api.openai.com/v1", responsesEngine)})
	// Defaults stay with the slot's own vendor.
	add(row{name: "embeddings openai default", run: embedRow("openai", "openai-api-key", "openai-base-url", ""), allowed: []string{"api.openai.com"}})
	add(row{name: "pi first-party default", run: piRow("openai", map[string]any{"api": "openai-completions"}, false), allowed: []string{"api.openai.com"}})
	add(row{name: "pi lunaroute decider default", run: piRow("lunaroute", map[string]any{"api": "openai-completions"}, true), allowed: []string{"gw.lunaroute.com"}})
	return append(rs, additionalBoundRows()...)
}

func quote(s string) string { return `"` + s + `"` }

// tripwire replaces http.DefaultTransport for the whole package: a path that
// ignores the client it was given (and would otherwise reach the real
// network, as OpenAI embeddings did before GEPPETTO-CREDSCOPE-001) is
// recorded here and fails its row instead of sending anything.
type tripwire struct {
	mu    sync.Mutex
	hosts []string
}

func (t *tripwire) RoundTrip(req *http.Request) (*http.Response, error) {
	t.mu.Lock()
	t.hosts = append(t.hosts, req.URL.Host)
	t.mu.Unlock()
	if req.Body != nil {
		_ = req.Body.Close()
	}
	return nil, errors.New("credscope tripwire: request bypassed the configured client")
}

func (t *tripwire) take() []string {
	t.mu.Lock()
	defer t.mu.Unlock()
	h := t.hosts
	t.hosts = nil
	return h
}

var bypass = &tripwire{}

func TestMain(m *testing.M) {
	http.DefaultTransport = bypass
	os.Exit(m.Run())
}

// TestNoCredentialLeavesItsOrigin is the invariant: for every path and
// configuration, the hosts contacted are a subset of the allowed ones, and
// a refused configuration contacts no host at all.
func TestNoCredentialLeavesItsOrigin(t *testing.T) {
	for _, r := range rows() {
		t.Run(r.name, func(t *testing.T) {
			redirect := map[string]string{}
			if r.redirectFrom != "" {
				redirect[r.redirectFrom] = "https://other.example/stolen"
			}
			rec := newRecorder(redirect)
			err := r.run(t, &http.Client{Transport: rec})
			if err == nil {
				t.Fatal("every row ends in an error (the recorder answers 400, 307 or the config is refused)")
			}
			if b := bypass.take(); len(b) > 0 {
				t.Fatalf("path bypassed the configured HTTP client and tried to reach %v (err %v)", b, err)
			}
			got := rec.hosts()
			for _, h := range got {
				if !contains(r.allowed, h) {
					t.Fatalf("contacted %s (allowed %v; all contacts %v; err %v)", h, r.allowed, got, err)
				}
			}
			if len(r.allowed) > 0 && len(got) == 0 {
				t.Fatalf("expected a request to %v, none was sent (err %v)", r.allowed, err)
			}
			if len(r.allowed) > 0 && !rec.contacts[r.allowed[0]] {
				t.Fatalf("the credential did not reach its own origin %s; the row is not exercising the credential path (err %v)", r.allowed[0], err)
			}
		})
	}
}

func contains(xs []string, x string) bool {
	for _, y := range xs {
		if x == y {
			return true
		}
	}
	return false
}

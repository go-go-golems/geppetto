package pi

import (
	"context"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-go-golems/geppetto/pkg/decision"
	"github.com/go-go-golems/geppetto/pkg/security"
	"github.com/go-go-golems/geppetto/pkg/turns"
)

// recordingTransport answers every request itself and records which secret
// markers reached which host. A host in redirect answers 307 to its value.
type recordingTransport struct {
	mu       sync.Mutex
	seen     map[string][]string // host -> markers
	redirect map[string]string
}

func (r *recordingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	var all strings.Builder
	for k, vs := range req.Header {
		all.WriteString(k + ":" + strings.Join(vs, ",") + "\n")
	}
	all.WriteString(req.URL.RawQuery)
	r.mu.Lock()
	if r.seen == nil {
		r.seen = map[string][]string{}
	}
	for _, m := range []string{"FW-SECRET", "ZAI-SECRET", "OA-SECRET", "GW-SECRET", "TEAM-SECRET"} {
		if strings.Contains(all.String(), m) {
			r.seen[req.URL.Host] = append(r.seen[req.URL.Host], m)
		}
	}
	if _, ok := r.seen[req.URL.Host]; !ok {
		r.seen[req.URL.Host] = nil
	}
	loc, redirect := r.redirect[req.URL.Host]
	r.mu.Unlock()
	if req.Body != nil {
		_, _ = io.Copy(io.Discard, req.Body)
		_ = req.Body.Close()
	}
	if redirect {
		return &http.Response{StatusCode: http.StatusTemporaryRedirect, Header: http.Header{"Location": {loc}}, Body: io.NopCloser(strings.NewReader("")), Request: req}, nil
	}
	return &http.Response{StatusCode: http.StatusBadRequest, Header: http.Header{"Content-Type": {"application/json"}},
		Body: io.NopCloser(strings.NewReader(`{"error":{"message":"probe"}}`)), Request: req}, nil
}

func (r *recordingTransport) hosts() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	var hs []string
	for h := range r.seen {
		hs = append(hs, h)
	}
	sort.Strings(hs)
	return hs
}

func credscopeAgentDir(t *testing.T, providers map[string]any) string {
	t.Helper()
	dir := t.TempDir()
	writeFile(t, dir, "settings.json", map[string]any{"defaultProvider": "openai", "defaultModel": "m"})
	writeFile(t, dir, "models.json", map[string]any{"providers": providers})
	return dir
}

func runModel(t *testing.T, a *AgentDir, ref string) error {
	t.Helper()
	r, err := a.Resolve(context.Background(), ref, ResolveOptions{})
	if err != nil {
		return err
	}
	e, err := r.Engine()
	require.NoError(t, err)
	tu := &turns.Turn{}
	turns.AppendBlock(tu, turns.NewUserTextBlock("hi"))
	_, err = e.RunInference(context.Background(), tu)
	return err
}

func model(api, key, baseURL string, extra map[string]any) map[string]any {
	p := map[string]any{"api": api, "apiKey": key, "models": []any{map[string]any{"id": "m"}}}
	if baseURL != "" {
		p["baseUrl"] = baseURL
	}
	for k, v := range extra {
		p[k] = v
	}
	return p
}

// A third-party provider without a baseUrl used to inherit the wire
// protocol's vendor URL (api.openai.com, api.anthropic.com) and send its key
// there. Now Resolve fails before anything is sent.
func TestResolveRefusesProtocolDefaultForForeignProviders(t *testing.T) {
	for _, tc := range []struct{ provider, api, key, baseURL string }{
		{"fireworks", "openai-completions", "FW-SECRET", ""},
		{"fireworks", "openai-completions", "FW-SECRET", " "},
		{"fireworks", "openai-responses", "FW-SECRET", "/"},
		{"zai", "anthropic-messages", "ZAI-SECRET", ""},
		{"zai", "anthropic-messages", "ZAI-SECRET", "api.z.ai/v1"}, // not absolute
	} {
		rec := &recordingTransport{}
		a, err := LoadAgentDir(credscopeAgentDir(t, map[string]any{tc.provider: model(tc.api, tc.key, tc.baseURL, nil)}),
			WithHTTPClient(&http.Client{Transport: rec}))
		require.NoError(t, err)
		err = runModel(t, a, tc.provider+"/m")
		var ne *NoEndpointError
		require.ErrorAs(t, err, &ne, "%s %q", tc.api, tc.baseURL)
		assert.Contains(t, err.Error(), "no baseUrl")
		assert.Empty(t, rec.hosts(), "%s %q: nothing may be sent", tc.api, tc.baseURL)
	}
}

// First-party providers keep their own default endpoint.
func TestResolveFirstPartyDefaults(t *testing.T) {
	rec := &recordingTransport{}
	a, err := LoadAgentDir(credscopeAgentDir(t, map[string]any{"openai": model("openai-completions", "OA-SECRET", "", nil)}),
		WithHTTPClient(&http.Client{Transport: rec}))
	require.NoError(t, err)
	_ = runModel(t, a, "openai/m")
	assert.Equal(t, []string{"api.openai.com"}, rec.hosts())
	assert.Equal(t, []string{"OA-SECRET"}, rec.seen["api.openai.com"])
}

// A redirect to another origin is refused; neither the key nor a models.json
// header reaches the redirect target (Go forwards custom headers on redirect).
func TestResolvePinsClientAcrossRedirects(t *testing.T) {
	rec := &recordingTransport{redirect: map[string]string{"gateway.example": "https://other.example/v1/chat/completions"}}
	a, err := LoadAgentDir(credscopeAgentDir(t, map[string]any{"gateway": model("openai-completions", "GW-SECRET", "https://gateway.example/v1",
		map[string]any{"headers": map[string]string{"x-team": "TEAM-SECRET"}})}),
		WithHTTPClient(&http.Client{Transport: rec}))
	require.NoError(t, err)
	err = runModel(t, a, "gateway/m")
	require.ErrorIs(t, err, security.ErrOriginNotAllowed)
	assert.Equal(t, []string{"gateway.example"}, rec.hosts(), "other.example must never be contacted")
}

// The decision path uses the same endpoint resolution: LunaRoute keeps its
// default, a foreign provider without baseUrl is refused, and the decider's
// client is pinned.
func TestDeciderUsesProviderEndpoint(t *testing.T) {
	rec := &recordingTransport{redirect: map[string]string{"gateway.example": "https://other.example/v1/systemone"}}
	a, err := LoadAgentDir(credscopeAgentDir(t, map[string]any{
		"lunaroute": model("openai-completions", "GW-SECRET", "", nil),
		"fireworks": model("openai-completions", "FW-SECRET", "", nil),
		"gateway":   model("openai-completions", "GW-SECRET", "https://gateway.example/v1", nil),
	}), WithHTTPClient(&http.Client{Transport: rec}))
	require.NoError(t, err)
	ctx := context.Background()
	q := decision.Request{State: "s", Questions: map[string]decision.Question{"q": {Type: decision.Noul, Instructions: "?"}}}

	d, err := a.Decider(ctx, "lunaroute/m", DecisionOptions{})
	require.NoError(t, err)
	_, _ = d.Decide(ctx, q)
	assert.Equal(t, []string{"gw.lunaroute.com"}, rec.hosts())

	_, err = a.Decider(ctx, "fireworks/m", DecisionOptions{})
	var ne *NoEndpointError
	require.ErrorAs(t, err, &ne)
	require.ErrorIs(t, err, decision.ErrInvalidRequest)

	rec.seen = nil
	d, err = a.Decider(ctx, "gateway/m", DecisionOptions{})
	require.NoError(t, err)
	_, err = d.Decide(ctx, q)
	require.Error(t, err) // systemone rejects redirects itself; the pin is the second line
	assert.Equal(t, []string{"gateway.example"}, rec.hosts())
}

func TestResolvedSettingsCannotRebindCredential(t *testing.T) {
	for _, protocol := range []string{APIOpenAICompletions, APIOpenAIResponses, APIAnthropicMessages, APIGoogleGenerativeAI} {
		t.Run(protocol, func(t *testing.T) {
			rec := &recordingTransport{}
			a, err := LoadAgentDir(credscopeAgentDir(t, map[string]any{"gateway": model(protocol, "GW-SECRET", "https://gateway.example/v1", nil)}), WithHTTPClient(&http.Client{Transport: rec}))
			require.NoError(t, err)
			r, err := a.Resolve(t.Context(), "gateway/m", ResolveOptions{})
			require.NoError(t, err)
			for _, key := range r.Settings.API.APIKeys {
				require.NotContains(t, key, "SECRET")
			}
			for name := range r.Settings.API.BaseUrls {
				r.Settings.API.BaseUrls[name] = "https://foreign.example/v1"
			}
			r.Settings.API.APIKeys["claude-api-key"] = "WRONG"
			e, err := r.Engine()
			require.NoError(t, err)
			_, _ = e.RunInference(t.Context(), &turns.Turn{Blocks: []turns.Block{turns.NewUserTextBlock("test")}})
			require.Equal(t, []string{"gateway.example"}, rec.hosts())
			require.Contains(t, rec.seen["gateway.example"], "GW-SECRET")
		})
	}
}

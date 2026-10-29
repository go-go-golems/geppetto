package pi

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-go-golems/geppetto/pkg/inference/engine"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/credentials"
	"github.com/go-go-golems/geppetto/pkg/turns"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeFile(t *testing.T, dir, name string, v any) {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, name), b, 0o600))
}

// agentDir writes a Pi agent directory shaped like a real one: a stored
// lunaroute catalog, a models.json provider with overrides, an unsupported
// provider, and a legacy top-level models.json key Pi ignores.
func agentDir(t *testing.T, baseURL string) string {
	t.Helper()
	dir := t.TempDir()
	writeFile(t, dir, "settings.json", map[string]any{
		"defaultProvider": "lunaroute", "defaultModel": "glm-5.3", "defaultThinkingLevel": "minimal",
		"enabledModels": []string{"lunaroute/glm-5.3"},
	})
	writeFile(t, dir, "models-store.json", map[string]any{
		"lunaroute": map[string]any{"models": []any{
			map[string]any{"id": "glm-5.3", "name": "GLM", "api": "openai-completions", "provider": "lunaroute",
				"baseUrl": "https://gw.lunaroute.com/v1", "reasoning": true, "input": []string{"text", "image"},
				"contextWindow": 131072, "maxTokens": 131072},
			map[string]any{"id": "djev", "api": "openai-completions", "provider": "lunaroute",
				"baseUrl": "https://gw.lunaroute.com/v1", "reasoning": false, "maxTokens": 1024},
		}},
		"amazon-bedrock": map[string]any{"models": []any{
			map[string]any{"id": "claude", "api": "bedrock-converse-stream", "provider": "amazon-bedrock"},
		}},
	})
	off := "none"
	writeFile(t, dir, "models.json", map[string]any{
		"fireworks": map[string]any{"api": "openai-completions", "apiKey": "ignored"},
		"providers": map[string]any{"lunaroute": map[string]any{
			"baseUrl": baseURL, "api": "openai-completions", "apiKey": "$LUNA_KEY",
			"headers": map[string]string{"x-team": "${TEAM}"},
			"models": []any{map[string]any{"id": "inkling", "reasoning": true, "maxOutputTokens": 65536,
				"compat":           map[string]any{"maxTokensField": "max_tokens", "supportsReasoningEffort": true},
				"thinkingLevelMap": map[string]any{"off": off, "low": "low", "high": "high", "xhigh": nil}}},
		}},
	})
	return dir
}

func env(vals map[string]string) Option {
	return WithEnv(func(k string) (string, bool) { v, ok := vals[k]; return v, ok })
}

func TestCatalogMerge(t *testing.T) {
	a, err := LoadAgentDir(agentDir(t, "https://proxy.example/v1"))
	require.NoError(t, err)
	assert.Equal(t, "minimal", a.Settings.DefaultThinkingLevel)
	assert.Equal(t, []string{"lunaroute/glm-5.3"}, a.Settings.EnabledModels)

	m, ok := a.Catalog.Lookup("lunaroute/glm-5.3")
	require.True(t, ok)
	assert.Equal(t, "https://proxy.example/v1", m.BaseURL, "models.json's provider baseUrl applies to stored models")
	assert.True(t, m.Supported && m.Reasoning && m.Images())
	assert.Equal(t, 131072, m.MaxTokens)
	assert.Equal(t, map[string]string{"x-team": "${TEAM}"}, m.Headers)

	in, ok := a.Catalog.Lookup("lunaroute/inkling")
	require.True(t, ok, "a models.json-only model joins the catalog")
	assert.Equal(t, 65536, in.MaxTokens, "maxOutputTokens stands in for maxTokens")
	assert.Equal(t, "max_tokens", in.Compat.MaxTokensField)
	assert.Equal(t, []string{"off", "minimal", "low", "medium", "high"}, in.ThinkingLevels(), "xhigh: null is not offered")

	b, ok := a.Catalog.Lookup("amazon-bedrock/claude")
	require.True(t, ok)
	assert.False(t, b.Supported)
	_, ok = a.Catalog.Lookup("fireworks/anything")
	assert.False(t, ok, "top-level models.json keys other than providers are ignored")
	_, err = a.Resolve(context.Background(), "amazon-bedrock/claude", ResolveOptions{})
	var ua *UnsupportedAPIError
	assert.ErrorAs(t, err, &ua)
	_, err = a.Resolve(context.Background(), "nobody/nothing", ResolveOptions{})
	assert.ErrorIs(t, err, ErrUnknownModel)
}

func TestMissingFilesAreEmpty(t *testing.T) {
	a, err := LoadAgentDir(t.TempDir())
	require.NoError(t, err)
	assert.Empty(t, a.Catalog.Models())
	require.NoError(t, os.WriteFile(filepath.Join(a.Dir, "models.json"), []byte("{oops"), 0o600))
	_, err = LoadAgentDir(a.Dir)
	assert.ErrorContains(t, err, "parse models.json")
}

func TestResolveValue(t *testing.T) {
	a, err := LoadAgentDir(t.TempDir(), env(map[string]string{"KEY": "k1", "B": "b2"}))
	require.NoError(t, err)
	ctx := context.Background()
	for in, want := range map[string]string{
		"literal": "literal", "$KEY": "k1", "${KEY}-x": "k1-x", "a$${B}": "a${B}", "$!x": "!x", "pre$KEY/$B": "prek1/b2", "$": "$", "${bad name}": "${bad name}",
	} {
		got, err := a.resolveValue(ctx, in, nil)
		require.NoError(t, err, in)
		assert.Equal(t, want, got, in)
	}
	got, err := a.resolveValue(ctx, "$KEY", map[string]string{"KEY": "scoped"})
	require.NoError(t, err)
	assert.Equal(t, "scoped", got, "the credential's env map wins")
	_, err = a.resolveValue(ctx, "$MISSING", nil)
	assert.ErrorContains(t, err, "MISSING")
	_, err = a.resolveValue(ctx, "!echo secret", nil)
	assert.ErrorContains(t, err, "WithCommandValues")

	c, err := LoadAgentDir(t.TempDir(), WithCommandValues())
	require.NoError(t, err)
	got, err = c.resolveValue(ctx, "!printf ' from-cmd \\n'", nil)
	require.NoError(t, err)
	assert.Equal(t, "from-cmd", got)
	_, err = c.resolveValue(ctx, "!exit 3", nil)
	assert.EqualError(t, err, "the configured command failed")
}

// sse writes an OpenAI Chat Completions stream.
func sse(w http.ResponseWriter, chunks ...string) {
	w.Header().Set("Content-Type", "text/event-stream")
	for _, c := range chunks {
		_, _ = io.WriteString(w, "data: "+c+"\n\n")
	}
	_, _ = io.WriteString(w, "data: [DONE]\n\n")
}

func TestResolveOpenAICompletionsEndToEnd(t *testing.T) {
	var got struct {
		path, auth, team, agent string
		body                    map[string]any
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got.path, got.auth, got.team, got.agent = r.URL.Path, r.Header.Get("Authorization"), r.Header.Get("x-team"), r.Header.Get("lunaroute-agent")
		_ = json.NewDecoder(r.Body).Decode(&got.body)
		sse(w,
			`{"id":"c1","model":"inkling","choices":[{"index":0,"delta":{"reasoning_content":"hmm"}}]}`,
			`{"id":"c1","model":"inkling","choices":[{"index":0,"delta":{"content":"Hel"}}]}`,
			`{"id":"c1","model":"inkling","choices":[{"index":0,"delta":{"content":"lo"},"finish_reason":"stop"}]}`,
			`{"id":"c1","model":"inkling","choices":[],"usage":{"prompt_tokens":5,"completion_tokens":7,"total_tokens":12}}`)
	}))
	defer srv.Close()
	a, err := LoadAgentDir(agentDir(t, srv.URL+"/v1"), env(map[string]string{"LUNA_KEY": "luna-secret", "TEAM": "slop"}),
		WithProviderHeaders("lunaroute", map[string]string{"lunaroute-agent": "slop"}), WithLocalProviders())
	require.NoError(t, err)

	r, err := a.Resolve(context.Background(), "lunaroute/inkling", ResolveOptions{Thinking: "high", MaxTokens: 4000})
	require.NoError(t, err)
	assert.Empty(t, r.Settings.API.APIKeys, "no key in settings")
	eng, err := r.Engine()
	require.NoError(t, err)
	out, err := eng.RunInference(context.Background(), &turns.Turn{Blocks: []turns.Block{turns.NewUserTextBlock("hi")}})
	require.NoError(t, err)

	assert.Equal(t, "/v1/chat/completions", got.path)
	assert.Equal(t, "Bearer luna-secret", got.auth)
	assert.Equal(t, "slop", got.team, "models.json header, resolved from the environment")
	assert.Equal(t, "slop", got.agent, "host-supplied provider header")
	assert.Equal(t, "inkling", got.body["model"])
	assert.Equal(t, "high", got.body["reasoning_effort"])
	assert.EqualValues(t, 4000, got.body["max_tokens"], "compat.maxTokensField = max_tokens")
	assert.NotContains(t, got.body, "max_completion_tokens")
	assert.NotContains(t, got.body, "temperature", "Pi sends no temperature")
	var text string
	for _, b := range out.Blocks {
		if b.Kind == turns.BlockKindLLMText {
			text, _ = b.Payload[turns.PayloadKeyText].(string)
		}
	}
	assert.Equal(t, "Hello", text)

	// Per-turn override of the response bound, as slop's round loop does.
	turn := &turns.Turn{Blocks: []turns.Block{turns.NewUserTextBlock("again")}}
	n := 77
	require.NoError(t, engine.KeyInferenceConfig.Set(&turn.Data, engine.InferenceConfig{MaxResponseTokens: &n}))
	_, err = eng.RunInference(context.Background(), turn)
	require.NoError(t, err)
	assert.EqualValues(t, 77, got.body["max_tokens"])
}

func TestCredentialPrecedence(t *testing.T) {
	dir := agentDir(t, "https://proxy.example/v1")
	a, err := LoadAgentDir(dir, env(map[string]string{"LUNA_KEY": "from-models-json"}))
	require.NoError(t, err)
	_, key, err := a.credential(context.Background(), "lunaroute")
	require.NoError(t, err)
	assert.Equal(t, "from-models-json", key)

	writeFile(t, dir, "auth.json", map[string]any{"lunaroute": map[string]any{"type": "oauth", "access": "gateway-key", "refresh": "r", "expires": 1}})
	_, key, err = a.credential(context.Background(), "lunaroute")
	require.NoError(t, err)
	assert.Equal(t, "gateway-key", key, "auth.json wins; an OAuth access token without a refresher is the key")

	_, _, err = a.credential(context.Background(), "zai")
	var nc *NoCredentialError
	assert.ErrorAs(t, err, &nc)
	assert.NotContains(t, err.Error(), "gateway-key")
}

func TestAuthFileSavePreservesFieldsAndUsesLock(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "auth.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"openai-codex":{"type":"oauth","access":"a1","refresh":"r1","expires":1000,"accountId":"acc","extra":{"x":1}},"zai":{"type":"api_key","key":"k"}}`), 0o644))
	f := NewAuthFile(path)
	e, ok, err := f.Entry("openai-codex")
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, "acc", e.AccountID)
	assert.Equal(t, time.UnixMilli(1000), e.Expires)

	s := &providerStore{file: f, provider: "openai-codex"}
	exp := time.UnixMilli(5_000_000)
	require.NoError(t, s.Save(context.Background(), credentials.Request{}, credentials.Credential{AccessToken: "a2", RefreshToken: "r2", ExpiresAt: exp}))
	c, err := s.Load(context.Background(), credentials.Request{})
	require.NoError(t, err)
	assert.Equal(t, credentials.Credential{AccessToken: "a2", RefreshToken: "r2", ExpiresAt: exp}, c)

	var all map[string]map[string]any
	raw, _ := os.ReadFile(path)
	require.NoError(t, json.Unmarshal(raw, &all))
	assert.Equal(t, map[string]any{"x": float64(1)}, all["openai-codex"]["extra"], "unknown fields survive")
	assert.Equal(t, "acc", all["openai-codex"]["accountId"])
	assert.Equal(t, "k", all["zai"]["key"], "other providers survive")
	st, _ := os.Stat(path)
	assert.Equal(t, os.FileMode(0o600), st.Mode().Perm())
	_, err = os.Stat(path + ".lock")
	assert.True(t, os.IsNotExist(err), "the lock is released")
}

func TestLockWaitsStaleAndCompromised(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "auth.json")
	l := fileLock{path: path + ".lock", stale: 300 * time.Millisecond, update: 50 * time.Millisecond, deadline: 150 * time.Millisecond}

	// A fresh foreign lock blocks until the deadline.
	require.NoError(t, os.Mkdir(l.path, 0o700))
	_, err := l.acquire(context.Background())
	assert.ErrorContains(t, err, "locked by another process")

	// A stale one is removed.
	old := time.Now().Add(-time.Second)
	require.NoError(t, os.Chtimes(l.path, old, old))
	h, err := l.acquire(context.Background())
	require.NoError(t, err)

	// Concurrent holders serialize.
	var wg sync.WaitGroup
	var order []int
	var mu sync.Mutex
	l2 := l
	l2.deadline = 2 * time.Second
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			h, err := l2.acquire(context.Background())
			if !assert.NoError(t, err) {
				return
			}
			mu.Lock()
			order = append(order, i)
			mu.Unlock()
			h.release()
		}(i)
	}
	time.Sleep(120 * time.Millisecond) // the holder keeps refreshing: not stale
	mu.Lock()
	assert.Empty(t, order, "nobody gets a held lock")
	mu.Unlock()
	h.release()
	wg.Wait()
	assert.Len(t, order, 2)

	// A lock taken over under us is compromised and kept.
	h, err = l.acquire(context.Background())
	require.NoError(t, err)
	other := time.Now().Add(time.Hour)
	require.NoError(t, os.Chtimes(l.path, other, other))
	time.Sleep(120 * time.Millisecond)
	assert.ErrorContains(t, h.check(), "taken over")
	h.release()
	_, err = os.Stat(l.path)
	assert.NoError(t, err, "a compromised lock is not ours to remove")
}

func jwt(claims map[string]any) string {
	b, _ := json.Marshal(claims)
	return "e30." + base64.RawURLEncoding.EncodeToString(b) + ".sig"
}

func TestCodexRefreshRotatesAndWritesBack(t *testing.T) {
	newAccess := jwt(map[string]any{codexClaim: map[string]any{"chatgpt_account_id": "acc-2"}})
	var form map[string][]string
	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		form = r.PostForm
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"access_token": newAccess, "refresh_token": "r2", "expires_in": 3600, "token_type": "Bearer"})
	}))
	defer tokenSrv.Close()

	var seen http.Header
	api := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		seen = r.Header.Clone()
		if r.URL.Host != "chatgpt.com" || r.URL.Path != "/backend-api/codex/responses" {
			t.Errorf("request to %s", r.URL)
		}
		body := "event: response.completed\ndata: {\"type\":\"response.completed\",\"response\":{\"id\":\"r\",\"status\":\"completed\",\"output\":[]}}\n\n"
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"text/event-stream"}}, Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
	})}

	dir := t.TempDir()
	writeFile(t, dir, "models-store.json", map[string]any{"openai-codex": map[string]any{"models": []any{
		map[string]any{"id": "gpt-x", "api": "openai-codex-responses", "baseUrl": "https://chatgpt.com/backend-api", "reasoning": true, "maxTokens": 128000},
	}}})
	oldAccess := jwt(map[string]any{codexClaim: map[string]any{"chatgpt_account_id": "acc-1"}})
	writeFile(t, dir, "auth.json", map[string]any{"openai-codex": map[string]any{"type": "oauth", "access": oldAccess, "refresh": "r1", "expires": time.Now().Add(-time.Minute).UnixMilli(), "accountId": "acc-1"}})

	a, err := LoadAgentDir(dir, WithCodexTokenURL(tokenSrv.URL), WithHTTPClient(api))
	require.NoError(t, err)
	r, err := a.Resolve(context.Background(), "openai-codex/gpt-x", ResolveOptions{Thinking: "medium"})
	require.NoError(t, err)
	assert.Equal(t, "medium", *r.Settings.Inference.ReasoningEffort)
	eng, err := r.Engine()
	require.NoError(t, err)
	_, _ = eng.RunInference(context.Background(), &turns.Turn{Blocks: []turns.Block{turns.NewUserTextBlock("ping")}})

	assert.Equal(t, []string{"refresh_token"}, form["grant_type"])
	assert.Equal(t, []string{"r1"}, form["refresh_token"])
	assert.Equal(t, []string{codexOAuth.ClientID}, form["client_id"])
	require.NotNil(t, seen, "the Codex request was sent")
	assert.Equal(t, "Bearer "+newAccess, seen.Get("Authorization"))
	assert.Equal(t, "acc-2", seen.Get("chatgpt-account-id"))

	e, _, err := a.Auth.Entry("openai-codex")
	require.NoError(t, err)
	assert.Equal(t, newAccess, e.Access)
	assert.Equal(t, "r2", e.Refresh, "the rotated refresh token is written back")
	assert.Equal(t, "acc-2", e.AccountID)
	assert.True(t, e.Expires.After(time.Now()))
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestAccountID(t *testing.T) {
	id, err := accountID(jwt(map[string]any{codexClaim: map[string]any{"chatgpt_account_id": "a"}}))
	require.NoError(t, err)
	assert.Equal(t, "a", id)
	for _, bad := range []string{"x", "a.b.c", jwt(map[string]any{"sub": "x"})} {
		_, err := accountID(bad)
		assert.Error(t, err)
	}
}

func TestThinkingConfig(t *testing.T) {
	s := func(v string) *string { return &v }
	yes, no := true, false
	m := func(api string, c Compat, lm map[string]*string) Model {
		return Model{API: api, Reasoning: true, Compat: c, ThinkingLevelMap: lm}
	}
	for name, tc := range map[string]struct {
		m      Model
		level  string
		effort *string
		typ    *string
		budget *int
	}{
		"plain reasoning effort":       {m(APIOpenAICompletions, Compat{}, nil), "high", s("high"), nil, nil},
		"mapped effort":                {m(APIOpenAICompletions, Compat{}, map[string]*string{"xhigh": s("max")}), "xhigh", s("max"), nil, nil},
		"null level not sent":          {m(APIOpenAICompletions, Compat{}, map[string]*string{"low": nil}), "low", nil, nil, nil},
		"off sends mapped off":         {m(APIOpenAICompletions, Compat{}, map[string]*string{"off": s("none")}), "off", s("none"), nil, nil},
		"effort unsupported":           {m(APIOpenAICompletions, Compat{SupportsReasoningEffort: &no}, nil), "high", nil, nil, nil},
		"zai on":                       {m(APIOpenAICompletions, Compat{ThinkingFormat: "zai", SupportsReasoningEffort: &no}, nil), "high", nil, s("enabled"), nil},
		"zai off":                      {m(APIOpenAICompletions, Compat{ThinkingFormat: "zai"}, nil), "off", nil, s("disabled"), nil},
		"deepseek on with effort":      {m(APIOpenAICompletions, Compat{ThinkingFormat: "deepseek", SupportsReasoningEffort: &yes}, nil), "medium", s("medium"), s("enabled"), nil},
		"deepseek off":                 {m(APIOpenAICompletions, Compat{ThinkingFormat: "deepseek"}, nil), "", nil, s("disabled"), nil},
		"deepseek off null keeps mode": {m(APIOpenAICompletions, Compat{ThinkingFormat: "deepseek"}, map[string]*string{"off": nil}), "off", nil, nil, nil},
		"responses effort":             {m(APIOpenAIResponses, Compat{}, nil), "low", s("low"), nil, nil},
		"responses off":                {m(APIOpenAICodexResponses, Compat{}, nil), "off", nil, nil, nil},
		"anthropic budget":             {m(APIAnthropicMessages, Compat{}, nil), "medium", nil, nil, ptr(8192)},
	} {
		c := thinkingConfig(tc.m, tc.level)
		assert.Equal(t, tc.effort, c.ReasoningEffort, name)
		assert.Equal(t, tc.typ, c.ThinkingType, name)
		assert.Equal(t, tc.budget, c.ThinkingBudget, name)
	}
	assert.Equal(t, engine.InferenceConfig{}, thinkingConfig(Model{API: APIOpenAICompletions}, "high"), "non-reasoning models get nothing")
}

func ptr(i int) *int { return &i }

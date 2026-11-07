// Command leak-probe shows, without any real network traffic, which hosts
// receive which credentials in today's Geppetto wiring. It replaces
// http.DefaultTransport with a recorder that answers every request itself,
// then drives each path the GEPPETTO-CREDSCOPE-001 guide lists. Each secret
// is a recognisable marker (FW-SECRET, …), so the output shows exactly where
// a marker ended up.
//
// Run from the geppetto repository root:
//
//	GOWORK=off go run ./ttmp/2026/11/05/GEPPETTO-CREDSCOPE-001--credentials-scoped-to-their-origin-no-key-reaches-a-host-it-was-not-issued-for/scripts/01-leak-probe
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	enginefactory "github.com/go-go-golems/geppetto/pkg/inference/engine/factory"

	"github.com/go-go-golems/geppetto/pkg/embeddings"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/providers/pi"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/types"
	"github.com/go-go-golems/geppetto/pkg/turns"
)

// hit is one request the recorder saw: host plus every marker found in its
// headers or URL query.
type hit struct {
	Host    string
	Path    string
	Markers []string
}

type recorder struct {
	mu   sync.Mutex
	hits []hit
	// redirect maps a host to the absolute URL it answers with 307.
	redirect map[string]string
}

var markers = []string{"FW-SECRET", "ZAI-SECRET", "GW-SECRET", "OR-SECRET", "CL-SECRET", "TEAM-SECRET"}

func (r *recorder) RoundTrip(req *http.Request) (*http.Response, error) {
	var found []string
	var all strings.Builder
	for k, vs := range req.Header {
		all.WriteString(k + ": " + strings.Join(vs, ",") + "\n")
	}
	all.WriteString(req.URL.RawQuery)
	for _, m := range markers {
		if strings.Contains(all.String(), m) {
			found = append(found, m)
		}
	}
	sort.Strings(found)
	r.mu.Lock()
	r.hits = append(r.hits, hit{Host: req.URL.Host, Path: req.URL.Path, Markers: found})
	loc, redirect := r.redirect[req.URL.Host]
	r.mu.Unlock()
	if req.Body != nil {
		_, _ = io.Copy(io.Discard, req.Body)
		_ = req.Body.Close()
	}
	if redirect {
		h := http.Header{}
		h.Set("Location", loc)
		return &http.Response{StatusCode: http.StatusTemporaryRedirect, Header: h, Body: io.NopCloser(strings.NewReader("")), Request: req}, nil
	}
	// 400 with a JSON body ends every engine's attempt without a retry.
	return &http.Response{StatusCode: http.StatusBadRequest, Header: http.Header{"Content-Type": {"application/json"}},
		Body: io.NopCloser(strings.NewReader(`{"error":{"message":"probe"}}`)), Request: req}, nil
}

func (r *recorder) take() []hit {
	r.mu.Lock()
	defer r.mu.Unlock()
	h := r.hits
	r.hits = nil
	return h
}

func userTurn() *turns.Turn {
	t := &turns.Turn{}
	turns.AppendBlock(t, turns.NewUserTextBlock("hello"))
	return t
}

func writeJSON(dir, name string, v any) {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), b, 0o600); err != nil {
		panic(err)
	}
}

// piDir writes a Pi agent directory with two providers that have a key but
// no baseUrl, and one gateway provider with a secret header.
func piDir() string {
	dir, err := os.MkdirTemp("", "credscope-probe-*")
	if err != nil {
		panic(err)
	}
	writeJSON(dir, "settings.json", map[string]any{"defaultProvider": "fireworks", "defaultModel": "m"})
	writeJSON(dir, "models.json", map[string]any{"providers": map[string]any{
		"fireworks": map[string]any{"api": "openai-completions", "apiKey": "FW-SECRET",
			"models": []any{map[string]any{"id": "m"}}},
		"zai": map[string]any{"api": "anthropic-messages", "apiKey": "ZAI-SECRET",
			"models": []any{map[string]any{"id": "m"}}},
		"gateway": map[string]any{"api": "openai-completions", "apiKey": "GW-SECRET", "baseUrl": "https://gateway.example/v1",
			"headers": map[string]string{"x-team": "TEAM-SECRET"},
			"models":  []any{map[string]any{"id": "m"}}},
	}})
	return dir
}

func report(name, expect string, hits []hit, err error) {
	fmt.Printf("== %s\n   expected: %s\n", name, expect)
	for _, h := range hits {
		fmt.Printf("   %-28s %-28s markers=%v\n", h.Host, h.Path, h.Markers)
	}
	if len(hits) == 0 {
		fmt.Printf("   (no request sent)\n")
	}
	if err != nil {
		msg := err.Error()
		if len(msg) > 120 {
			msg = msg[:120] + "…"
		}
		fmt.Printf("   error: %s\n", msg)
	}
}

func main() {
	rec := &recorder{redirect: map[string]string{}}
	http.DefaultTransport = rec
	ctx := context.Background()

	dir := piDir()
	defer func() { _ = os.RemoveAll(dir) }()
	agent, err := pi.LoadAgentDir(dir)
	if err != nil {
		panic(err)
	}

	runPi := func(ref string) error {
		r, err := agent.Resolve(ctx, ref, pi.ResolveOptions{})
		if err != nil {
			return err
		}
		e, err := r.Engine()
		if err != nil {
			return err
		}
		_, err = e.RunInference(ctx, userTurn())
		return err
	}

	// Row 2a: protocol-keyed default URL for an OpenAI-compatible provider.
	err = runPi("fireworks/m")
	report("pi.Resolve fireworks/m (openai-completions, no baseUrl)", "no request, or only to fireworks' own host", rec.take(), err)

	// Row 2b: the same for an Anthropic-compatible provider.
	err = runPi("zai/m")
	report("pi.Resolve zai/m (anthropic-messages, no baseUrl)", "no request, or only to zai's own host", rec.take(), err)

	// Row 6: models.json headers follow a redirect to another host.
	rec.redirect["gateway.example"] = "https://other.example/v1/chat/completions"
	err = runPi("gateway/m")
	report("pi.Resolve gateway/m, gateway answers 307 to other.example", "GW-SECRET and TEAM-SECRET only at gateway.example", rec.take(), err)
	delete(rec.redirect, "gateway.example")

	// Row 3: OpenAI embeddings ignore openai-base-url.
	s, err := settings.NewInferenceSettings()
	if err != nil {
		panic(err)
	}
	s.API.APIKeys["openai-api-key"] = "GW-SECRET"
	s.API.BaseUrls["openai-base-url"] = "https://gateway.example/v1"
	s.Embeddings.Type = "openai"
	s.Embeddings.Engine = "text-embedding-3-small"
	p, err := embeddings.NewSettingsFactoryFromInferenceSettings(s).NewProvider()
	if err == nil {
		_, err = p.GenerateEmbedding(ctx, "hello")
	}
	report("embeddings openai, openai-base-url=https://gateway.example/v1", "GW-SECRET only at gateway.example", rec.take(), err)

	// Row 4: Responses picks key and URL from different slots.
	s, err = settings.NewInferenceSettings()
	if err != nil {
		panic(err)
	}
	t := types.ApiTypeOpenResponses
	s.Chat.ApiType = &t
	model := "m"
	s.Chat.Engine = &model
	s.API.APIKeys["open-responses-api-key"] = "OR-SECRET"
	s.API.BaseUrls["openai-base-url"] = "https://gateway.example/v1"
	if e, err2 := enginefactory.NewEngineFromSettings(s); err2 == nil {
		_, err = e.RunInference(ctx, userTurn())
	} else {
		err = err2
	}
	report("open-responses key, only openai-base-url set", "OR-SECRET not sent to the openai slot's gateway", rec.take(), err)

	// Row 7: Claude x-api-key follows a redirect to another host.
	s, err = settings.NewInferenceSettings()
	if err != nil {
		panic(err)
	}
	ct := types.ApiTypeClaude
	s.Chat.ApiType = &ct
	s.Chat.Engine = &model
	s.API.APIKeys["claude-api-key"] = "CL-SECRET"
	s.API.BaseUrls["claude-base-url"] = "https://gateway.example"
	rec.redirect["gateway.example"] = "https://other.example/v1/messages"
	if e, err2 := enginefactory.NewEngineFromSettings(s); err2 == nil {
		_, err = e.RunInference(ctx, userTurn())
	} else {
		err = err2
	}
	report("claude, gateway answers 307 to other.example", "CL-SECRET only at gateway.example", rec.take(), err)
	delete(rec.redirect, "gateway.example")

	// Row 1 after the PR #444 fixes: the decision path refuses blank URLs.
	writeJSON(dir, "models.json", map[string]any{"providers": map[string]any{
		"fireworks": map[string]any{"api": "openai-completions", "apiKey": "FW-SECRET", "baseUrl": " ",
			"models": []any{map[string]any{"id": "m"}}},
	}})
	agent, err = pi.LoadAgentDir(dir)
	if err != nil {
		panic(err)
	}
	_, err = agent.Decider(ctx, "fireworks/m", pi.DecisionOptions{})
	report("pi.Decider fireworks/m, baseUrl \" \" (after #444 fixes)", "refused before any request", rec.take(), err)
}

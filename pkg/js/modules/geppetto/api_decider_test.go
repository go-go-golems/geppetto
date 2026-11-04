package geppetto

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/dop251/goja"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const deciderReply = `{"model":"djev","answers":{
 "route":{"type":"choice","choice":"semantic","confidence":0.9,"probabilities":{"z":0.01,"semantic":0.98,"keyword":0.01}},
 "needs":{"type":"noul","noul":0},
 "hard":{"type":"score","score":1.5,"confidence":0.7,"probabilities":{"0":0.1,"1":0.3,"2":0.6}}},
 "usage":{"input_tokens":12,"output_tokens":3}}`

// newDeciderTestServer serves System One and writes a profile binding djev to it.
func newDeciderTestServer(t *testing.T) (*sync.Map, string) {
	t.Helper()
	seen := &sync.Map{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		seen.Store("body", string(b))
		seen.Store("auth", r.Header.Get("Authorization"))
		seen.Store("path", r.URL.Path)
		w.Header().Set("X-Request-Id", "rid-1")
		_, _ = io.WriteString(w, deciderReply)
	}))
	t.Cleanup(srv.Close)
	profilePath := filepath.Join(t.TempDir(), "profiles.yaml")
	require.NoError(t, os.WriteFile(profilePath, []byte(fmt.Sprintf(`slug: decision-test
profiles:
  djev:
    inference_settings:
      api:
        api_keys:
          systemone-api-key: test-key
        base_urls:
          systemone-base-url: %q
        allow_http:
          decision: true
        allow_local_networks:
          decision: true
      decision:
        type: systemone
        engine: djev
`, srv.URL+"/v1")), 0o644))
	return seen, profilePath
}

const deciderRequestJS = `({
	state: { query: "how does rebalance work?" },
	questions: {
		route: { type: "choice", instructions: "Which index?", options: { z: "last letter", semantic: "concepts", keyword: "identifiers" } },
		needs: { type: "noul", instructions: "Needs documents?", true: "corpus facts" },
		hard:  { type: "score", instructions: "How hard?", levels: ["easy", "medium", "hard"] },
	},
})`

func TestDeciderDecide(t *testing.T) {
	seen, profilePath := newDeciderTestServer(t)
	rt := newJSRuntime(t, Options{})
	require.NoError(t, rt.vm.Set("profilePath", profilePath))
	got := mustEvalExprExport(t, rt, `(() => {
		const gp = require("geppetto");
		const d = gp.decider(gp.inferenceProfiles.load(globalThis.profilePath).resolve("djev"));
		const m = d.model();
		const r = d.decide(`+deciderRequestJS+`);
		return JSON.stringify({ model: m, choice: r.answers.route.choice, needs: [r.answers.needs.type, r.answers.needs.probability], levels: r.answers.hard.levelProbabilities,
			usage: r.usage, requestId: r.requestId, provider: r.provider, keys: Object.keys(d) });
	})()`)
	s := got.(string)
	for _, want := range []string{
		`"model":{"provider":"systemone","name":"djev"}`,
		`"choice":"semantic"`, `"needs":["noul",0]`, `"levels":[0.1,0.3,0.6]`,
		`"usage":{"inputTokens":12,"outputTokens":3}`, `"requestId":"rid-1"`, `"keys":["decide","decideAsync","model"]`,
	} {
		assert.Contains(t, s, want)
	}
	body, _ := seen.Load("body")
	assert.Contains(t, body, `"criteria":{"z":"last letter","semantic":"concepts","keyword":"identifiers"}`, "JS insertion order must reach the wire")
	auth, _ := seen.Load("auth")
	assert.Equal(t, "Bearer test-key", auth)
}

func TestDeciderRejectsBadRequests(t *testing.T) {
	_, profilePath := newDeciderTestServer(t)
	rt := newJSRuntime(t, Options{})
	require.NoError(t, rt.vm.Set("profilePath", profilePath))
	got := mustEvalExprExport(t, rt, `(() => {
		const gp = require("geppetto");
		const d = gp.decider(gp.inferenceProfiles.load(globalThis.profilePath).resolve("djev"));
		const out = [];
		for (const req of [
			{ state: "s", questions: { q: { type: "noul", instructions: "?" } }, topN: 3 },
			{ state: "s", questions: { q: { type: "noul", instruction: "?" } } },
			{ state: "s", questions: { q: { type: "choice", instructions: "?" } } },
			{ state: "s", questions: { q: { type: "choice", instructions: "?", options: [{ name: "a", weight: 1 }] } } },
			"not an object",
		]) {
			try { d.decide(req); out.push("accepted"); } catch (e) { out.push(String(e)); }
		}
		return JSON.stringify(out);
	})()`)
	s := got.(string)
	assert.NotContains(t, s, "accepted")
	assert.Contains(t, s, `unknown key \"topN\"`)
	assert.Contains(t, s, `unknown key \"instruction\"`)
	assert.Contains(t, s, "needs 1-255 options")
	assert.Contains(t, s, "only name and description")
}

func TestDeciderAsyncResolves(t *testing.T) {
	_, profilePath := newDeciderTestServer(t)
	rt := newJSRuntime(t, Options{})
	require.NoError(t, rt.vm.Set("profilePath", profilePath))
	_, err := rt.runtimeOwner.Call(context.Background(), "test.decideAsync", func(_ context.Context, vm *goja.Runtime) (any, error) {
		_, runErr := vm.RunString(`
			const gp = require("geppetto");
			const d = gp.decider(gp.inferenceProfiles.load(globalThis.profilePath).resolve("djev"));
			globalThis.done = false;
			const h = d.decideAsync(` + deciderRequestJS + `);
			if (typeof h.cancel !== "function" || typeof h.close !== "function") throw new Error("handle methods missing");
			h.promise.then(r => { globalThis.done = true; globalThis.choice = r.answers.route.choice; },
			               e => { globalThis.done = true; globalThis.err = String(e); });
		`)
		return nil, runErr
	})
	require.NoError(t, err)
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if done, _ := mustEvalExprExport(t, rt, `globalThis.done === true`).(bool); done {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	got := mustEvalExprExport(t, rt, `JSON.stringify({ done: globalThis.done, choice: globalThis.choice, err: globalThis.err })`).(string)
	assert.Equal(t, `{"done":true,"choice":"semantic"}`, strings.TrimSpace(got))
}

func TestDeciderRequiresDecisionProfile(t *testing.T) {
	_, profilePath := newRerankerTestServer(t, "bge")
	rt := newJSRuntime(t, Options{})
	require.NoError(t, rt.vm.Set("profilePath", profilePath))
	got := mustEvalExprExport(t, rt, `(() => {
		const gp = require("geppetto");
		try { gp.decider(gp.inferenceProfiles.load(globalThis.profilePath).resolve("bge-reranker")); return "built"; }
		catch (e) { return String(e); }
	})()`).(string)
	assert.Contains(t, got, "not decision-capable")
}

package geppetto

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dop251/goja"
	"github.com/go-go-golems/geppetto/pkg/security"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newRerankerTestServer returns an httptest server that responds with a valid
// rerank response, and writes a profile YAML that points a reranker at it.
func newRerankerTestServer(t *testing.T, model string) (*httptest.Server, string) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Model     string   `json:"model"`
			Query     string   `json:"query"`
			Documents []string `json:"documents"`
			TopN      int      `json:"top_n"`
		}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		results := make([]string, 0, req.TopN)
		for i := 0; i < req.TopN; i++ {
			results = append(results, fmt.Sprintf(`{"index":%d,"relevance_score":%d.0}`, i, req.TopN-i))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"model":%q,"object":"list","usage":{"prompt_tokens":10,"total_tokens":10},"results":[%s]}`,
			model, strings.Join(results, ","))
	}))
	t.Cleanup(srv.Close)

	profilePath := filepath.Join(t.TempDir(), "profiles.yaml")
	profile := fmt.Sprintf(`slug: rerank-test
profiles:
  bge-reranker:
    inference_settings:
      api:
        base_urls:
          rerank-base-url: %q
        allow_http:
          rerank: true
        allow_local_networks:
          rerank: true
      rerank:
        type: llamacpp
        engine: %q
        max_request_bytes: 1048576
        max_response_bytes: 1048576
`, srv.URL, model)
	require.NoError(t, os.WriteFile(profilePath, []byte(profile), 0o644))
	return srv, profilePath
}

func TestRerankerBuilderFromRegistryProfile_ExposesModel(t *testing.T) {
	const model = "qllama/bge-reranker-v2-m3:q4_k_m"
	_, profilePath := newRerankerTestServer(t, model)

	rt := newJSRuntime(t, Options{})
	require.NoError(t, rt.vm.Set("profilePath", profilePath))
	mustRunJS(t, rt, `
		const gp = require("geppetto");
		const settings = gp.inferenceProfiles.load(globalThis.profilePath).resolve("bge-reranker");
		const reranker = gp.reranker(settings);
		const model = reranker.model();
		if (model.provider !== "llama.cpp") throw new Error("wrong provider: " + model.provider);
		if (model.name !== "`+model+`") throw new Error("wrong model: " + model.name);
	`)
}

func TestRerankerBuilder_RejectsMissingSettings(t *testing.T) {
	rt := newJSRuntime(t, Options{})
	mustRunJS(t, rt, `
		const gp = require("geppetto");
		let ok = false;
		try { gp.reranker(); } catch (err) { ok = String(err).includes("requires"); }
		if (!ok) throw new Error("expected reranker() to reject missing settings");
	`)
}

func TestRerankerSync_RerankMapsIndexToDocumentID(t *testing.T) {
	const model = "bge-reranker-test"
	_, profilePath := newRerankerTestServer(t, model)

	rt := newJSRuntime(t, Options{})
	require.NoError(t, rt.vm.Set("profilePath", profilePath))
	mustRunJS(t, rt, `
		const gp = require("geppetto");
		const settings = gp.inferenceProfiles.load(globalThis.profilePath).resolve("bge-reranker");
		const reranker = gp.reranker(settings);
		const response = reranker.rerank(
			"How does TTC calculate a payroll adjustment?",
			[
				{id: "chunk-001", text: "A payroll adjustment corrects wages or deductions."},
				{id: "chunk-002", text: "Cypress trees tolerate dry conditions."}
			],
			{topN: 2}
		);
		if (response.provider !== "llama.cpp") throw new Error("wrong provider: " + response.provider);
		if (response.model !== "`+model+`") throw new Error("wrong model: " + response.model);
		if (response.results.length !== 2) throw new Error("expected 2 results, got " + response.results.length);
		if (response.results[0].rank !== 1) throw new Error("first result rank should be 1");
		if (response.results[0].documentId !== "chunk-001") throw new Error("first documentId mismatch: " + response.results[0].documentId);
		if (response.results[0].index !== 0) throw new Error("first index mismatch");
		if (response.results[1].rank !== 2) throw new Error("second result rank should be 2");
		if (response.results[1].documentId !== "chunk-002") throw new Error("second documentId mismatch");
		if (!response.usage) throw new Error("usage should be present");
		if (response.usage.inputTokens !== 10) throw new Error("wrong input tokens: " + response.usage.inputTokens);
		if (response.durationMs === undefined) throw new Error("durationMs should be present");
	`)
}

func TestRerankerSync_RejectsEmptyQuery(t *testing.T) {
	const model = "bge-reranker-test"
	_, profilePath := newRerankerTestServer(t, model)

	rt := newJSRuntime(t, Options{})
	require.NoError(t, rt.vm.Set("profilePath", profilePath))
	mustRunJS(t, rt, `
		const gp = require("geppetto");
		const settings = gp.inferenceProfiles.load(globalThis.profilePath).resolve("bge-reranker");
		const reranker = gp.reranker(settings);
		let ok = false;
		try {
			reranker.rerank("", [{id: "a", text: "x"}], {topN: 1});
		} catch (err) { ok = String(err).includes("query is required"); }
		if (!ok) throw new Error("expected empty query rejection");
	`)
}

func TestRerankerSync_RejectsMalformedDocuments(t *testing.T) {
	const model = "bge-reranker-test"
	_, profilePath := newRerankerTestServer(t, model)

	rt := newJSRuntime(t, Options{})
	require.NoError(t, rt.vm.Set("profilePath", profilePath))
	mustRunJS(t, rt, `
		const gp = require("geppetto");
		const settings = gp.inferenceProfiles.load(globalThis.profilePath).resolve("bge-reranker");
		const reranker = gp.reranker(settings);

		// Missing id.
		let ok = false;
		try { reranker.rerank("q", [{text: "x"}], {topN: 1}); } catch (err) { ok = String(err).includes("id"); }
		if (!ok) throw new Error("expected missing id rejection");

		// Missing text. The protected caller ID must not enter the error.
		ok = false;
		try {
			reranker.rerank("q", [{id: "CALLER-ID-MUST-NOT-LEAK"}], {topN: 1});
		} catch (err) {
			ok = String(err).includes("text") && !String(err).includes("CALLER-ID-MUST-NOT-LEAK");
		}
		if (!ok) throw new Error("expected safe missing text rejection");

		// Empty documents.
		ok = false;
		try { reranker.rerank("q", [], {topN: 1}); } catch (err) { ok = String(err).includes("at least one document"); }
		if (!ok) throw new Error("expected empty documents rejection");
	`)
}

func TestRerankerSync_RejectsInvalidOptions(t *testing.T) {
	const model = "bge-reranker-test"
	_, profilePath := newRerankerTestServer(t, model)

	rt := newJSRuntime(t, Options{})
	require.NoError(t, rt.vm.Set("profilePath", profilePath))
	mustRunJS(t, rt, `
		const gp = require("geppetto");
		const settings = gp.inferenceProfiles.load(globalThis.profilePath).resolve("bge-reranker");
		const reranker = gp.reranker(settings);
		const docs = [{id: "a", text: "x"}, {id: "b", text: "y"}];

		// Missing topN.
		let ok = false;
		try { reranker.rerank("q", docs, {}); } catch (err) { ok = String(err).includes("topN is required"); }
		if (!ok) throw new Error("expected missing topN rejection");

		// topN out of range.
		ok = false;
		try { reranker.rerank("q", docs, {topN: 5}); } catch (err) { ok = String(err).includes("between 1 and 2"); }
		if (!ok) throw new Error("expected topN range rejection");

		// Unknown option key.
		ok = false;
		try { reranker.rerank("q", docs, {topN: 1, unknown: true}); } catch (err) { ok = String(err).includes("unknown key"); }
		if (!ok) throw new Error("expected unknown key rejection");

		// Non-integral topN.
		ok = false;
		try { reranker.rerank("q", docs, {topN: 1.5}); } catch (err) { ok = String(err).includes("safe integer"); }
		if (!ok) throw new Error("expected non-integral topN rejection");

		// Values above JavaScript's exact integer range must not be silently rounded.
		ok = false;
		try { reranker.rerank("q", docs, {topN: 9007199254740992}); } catch (err) { ok = String(err).includes("safe integer"); }
		if (!ok) throw new Error("expected unsafe topN rejection");
	`)
}

func TestRerankerSync_DoesNotLeakQueryOrDocumentTextInErrors(t *testing.T) {
	// Server that returns a 500 with a body containing a secret.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"internal secret token: SUPERSECRET"}`))
	}))
	t.Cleanup(srv.Close)

	profilePath := filepath.Join(t.TempDir(), "profiles.yaml")
	profile := fmt.Sprintf(`slug: rerank-leak-test
profiles:
  bge-reranker:
    inference_settings:
      api:
        base_urls:
          rerank-base-url: %q
        allow_http:
          rerank: true
        allow_local_networks:
          rerank: true
      rerank:
        type: llamacpp
        engine: "bge-model"
`, srv.URL)
	require.NoError(t, os.WriteFile(profilePath, []byte(profile), 0o644))

	rt := newJSRuntime(t, Options{})
	require.NoError(t, rt.vm.Set("profilePath", profilePath))
	// Run the throwing JS and capture the error via a Promise-like wrapper.
	_, err := rt.vm.RunString(`
		const gp = require("geppetto");
		const settings = gp.inferenceProfiles.load(globalThis.profilePath).resolve("bge-reranker");
		const reranker = gp.reranker(settings);
		reranker.rerank("SECRET-QUERY", [{id: "a", text: "SECRET-DOC-TEXT"}], {topN: 1});
	`)
	require.Error(t, err)
	errStr := err.Error()
	assert.NotContains(t, errStr, "SUPERSECRET")
	assert.NotContains(t, errStr, "SECRET-QUERY")
	assert.NotContains(t, errStr, "SECRET-DOC-TEXT")
}

func TestRerankerSync_NegativeScoresPreserved(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"model":"bge","results":[{"index":0,"relevance_score":-3.32},{"index":1,"relevance_score":-9.83}]}`))
	}))
	t.Cleanup(srv.Close)

	profilePath := filepath.Join(t.TempDir(), "profiles.yaml")
	profile := fmt.Sprintf(`slug: rerank-neg-test
profiles:
  bge-reranker:
    inference_settings:
      api:
        base_urls:
          rerank-base-url: %q
        allow_http:
          rerank: true
        allow_local_networks:
          rerank: true
      rerank:
        type: llamacpp
        engine: "bge"
`, srv.URL)
	require.NoError(t, os.WriteFile(profilePath, []byte(profile), 0o644))

	rt := newJSRuntime(t, Options{})
	require.NoError(t, rt.vm.Set("profilePath", profilePath))
	mustRunJS(t, rt, `
		const gp = require("geppetto");
		const settings = gp.inferenceProfiles.load(globalThis.profilePath).resolve("bge-reranker");
		const reranker = gp.reranker(settings);
		const response = reranker.rerank("q", [{id: "a", text: "x"}, {id: "b", text: "y"}], {topN: 2});
		if (response.results[0].score !== -3.32) throw new Error("expected -3.32, got " + response.results[0].score);
		if (response.results[1].score !== -9.83) throw new Error("expected -9.83, got " + response.results[1].score);
	`)
}

func TestReranker_HiddenRefIsNotEnumerable(t *testing.T) {
	const model = "bge-reranker-test"
	_, profilePath := newRerankerTestServer(t, model)

	rt := newJSRuntime(t, Options{})
	require.NoError(t, rt.vm.Set("profilePath", profilePath))
	mustRunJS(t, rt, `
		const gp = require("geppetto");
		const settings = gp.inferenceProfiles.load(globalThis.profilePath).resolve("bge-reranker");
		const reranker = gp.reranker(settings);
		const keys = Object.keys(reranker);
		if (keys.includes("__geppetto_ref")) throw new Error("hidden ref should not be enumerable: " + keys.join(","));
		if (!keys.includes("rerank") || !keys.includes("rerankAsync") || !keys.includes("model")) {
			throw new Error("missing expected methods: " + keys.join(","));
		}
	`)
}

func TestRerankerAsync_ResolvesPromise(t *testing.T) {
	const model = "bge-reranker-async"
	_, profilePath := newRerankerTestServer(t, model)

	rt := newJSRuntime(t, Options{})
	require.NoError(t, rt.vm.Set("profilePath", profilePath))

	_, err := rt.runtimeOwner.Call(context.Background(), "test.rerankerAsync", func(_ context.Context, vm *goja.Runtime) (any, error) {
		_, runErr := vm.RunString(`
			const gp = require("geppetto");
			globalThis.rerankAsyncDone = false;
			const settings = gp.inferenceProfiles.load(globalThis.profilePath).resolve("bge-reranker");
			const reranker = gp.reranker(settings);
			const handle = reranker.rerankAsync(
				"query",
				[{id: "a", text: "x"}, {id: "b", text: "y"}],
				{topN: 2}
			);
			if (typeof handle.cancel !== "function") throw new Error("handle.cancel missing");
			if (typeof handle.close !== "function") throw new Error("handle.close missing");
			handle.promise.then(
				resp => { globalThis.rerankAsyncDone = true; globalThis.rerankAsyncResp = resp; },
				err => { globalThis.rerankAsyncDone = true; globalThis.rerankAsyncError = String(err); }
			);
		`)
		return nil, runErr
	})
	require.NoError(t, err)

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		v := mustEvalExprExport(t, rt, `globalThis.rerankAsyncDone === true`)
		if done, ok := v.(bool); ok && done {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	got := mustEvalExprExport(t, rt, `JSON.stringify({done: globalThis.rerankAsyncDone, error: globalThis.rerankAsyncError, firstDoc: globalThis.rerankAsyncResp && globalThis.rerankAsyncResp.results && globalThis.rerankAsyncResp.results[0] && globalThis.rerankAsyncResp.results[0].documentId})`)
	result, ok := got.(string)
	require.True(t, ok, "expected string result, got %T", got)
	assert.Contains(t, result, `"done":true`)
	assert.NotContains(t, result, `"error":"`)
	assert.Contains(t, result, `"firstDoc":"a"`)
}

func TestRerankerAsync_CancelIsIdempotent(t *testing.T) {
	const model = "bge-reranker-cancel"
	_, profilePath := newRerankerTestServer(t, model)

	rt := newJSRuntime(t, Options{})
	require.NoError(t, rt.vm.Set("profilePath", profilePath))
	// Repeated cancel/close must not panic.
	mustRunJS(t, rt, `
		const gp = require("geppetto");
		const settings = gp.inferenceProfiles.load(globalThis.profilePath).resolve("bge-reranker");
		const reranker = gp.reranker(settings);
		const handle = reranker.rerankAsync("q", [{id: "a", text: "x"}], {topN: 1});
		handle.cancel();
		handle.cancel();
		handle.close();
		handle.close();
	`)
}

// Compile-time assertion that security.OutboundURLOptions is imported and used
// (guards against accidental removal of the local-network opt-in documentation).
var _ security.OutboundURLOptions = security.OutboundURLOptions{}

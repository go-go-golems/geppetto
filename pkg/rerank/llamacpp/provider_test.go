package llamacpp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-go-golems/geppetto/pkg/rerank"
	"github.com/go-go-golems/geppetto/pkg/security"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testModel = "qllama/bge-reranker-v2-m3:q4_k_m"

func baseRequest(query string, docs []rerank.Document, topN int) rerank.Request {
	return rerank.Request{Query: query, Documents: docs, TopN: topN}
}

func mustNewProvider(t *testing.T, opts Options) *Provider {
	t.Helper()
	p, err := New(opts)
	require.NoError(t, err)
	return p
}

// newTestServer returns an httptest server and a provider pointed at it with
// local HTTP/local networks explicitly allowed.
func newTestServer(t *testing.T, handler http.HandlerFunc) (*Provider, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	p := mustNewProvider(t, Options{
		BaseURL:     srv.URL,
		Model:       testModel,
		OutboundURL: security.OutboundURLOptions{AllowHTTP: true, AllowLocalNetworks: true},
	})
	return p, srv
}

func TestNew_RequiresBaseURL(t *testing.T) {
	_, err := New(Options{Model: testModel})
	require.ErrorIs(t, err, rerank.ErrInvalidRequest)
	assert.Contains(t, err.Error(), "base URL is required")
}

func TestNew_RequiresModel(t *testing.T) {
	_, err := New(Options{BaseURL: "http://127.0.0.1:18012"})
	require.ErrorIs(t, err, rerank.ErrInvalidRequest)
	assert.Contains(t, err.Error(), "model is required")
}

func TestNew_RejectsBadScheme(t *testing.T) {
	_, err := New(Options{BaseURL: "ftp://example.com", Model: testModel})
	require.ErrorIs(t, err, rerank.ErrInvalidRequest)
	assert.Contains(t, err.Error(), "scheme")
}

func TestNew_RejectsUserinfo(t *testing.T) {
	_, err := New(Options{BaseURL: "http://user:pass@127.0.0.1:18012", Model: testModel})
	require.ErrorIs(t, err, rerank.ErrInvalidRequest)
	assert.Contains(t, err.Error(), "userinfo")
}

func TestNew_RejectsQueryAndFragment(t *testing.T) {
	_, err := New(Options{BaseURL: "http://127.0.0.1:18012?x=1", Model: testModel})
	require.ErrorIs(t, err, rerank.ErrInvalidRequest)
	assert.Contains(t, err.Error(), "query or fragment")
}

func TestNew_DeniesLocalHTTPByDefault(t *testing.T) {
	_, err := New(Options{BaseURL: "http://127.0.0.1:18012", Model: testModel})
	require.ErrorIs(t, err, rerank.ErrInvalidRequest)
	assert.Contains(t, err.Error(), "outbound URL policy")
}

func TestNew_RejectsNonPositiveLimits(t *testing.T) {
	_, err := New(Options{
		BaseURL: "http://127.0.0.1:18012", Model: testModel,
		OutboundURL:     security.OutboundURLOptions{AllowHTTP: true, AllowLocalNetworks: true},
		MaxRequestBytes: -1,
	})
	require.ErrorIs(t, err, rerank.ErrInvalidRequest)
	assert.Contains(t, err.Error(), "max_request_bytes")
}

func TestModel_ReturnsConfiguredIdentity(t *testing.T) {
	p := mustNewProvider(t, Options{
		BaseURL:     "http://127.0.0.1:18012",
		Model:       testModel,
		OutboundURL: security.OutboundURLOptions{AllowHTTP: true, AllowLocalNetworks: true},
	})
	m := p.Model()
	assert.Equal(t, "llama.cpp", m.Provider)
	assert.Equal(t, testModel, m.Name)
}

func TestRerank_ExactRequestShapeAndMapping(t *testing.T) {
	var gotRequest request
	p, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		assert.True(t, strings.HasSuffix(r.URL.Path, "/v1/rerank"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		require.NoError(t, json.NewDecoder(r.Body).Decode(&gotRequest))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"model": "` + testModel + `",
			"object": "list",
			"usage": {"prompt_tokens": 96, "total_tokens": 96},
			"results": [
				{"index": 0, "relevance_score": -3.32784366607666},
				{"index": 1, "relevance_score": -9.837879180908203}
			]
		}`))
	})

	docs := []rerank.Document{
		{ID: "chunk-001", Text: "A payroll adjustment corrects wages or deductions."},
		{ID: "chunk-002", Text: "Cypress trees tolerate dry conditions."},
	}
	resp, err := p.Rerank(context.Background(), baseRequest("How does TTC calculate a payroll adjustment?", docs, 2))
	require.NoError(t, err)

	// Caller IDs never enter the payload; only document text in array order.
	assert.Equal(t, testModel, gotRequest.Model)
	assert.Equal(t, "How does TTC calculate a payroll adjustment?", gotRequest.Query)
	assert.Equal(t, []string{docs[0].Text, docs[1].Text}, gotRequest.Documents)
	assert.Equal(t, 2, gotRequest.TopN)

	assert.Equal(t, "llama.cpp", resp.Provider)
	assert.Equal(t, testModel, resp.Model)
	require.Len(t, resp.Results, 2)
	// Descending score: -3.32 (idx 0) then -9.83 (idx 1).
	assert.Equal(t, "chunk-001", resp.Results[0].DocumentID)
	assert.Equal(t, -3.32784366607666, resp.Results[0].Score)
	assert.Equal(t, 1, resp.Results[0].Rank)
	assert.Equal(t, "chunk-002", resp.Results[1].DocumentID)
	assert.Equal(t, 2, resp.Results[1].Rank)

	require.NotNil(t, resp.Usage)
	assert.Equal(t, 96, resp.Usage.InputTokens)
	assert.Equal(t, 96, resp.Usage.TotalTokens)
	require.NotNil(t, resp.DurationMs)
}

func TestRerank_LoadsSanitizedFixture(t *testing.T) {
	p, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(fixtureBytes(t))
	})
	docs := []rerank.Document{
		{ID: "d0", Text: "a"},
		{ID: "d1", Text: "b"},
		{ID: "d2", Text: "c"},
	}
	resp, err := p.Rerank(context.Background(), baseRequest("q", docs, 3))
	require.NoError(t, err)
	require.Len(t, resp.Results, 3)
	assert.Equal(t, "d0", resp.Results[0].DocumentID)
	assert.Equal(t, "d2", resp.Results[2].DocumentID)
	require.NotNil(t, resp.Usage)
	assert.Equal(t, 96, resp.Usage.InputTokens)
}

func TestRerank_RejectsWrongCardinality(t *testing.T) {
	p, _ := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"results":[{"index":0,"relevance_score":1.0}]}`))
	})
	docs := []rerank.Document{{ID: "a", Text: "x"}, {ID: "b", Text: "y"}}
	_, err := p.Rerank(context.Background(), baseRequest("q", docs, 2))
	require.ErrorIs(t, err, rerank.ErrInvalidResponse)
}

func TestRerank_RejectsMissingIndexOrScore(t *testing.T) {
	p, _ := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"results":[{"relevance_score":1.0}]}`))
	})
	docs := []rerank.Document{{ID: "a", Text: "x"}}
	_, err := p.Rerank(context.Background(), baseRequest("q", docs, 1))
	require.ErrorIs(t, err, rerank.ErrInvalidResponse)

	p2, _ := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"results":[{"index":0}]}`))
	})
	_, err = p2.Rerank(context.Background(), baseRequest("q", docs, 1))
	require.ErrorIs(t, err, rerank.ErrInvalidResponse)
}

func TestRerank_RejectsOutOfRangeAndDuplicateIndex(t *testing.T) {
	docs := []rerank.Document{{ID: "a", Text: "x"}}

	pOutOfRange, _ := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"results":[{"index":5,"relevance_score":1.0}]}`))
	})
	_, err := pOutOfRange.Rerank(context.Background(), baseRequest("q", docs, 1))
	require.ErrorIs(t, err, rerank.ErrInvalidResponse)

	pDup, _ := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"results":[{"index":0,"relevance_score":1.0},{"index":0,"relevance_score":0.5}]}`))
	})
	docs2 := []rerank.Document{{ID: "a", Text: "x"}, {ID: "b", Text: "y"}}
	_, err = pDup.Rerank(context.Background(), baseRequest("q", docs2, 2))
	require.ErrorIs(t, err, rerank.ErrInvalidResponse)
}

func TestRerank_RejectsTrailingJSON(t *testing.T) {
	p, _ := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"results":[{"index":0,"relevance_score":1.0}]}{"trailing":true}`))
	})
	docs := []rerank.Document{{ID: "a", Text: "x"}}
	_, err := p.Rerank(context.Background(), baseRequest("q", docs, 1))
	require.ErrorIs(t, err, rerank.ErrInvalidResponse)
	assert.Contains(t, err.Error(), "trailing")
}

func TestRerank_RejectsUnknownFields(t *testing.T) {
	p, _ := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"results":[{"index":0,"relevance_score":1.0,"extra":true}]}`))
	})
	docs := []rerank.Document{{ID: "a", Text: "x"}}
	_, err := p.Rerank(context.Background(), baseRequest("q", docs, 1))
	require.ErrorIs(t, err, rerank.ErrInvalidResponse)
}

func TestRerank_RejectsResponseModelMismatch(t *testing.T) {
	p, _ := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"model":"different/model","results":[{"index":0,"relevance_score":1.0}]}`))
	})
	docs := []rerank.Document{{ID: "a", Text: "x"}}
	_, err := p.Rerank(context.Background(), baseRequest("q", docs, 1))
	require.ErrorIs(t, err, rerank.ErrInvalidResponse)
	assert.Contains(t, err.Error(), "does not match effective model")
}

func TestRerank_Non2xxDoesNotLeakBody(t *testing.T) {
	p, _ := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"internal secret token: SUPERSECRET"}`))
	})
	docs := []rerank.Document{{ID: "a", Text: "secret-doc-text"}}
	_, err := p.Rerank(context.Background(), baseRequest("secret-query", docs, 1))
	require.ErrorIs(t, err, rerank.ErrUnavailable)
	assert.NotContains(t, err.Error(), "SUPERSECRET")
	assert.NotContains(t, err.Error(), "secret-doc-text")
	assert.NotContains(t, err.Error(), "secret-query")
	assert.Contains(t, err.Error(), "status 500")
}

func TestRerank_RejectsOversizedResponse(t *testing.T) {
	p, _ := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"results":[{"index":0,"relevance_score":1.0}]`))
		// Pad well past the small limit.
		_, _ = w.Write(padBytes(2000))
	})
	p2 := mustNewProvider(t, Options{
		BaseURL:          p.baseURL,
		Model:            testModel,
		OutboundURL:      security.OutboundURLOptions{AllowHTTP: true, AllowLocalNetworks: true},
		MaxResponseBytes: 100,
	})
	docs := []rerank.Document{{ID: "a", Text: "x"}}
	_, err := p2.Rerank(context.Background(), baseRequest("q", docs, 1))
	require.ErrorIs(t, err, rerank.ErrResponseTooLarge)
}

func TestRerank_RejectsOversizedRequest(t *testing.T) {
	p, _ := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"results":[{"index":0,"relevance_score":1.0}]}`))
	})
	p2 := mustNewProvider(t, Options{
		BaseURL:         p.baseURL,
		Model:           testModel,
		OutboundURL:     security.OutboundURLOptions{AllowHTTP: true, AllowLocalNetworks: true},
		MaxRequestBytes: 16,
	})
	big := strings.Repeat("x", 200)
	docs := []rerank.Document{{ID: "a", Text: big}}
	_, err := p2.Rerank(context.Background(), baseRequest("q", docs, 1))
	require.ErrorIs(t, err, rerank.ErrRequestTooLarge)
}

func TestRerank_ContextCancellation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(200 * time.Millisecond)
		_, _ = w.Write([]byte(`{"results":[{"index":0,"relevance_score":1.0}]}`))
	}))
	t.Cleanup(srv.Close)
	p := mustNewProvider(t, Options{
		BaseURL:     srv.URL,
		Model:       testModel,
		OutboundURL: security.OutboundURLOptions{AllowHTTP: true, AllowLocalNetworks: true},
	})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := p.Rerank(ctx, baseRequest("q", []rerank.Document{{ID: "a", Text: "x"}}, 1))
	require.Error(t, err)
	// Cancellation surfaces as a transport/unavailable error, never as a
	// successful response.
}

func TestRerank_RejectsRedirect(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"results":[{"index":0,"relevance_score":1.0}]}`))
	}))
	t.Cleanup(target.Close)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusFound)
	}))
	t.Cleanup(srv.Close)
	p := mustNewProvider(t, Options{
		BaseURL:     srv.URL,
		Model:       testModel,
		OutboundURL: security.OutboundURLOptions{AllowHTTP: true, AllowLocalNetworks: true},
	})
	docs := []rerank.Document{{ID: "a", Text: "x"}}
	_, err := p.Rerank(context.Background(), baseRequest("q", docs, 1))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "redirect")
}

func TestRerank_CostIsNilWithoutPricing(t *testing.T) {
	p, _ := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"usage":{"prompt_tokens":10,"total_tokens":10},"results":[{"index":0,"relevance_score":1.0}]}`))
	})
	docs := []rerank.Document{{ID: "a", Text: "x"}}
	resp, err := p.Rerank(context.Background(), baseRequest("q", docs, 1))
	require.NoError(t, err)
	assert.Nil(t, resp.Cost)
}

func TestRerank_CostIsComputedWithPricing(t *testing.T) {
	rate := 0.0 // explicitly free/local
	p, _ := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"usage":{"prompt_tokens":96,"total_tokens":96},"results":[{"index":0,"relevance_score":1.0}]}`))
	})
	p2 := mustNewProvider(t, Options{
		BaseURL:        p.baseURL,
		Model:          testModel,
		OutboundURL:    security.OutboundURLOptions{AllowHTTP: true, AllowLocalNetworks: true},
		CostPerMTokens: &rate,
	})
	docs := []rerank.Document{{ID: "a", Text: "x"}}
	resp, err := p2.Rerank(context.Background(), baseRequest("q", docs, 1))
	require.NoError(t, err)
	require.NotNil(t, resp.Cost)
	assert.Equal(t, 0.0, *resp.Cost)
}

func TestRerank_InjectedClientIsNotMutated(t *testing.T) {
	injected := &http.Client{}
	// The caller's client starts with a nil CheckRedirect. The provider must
	// not mutate the injected client in place; it clones before overriding
	// CheckRedirect, so the injected client remains nil here.
	require.Nil(t, injected.CheckRedirect, "precondition: injected client should start with nil CheckRedirect")
	_ = mustNewProvider(t, Options{
		BaseURL:     "http://127.0.0.1:18012",
		Model:       testModel,
		HTTPClient:  injected,
		OutboundURL: security.OutboundURLOptions{AllowHTTP: true, AllowLocalNetworks: true},
	})
	// The caller's client must still have a nil CheckRedirect (not mutated).
	assert.Nil(t, injected.CheckRedirect, "injected client CheckRedirect was mutated in place")
}

func TestReadAtMost_Limits(t *testing.T) {
	out, err := readAtMost(strings.NewReader("hello"), 3)
	require.NoError(t, err)
	assert.Len(t, out, 4) // limit+1
}

// padBytes returns a byte slice of n spaces (helper for oversized-response tests).
func padBytes(n int) []byte {
	return []byte(strings.Repeat(" ", n))
}

// fixtureJSON is the sanitized BGE reranker response (see testdata/
// bge-reranker-v2-m3-response.json), inlined here so the test does not depend
// on file IO ordering.
const fixtureJSON = `{
  "model": "qllama/bge-reranker-v2-m3:q4_k_m",
  "object": "list",
  "usage": {"prompt_tokens": 96, "total_tokens": 96},
  "results": [
    {"index": 0, "relevance_score": -3.32784366607666},
    {"index": 1, "relevance_score": -9.837879180908203},
    {"index": 2, "relevance_score": -11.012685775756836}
  ]
}`

func fixtureBytes(t *testing.T) []byte {
	t.Helper()
	return []byte(fixtureJSON)
}

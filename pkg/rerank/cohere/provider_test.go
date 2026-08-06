package cohere

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-go-golems/geppetto/pkg/rerank"
	"github.com/go-go-golems/geppetto/pkg/security"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testModel  = "rerank-v3.5"
	testAPIKey = "test-cohere-key"
)

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
// local HTTP/local networks explicitly allowed (the hosted default policy
// correctly denies plain-HTTP loopback, so tests opt in explicitly).
func newTestServer(t *testing.T, handler http.HandlerFunc) (*Provider, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	p := mustNewProvider(t, Options{
		APIKey:      testAPIKey,
		BaseURL:     srv.URL,
		Model:       testModel,
		OutboundURL: security.OutboundURLOptions{AllowHTTP: true, AllowLocalNetworks: true},
	})
	return p, srv
}

// writeWire encodes a wire response exactly as Cohere would.
func writeWire(w http.ResponseWriter, body string) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(body))
}

// --- Constructor tests (matrix row 19) ---

func TestNew_RequiresAPIKey(t *testing.T) {
	_, err := New(Options{Model: testModel})
	require.ErrorIs(t, err, rerank.ErrInvalidRequest)
	assert.Contains(t, err.Error(), "api key is required")
}

func TestNew_RequiresModel(t *testing.T) {
	_, err := New(Options{APIKey: testAPIKey})
	require.ErrorIs(t, err, rerank.ErrInvalidRequest)
	assert.Contains(t, err.Error(), "model is required")
}

func TestNew_DefaultsToHostedBaseURL(t *testing.T) {
	p := mustNewProvider(t, Options{APIKey: testAPIKey, Model: testModel})
	assert.Equal(t, DefaultBaseURL, p.baseURL)
	assert.Equal(t, "https://api.cohere.com/v2/rerank", p.endpoint)
}

func TestNew_RejectsBadScheme(t *testing.T) {
	_, err := New(Options{APIKey: testAPIKey, BaseURL: "ftp://example.com", Model: testModel})
	require.ErrorIs(t, err, rerank.ErrInvalidRequest)
	assert.Contains(t, err.Error(), "scheme")
}

func TestNew_RejectsUserinfo(t *testing.T) {
	_, err := New(Options{APIKey: testAPIKey, BaseURL: "https://user:pass@api.cohere.com", Model: testModel})
	require.ErrorIs(t, err, rerank.ErrInvalidRequest)
	assert.Contains(t, err.Error(), "userinfo")
}

func TestNew_RejectsQueryAndFragment(t *testing.T) {
	_, err := New(Options{APIKey: testAPIKey, BaseURL: "https://api.cohere.com?x=1", Model: testModel})
	require.ErrorIs(t, err, rerank.ErrInvalidRequest)
	assert.Contains(t, err.Error(), "query or fragment")
}

func TestNew_RejectsAmbiguousPath(t *testing.T) {
	for _, rawURL := range []string{
		"https://api.cohere.com/a/../b",
		"https://api.cohere.com//prefix",
		"https://api.cohere.com/%2e%2e",
	} {
		t.Run(rawURL, func(t *testing.T) {
			_, err := New(Options{APIKey: testAPIKey, BaseURL: rawURL, Model: testModel})
			require.ErrorIs(t, err, rerank.ErrInvalidRequest)
		})
	}
}

func TestNew_MalformedURLDoesNotLeakUserinfo(t *testing.T) {
	_, err := New(Options{APIKey: testAPIKey, BaseURL: "http://user:SUPERSECRET@[", Model: testModel})
	require.ErrorIs(t, err, rerank.ErrInvalidRequest)
	assert.NotContains(t, err.Error(), "SUPERSECRET")
}

func TestNew_DeniesLocalHTTPByDefault(t *testing.T) {
	_, err := New(Options{APIKey: testAPIKey, BaseURL: "http://127.0.0.1:18012", Model: testModel})
	require.ErrorIs(t, err, rerank.ErrInvalidRequest)
	assert.Contains(t, err.Error(), "outbound URL policy")
}

func TestNew_RejectsNonPositiveLimits(t *testing.T) {
	_, err := New(Options{
		APIKey: testAPIKey, Model: testModel,
		MaxRequestBytes: -1,
	})
	require.ErrorIs(t, err, rerank.ErrInvalidRequest)
	assert.Contains(t, err.Error(), "max_request_bytes")

	_, err = New(Options{
		APIKey: testAPIKey, Model: testModel,
		MaxResponseBytes: -1,
	})
	require.ErrorIs(t, err, rerank.ErrInvalidRequest)
	assert.Contains(t, err.Error(), "max_response_bytes")
}

func TestModel_ReturnsConfiguredIdentity(t *testing.T) {
	p := mustNewProvider(t, Options{APIKey: testAPIKey, Model: testModel})
	m := p.Model()
	assert.Equal(t, ProviderName, m.Provider)
	assert.Equal(t, testModel, m.Name)
}

// --- Happy path (matrix rows 1, 2, 18) ---

func TestRerank_HappyPath(t *testing.T) {
	docs := []rerank.Document{
		{ID: "doc-nevada", Text: "Carson City is the capital of Nevada."},
		{ID: "doc-dc", Text: "Washington, D.C. is the capital of the United States."},
		{ID: "doc-mariana", Text: "The Northern Mariana Islands' capital is Saipan."},
		{ID: "doc-grammar", Text: "Capitalization is the use of a capital letter."},
	}

	var sawAuth, sawClientName, sawBody bool
	p, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		sawAuth = r.Header.Get("Authorization") == "Bearer "+testAPIKey
		sawClientName = r.Header.Get("X-Client-Name") == clientName
		assert.Equal(t, "/v2/rerank", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		var wireReq request
		require.NoError(t, json.NewDecoder(r.Body).Decode(&wireReq))
		assert.Equal(t, testModel, wireReq.Model)
		assert.Equal(t, "capital of the united states", wireReq.Query)
		assert.Equal(t, 2, wireReq.TopN)
		// Caller document IDs must never enter the provider payload; the wire
		// DTO carries only text, verified structurally by decoding into the
		// unexported request type.
		require.Len(t, wireReq.Documents, 4)
		assert.Equal(t, docs[1].Text, wireReq.Documents[1])
		sawBody = true

		writeWire(w, `{
			"id": "req-9f7c",
			"results": [
				{"index": 1, "relevance_score": 0.98},
				{"index": 0, "relevance_score": 0.42}
			],
			"meta": {
				"api_version": {"version": "2.0", "is_experimental": false},
				"billed_units": {"search_units": 1}
			}
		}`)
	})

	resp, err := p.Rerank(context.Background(), baseRequest("capital of the united states", docs, 2))
	require.NoError(t, err)
	assert.True(t, sawAuth, "Authorization header not sent")
	assert.True(t, sawClientName, "X-Client-Name header not sent")
	assert.True(t, sawBody)

	assert.Equal(t, ProviderName, resp.Provider)
	assert.Equal(t, testModel, resp.Model)
	require.Len(t, resp.Results, 2)
	// Sorted by score descending, ranks assigned from 1, IDs mapped back.
	assert.Equal(t, "doc-dc", resp.Results[0].DocumentID)
	assert.Equal(t, 1, resp.Results[0].Index)
	assert.InDelta(t, 0.98, resp.Results[0].Score, 1e-9)
	assert.Equal(t, 1, resp.Results[0].Rank)
	assert.Equal(t, "doc-nevada", resp.Results[1].DocumentID)
	assert.Equal(t, 2, resp.Results[1].Rank)

	assert.Equal(t, "req-9f7c", resp.RequestID)
	require.NotNil(t, resp.DurationMs)
	assert.Nil(t, resp.Usage, "cohere reports search units, not tokens; Usage must stay nil (DR-3)")
	assert.Nil(t, resp.Cost, "no per-search rate configured; cost must be unknown, not zero")
}

func TestRerank_FullCardinality(t *testing.T) {
	docs := []rerank.Document{
		{ID: "a", Text: "alpha"},
		{ID: "b", Text: "beta"},
		{ID: "c", Text: "gamma"},
	}
	p, _ := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		writeWire(w, `{"id":"r1","results":[
			{"index":2,"relevance_score":0.3},
			{"index":0,"relevance_score":0.9},
			{"index":1,"relevance_score":0.6}
		]}`)
	})
	resp, err := p.Rerank(context.Background(), baseRequest("q", docs, 3))
	require.NoError(t, err)
	require.Len(t, resp.Results, 3)
	assert.Equal(t, []string{"a", "b", "c"},
		[]string{resp.Results[0].DocumentID, resp.Results[1].DocumentID, resp.Results[2].DocumentID})
}

func TestRerank_DeterministicTieBreak(t *testing.T) {
	docs := []rerank.Document{
		{ID: "b", Text: "beta"},
		{ID: "a", Text: "alpha"},
	}
	p, _ := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		writeWire(w, `{"results":[
			{"index":0,"relevance_score":0.5},
			{"index":1,"relevance_score":0.5}
		]}`)
	})
	resp, err := p.Rerank(context.Background(), baseRequest("q", docs, 2))
	require.NoError(t, err)
	// Equal scores: input index ascending wins.
	assert.Equal(t, "b", resp.Results[0].DocumentID)
	assert.Equal(t, "a", resp.Results[1].DocumentID)
}

func TestRerank_CostFromSearchUnits(t *testing.T) {
	rate := 0.002 // $2.00 per 1000 searches
	docs := []rerank.Document{{ID: "a", Text: "alpha"}}
	p := mustNewProvider(t, Options{
		APIKey:        testAPIKey,
		Model:         testModel,
		CostPerSearch: &rate,
		// Point at a local test server through the BaseURL override.
		BaseURL:     testServerURL(t, `{"id":"r","results":[{"index":0,"relevance_score":1.0}],"meta":{"billed_units":{"search_units":2}}}`),
		OutboundURL: security.OutboundURLOptions{AllowHTTP: true, AllowLocalNetworks: true},
	})
	resp, err := p.Rerank(context.Background(), baseRequest("q", docs, 1))
	require.NoError(t, err)
	require.NotNil(t, resp.Cost)
	assert.InDelta(t, 0.004, *resp.Cost, 1e-12)
}

// testServerURL starts a fixed-response server and returns its URL.
func testServerURL(t *testing.T, body string) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeWire(w, body)
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

// --- Request validation (matrix rows 4, 5, 6) ---

func TestRerank_RejectsInvalidRequests(t *testing.T) {
	p, _ := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		t.Error("server must not be reached for invalid requests")
	})
	docs := []rerank.Document{{ID: "a", Text: "alpha"}}

	cases := map[string]rerank.Request{
		"empty query":      baseRequest("   ", docs, 1),
		"no documents":     baseRequest("q", nil, 1),
		"empty doc id":     baseRequest("q", []rerank.Document{{ID: "", Text: "x"}}, 1),
		"empty doc text":   baseRequest("q", []rerank.Document{{ID: "a", Text: " "}}, 1),
		"duplicate doc id": baseRequest("q", []rerank.Document{{ID: "a", Text: "x"}, {ID: "a", Text: "y"}}, 1),
		"top_n zero":       baseRequest("q", docs, 0),
		"top_n too large":  baseRequest("q", docs, 2),
		"model conflict":   {Model: "other-model", Query: "q", Documents: docs, TopN: 1},
	}
	for name, req := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := p.Rerank(context.Background(), req)
			require.ErrorIs(t, err, rerank.ErrInvalidRequest)
		})
	}
}

func TestRerank_ErrorsDoNotEchoDocumentIDs(t *testing.T) {
	p, _ := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {})
	_, err := p.Rerank(context.Background(), baseRequest("q", []rerank.Document{
		{ID: "SECRET-DOC-ID", Text: "x"},
		{ID: "SECRET-DOC-ID", Text: "y"},
	}, 1))
	require.ErrorIs(t, err, rerank.ErrInvalidRequest)
	assert.NotContains(t, err.Error(), "SECRET-DOC-ID")
}

// --- Response validation (matrix rows 7-10, 14, 15) ---

func TestRerank_RejectsCardinalityMismatch(t *testing.T) {
	docs := []rerank.Document{{ID: "a", Text: "x"}, {ID: "b", Text: "y"}}
	p, _ := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		writeWire(w, `{"results":[{"index":0,"relevance_score":0.9}]}`)
	})
	_, err := p.Rerank(context.Background(), baseRequest("q", docs, 2))
	require.ErrorIs(t, err, rerank.ErrInvalidResponse)
}

func TestRerank_RejectsMissingIndexOrScore(t *testing.T) {
	docs := []rerank.Document{{ID: "a", Text: "x"}}
	for name, body := range map[string]string{
		"missing index": `{"results":[{"relevance_score":0.9}]}`,
		"missing score": `{"results":[{"index":0}]}`,
	} {
		t.Run(name, func(t *testing.T) {
			p := mustNewProvider(t, Options{
				APIKey: testAPIKey, BaseURL: testServerURL(t, body), Model: testModel,
				OutboundURL: security.OutboundURLOptions{AllowHTTP: true, AllowLocalNetworks: true},
			})
			_, err := p.Rerank(context.Background(), baseRequest("q", docs, 1))
			require.ErrorIs(t, err, rerank.ErrInvalidResponse)
		})
	}
}

func TestRerank_RejectsOutOfRangeAndDuplicateIndex(t *testing.T) {
	docs := []rerank.Document{{ID: "a", Text: "x"}, {ID: "b", Text: "y"}}
	for name, body := range map[string]string{
		"index out of range": `{"results":[{"index":0,"relevance_score":0.9},{"index":5,"relevance_score":0.1}]}`,
		"duplicate index":    `{"results":[{"index":1,"relevance_score":0.9},{"index":1,"relevance_score":0.1}]}`,
	} {
		t.Run(name, func(t *testing.T) {
			p := mustNewProvider(t, Options{
				APIKey: testAPIKey, BaseURL: testServerURL(t, body), Model: testModel,
				OutboundURL: security.OutboundURLOptions{AllowHTTP: true, AllowLocalNetworks: true},
			})
			_, err := p.Rerank(context.Background(), baseRequest("q", docs, 2))
			require.ErrorIs(t, err, rerank.ErrInvalidResponse)
		})
	}
}

func TestRerank_RejectsNonFiniteScore(t *testing.T) {
	docs := []rerank.Document{{ID: "a", Text: "x"}}
	p := mustNewProvider(t, Options{
		APIKey:      testAPIKey,
		BaseURL:     testServerURL(t, `{"results":[{"index":0,"relevance_score":1e999}]}`),
		Model:       testModel,
		OutboundURL: security.OutboundURLOptions{AllowHTTP: true, AllowLocalNetworks: true},
	})
	// 1e999 overflows float64; strict decode or score validation must reject it.
	_, err := p.Rerank(context.Background(), baseRequest("q", docs, 1))
	require.ErrorIs(t, err, rerank.ErrInvalidResponse)
}

func TestRerank_RejectsTrailingJSON(t *testing.T) {
	docs := []rerank.Document{{ID: "a", Text: "x"}}
	p := mustNewProvider(t, Options{
		APIKey:      testAPIKey,
		BaseURL:     testServerURL(t, `{"results":[{"index":0,"relevance_score":1.0}]} {"extra":true}`),
		Model:       testModel,
		OutboundURL: security.OutboundURLOptions{AllowHTTP: true, AllowLocalNetworks: true},
	})
	_, err := p.Rerank(context.Background(), baseRequest("q", docs, 1))
	require.ErrorIs(t, err, rerank.ErrInvalidResponse)
}

func TestRerank_RejectsUnknownFields(t *testing.T) {
	docs := []rerank.Document{{ID: "a", Text: "x"}}
	p := mustNewProvider(t, Options{
		APIKey:      testAPIKey,
		BaseURL:     testServerURL(t, `{"results":[{"index":0,"relevance_score":1.0}],"surprise":true}`),
		Model:       testModel,
		OutboundURL: security.OutboundURLOptions{AllowHTTP: true, AllowLocalNetworks: true},
	})
	_, err := p.Rerank(context.Background(), baseRequest("q", docs, 1))
	require.ErrorIs(t, err, rerank.ErrInvalidResponse)
}

// --- Transport and bounds (matrix rows 11, 12, 13, 16, 17) ---

func TestRerank_Non2xxDoesNotLeakBody(t *testing.T) {
	docs := []rerank.Document{{ID: "a", Text: "x"}}
	p := mustNewProvider(t, Options{
		APIKey: testAPIKey,
		BaseURL: testServerURLWithStatus(t, http.StatusTooManyRequests,
			`{"message":"rate limited, query was SECRETQUERY"}`),
		Model:       testModel,
		OutboundURL: security.OutboundURLOptions{AllowHTTP: true, AllowLocalNetworks: true},
	})
	_, err := p.Rerank(context.Background(), baseRequest("SECRETQUERY", docs, 1))
	require.ErrorIs(t, err, rerank.ErrUnavailable)
	assert.Contains(t, err.Error(), "429")
	assert.NotContains(t, err.Error(), "SECRETQUERY")
	assert.NotContains(t, err.Error(), "rate limited")
}

func testServerURLWithStatus(t *testing.T, status int, body string) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
		_, _ = io.WriteString(w, body)
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

func TestRerank_ResponseTooLarge(t *testing.T) {
	docs := []rerank.Document{{ID: "a", Text: "x"}}
	p := mustNewProvider(t, Options{
		APIKey: testAPIKey,
		BaseURL: testServerURL(t,
			`{"results":[{"index":0,"relevance_score":1.0}],"id":"`+strings.Repeat("x", 2048)+`"}`),
		Model:            testModel,
		MaxResponseBytes: 64,
		OutboundURL:      security.OutboundURLOptions{AllowHTTP: true, AllowLocalNetworks: true},
	})
	_, err := p.Rerank(context.Background(), baseRequest("q", docs, 1))
	require.ErrorIs(t, err, rerank.ErrResponseTooLarge)
}

func TestRerank_RequestTooLarge(t *testing.T) {
	docs := []rerank.Document{{ID: "a", Text: strings.Repeat("x", 1024)}}
	p, _ := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		t.Error("server must not be reached for oversized requests")
	})
	p.maxRequestBytes = 16
	_, err := p.Rerank(context.Background(), baseRequest("q", docs, 1))
	require.ErrorIs(t, err, rerank.ErrRequestTooLarge)
}

func TestRerank_RejectsRedirect(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeWire(w, `{"results":[{"index":0,"relevance_score":1.0}]}`)
	}))
	t.Cleanup(target.Close)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL+"?token=REDIRECTSECRET", http.StatusFound)
	}))
	t.Cleanup(srv.Close)
	p := mustNewProvider(t, Options{
		APIKey:      testAPIKey,
		BaseURL:     srv.URL,
		Model:       testModel,
		OutboundURL: security.OutboundURLOptions{AllowHTTP: true, AllowLocalNetworks: true},
	})
	docs := []rerank.Document{{ID: "a", Text: "x"}}
	_, err := p.Rerank(context.Background(), baseRequest("q", docs, 1))
	require.ErrorIs(t, err, rerank.ErrUnavailable)
	assert.NotContains(t, err.Error(), "REDIRECTSECRET")
}

func TestRerank_RedactsTransportError(t *testing.T) {
	docs := []rerank.Document{{ID: "a", Text: "x"}}
	// Start and immediately close a server to force a connection refusal.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {}))
	url := srv.URL
	srv.Close()
	p := mustNewProvider(t, Options{
		APIKey:      testAPIKey,
		BaseURL:     url,
		Model:       testModel,
		OutboundURL: security.OutboundURLOptions{AllowHTTP: true, AllowLocalNetworks: true},
	})
	_, err := p.Rerank(context.Background(), baseRequest("q", docs, 1))
	require.ErrorIs(t, err, rerank.ErrUnavailable)
	assert.NotContains(t, err.Error(), "127.0.0.1", "transport error must be redacted, not echoed")
}

func TestRerank_InjectedClientNotMutated(t *testing.T) {
	injected := &http.Client{}
	require.Nil(t, injected.CheckRedirect, "precondition: injected client starts with nil CheckRedirect")
	p := mustNewProvider(t, Options{
		APIKey:      testAPIKey,
		Model:       testModel,
		HTTPClient:  injected,
		BaseURL:     testServerURL(t, `{"results":[{"index":0,"relevance_score":1.0}]}`),
		OutboundURL: security.OutboundURLOptions{AllowHTTP: true, AllowLocalNetworks: true},
	})
	docs := []rerank.Document{{ID: "a", Text: "x"}}
	_, err := p.Rerank(context.Background(), baseRequest("q", docs, 1))
	require.NoError(t, err)
	assert.Nil(t, injected.CheckRedirect, "injected client CheckRedirect was mutated in place")
}

func TestRerank_ContextCancellation(t *testing.T) {
	docs := []rerank.Document{{ID: "a", Text: "x"}}
	p, _ := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		t.Error("server must not be reached for cancelled contexts")
	})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := p.Rerank(ctx, baseRequest("q", docs, 1))
	require.Error(t, err)
}

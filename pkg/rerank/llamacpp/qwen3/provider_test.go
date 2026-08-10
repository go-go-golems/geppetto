package qwen3

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-go-golems/geppetto/pkg/rerank"
	"github.com/go-go-golems/geppetto/pkg/security"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testModel = "qwen3-reranker-4b"

func localOptions(server *httptest.Server) Options {
	return Options{
		BaseURL:     server.URL,
		Model:       testModel,
		OutboundURL: security.OutboundURLOptions{AllowHTTP: true, AllowLocalNetworks: true},
	}
}

func mustNewProvider(t *testing.T, options Options) *Provider {
	t.Helper()
	provider, err := New(options)
	require.NoError(t, err)
	return provider
}

func qwenRequest(documents ...rerank.Document) rerank.Request {
	return rerank.Request{Query: "query", Documents: documents, TopN: len(documents)}
}

func completionBody(yes, no float64, usage bool) map[string]any {
	body := map[string]any{
		"completion_probabilities": []map[string]any{{
			"top_logprobs": []map[string]any{
				{"token": "yes", "logprob": yes},
				{"token": "No", "logprob": no},
			},
		}},
	}
	if usage {
		body["tokens_evaluated"] = 10
		body["tokens_predicted"] = 1
	}
	return body
}

func TestNewValidatesConfigurationAndIdentity(t *testing.T) {
	t.Parallel()

	local := security.OutboundURLOptions{AllowHTTP: true, AllowLocalNetworks: true}
	negative := -1.0
	for _, test := range []struct {
		name    string
		options Options
	}{
		{name: "missing URL", options: Options{Model: testModel}},
		{name: "missing model", options: Options{BaseURL: "https://example.com"}},
		{name: "local denied by default", options: Options{BaseURL: "http://127.0.0.1:8080", Model: testModel}},
		{name: "negative request bound", options: Options{BaseURL: "http://127.0.0.1:8080", Model: testModel, OutboundURL: local, MaxRequestBytes: -1}},
		{name: "negative response bound", options: Options{BaseURL: "http://127.0.0.1:8080", Model: testModel, OutboundURL: local, MaxResponseBytes: -1}},
		{name: "negative concurrency", options: Options{BaseURL: "http://127.0.0.1:8080", Model: testModel, OutboundURL: local, MaxConcurrency: -1}},
		{name: "negative cost", options: Options{BaseURL: "http://127.0.0.1:8080", Model: testModel, OutboundURL: local, CostPerMTokens: &negative}},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			_, err := New(test.options)
			require.ErrorIs(t, err, rerank.ErrInvalidRequest)
		})
	}

	provider := mustNewProvider(t, Options{
		BaseURL: "http://127.0.0.1:8080", Model: testModel, OutboundURL: local,
	})
	assert.Equal(t, rerank.Model{Provider: ProviderName, Name: testModel}, provider.Model())
}

func TestRerankPreservesOfficialPromptScoringConcurrencyAndOrdering(t *testing.T) {
	var active atomic.Int32
	var maximum atomic.Int32
	var requestsMu sync.Mutex
	requests := make([]completionRequest, 0, 6)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		require.Equal(t, http.MethodPost, request.Method)
		require.Equal(t, "/completion", request.URL.Path)
		require.Equal(t, "application/json", request.Header.Get("Content-Type"))
		current := active.Add(1)
		defer active.Add(-1)
		for prior := maximum.Load(); current > prior && !maximum.CompareAndSwap(prior, current); prior = maximum.Load() {
		}
		time.Sleep(15 * time.Millisecond)

		var decoded completionRequest
		require.NoError(t, json.NewDecoder(request.Body).Decode(&decoded))
		requestsMu.Lock()
		requests = append(requests, decoded)
		requestsMu.Unlock()
		require.Equal(t, 1, decoded.NPredict)
		require.Equal(t, 20, decoded.NProbs)
		require.Zero(t, decoded.Temperature)

		yes, no := -2.0, -0.2
		if strings.Contains(decoded.Prompt, "relevant-doc") {
			yes, no = -0.2, -2.0
		}
		if strings.Contains(decoded.Prompt, "tie-doc") {
			yes, no = -1.0, -1.0
		}
		_ = json.NewEncoder(w).Encode(completionBody(yes, no, true))
	}))
	t.Cleanup(server.Close)

	options := localOptions(server)
	options.MaxConcurrency = 2
	provider := mustNewProvider(t, options)
	documents := []rerank.Document{
		{ID: "boring", Text: "boring-doc"},
		{ID: "relevant", Text: "relevant-doc"},
		{ID: "tie-a", Text: "tie-doc-a"},
		{ID: "tie-b", Text: "tie-doc-b"},
	}
	response, err := provider.Rerank(t.Context(), rerank.Request{
		Model: testModel, Query: "query", Documents: documents, TopN: 3,
	})
	require.NoError(t, err)

	require.Len(t, response.Results, 3)
	assert.Equal(t, "relevant", response.Results[0].DocumentID)
	assert.Equal(t, "tie-a", response.Results[1].DocumentID)
	assert.Equal(t, "tie-b", response.Results[2].DocumentID)
	for i, result := range response.Results {
		assert.Equal(t, i+1, result.Rank)
	}
	wantScore := math.Exp(-0.2) / (math.Exp(-0.2) + math.Exp(-2.0))
	assert.InDelta(t, wantScore, response.Results[0].Score, 1e-12)
	assert.Equal(t, ProviderName, response.Provider)
	assert.Equal(t, testModel, response.Model)
	require.NotNil(t, response.DurationMs)
	require.NotNil(t, response.Usage)
	assert.Equal(t, 40, response.Usage.InputTokens)
	assert.Equal(t, 44, response.Usage.TotalTokens)
	assert.Nil(t, response.Cost)
	assert.Equal(t, int32(2), maximum.Load())
	require.Len(t, requests, 4, "all candidates must be scored before TopN")

	wantPrompt := "<|im_start|>system\n" + officialSystemPrompt + "<|im_end|>\n" +
		"<|im_start|>user\n<Instruct>: " + officialInstruction +
		"\n<Query>: query\n<Document>: relevant-doc<|im_end|>\n" +
		"<|im_start|>assistant\n<think>\n\n</think>\n\n"
	found := false
	for _, request := range requests {
		if request.Prompt == wantPrompt {
			found = true
		}
	}
	assert.True(t, found, "official prompt bytes were not preserved")
}

func TestNormalizedYesProbabilityParityAndValidation(t *testing.T) {
	t.Parallel()

	ptr := func(value float64) *float64 { return &value }
	for _, test := range []struct {
		name    string
		values  []tokenLogprob
		want    float64
		wantErr bool
	}{
		{
			name: "case variants aggregate",
			values: []tokenLogprob{
				{Token: " yes", Logprob: ptr(-1)},
				{Token: "YES", Logprob: ptr(-2)},
				{Token: "No", Logprob: ptr(-3)},
			},
			want: (math.Exp(-1) + math.Exp(-2)) / (math.Exp(-1) + math.Exp(-2) + math.Exp(-3)),
		},
		{name: "no labels preserves historical zero", values: []tokenLogprob{{Token: "maybe", Logprob: ptr(-1)}}, want: 0},
		{name: "yes only", values: []tokenLogprob{{Token: "yes", Logprob: ptr(1000)}}, want: 1},
		{name: "no only", values: []tokenLogprob{{Token: "no", Logprob: ptr(1000)}}, want: 0},
		{name: "missing relevant logprob", values: []tokenLogprob{{Token: "yes"}}, wantErr: true},
		{name: "non-finite relevant logprob", values: []tokenLogprob{{Token: "yes", Logprob: ptr(math.NaN())}}, wantErr: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got, err := normalizedYesProbability(test.values)
			if test.wantErr {
				require.ErrorIs(t, err, rerank.ErrInvalidResponse)
				return
			}
			require.NoError(t, err)
			assert.InDelta(t, test.want, got, 1e-12)
		})
	}
}

func TestRerankValidatesRequestBeforeTransport(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		calls.Add(1)
	}))
	t.Cleanup(server.Close)
	provider := mustNewProvider(t, localOptions(server))
	for _, request := range []rerank.Request{
		{Query: "", Documents: []rerank.Document{{ID: "a", Text: "x"}}, TopN: 1},
		{Query: "q", TopN: 1},
		{Model: "other", Query: "q", Documents: []rerank.Document{{ID: "a", Text: "x"}}, TopN: 1},
		{Query: "q", Documents: []rerank.Document{{ID: "", Text: "x"}}, TopN: 1},
		{Query: "q", Documents: []rerank.Document{{ID: "a", Text: "x"}, {ID: "a", Text: "y"}}, TopN: 2},
		{Query: "q", Documents: []rerank.Document{{ID: "a", Text: ""}}, TopN: 1},
		{Query: "q", Documents: []rerank.Document{{ID: "a", Text: "x"}}, TopN: 0},
		{Query: "q", Documents: []rerank.Document{{ID: "a", Text: "x"}}, TopN: 2},
	} {
		_, err := provider.Rerank(t.Context(), request)
		require.ErrorIs(t, err, rerank.ErrInvalidRequest, "%#v", request)
	}
	assert.Zero(t, calls.Load())
}

func TestRerankUsageAndCostAreAllOrNothing(t *testing.T) {
	t.Parallel()

	var requestNumber atomic.Int32
	var allUsage atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		withUsage := allUsage.Load() || requestNumber.Add(1) != 2
		_ = json.NewEncoder(w).Encode(completionBody(-0.2, -2, withUsage))
	}))
	t.Cleanup(server.Close)
	rate := 2.0
	options := localOptions(server)
	options.CostPerMTokens = &rate
	provider := mustNewProvider(t, options)
	response, err := provider.Rerank(t.Context(), qwenRequest(
		rerank.Document{ID: "a", Text: "a"},
		rerank.Document{ID: "b", Text: "b"},
	))
	require.NoError(t, err)
	assert.Nil(t, response.Usage)
	assert.Nil(t, response.Cost)

	requestNumber.Store(0)
	allUsage.Store(true)
	response, err = provider.Rerank(t.Context(), qwenRequest(
		rerank.Document{ID: "a", Text: "a"},
		rerank.Document{ID: "b", Text: "b"},
	))
	require.NoError(t, err)
	require.NotNil(t, response.Usage)
	require.NotNil(t, response.Cost)
	assert.Equal(t, 20, response.Usage.InputTokens)
	assert.Equal(t, 0.00004, *response.Cost)
}

func TestRerankRejectsInvalidResponses(t *testing.T) {
	t.Parallel()

	responses := map[string]string{
		"invalid JSON":          `{`,
		"trailing JSON":         `{"completion_probabilities":[{"top_logprobs":[]}]} {}`,
		"missing probability":   `{}`,
		"negative input usage":  `{"tokens_evaluated":-1,"tokens_predicted":1,"completion_probabilities":[{"top_logprobs":[]}]}`,
		"negative output usage": `{"tokens_evaluated":1,"tokens_predicted":-1,"completion_probabilities":[{"top_logprobs":[]}]}`,
		"missing yes logprob":   `{"completion_probabilities":[{"top_logprobs":[{"token":"yes"}]}]}`,
	}
	for name, body := range responses {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, _ = io.WriteString(w, body)
			}))
			t.Cleanup(server.Close)
			provider := mustNewProvider(t, localOptions(server))
			_, err := provider.Rerank(t.Context(), qwenRequest(rerank.Document{ID: "a", Text: "secret document"}))
			require.ErrorIs(t, err, rerank.ErrInvalidResponse)
			assert.NotContains(t, err.Error(), "secret document")
		})
	}
}

func TestRerankEnforcesTransportAndBodyBoundsWithoutLeakingContent(t *testing.T) {
	t.Parallel()

	t.Run("non-2xx", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = io.WriteString(w, `{"error":"SUPERSECRET"}`)
		}))
		t.Cleanup(server.Close)
		provider := mustNewProvider(t, localOptions(server))
		_, err := provider.Rerank(t.Context(), qwenRequest(rerank.Document{ID: "a", Text: "PRIVATE-DOC"}))
		require.ErrorIs(t, err, rerank.ErrUnavailable)
		assert.NotContains(t, err.Error(), "SUPERSECRET")
		assert.NotContains(t, err.Error(), "PRIVATE-DOC")
	})

	t.Run("oversized request", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
			t.Fatal("transport must not be called")
		}))
		t.Cleanup(server.Close)
		options := localOptions(server)
		options.MaxRequestBytes = 32
		provider := mustNewProvider(t, options)
		_, err := provider.Rerank(t.Context(), qwenRequest(rerank.Document{ID: "a", Text: strings.Repeat("x", 100)}))
		require.ErrorIs(t, err, rerank.ErrRequestTooLarge)
	})

	t.Run("oversized response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = io.WriteString(w, strings.Repeat(" ", 200))
		}))
		t.Cleanup(server.Close)
		options := localOptions(server)
		options.MaxResponseBytes = 32
		provider := mustNewProvider(t, options)
		_, err := provider.Rerank(t.Context(), qwenRequest(rerank.Document{ID: "a", Text: "x"}))
		require.ErrorIs(t, err, rerank.ErrResponseTooLarge)
	})
}

func TestRerankPropagatesCancellationAndRejectsRedirects(t *testing.T) {
	t.Parallel()

	t.Run("cancellation", func(t *testing.T) {
		started := make(chan struct{})
		client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
			close(started)
			<-request.Context().Done()
			return nil, request.Context().Err()
		})}
		provider := mustNewProvider(t, Options{
			BaseURL: "http://127.0.0.1:18012", Model: testModel, HTTPClient: client,
			OutboundURL: security.OutboundURLOptions{AllowHTTP: true, AllowLocalNetworks: true},
		})
		ctx, cancel := context.WithCancel(context.Background())
		errorsChannel := make(chan error, 1)
		go func() {
			_, err := provider.Rerank(ctx, qwenRequest(rerank.Document{ID: "a", Text: "x"}))
			errorsChannel <- err
		}()
		select {
		case <-started:
		case <-time.After(time.Second):
			t.Fatal("request did not start")
		}
		cancel()
		select {
		case err := <-errorsChannel:
			require.ErrorIs(t, err, rerank.ErrUnavailable)
		case <-time.After(time.Second):
			t.Fatal("request did not stop")
		}
	})

	t.Run("redirect", func(t *testing.T) {
		target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_ = json.NewEncoder(w).Encode(completionBody(-0.2, -2, false))
		}))
		t.Cleanup(target.Close)
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
			http.Redirect(w, request, target.URL+"?token=REDIRECTSECRET", http.StatusFound)
		}))
		t.Cleanup(server.Close)
		provider := mustNewProvider(t, localOptions(server))
		_, err := provider.Rerank(t.Context(), qwenRequest(rerank.Document{ID: "a", Text: "x"}))
		require.ErrorIs(t, err, rerank.ErrUnavailable)
		assert.NotContains(t, err.Error(), "REDIRECTSECRET")
	})
}

func TestInjectedClientIsNotMutated(t *testing.T) {
	t.Parallel()

	client := &http.Client{}
	_ = mustNewProvider(t, Options{
		BaseURL: "http://127.0.0.1:18012", Model: testModel, HTTPClient: client,
		OutboundURL: security.OutboundURLOptions{AllowHTTP: true, AllowLocalNetworks: true},
	})
	assert.Nil(t, client.CheckRedirect)
}

func TestDecodeResponseAllowsAdditionalLlamaCppObservationFields(t *testing.T) {
	t.Parallel()

	response, err := decodeResponse([]byte(`{
		"content":"yes",
		"timings":{"prompt_ms":1},
		"completion_probabilities":[{"top_logprobs":[]}]
	}`))
	require.NoError(t, err)
	require.Len(t, response.CompletionProbabilities, 1)
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (function roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return function(request)
}

func TestProviderErrorClassificationUsesStableSentinels(t *testing.T) {
	t.Parallel()

	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("TRANSPORTSECRET")
	})}
	provider := mustNewProvider(t, Options{
		BaseURL: "http://127.0.0.1:18012", Model: testModel, HTTPClient: client,
		OutboundURL: security.OutboundURLOptions{AllowHTTP: true, AllowLocalNetworks: true},
	})
	_, err := provider.Rerank(t.Context(), qwenRequest(rerank.Document{ID: "a", Text: "DOCUMENTSECRET"}))
	require.ErrorIs(t, err, rerank.ErrUnavailable)
	assert.NotContains(t, err.Error(), "TRANSPORTSECRET")
	assert.NotContains(t, err.Error(), "DOCUMENTSECRET")
}

package llamacpp

import (
	"context"
	"os"
	"testing"

	"github.com/go-go-golems/geppetto/pkg/rerank"
	"github.com/go-go-golems/geppetto/pkg/security"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLive_RerankAgainstRealLlamaCpp is an opt-in test that runs against a real
// llama.cpp server. It is skipped unless GEPPETTO_LIVE_RERANK=1 is set exactly.
// It never falls back to a fixture and never starts external services itself.
//
// Set:
//
//	GEPPETTO_LIVE_RERANK=1 \
//	GEPPETTO_RERANK_BASE_URL=http://127.0.0.1:18012 \
//	GEPPETTO_RERANK_MODEL=qllama/bge-reranker-v2-m3:q4_k_m \
//	go test ./pkg/rerank/llamacpp -run TestLive -v -count=1
func TestLive_RerankAgainstRealLlamaCpp(t *testing.T) {
	if os.Getenv("GEPPETTO_LIVE_RERANK") != "1" {
		t.Skip("skipping live rerank test; set GEPPETTO_LIVE_RERANK=1 to run")
	}
	baseURL := os.Getenv("GEPPETTO_RERANK_BASE_URL")
	model := os.Getenv("GEPPETTO_RERANK_MODEL")
	if baseURL == "" || model == "" {
		t.Fatal("GEPPETTO_RERANK_BASE_URL and GEPPETTO_RERANK_MODEL must be set for live test")
	}

	provider, err := New(Options{
		BaseURL:     baseURL,
		Model:       model,
		OutboundURL: security.OutboundURLOptions{AllowHTTP: true, AllowLocalNetworks: true},
	})
	require.NoError(t, err)

	docs := []rerank.Document{
		{ID: "chunk-001", Text: "A payroll adjustment corrects wages or deductions."},
		{ID: "chunk-002", Text: "Cypress trees tolerate dry conditions."},
		{ID: "chunk-003", Text: "Weather forecasts predict rain."},
	}
	resp, err := provider.Rerank(context.Background(), rerank.Request{
		Query:     "How does TTC calculate a payroll adjustment?",
		Documents: docs,
		TopN:      3,
	})
	require.NoError(t, err)

	assert.Equal(t, ProviderName, resp.Provider)
	assert.Equal(t, model, resp.Model)
	require.Len(t, resp.Results, 3)
	// Ranks assigned from 1.
	for i, r := range resp.Results {
		assert.Equal(t, i+1, r.Rank, "rank should be position+1")
	}
	// Scores must be finite (already enforced by validation); record them for
	// the live qualification record.
	t.Logf("live rerank results: %+v", resp.Results)
	t.Logf("live rerank usage: %+v", resp.Usage)
	// The payroll document should rank first for the payroll query.
	assert.Equal(t, "chunk-001", resp.Results[0].DocumentID,
		"payroll document should rank first for the payroll query")
}

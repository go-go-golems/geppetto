package cohere

import (
	"context"
	"os"
	"testing"

	"github.com/go-go-golems/geppetto/pkg/rerank"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLive_RerankAgainstRealCohere is an opt-in test that runs against the
// real Cohere v2 /rerank API. It is skipped unless GEPPETTO_LIVE_RERANK=1 is
// set exactly. It never falls back to a fixture and never starts external
// services itself.
//
// Set:
//
//	GEPPETTO_LIVE_RERANK=1 \
//	COHERE_API_KEY=<your-key> \
//	GEPPETTO_RERANK_MODEL=rerank-v3.5 \
//	go test ./pkg/rerank/cohere -run TestLive -v -count=1
func TestLive_RerankAgainstRealCohere(t *testing.T) {
	if os.Getenv("GEPPETTO_LIVE_RERANK") != "1" {
		t.Skip("skipping live rerank test; set GEPPETTO_LIVE_RERANK=1 to run")
	}
	apiKey := os.Getenv("COHERE_API_KEY")
	model := os.Getenv("GEPPETTO_RERANK_MODEL")
	if apiKey == "" || model == "" {
		t.Skip("skipping live rerank test; set COHERE_API_KEY and GEPPETTO_RERANK_MODEL")
	}

	p, err := New(Options{
		APIKey: apiKey,
		Model:  model,
		// No BaseURL: the live test targets the canonical hosted endpoint and
		// the default outbound policy (HTTPS, no local networks) applies.
	})
	require.NoError(t, err)

	docs := []rerank.Document{
		{ID: "nevada", Text: "Carson City is the capital city of the American state of Nevada."},
		{ID: "mariana", Text: "The Commonwealth of the Northern Mariana Islands is a group of islands in the Pacific Ocean. Its capital is Saipan."},
		{ID: "grammar", Text: "Capitalization in English grammar is the use of a capital letter at the start of a word."},
		{ID: "dc", Text: "Washington, D.C. is the capital of the United States."},
		{ID: "punishment", Text: "Capital punishment has existed in the United States since before the United States was a country."},
	}

	resp, err := p.Rerank(context.Background(), rerank.Request{
		Query:     "What is the capital of the United States?",
		Documents: docs,
		TopN:      3,
	})
	require.NoError(t, err)

	assert.Equal(t, ProviderName, resp.Provider)
	assert.Equal(t, model, resp.Model)
	require.Len(t, resp.Results, 3)
	for i, result := range resp.Results {
		assert.Equal(t, i+1, result.Rank)
		assert.NotEmpty(t, result.DocumentID)
		assert.GreaterOrEqual(t, result.Score, 0.0)
		assert.LessOrEqual(t, result.Score, 1.0, "cohere relevance scores are normalized to [0,1]")
		if i > 0 {
			assert.GreaterOrEqual(t, resp.Results[i-1].Score, result.Score, "results must be score-sorted")
		}
	}
	// The most relevant document must win by a wide margin on this fixture.
	assert.Equal(t, "dc", resp.Results[0].DocumentID)

	// Cohere bills in search units, not tokens: Usage stays nil (DR-3), and no
	// per-search rate is configured here, so Cost stays unknown.
	assert.Nil(t, resp.Usage)
	assert.Nil(t, resp.Cost)
	assert.NotEmpty(t, resp.RequestID, "cohere carries the request id in the response body")
	require.NotNil(t, resp.DurationMs)

	t.Logf("live cohere rerank: top=%s score=%.4f requestID=%s durationMs=%d",
		resp.Results[0].DocumentID, resp.Results[0].Score, resp.RequestID, *resp.DurationMs)
}

package embeddings

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCohereReranker_Rerank(t *testing.T) {
	// Create mock documents
	documents := []string{
		"Carson City is the capital city of the American state of Nevada.",
		"The Commonwealth of the Northern Mariana Islands is a group of islands in the Pacific Ocean. Its capital is Saipan.",
		"Capitalization in English grammar is the use of a capital letter at the start of a word.",
		"Washington, D.C. is the capital of the United States.",
		"Capital punishment has existed in the United States since before the United States was a country.",
	}

	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		assert.Equal(t, "/v2/rerank", r.URL.Path)
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "Bearer test-api-key", r.Header.Get("Authorization"))

		// Parse request body
		var req CohereRerankRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)

		// Verify request body
		assert.Equal(t, "test-model", req.Model)
		assert.Equal(t, "What is the capital of the United States?", req.Query)
		assert.Equal(t, documents, req.Documents)
		assert.NotNil(t, req.TopN)
		assert.Equal(t, 3, *req.TopN)

		// Create mock response
		resp := CohereRerankResponse{
			ID: "test-id",
			Results: []struct {
				Index          int     `json:"index"`
				RelevanceScore float64 `json:"relevance_score"`
			}{
				{Index: 3, RelevanceScore: 0.999071},
				{Index: 4, RelevanceScore: 0.786787},
				{Index: 0, RelevanceScore: 0.327131},
			},
			Meta: struct {
				APIVersion struct {
					Version        string `json:"version"`
					IsExperimental bool   `json:"is_experimental"`
				} `json:"api_version"`
				BilledUnits struct {
					SearchUnits int `json:"search_units"`
				} `json:"billed_units"`
			}{
				APIVersion: struct {
					Version        string `json:"version"`
					IsExperimental bool   `json:"is_experimental"`
				}{
					Version:        "2",
					IsExperimental: false,
				},
				BilledUnits: struct {
					SearchUnits int `json:"search_units"`
				}{
					SearchUnits: 1,
				},
			},
		}

		// Write response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Errorf("Failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	// Create reranker using mock server
	reranker := NewCohereReranker(
		"test-api-key",
		"test-model",
		WithCohereRerankBaseURL(server.URL+"/v2/rerank"),
	)

	// Test Rerank
	results, err := reranker.Rerank(
		context.Background(),
		"What is the capital of the United States?",
		documents,
		WithTopN(3),
	)
	require.NoError(t, err)
	require.Len(t, results, 3)

	// Verify results
	assert.Equal(t, 3, results[0].Index)
	assert.Equal(t, documents[3], results[0].Document)
	assert.Equal(t, 0.999071, results[0].Score)

	assert.Equal(t, 4, results[1].Index)
	assert.Equal(t, documents[4], results[1].Document)
	assert.Equal(t, 0.786787, results[1].Score)

	assert.Equal(t, 0, results[2].Index)
	assert.Equal(t, documents[0], results[2].Document)
	assert.Equal(t, 0.327131, results[2].Score)
}

func TestCohereReranker_EmptyDocuments(t *testing.T) {
	// Create reranker
	reranker := NewCohereReranker(
		"test-api-key",
		"test-model",
	)

	// Test with empty documents
	results, err := reranker.Rerank(
		context.Background(),
		"test query",
		[]string{},
	)
	require.NoError(t, err)
	assert.Empty(t, results)
}

func TestCohereReranker_GetModel(t *testing.T) {
	reranker := NewCohereReranker(
		"test-api-key",
		"rerank-v3.5",
	)

	model := reranker.GetModel()
	assert.Equal(t, "rerank-v3.5", model.Name)
}

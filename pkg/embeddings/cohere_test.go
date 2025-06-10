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

func TestCohereProvider_GenerateEmbedding(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		assert.Equal(t, "/v2/embed", r.URL.Path)
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "Bearer test-api-key", r.Header.Get("Authorization"))

		// Parse request body
		var req CohereEmbedRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)

		// Verify request body
		assert.Equal(t, "test-model", req.Model)
		assert.Equal(t, "search_document", req.InputType)
		assert.Equal(t, []string{"test text"}, req.Texts)
		assert.Equal(t, 384, req.OutputDimension)

		// Create mock response
		mockEmbedding := make([]float32, 384)
		for i := range mockEmbedding {
			mockEmbedding[i] = float32(i) * 0.01
		}

		resp := CohereEmbedResponse{
			ID:    "test-id",
			Texts: []string{"test text"},
			Embeddings: CohereEmbeddingResult{
				Float: [][]float32{mockEmbedding},
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

	// Create provider using mock server
	provider := NewCohereProvider(
		"test-api-key",
		"test-model",
		384,
		WithCohereBaseURL(server.URL+"/v2/embed"),
	)

	// Test GenerateEmbedding
	embedding, err := provider.GenerateEmbedding(context.Background(), "test text")
	require.NoError(t, err)
	require.Len(t, embedding, 384)

	// Verify first few values
	assert.Equal(t, float32(0.0), embedding[0])
	assert.Equal(t, float32(0.01), embedding[1])
	assert.Equal(t, float32(0.02), embedding[2])
}

func TestCohereProvider_GenerateBatchEmbeddings(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Parse request body
		var req CohereEmbedRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)

		// Verify request has multiple texts
		assert.Equal(t, []string{"text 1", "text 2", "text 3"}, req.Texts)

		// Create mock embeddings
		mockEmbeddings := make([][]float32, len(req.Texts))
		for i := range mockEmbeddings {
			mockEmbeddings[i] = make([]float32, 256)
			for j := range mockEmbeddings[i] {
				mockEmbeddings[i][j] = float32(i*1000+j) * 0.001
			}
		}

		resp := CohereEmbedResponse{
			ID:    "test-batch-id",
			Texts: req.Texts,
			Embeddings: CohereEmbeddingResult{
				Float: mockEmbeddings,
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

	// Create provider using mock server
	provider := NewCohereProvider(
		"test-api-key",
		"test-model",
		256,
		WithCohereBaseURL(server.URL+"/v2/embed"),
		WithCohereInputType("classification"),
	)

	// Test GenerateBatchEmbeddings
	texts := []string{"text 1", "text 2", "text 3"}
	embeddings, err := provider.GenerateBatchEmbeddings(context.Background(), texts)
	require.NoError(t, err)
	require.Len(t, embeddings, 3)

	// Verify dimensions
	for i, embedding := range embeddings {
		require.Len(t, embedding, 256)
		// Check first value of each embedding
		assert.Equal(t, float32(i*1000)*0.001, embedding[0])
	}

	// Test empty batch
	emptyEmbeddings, err := provider.GenerateBatchEmbeddings(context.Background(), []string{})
	require.NoError(t, err)
	assert.Empty(t, emptyEmbeddings)
}

func TestCohereProvider_GetModel(t *testing.T) {
	provider := NewCohereProvider(
		"test-api-key",
		"embed-v4.0",
		1024,
	)

	model := provider.GetModel()
	assert.Equal(t, "embed-v4.0", model.Name)
	assert.Equal(t, 1024, model.Dimensions)
}

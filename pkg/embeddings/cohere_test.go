package embeddings

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-go-golems/geppetto/pkg/steps/ai/connection"
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

	// Create provider using mock server. The base URL is the server root; the
	// provider appends /v2/embed itself (base-URL semantics shared with the
	// rerank adapter).
	provider, err := NewCohereProvider(testEmbeddingConnection(t, "cohere", "test-api-key", server.URL, nil), "test-model", 384)
	require.NoError(t, err)

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
	provider, err := NewCohereProvider(testEmbeddingConnection(t, "cohere", "test-api-key", server.URL, nil), "test-model", 256, WithCohereInputType("classification"))
	require.NoError(t, err)

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
	provider, err := NewCohereProvider(testEmbeddingConnection(t, "cohere", "test-api-key", defaultCohereBaseURL, nil), "embed-v4.0", 1024)
	require.NoError(t, err)

	model := provider.GetModel()
	assert.Equal(t, "embed-v4.0", model.Name)
	assert.Equal(t, 1024, model.Dimensions)
}

// TestCohereProvider_RejectsCardinalityMismatch verifies the
// one-result-per-input contract: cache wrappers would silently corrupt or
// panic on short/oversized responses, so the provider must reject both.
func TestCohereProvider_RejectsCardinalityMismatch(t *testing.T) {
	for _, tc := range []struct {
		name      string
		returned  int
		requested int
	}{
		{"short response", 1, 2},
		{"oversized response", 3, 2},
		{"empty response", 0, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				embeddings := make([][]float32, tc.returned)
				for i := range embeddings {
					embeddings[i] = []float32{0.1}
				}
				resp := CohereEmbedResponse{
					Embeddings: CohereEmbeddingResult{Float: embeddings},
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(resp)
			}))
			defer server.Close()

			provider, err := NewCohereProvider(testEmbeddingConnection(t, "cohere", "key", server.URL, nil), "model", 0)
			require.NoError(t, err)

			texts := make([]string, tc.requested)
			for i := range texts {
				texts[i] = fmt.Sprintf("text %d", i)
			}
			_, err = provider.GenerateBatchEmbeddings(context.Background(), texts)
			require.Error(t, err)
			assert.Contains(t, err.Error(), fmt.Sprintf("returned %d embeddings for %d texts", tc.returned, tc.requested))
		})
	}
}

// TestCohereProvider_OutboundURLPolicy verifies the endpoint is validated at
// construction time: plaintext HTTP and loopback targets are rejected unless
// explicitly allowed.
func TestCohereProvider_OutboundURLPolicy(t *testing.T) {
	_, err := connection.New(connection.Config{Provider: "cohere", BaseURL: "http://127.0.0.1:9999", Auth: connection.StaticBearer("key"), Routes: CohereRoutes()})
	require.Error(t, err)
	_, err = NewCohereProvider(testEmbeddingConnection(t, "cohere", "key", "http://127.0.0.1:9999", nil), "model", 0)
	require.NoError(t, err)
	_, err = NewCohereProvider(testEmbeddingConnection(t, "cohere", "key", defaultCohereBaseURL, nil), "model", 0)
	require.NoError(t, err)
}

// TestCohereProvider_EndpointDerivedFromBase verifies the provider treats the
// override as a base URL and appends /v2/embed, keeping semantics consistent
// with the rerank adapter sharing the same cohere-base-url key.
func TestCohereProvider_EndpointDerivedFromBase(t *testing.T) {
	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		resp := CohereEmbedResponse{
			Embeddings: CohereEmbeddingResult{Float: [][]float32{{0.1}}},
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	provider, err := NewCohereProvider(testEmbeddingConnection(t, "cohere", "key", server.URL, nil), "model", 0)
	require.NoError(t, err)

	_, err = provider.GenerateEmbedding(context.Background(), "hello")
	require.NoError(t, err)
	assert.Equal(t, "/v2/embed", gotPath)
}

// TestCohereProvider_UsesConfiguredHTTPClient verifies the injected client's
// behavior applies (here: a short timeout trips on a stalling server) instead
// of a zero-timeout default client blocking forever.
func TestCohereProvider_UsesConfiguredHTTPClient(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond)
	}))
	defer server.Close()

	client := &http.Client{Timeout: 50 * time.Millisecond}
	provider, err := NewCohereProvider(testEmbeddingConnection(t, "cohere", "key", server.URL, client), "model", 0)
	require.NoError(t, err)

	_, err = provider.GenerateEmbedding(context.Background(), "hello")
	require.Error(t, err)
}

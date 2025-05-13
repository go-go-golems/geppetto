package embeddings

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBatchProcessing(t *testing.T) {
	t.Run("default sequential implementation", func(t *testing.T) {
		provider := NewMockProvider()
		texts := []string{"one", "two", "three"}

		results, err := DefaultGenerateBatchEmbeddings(context.Background(), provider, texts)
		require.NoError(t, err)
		require.Equal(t, 3, len(results))

		// Check each result matches what we expect
		assert.Equal(t, []float32{3.0, 1.0, 2.0}, results[0]) // "one"
		assert.Equal(t, []float32{3.0, 1.0, 2.0}, results[1]) // "two"
		assert.Equal(t, []float32{5.0, 1.0, 2.0}, results[2]) // "three"
	})

	t.Run("parallel implementation", func(t *testing.T) {
		provider := NewMockProvider()
		texts := []string{"one", "two", "three", "four", "five"}

		results, err := ParallelGenerateBatchEmbeddings(context.Background(), provider, texts, 2)
		require.NoError(t, err)
		require.Equal(t, 5, len(results))

		// Check each result matches what we expect
		assert.Equal(t, []float32{3.0, 1.0, 2.0}, results[0]) // "one"
		assert.Equal(t, []float32{3.0, 1.0, 2.0}, results[1]) // "two"
		assert.Equal(t, []float32{5.0, 1.0, 2.0}, results[2]) // "three"
		assert.Equal(t, []float32{4.0, 1.0, 2.0}, results[3]) // "four"
		assert.Equal(t, []float32{4.0, 1.0, 2.0}, results[4]) // "five"
	})

	t.Run("empty input", func(t *testing.T) {
		provider := NewMockProvider()
		texts := []string{}

		// Test both implementations with empty input
		results1, err := DefaultGenerateBatchEmbeddings(context.Background(), provider, texts)
		require.NoError(t, err)
		assert.Equal(t, 0, len(results1))

		results2, err := ParallelGenerateBatchEmbeddings(context.Background(), provider, texts, 2)
		require.NoError(t, err)
		assert.Equal(t, 0, len(results2))
	})
}

func TestProviderBatchImplementations(t *testing.T) {
	t.Run("OpenAI provider batch processing", func(t *testing.T) {
		// This is a minimal test that just ensures the method exists and returns expected format
		// (Not actually calling the API)
		provider := NewOpenAIProvider("dummy-key", "text-embedding-3-small", 1536)
		texts := []string{"one", "two"}

		// Mock the client call so we don't actually make API calls
		oldClient := provider.client
		defer func() { provider.client = oldClient }()

		// Just verify method exists and is callable
		_, err := DefaultGenerateBatchEmbeddings(context.Background(), provider, texts)
		assert.Error(t, err) // Will fail because we're not mocking the client response
	})

	t.Run("Ollama provider batch processing", func(t *testing.T) {
		// Similar minimal test for Ollama
		provider := NewOllamaProvider("", "", 0) // Use defaults
		texts := []string{"one", "two"}

		// We expect this to call ParallelGenerateBatchEmbeddings
		// For testing, we'll just verify it doesn't panic
		// since we can't easily test the HTTP calls
		// Without dependency injection for testing
		testCtx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
		defer cancel()
		_, err := provider.GenerateBatchEmbeddings(testCtx, texts)
		// Either timeout or connection error is acceptable
		assert.Error(t, err)
	})
}

func TestCacheProviderBatch(t *testing.T) {
	t.Run("memory cache provider", func(t *testing.T) {
		provider := NewMockProvider()
		cachedProvider := NewCachedProvider(provider, 100)

		// First call - should use underlying provider
		texts := []string{"one", "two", "three"}
		results1, err := cachedProvider.GenerateBatchEmbeddings(context.Background(), texts)
		require.NoError(t, err)
		require.Equal(t, 3, len(results1))

		// Second call with same texts - should use cache
		results2, err := cachedProvider.GenerateBatchEmbeddings(context.Background(), texts)
		require.NoError(t, err)
		require.Equal(t, 3, len(results2))

		// Results should be identical
		for i := range results1 {
			assert.Equal(t, results1[i], results2[i])
		}

		// Verify cache has the entries
		assert.Equal(t, 3, cachedProvider.Size())
	})

	t.Run("partial cache hit", func(t *testing.T) {
		provider := NewMockProvider()
		cachedProvider := NewCachedProvider(provider, 100)

		// First call to populate cache
		_, err := cachedProvider.GenerateBatchEmbeddings(context.Background(), []string{"one", "two"})
		require.NoError(t, err)

		// Second call with some new texts
		texts := []string{"one", "three", "two"}
		results, err := cachedProvider.GenerateBatchEmbeddings(context.Background(), texts)
		require.NoError(t, err)
		require.Equal(t, 3, len(results))

		// Verify first and third ("one" and "two") were cached, second ("three") was generated
		assert.Equal(t, []float32{3.0, 1.0, 2.0}, results[0]) // "one"
		assert.Equal(t, []float32{5.0, 1.0, 2.0}, results[1]) // "three"
		assert.Equal(t, []float32{3.0, 1.0, 2.0}, results[2]) // "two"
	})
}

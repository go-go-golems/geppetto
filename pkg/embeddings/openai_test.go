package embeddings

import (
	"testing"

	"github.com/sashabaranov/go-openai"
	"github.com/stretchr/testify/assert"
)

func TestSupportsOpenAIDimensionsOverride(t *testing.T) {
	assert.True(t, supportsOpenAIDimensionsOverride(openai.SmallEmbedding3))
	assert.True(t, supportsOpenAIDimensionsOverride(openai.LargeEmbedding3))
	assert.False(t, supportsOpenAIDimensionsOverride(openai.AdaEmbeddingV2))
}

func TestOpenAIProviderNewRequestDimensions(t *testing.T) {
	t.Run("text-embedding-3 includes dimensions", func(t *testing.T) {
		p := NewOpenAIProvider("dummy", openai.SmallEmbedding3, 1536)

		req := p.newRequest([]string{"hello"})
		assert.Equal(t, openai.SmallEmbedding3, req.Model)
		assert.Equal(t, 1536, req.Dimensions)
	})

	t.Run("text-embedding-ada-002 omits dimensions", func(t *testing.T) {
		p := NewOpenAIProvider("dummy", openai.AdaEmbeddingV2, 1536)

		req := p.newRequest([]string{"hello"})
		assert.Equal(t, openai.AdaEmbeddingV2, req.Model)
		assert.Equal(t, 0, req.Dimensions)
	})
}

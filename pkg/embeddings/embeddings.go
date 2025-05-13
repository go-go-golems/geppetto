package embeddings

import "context"

// EmbeddingModel contains metadata about the embedding model
type EmbeddingModel struct {
	Name       string
	Dimensions int
}

// Provider defines the interface for generating embeddings
type Provider interface {
	// GenerateEmbedding creates an embedding vector for the given text
	GenerateEmbedding(ctx context.Context, text string) ([]float32, error)

	// GenerateBatchEmbeddings creates embedding vectors for multiple texts at once
	// This is typically more efficient than calling GenerateEmbedding multiple times
	GenerateBatchEmbeddings(ctx context.Context, texts []string) ([][]float32, error)

	// GetModel returns information about the embedding model being used
	GetModel() EmbeddingModel
}

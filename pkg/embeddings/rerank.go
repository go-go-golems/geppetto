package embeddings

import "context"

// RankResult represents a single document ranking result
type RankResult struct {
	// Index is the position of the document in the original input list
	Index int `json:"index"`

	// Document is the original document text
	Document string `json:"document"`

	// Score is the relevance score between 0 and 1 (higher = more relevant)
	Score float64 `json:"score"`
}

// RerankerModel contains metadata about the reranking model
type RerankerModel struct {
	Name string
}

// RerankOption represents a configuration option for reranking
type RerankOption func(*rerankOptions)

type rerankOptions struct {
	topN            *int
	maxTokensPerDoc *int
}

// WithTopN limits the number of results returned
func WithTopN(n int) RerankOption {
	return func(o *rerankOptions) {
		o.topN = &n
	}
}

// WithMaxTokensPerDoc limits the number of tokens per document
func WithMaxTokensPerDoc(n int) RerankOption {
	return func(o *rerankOptions) {
		o.maxTokensPerDoc = &n
	}
}

// Reranker defines the interface for reranking documents based on their relevance to a query
type Reranker interface {
	// Rerank reorders the provided documents based on their relevance to the query
	Rerank(ctx context.Context, query string, documents []string, options ...RerankOption) ([]RankResult, error)

	// GetModel returns information about the reranking model being used
	GetModel() RerankerModel
}

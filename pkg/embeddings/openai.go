package embeddings

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/sashabaranov/go-openai"
)

type OpenAIProvider struct {
	client     *openai.Client
	model      openai.EmbeddingModel
	dimensions int
}

var _ Provider = &OpenAIProvider{}

// OpenAIOption configures NewOpenAIProvider.
type OpenAIOption func(*openai.ClientConfig)

// WithOpenAIBaseURL sends requests to baseURL (an OpenAI-compatible API
// root such as https://gateway.example/v1) instead of OpenAI's.
func WithOpenAIBaseURL(baseURL string) OpenAIOption {
	return func(c *openai.ClientConfig) {
		if u := strings.TrimSpace(baseURL); u != "" {
			c.BaseURL = strings.TrimRight(u, "/")
		}
	}
}

// WithOpenAIHTTPClient sets the HTTP client. The settings factory passes a
// client pinned to the base URL's origin (GEPPETTO-CREDSCOPE-001).
func WithOpenAIHTTPClient(c *http.Client) OpenAIOption {
	return func(cfg *openai.ClientConfig) {
		if c != nil {
			cfg.HTTPClient = c
		}
	}
}

func NewOpenAIProvider(apiKey string, model openai.EmbeddingModel, dimensions int, opts ...OpenAIOption) *OpenAIProvider {
	if model == "" {
		model = openai.SmallEmbedding3
	}
	if dimensions <= 0 {
		dimensions = 1536 // Default for Ada-002
	}
	cfg := openai.DefaultConfig(apiKey)
	for _, o := range opts {
		if o != nil {
			o(&cfg)
		}
	}

	return &OpenAIProvider{
		client:     openai.NewClientWithConfig(cfg),
		model:      model,
		dimensions: dimensions,
	}
}

func (p *OpenAIProvider) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	resp, err := p.client.CreateEmbeddings(ctx, p.newRequest([]string{text}))
	if err != nil {
		return nil, err
	}

	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("no embedding data received from OpenAI")
	}

	return resp.Data[0].Embedding, nil
}

func (p *OpenAIProvider) GenerateBatchEmbeddings(ctx context.Context, texts []string) ([][]float32, error) {
	// Handle empty input
	if len(texts) == 0 {
		return [][]float32{}, nil
	}

	// OpenAI API has native batch support
	resp, err := p.client.CreateEmbeddings(ctx, p.newRequest(texts))
	if err != nil {
		return nil, err
	}

	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("no embedding data received from OpenAI")
	}

	if len(resp.Data) != len(texts) {
		return nil, fmt.Errorf("received %d embeddings but expected %d", len(resp.Data), len(texts))
	}

	// Extract embeddings from response
	results := make([][]float32, len(texts))
	for i, data := range resp.Data {
		results[i] = data.Embedding
	}

	return results, nil
}

func (p *OpenAIProvider) GetModel() EmbeddingModel {
	return EmbeddingModel{
		Name:       string(p.model),
		Dimensions: p.dimensions,
	}
}

func (p *OpenAIProvider) newRequest(input []string) openai.EmbeddingRequest {
	req := openai.EmbeddingRequest{
		Input: input,
		Model: p.model,
	}
	if supportsOpenAIDimensionsOverride(p.model) {
		req.Dimensions = p.dimensions
	}
	return req
}

func supportsOpenAIDimensionsOverride(model openai.EmbeddingModel) bool {
	// OpenAI currently supports `dimensions` override on the text-embedding-3 family.
	return strings.HasPrefix(string(model), "text-embedding-3")
}

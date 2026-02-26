package embeddings

import (
	"context"
	"fmt"
	"strings"

	"github.com/sashabaranov/go-openai"
)

type OpenAIProvider struct {
	client     *openai.Client
	model      openai.EmbeddingModel
	dimensions int
}

var _ Provider = &OpenAIProvider{}

func NewOpenAIProvider(apiKey string, model openai.EmbeddingModel, dimensions int) *OpenAIProvider {
	if model == "" {
		model = openai.SmallEmbedding3
	}
	if dimensions <= 0 {
		dimensions = 1536 // Default for Ada-002
	}

	return &OpenAIProvider{
		client:     openai.NewClient(apiKey),
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

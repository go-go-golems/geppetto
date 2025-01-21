package embeddings

import (
	"context"
	"fmt"
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
	resp, err := p.client.CreateEmbeddings(ctx, openai.EmbeddingRequest{
		Input: []string{text},
		Model: p.model,
	})
	if err != nil {
		return nil, err
	}

	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("no embedding data received from OpenAI")
	}

	return resp.Data[0].Embedding, nil
}

func (p *OpenAIProvider) GetModel() EmbeddingModel {
	return EmbeddingModel{
		Name:       string(p.model),
		Dimensions: p.dimensions,
	}
}

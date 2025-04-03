package embeddings

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

type OllamaProvider struct {
	baseURL    string
	model      string
	dimensions int
}

type ollamaRequest struct {
	Model    string `json:"model"`
	Prompt   string `json:"prompt"`
	Template string `json:"template,omitempty"`
}

type ollamaResponse struct {
	Embedding []float32 `json:"embedding"`
}

var _ Provider = &OllamaProvider{}

func NewOllamaProvider(baseURL string, model string, dimensions int) *OllamaProvider {
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	if model == "" {
		model = "all-minilm"
	}
	if dimensions <= 0 {
		dimensions = 384 // Default for all-minilm
	}

	return &OllamaProvider{
		baseURL:    baseURL,
		model:      model,
		dimensions: dimensions,
	}
}

func (p *OllamaProvider) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	reqBody := ollamaRequest{
		Model:  p.model,
		Prompt: text,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/api/embeddings", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("Failed to close response body: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var result ollamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Embedding, nil
}

func (p *OllamaProvider) GetModel() EmbeddingModel {
	return EmbeddingModel{
		Name:       p.model,
		Dimensions: p.dimensions,
	}
}

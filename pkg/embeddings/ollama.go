package embeddings

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-go-golems/geppetto/pkg/steps/ai/connection"
	"github.com/pkg/errors"
)

type OllamaProvider struct {
	connection *connection.Connection
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

func OllamaRoutes() connection.Routes {
	return connection.Routes{"embed": connection.Relative("api/embeddings")}
}

func NewOllamaProvider(c *connection.Connection, model string, dimensions int) (*OllamaProvider, error) {
	if c == nil {
		return nil, errors.New("Ollama embeddings require a bound connection")
	}
	if _, err := c.Resolve("embed"); err != nil {
		return nil, err
	}
	if model == "" {
		model = "all-minilm"
	}
	if dimensions <= 0 {
		dimensions = 384 // Default for all-minilm
	}

	return &OllamaProvider{
		connection: c,
		model:      model,
		dimensions: dimensions,
	}, nil
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

	req, err := p.connection.NewRequest(ctx, http.MethodPost, "embed", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// #nosec G704 -- endpoint URL is validated above and local Ollama is intentionally allowed.
	resp, err := p.connection.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Warn().Err(err).Msg("failed to close response body")
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

func (p *OllamaProvider) GenerateBatchEmbeddings(ctx context.Context, texts []string) ([][]float32, error) {
	// Ollama doesn't currently support native batch embedding, so we use the parallel helper
	// with a concurrency limit of 4 to avoid overwhelming the server
	return ParallelGenerateBatchEmbeddings(ctx, p, texts, 4)
}

func (p *OllamaProvider) GetModel() EmbeddingModel {
	return EmbeddingModel{
		Name:       p.model,
		Dimensions: p.dimensions,
	}
}

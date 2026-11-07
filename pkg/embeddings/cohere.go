package embeddings

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	providerdefaults "github.com/go-go-golems/geppetto/pkg/steps/ai/providers/defaults"
	"io"
	"net/http"

	"github.com/go-go-golems/geppetto/pkg/steps/ai/connection"
	"github.com/pkg/errors"
)

// CohereProvider implements the Provider interface for Cohere embeddings API
type CohereProvider struct {
	connection *connection.Connection
	model      string
	inputType  string
	dimensions int
}

const (
	// defaultCohereBaseURL is the canonical hosted Cohere API root. The embed
	// endpoint path is appended to the base URL so a single cohere-base-url
	// override serves every Cohere capability (embeddings, reranking) with the
	// same base-URL semantics (mirrors pkg/rerank/cohere).
	defaultCohereBaseURL = providerdefaults.CohereBaseURL
	cohereEmbedPath      = "/v2/embed"
)

// CohereEmbedRequest represents the request structure for Cohere's embed API
type CohereEmbedRequest struct {
	Model           string   `json:"model"`
	InputType       string   `json:"input_type"`
	Texts           []string `json:"texts,omitempty"`
	OutputDimension int      `json:"output_dimension,omitempty"`
	EmbeddingTypes  []string `json:"embedding_types,omitempty"`
	Truncate        string   `json:"truncate,omitempty"`
}

// CohereEmbedResponse represents the response structure from Cohere's embed API
type CohereEmbedResponse struct {
	ID         string                `json:"id"`
	Embeddings CohereEmbeddingResult `json:"embeddings"`
	Texts      []string              `json:"texts"`
	Meta       struct {
		APIVersion struct {
			Version        string `json:"version"`
			IsExperimental bool   `json:"is_experimental"`
		} `json:"api_version"`
	} `json:"meta"`
}

// CohereEmbeddingResult contains the different embedding formats returned by Cohere
type CohereEmbeddingResult struct {
	Float [][]float32 `json:"float"`
}

// CohereRoutes declares embedding operations below the shared Cohere API root.
func CohereRoutes() connection.Routes {
	return connection.Routes{"embed": connection.Relative("v2/embed")}
}

func NewCohereProvider(c *connection.Connection, model string, dimensions int, options ...func(*CohereProvider)) (*CohereProvider, error) {
	if c == nil {
		return nil, errors.New("Cohere embeddings require a bound connection")
	}
	if _, err := c.Resolve("embed"); err != nil {
		return nil, err
	}
	provider := &CohereProvider{connection: c, model: model, inputType: "search_document", dimensions: dimensions}
	for _, option := range options {
		if option != nil {
			option(provider)
		}
	}
	return provider, nil
}

// WithCohereInputType sets the input type for the embeddings
func WithCohereInputType(inputType string) func(*CohereProvider) {
	return func(p *CohereProvider) {
		p.inputType = inputType
	}
}

// GenerateEmbedding implements the Provider interface
func (p *CohereProvider) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	// Call batch implementation with a single text
	embeddings, err := p.GenerateBatchEmbeddings(ctx, []string{text})
	if err != nil {
		return nil, err
	}

	// Return the first (and only) embedding
	if len(embeddings) == 0 {
		return nil, fmt.Errorf("no embeddings returned from Cohere API")
	}
	return embeddings[0], nil
}

// GenerateBatchEmbeddings implements the Provider interface
func (p *CohereProvider) GenerateBatchEmbeddings(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return [][]float32{}, nil
	}

	// Prepare the request
	request := CohereEmbedRequest{
		Model:          p.model,
		InputType:      p.inputType,
		Texts:          texts,
		EmbeddingTypes: []string{"float"},
		Truncate:       "END",
	}

	// Add output dimension if specified
	if p.dimensions > 0 {
		request.OutputDimension = p.dimensions
	}

	// Marshal the request to JSON
	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %w", err)
	}

	// Create HTTP request
	httpReq, err := p.connection.NewRequest(
		ctx,
		"POST",
		"embed",
		bytes.NewBuffer(requestBody),
	)
	if err != nil {
		return nil, fmt.Errorf("error creating HTTP request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("X-Client-Name", "go-go-golems/geppetto")

	// Send the request with the configured client (host-owned timeouts and
	// transport policy apply).
	resp, err := p.connection.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("error sending request to Cohere API: %w", err)
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("error closing response body: %w", cerr)
		}
	}()

	// Check for HTTP errors. The response body is intentionally not included
	// in the error: provider error payloads may echo request text, and errors
	// must never carry query/document content (same safety contract as
	// pkg/rerank).
	if resp.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<20))
		return nil, fmt.Errorf("cohere embed API returned status %d", resp.StatusCode)
	}

	// Decode the response
	var response CohereEmbedResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	// Enforce the one-result-per-input contract. A short response would leave
	// nil entries in cache wrappers (CachedProvider, DiskCacheProvider), while
	// an oversized response makes those wrappers index past missedIndices and
	// panic; reject both rather than propagating a corrupted mapping.
	if len(response.Embeddings.Float) != len(texts) {
		return nil, fmt.Errorf("cohere embed API returned %d embeddings for %d texts", len(response.Embeddings.Float), len(texts))
	}

	return response.Embeddings.Float, nil
}

// GetModel implements the Provider interface
func (p *CohereProvider) GetModel() EmbeddingModel {
	return EmbeddingModel{
		Name:       p.model,
		Dimensions: p.dimensions,
	}
}

// Ensure CohereProvider implements Provider interface
var _ Provider = &CohereProvider{}

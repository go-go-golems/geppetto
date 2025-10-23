package embeddings

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// CohereProvider implements the Provider interface for Cohere embeddings API
type CohereProvider struct {
	apiKey     string
	baseURL    string
	model      string
	inputType  string
	dimensions int
}

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

// NewCohereProvider creates a new Provider that uses Cohere's embedding API
func NewCohereProvider(apiKey, model string, dimensions int, options ...func(*CohereProvider)) *CohereProvider {
	provider := &CohereProvider{
		apiKey:     apiKey,
		baseURL:    "https://api.cohere.com/v2/embed",
		model:      model,
		inputType:  "search_document", // Default input type
		dimensions: dimensions,
	}

	// Apply options
	for _, option := range options {
		option(provider)
	}

	return provider
}

// WithCohereBaseURL sets a custom base URL for the Cohere API
func WithCohereBaseURL(baseURL string) func(*CohereProvider) {
	return func(p *CohereProvider) {
		p.baseURL = baseURL
	}
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
	httpReq, err := http.NewRequestWithContext(
		ctx,
		"POST",
		p.baseURL,
		bytes.NewBuffer(requestBody),
	)
	if err != nil {
		return nil, fmt.Errorf("error creating HTTP request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	httpReq.Header.Set("X-Client-Name", "go-go-golems/geppetto")

	// Send the request
	httpClient := &http.Client{}
	resp, err := httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("error sending request to Cohere API: %w", err)
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("error closing response body: %w", cerr)
		}
	}()

	// Check for HTTP errors
	if resp.StatusCode != http.StatusOK {
		var errorResponse map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&errorResponse); err == nil {
			return nil, fmt.Errorf("cohere API error (status %d): %v", resp.StatusCode, errorResponse)
		}
		return nil, fmt.Errorf("cohere API error (status %d)", resp.StatusCode)
	}

	// Decode the response
	var response CohereEmbedResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	// Verify embeddings format
	if len(response.Embeddings.Float) == 0 {
		return nil, fmt.Errorf("no float embeddings in response")
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

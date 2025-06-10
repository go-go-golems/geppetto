package embeddings

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// CohereReranker implements the Reranker interface using Cohere's rerank API
type CohereReranker struct {
	apiKey  string
	baseURL string
	model   string
}

// CohereRerankRequest represents the request structure for Cohere's rerank API
type CohereRerankRequest struct {
	Model           string   `json:"model"`
	Query           string   `json:"query"`
	Documents       []string `json:"documents"`
	TopN            *int     `json:"top_n,omitempty"`
	MaxTokensPerDoc *int     `json:"max_tokens_per_doc,omitempty"`
}

// CohereRerankResponse represents the response structure from Cohere's rerank API
type CohereRerankResponse struct {
	Results []struct {
		Index          int     `json:"index"`
		RelevanceScore float64 `json:"relevance_score"`
	} `json:"results"`
	ID   string `json:"id"`
	Meta struct {
		APIVersion struct {
			Version        string `json:"version"`
			IsExperimental bool   `json:"is_experimental"`
		} `json:"api_version"`
		BilledUnits struct {
			SearchUnits int `json:"search_units"`
		} `json:"billed_units"`
	} `json:"meta"`
}

// NewCohereReranker creates a new Reranker that uses Cohere's rerank API
func NewCohereReranker(apiKey, model string, options ...func(*CohereReranker)) *CohereReranker {
	reranker := &CohereReranker{
		apiKey:  apiKey,
		baseURL: "https://api.cohere.com/v2/rerank",
		model:   model,
	}

	// Apply options
	for _, option := range options {
		option(reranker)
	}

	return reranker
}

// WithCohereRerankBaseURL sets a custom base URL for the Cohere rerank API
func WithCohereRerankBaseURL(baseURL string) func(*CohereReranker) {
	return func(r *CohereReranker) {
		r.baseURL = baseURL
	}
}

// Rerank implements the Reranker interface
func (r *CohereReranker) Rerank(ctx context.Context, query string, documents []string, options ...RerankOption) ([]RankResult, error) {
	if len(documents) == 0 {
		return []RankResult{}, nil
	}

	// Parse options
	opts := &rerankOptions{}
	for _, option := range options {
		option(opts)
	}

	// Prepare the request
	request := CohereRerankRequest{
		Model:           r.model,
		Query:           query,
		Documents:       documents,
		TopN:            opts.topN,
		MaxTokensPerDoc: opts.maxTokensPerDoc,
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
		r.baseURL,
		bytes.NewBuffer(requestBody),
	)
	if err != nil {
		return nil, fmt.Errorf("error creating HTTP request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+r.apiKey)
	httpReq.Header.Set("X-Client-Name", "go-go-golems/geppetto")

	// Send the request
	httpClient := &http.Client{}
	resp, err := httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("error sending request to Cohere Rerank API: %w", err)
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
			return nil, fmt.Errorf("cohere rerank API error (status %d): %v", resp.StatusCode, errorResponse)
		}
		return nil, fmt.Errorf("cohere rerank API error (status %d)", resp.StatusCode)
	}

	// Decode the response
	var response CohereRerankResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	// Convert to RankResults
	results := make([]RankResult, len(response.Results))
	for i, result := range response.Results {
		results[i] = RankResult{
			Index:    result.Index,
			Document: documents[result.Index],
			Score:    result.RelevanceScore,
		}
	}

	return results, nil
}

// GetModel implements the Reranker interface
func (r *CohereReranker) GetModel() RerankerModel {
	return RerankerModel{
		Name: r.model,
	}
}

// Ensure CohereReranker implements Reranker interface
var _ Reranker = &CohereReranker{}

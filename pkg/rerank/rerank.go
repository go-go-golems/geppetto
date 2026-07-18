package rerank

import "context"

// Document is a single retrieval candidate supplied to a cross-encoder
// reranker. ID is the durable, caller-controlled identity; Text is the exact
// text submitted to the provider.
//
// Application metadata and first-stage retrieval scores remain outside the
// provider request. This keeps the interface reusable and reduces accidental
// disclosure of application context to providers.
type Document struct {
	ID   string `json:"id" yaml:"id"`
	Text string `json:"text" yaml:"text"`
}

// Request is the transport-neutral rerank request.
//
// Model is explicit even when a provider instance has a configured model.
// Validation requires it to either be empty (filled from the provider default)
// or exactly equal the configured provider model. This prevents silent model
// drift between the caller's expectation and the provider's binding.
//
// TopN controls response cardinality. Some providers return only the
// highest-scoring TopN documents. Callers that require one score per submitted
// document must set TopN == len(Documents). The package does not silently
// default TopN: explicit cardinality lets complete-score callers prove they
// requested all candidates.
type Request struct {
	Model     string     `json:"model,omitempty" yaml:"model,omitempty"`
	Query     string     `json:"query" yaml:"query"`
	Documents []Document `json:"documents" yaml:"documents"`
	TopN      int        `json:"top_n" yaml:"top_n"`
}

// Result is one scored document mapped back to the caller's durable identity.
//
// DocumentID is the caller-supplied Document.ID. Index is the zero-based
// provider array position of the submitted document. Score is the provider's
// raw relevance score (may be negative). Rank is assigned from 1 after
// deterministic sorting.
type Result struct {
	DocumentID string  `json:"document_id" yaml:"document_id"`
	Index      int     `json:"index" yaml:"index"`
	Score      float64 `json:"score" yaml:"score"`
	Rank       int     `json:"rank" yaml:"rank"`
}

// Usage reports provider-reported token consumption for a rerank call.
//
// Reranking usually consumes input tokens but produces no generated output
// tokens. Provider responses may report prompt or total tokens. A zero
// Usage value means the provider reported zero tokens; a nil *Usage means the
// provider did not report usage at all.
type Usage struct {
	InputTokens int `json:"input_tokens,omitempty" yaml:"input_tokens,omitempty"`
	TotalTokens int `json:"total_tokens,omitempty" yaml:"total_tokens,omitempty"`
}

// Response is the rich rerank response carrying scores plus provider
// observations needed by downstream scientific runs.
//
// Provider and Model identify what actually answered. Results are sorted and
// ranked. Cost is nil when pricing is unknown and a pointer to zero when the
// provider is explicitly free/local under the selected pricing policy. nil and
// zero cost are intentionally distinguishable.
type Response struct {
	Provider   string   `json:"provider" yaml:"provider"`
	Model      string   `json:"model" yaml:"model"`
	Results    []Result `json:"results" yaml:"results"`
	Usage      *Usage   `json:"usage,omitempty" yaml:"usage,omitempty"`
	Cost       *float64 `json:"cost,omitempty" yaml:"cost,omitempty"`
	RequestID  string   `json:"request_id,omitempty" yaml:"request_id,omitempty"`
	DurationMs *int64   `json:"duration_ms,omitempty" yaml:"duration_ms,omitempty"`
}

// Model identifies a provider instance's configured model. It is returned by
// Provider.Model so callers can record the exact provider/model identity that
// answered a request.
type Model struct {
	Provider string `json:"provider" yaml:"provider"`
	Name     string `json:"name" yaml:"name"`
}

// Provider is the transport-neutral rerank provider interface.
//
// Implementations must:
//
//   - validate the request against the provider's configured model and limits;
//   - map provider array indices back to caller document IDs;
//   - validate and deterministically order the response;
//   - preserve provider usage, model, cost, duration, and request ID where
//     available, leaving unavailable values nil rather than fabricating zero.
type Provider interface {
	// Rerank scores and reorders documents for a single query.
	Rerank(ctx context.Context, in Request) (Response, error)
	// Model returns the provider's configured provider/model identity.
	Model() Model
}

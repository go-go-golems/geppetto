// Package llamacpp implements a strict llama.cpp /v1/rerank adapter for the
// transport-neutral rerank.Provider interface.
//
// The adapter enforces:
//   - bounded request encoding (MaxRequestBytes) before sending;
//   - bounded response reading (MaxResponseBytes) before decoding;
//   - strict JSON decoding that rejects trailing data;
//   - outbound URL policy via security.ValidateOutboundURL (scheme, host,
//     userinfo, local-network opt-in);
//   - redirect rejection (local model endpoints should not redirect);
//   - context cancellation propagation;
//   - safe errors that never include query/document text, credentials, or
//     response bodies.
//
// Caller document IDs never enter the provider payload; the adapter retains a
// local index-to-ID table and maps results back to caller identity.
package llamacpp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-go-golems/geppetto/pkg/rerank"
	"github.com/go-go-golems/geppetto/pkg/rerank/llamacpp/internal/transport"
	"github.com/go-go-golems/geppetto/pkg/security"
)

const (
	// ProviderName is the provider identity reported in rerank.Response.
	ProviderName = "llama.cpp"

	// DefaultMaxRequestBytes is the conservative default request body bound.
	DefaultMaxRequestBytes int64 = 2 << 20 // 2 MiB
	// DefaultMaxResponseBytes is the conservative default response body bound.
	DefaultMaxResponseBytes int64 = 1 << 20 // 1 MiB

	// rerankPath is the llama.cpp reranking route appended to the base URL.
	rerankPath = "v1/rerank"
)

// Options configures the llama.cpp rerank provider.
//
// BaseURL and Model are required and have no defaults; a generic library must
// not silently point at localhost. HTTPClient is optional and, when injected,
// is cloned (not mutated) so the caller's CheckRedirect and transport remain
// intact. OutboundURL controls scheme and local-network policy; HTTP and local
// networks are denied by default. MaxRequestBytes and MaxResponseBytes bound
// memory and transport. CostPerMTokens is optional; when nil, the response
// Cost is nil (unknown) rather than zero.
type Options struct {
	BaseURL          string
	Model            string
	HTTPClient       *http.Client
	OutboundURL      security.OutboundURLOptions
	MaxRequestBytes  int64
	MaxResponseBytes int64
	CostPerMTokens   *float64
}

// Provider is the llama.cpp rerank provider.
type Provider struct {
	baseURL          string
	endpoint         string
	model            string
	client           *http.Client
	outboundURL      security.OutboundURLOptions
	maxRequestBytes  int64
	maxResponseBytes int64
	costPerMTokens   *float64
}

var _ rerank.Provider = (*Provider)(nil)

// New constructs a llama.cpp rerank provider.
//
// It validates BaseURL (scheme, host, no userinfo, no query/fragment),
// requires Model, bounds MaxRequestBytes/MaxResponseBytes to positive defaults
// when zero, and clones an injected HTTPClient so the caller's redirect policy
// is replaced with the adapter's redirect-rejection policy without mutating
// the caller's client. The final endpoint is built with url.JoinPath and
// re-validated under the outbound URL policy.
func New(options Options) (*Provider, error) {
	baseURL, err := transport.ParseAndValidateBaseURL("llamacpp", options.BaseURL)
	if err != nil {
		return nil, err
	}

	model := strings.TrimSpace(options.Model)
	if model == "" {
		return nil, fmt.Errorf("llamacpp model is required: %w", rerank.ErrInvalidRequest)
	}

	maxRequestBytes := options.MaxRequestBytes
	if maxRequestBytes == 0 {
		maxRequestBytes = DefaultMaxRequestBytes
	}
	if maxRequestBytes < 1 {
		return nil, fmt.Errorf("llamacpp max_request_bytes must be positive: %w", rerank.ErrInvalidRequest)
	}
	maxResponseBytes := options.MaxResponseBytes
	if maxResponseBytes == 0 {
		maxResponseBytes = DefaultMaxResponseBytes
	}
	if maxResponseBytes < 1 {
		return nil, fmt.Errorf("llamacpp max_response_bytes must be positive: %w", rerank.ErrInvalidRequest)
	}

	endpoint, err := transport.Endpoint("llamacpp", baseURL, rerankPath, options.OutboundURL)
	if err != nil {
		return nil, err
	}

	client := transport.CloneClientWithRedirectRejection(options.HTTPClient)

	return &Provider{
		baseURL:          baseURL,
		endpoint:         endpoint,
		model:            model,
		client:           client,
		outboundURL:      options.OutboundURL,
		maxRequestBytes:  maxRequestBytes,
		maxResponseBytes: maxResponseBytes,
		costPerMTokens:   options.CostPerMTokens,
	}, nil
}

// Model returns the provider's configured provider/model identity.
func (p *Provider) Model() rerank.Model {
	return rerank.Model{Provider: ProviderName, Name: p.model}
}

// Rerank scores and reorders documents via the llama.cpp /v1/rerank endpoint.
func (p *Provider) Rerank(ctx context.Context, in rerank.Request) (rerank.Response, error) {
	started := time.Now()

	providerModel := p.Model()
	if err := rerank.ValidateRequest(in, providerModel); err != nil {
		return rerank.Response{}, err
	}

	effectiveModel := rerank.ResolveModel(in, providerModel)
	documents := make([]string, len(in.Documents))
	for i, doc := range in.Documents {
		documents[i] = doc.Text
	}

	payload, err := json.Marshal(request{
		Model:     effectiveModel,
		Query:     in.Query,
		Documents: documents,
		TopN:      in.TopN,
	})
	if err != nil {
		return rerank.Response{}, fmt.Errorf("llamacpp encode request: %w", err)
	}
	if int64(len(payload)) > p.maxRequestBytes {
		return rerank.Response{}, fmt.Errorf("llamacpp encoded request is %d bytes, limit is %d: %w",
			len(payload), p.maxRequestBytes, rerank.ErrRequestTooLarge)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.endpoint, bytes.NewReader(payload))
	if err != nil {
		return rerank.Response{}, fmt.Errorf("llamacpp could not create provider request: %w", rerank.ErrUnavailable)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	httpResp, err := p.client.Do(httpReq)
	if err != nil {
		return rerank.Response{}, transport.RedactedTransportError("llamacpp")
	}
	defer func() { _ = httpResp.Body.Close() }()

	if httpResp.StatusCode < http.StatusOK || httpResp.StatusCode >= http.StatusMultipleChoices {
		transport.DrainBounded(httpResp.Body, p.maxResponseBytes)
		return rerank.Response{}, fmt.Errorf("llamacpp endpoint returned status %d: %w",
			httpResp.StatusCode, rerank.ErrUnavailable)
	}

	raw, tooLarge, err := transport.ReadAtMost(httpResp.Body, p.maxResponseBytes)
	if err != nil {
		return rerank.Response{}, fmt.Errorf("llamacpp could not read provider response: %w", rerank.ErrUnavailable)
	}
	if tooLarge {
		return rerank.Response{}, fmt.Errorf("llamacpp response body exceeds %d bytes: %w",
			p.maxResponseBytes, rerank.ErrResponseTooLarge)
	}

	wire, err := decodeStrict(raw)
	if err != nil {
		return rerank.Response{}, fmt.Errorf("llamacpp decode response: %w: %w", err, rerank.ErrInvalidResponse)
	}

	results, err := rerank.ValidateAndMapResults(in.Documents, in.TopN, toRawResults(wire.Results))
	if err != nil {
		return rerank.Response{}, err
	}

	// Model mismatch: when the response declares a model, it must match the
	// effective request/provider model.
	if wire.Model != "" && wire.Model != effectiveModel {
		return rerank.Response{}, fmt.Errorf("llamacpp response model %q does not match effective model %q: %w",
			wire.Model, effectiveModel, rerank.ErrInvalidResponse)
	}

	usage := mapUsage(wire.Usage)
	durationMs := time.Since(started).Milliseconds()

	return rerank.Response{
		Provider:   ProviderName,
		Model:      effectiveModel,
		Results:    results,
		Usage:      usage,
		Cost:       computeInputCost(usage, p.costPerMTokens),
		RequestID:  httpResp.Header.Get("X-Request-Id"),
		DurationMs: &durationMs,
	}, nil
}

// decodeStrict decodes exactly one JSON value. It rejects unknown fields and
// every non-whitespace byte after that value without returning provider body
// content in an error.
func decodeStrict(raw []byte) (*response, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	var wire response
	if err := dec.Decode(&wire); err != nil {
		return nil, fmt.Errorf("invalid JSON response")
	}
	var extra any
	if err := dec.Decode(&extra); !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("trailing data after rerank response")
	}
	return &wire, nil
}

// toRawResults converts wire items (with pointer fields) into rerank.RawResult
// presence-flagged values.
func toRawResults(items []item) []rerank.RawResult {
	out := make([]rerank.RawResult, 0, len(items))
	for _, it := range items {
		r := rerank.RawResult{}
		if it.Index != nil {
			r.Index = *it.Index
			r.HasIndex = true
		}
		if it.RelevanceScore != nil {
			r.Score = *it.RelevanceScore
			r.HasScore = true
		}
		out = append(out, r)
	}
	return out
}

// mapUsage converts the wire usage into the rerank Usage. Returns nil when the
// provider did not report usage.
func mapUsage(u *usage) *rerank.Usage {
	if u == nil {
		return nil
	}
	return &rerank.Usage{
		InputTokens: u.PromptTokens,
		TotalTokens: u.TotalTokens,
	}
}

// computeInputCost computes the input-token cost when a per-million-token rate is
// configured and the provider reported usage. Returns nil (unknown cost) when
// either is absent, preserving the nil-vs-zero distinction.
func computeInputCost(u *rerank.Usage, costPerMTokens *float64) *float64 {
	if u == nil || costPerMTokens == nil {
		return nil
	}
	tokens := u.InputTokens
	if tokens == 0 {
		tokens = u.TotalTokens
	}
	cost := *costPerMTokens * float64(tokens) / 1_000_000
	return &cost
}

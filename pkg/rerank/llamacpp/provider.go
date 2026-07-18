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
	"net/url"
	"strings"
	"time"

	"github.com/go-go-golems/geppetto/pkg/rerank"
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
	baseURL := strings.TrimSpace(options.BaseURL)
	if baseURL == "" {
		return nil, fmt.Errorf("llamacpp base URL is required: %w", rerank.ErrInvalidRequest)
	}
	parsed, err := url.Parse(baseURL)
	if err != nil {
		// url.Parse errors include the original URL. Never wrap them: a malformed
		// URL can contain endpoint credentials or private topology.
		return nil, fmt.Errorf("llamacpp base URL is malformed: %w", rerank.ErrInvalidRequest)
	}
	if err := validateBaseURL(parsed); err != nil {
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

	endpoint, err := url.JoinPath(baseURL, rerankPath)
	if err != nil {
		return nil, fmt.Errorf("llamacpp endpoint construction failed: %w: %w", err, rerank.ErrInvalidRequest)
	}
	if err := security.ValidateOutboundURL(endpoint, options.OutboundURL); err != nil {
		return nil, fmt.Errorf("llamacpp endpoint rejected by outbound URL policy: %w: %w", err, rerank.ErrInvalidRequest)
	}

	client := cloneClientWithRedirectRejection(options.HTTPClient)

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

// validateBaseURL permits an optional unambiguous path prefix, but rejects
// encoded paths, repeated separators, and dot segments. url.JoinPath would
// otherwise normalize those forms after validation, making the configured
// target ambiguous to reviewers and security policy.
func validateBaseURL(parsed *url.URL) error {
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("llamacpp base URL scheme must be http or https: %w", rerank.ErrInvalidRequest)
	}
	if parsed.Host == "" {
		return fmt.Errorf("llamacpp base URL host is required: %w", rerank.ErrInvalidRequest)
	}
	if parsed.User != nil {
		return fmt.Errorf("llamacpp base URL must not contain userinfo: %w", rerank.ErrInvalidRequest)
	}
	if parsed.RawQuery != "" || parsed.Fragment != "" {
		return fmt.Errorf("llamacpp base URL must not contain query or fragment: %w", rerank.ErrInvalidRequest)
	}
	if parsed.RawPath != "" || strings.Contains(parsed.Path, "//") {
		return fmt.Errorf("llamacpp base URL path must be unambiguous: %w", rerank.ErrInvalidRequest)
	}
	for _, segment := range strings.Split(parsed.Path, "/") {
		if segment == "." || segment == ".." {
			return fmt.Errorf("llamacpp base URL path must not contain dot segments: %w", rerank.ErrInvalidRequest)
		}
	}
	return nil
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
		return rerank.Response{}, redactTransportError(err)
	}
	defer func() { _ = httpResp.Body.Close() }()

	if httpResp.StatusCode < http.StatusOK || httpResp.StatusCode >= http.StatusMultipleChoices {
		drainBounded(httpResp.Body, p.maxResponseBytes)
		return rerank.Response{}, fmt.Errorf("llamacpp endpoint returned status %d: %w",
			httpResp.StatusCode, rerank.ErrUnavailable)
	}

	raw, tooLarge, err := readAtMost(httpResp.Body, p.maxResponseBytes)
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

// redactTransportError intentionally discards the original transport error.
// net/url errors can contain a redirect target, proxy URL, userinfo, or query
// parameters. The stable sentinel is sufficient for callers to classify the
// failure without serializing protected operational data.
func redactTransportError(_ error) error {
	return fmt.Errorf("llamacpp provider transport failed: %w", rerank.ErrUnavailable)
}

// drainBounded reads and discards a non-2xx body up to the limit so the
// connection can be reused, without ever surfacing the body in an error.
func drainBounded(body io.Reader, limit int64) {
	_, _, _ = readAtMost(body, limit)
}

// readAtMost reads up to limit+1 bytes from r. tooLarge is true only when the
// body exceeded the limit; a read error remains distinguishable from a limit
// violation and is classified by the caller as provider unavailability.
func readAtMost(r io.Reader, limit int64) ([]byte, bool, error) {
	lr := &io.LimitedReader{R: r, N: limit + 1}
	body, err := io.ReadAll(lr)
	if err != nil {
		return nil, false, err
	}
	return body, int64(len(body)) > limit, nil
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

// cloneClientWithRedirectRejection returns an http.Client that rejects every
// redirect. If options.HTTPClient is nil, a new client is constructed. When
// injected, the client is shallow-copied so its Transport, Jar, and Timeout
// are retained while CheckRedirect is replaced. The caller's client is never
// mutated in place.
func cloneClientWithRedirectRejection(injected *http.Client) *http.Client {
	rejectRedirect := func(_ *http.Request, _ []*http.Request) error {
		return fmt.Errorf("rerank provider rejects redirects")
	}
	if injected == nil {
		return &http.Client{
			CheckRedirect: rejectRedirect,
			Timeout:       0,
		}
	}
	cloned := *injected
	cloned.CheckRedirect = rejectRedirect
	return &cloned
}

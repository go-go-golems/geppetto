package cohere

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	providerdefaults "github.com/go-go-golems/geppetto/pkg/steps/ai/providers/defaults"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-go-golems/geppetto/pkg/rerank"
	"github.com/go-go-golems/geppetto/pkg/security"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/connection"
)

const (
	// ProviderName is the provider identity reported in rerank.Response.
	ProviderName = "cohere"

	// DefaultBaseURL is the canonical hosted Cohere API base URL. Unlike the
	// llama.cpp adapter, a hosted provider has exactly one canonical endpoint,
	// so a default is unambiguous; the BaseURL option remains as an override
	// for proxies and tests (GEPPETTO-RERANKER-002 decision record DR-5).
	DefaultBaseURL = providerdefaults.CohereBaseURL

	// DefaultMaxRequestBytes is the conservative default request body bound.
	DefaultMaxRequestBytes int64 = 2 << 20 // 2 MiB
	// DefaultMaxResponseBytes is the conservative default response body bound.
	DefaultMaxResponseBytes int64 = 1 << 20 // 1 MiB

	// rerankPath is the Cohere reranking route appended to the base URL.
	rerankPath = "v2/rerank"

	// clientName identifies this client to Cohere for support diagnostics.
	clientName = "go-go-golems/geppetto"
)

// Options configures the Cohere rerank provider.
//
// APIKey and Model are required. BaseURL is optional and defaults to
// DefaultBaseURL. HTTPClient is optional and, when injected, is cloned (not
// mutated) so the caller's CheckRedirect and transport remain intact.
// OutboundURL controls scheme and local-network policy; HTTP and local
// networks are denied by default, which matches the hosted endpoint.
// MaxRequestBytes and MaxResponseBytes bound memory and transport.
// CostPerSearch is optional; when nil or when the response reports no billed
// units, the response Cost is nil (unknown) rather than zero.
type Options struct {
	Connection       *connection.Connection
	APIKey           string
	BaseURL          string
	Model            string
	HTTPClient       *http.Client
	OutboundURL      security.OutboundURLOptions
	MaxRequestBytes  int64
	MaxResponseBytes int64
	CostPerSearch    *float64
}

// Provider is the Cohere rerank provider.
type Provider struct {
	baseURL          string
	endpoint         string
	connection       *connection.Connection
	model            string
	maxRequestBytes  int64
	maxResponseBytes int64
	costPerSearch    *float64
}

var _ rerank.Provider = (*Provider)(nil)

// New constructs a Cohere rerank provider.
//
// It requires APIKey and Model, defaults BaseURL to the canonical hosted
// endpoint, applies the same base-URL hygiene as the llama.cpp adapter
// (scheme, host, no userinfo, no query/fragment, no dot segments), bounds
// MaxRequestBytes/MaxResponseBytes to positive defaults when zero, and clones
// an injected HTTPClient so the caller's redirect policy is replaced with the
// adapter's redirect-rejection policy without mutating the caller's client.
// The final endpoint is built with url.JoinPath and re-validated under the
// outbound URL policy.
func New(options Options) (*Provider, error) {
	c := options.Connection
	if c != nil && (options.APIKey != "" || options.BaseURL != "" || options.HTTPClient != nil) {
		return nil, fmt.Errorf("connection already owns endpoint, authentication and client: %w", rerank.ErrInvalidRequest)
	}
	apiKey := strings.TrimSpace(options.APIKey)
	if apiKey == "" && c == nil {
		return nil, fmt.Errorf("cohere api key is required: %w", rerank.ErrInvalidRequest)
	}

	baseURL := strings.TrimSpace(options.BaseURL)
	if c != nil {
		baseURL = c.BaseURL()
	}
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	parsed, err := url.Parse(baseURL)
	if err != nil {
		// url.Parse errors include the original URL. Never wrap them: a malformed
		// URL can contain endpoint credentials or private topology.
		return nil, fmt.Errorf("cohere base URL is malformed: %w", rerank.ErrInvalidRequest)
	}
	if err := validateBaseURL(parsed); err != nil {
		return nil, err
	}

	model := strings.TrimSpace(options.Model)
	if model == "" {
		return nil, fmt.Errorf("cohere model is required: %w", rerank.ErrInvalidRequest)
	}

	maxRequestBytes := options.MaxRequestBytes
	if maxRequestBytes == 0 {
		maxRequestBytes = DefaultMaxRequestBytes
	}
	if maxRequestBytes < 1 {
		return nil, fmt.Errorf("cohere max_request_bytes must be positive: %w", rerank.ErrInvalidRequest)
	}
	maxResponseBytes := options.MaxResponseBytes
	if maxResponseBytes == 0 {
		maxResponseBytes = DefaultMaxResponseBytes
	}
	if maxResponseBytes < 1 {
		return nil, fmt.Errorf("cohere max_response_bytes must be positive: %w", rerank.ErrInvalidRequest)
	}

	endpoint, err := url.JoinPath(baseURL, rerankPath)
	if err != nil {
		return nil, fmt.Errorf("cohere endpoint construction failed: %w: %w", err, rerank.ErrInvalidRequest)
	}
	if c == nil {
		if err := security.ValidateOutboundURL(endpoint, options.OutboundURL); err != nil {
			return nil, fmt.Errorf("cohere endpoint rejected by outbound URL policy: %w: %w", err, rerank.ErrInvalidRequest)
		}
	}

	if c == nil {
		c, err = connection.New(connection.Config{Provider: ProviderName, BaseURL: baseURL, Routes: Routes(), Auth: connection.StaticBearer(apiKey), HTTPClient: options.HTTPClient, OutboundURL: options.OutboundURL, RejectRedirects: true})
		if err != nil {
			return nil, fmt.Errorf("cohere connection: %w: %w", err, rerank.ErrInvalidRequest)
		}
	}
	if _, err := c.Resolve("rerank"); err != nil {
		return nil, err
	}

	return &Provider{
		baseURL:          baseURL,
		endpoint:         endpoint,
		connection:       c,
		model:            model,
		maxRequestBytes:  maxRequestBytes,
		maxResponseBytes: maxResponseBytes,
		costPerSearch:    options.CostPerSearch,
	}, nil
}

// validateBaseURL permits an optional unambiguous path prefix, but rejects
// encoded paths, repeated separators, and dot segments. url.JoinPath would
// otherwise normalize those forms after validation, making the configured
// target ambiguous to reviewers and security policy.
func validateBaseURL(parsed *url.URL) error {
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("cohere base URL scheme must be http or https: %w", rerank.ErrInvalidRequest)
	}
	if parsed.Host == "" {
		return fmt.Errorf("cohere base URL host is required: %w", rerank.ErrInvalidRequest)
	}
	if parsed.User != nil {
		return fmt.Errorf("cohere base URL must not contain userinfo: %w", rerank.ErrInvalidRequest)
	}
	if parsed.RawQuery != "" || parsed.Fragment != "" {
		return fmt.Errorf("cohere base URL must not contain query or fragment: %w", rerank.ErrInvalidRequest)
	}
	if parsed.RawPath != "" || strings.Contains(parsed.Path, "//") {
		return fmt.Errorf("cohere base URL path must be unambiguous: %w", rerank.ErrInvalidRequest)
	}
	for _, segment := range strings.Split(parsed.Path, "/") {
		if segment == "." || segment == ".." {
			return fmt.Errorf("cohere base URL path must not contain dot segments: %w", rerank.ErrInvalidRequest)
		}
	}
	return nil
}

// Model returns the provider's configured provider/model identity.
func (p *Provider) Model() rerank.Model {
	return rerank.Model{Provider: ProviderName, Name: p.model}
}

// Rerank scores and reorders documents via the Cohere v2 /rerank endpoint.
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
		return rerank.Response{}, fmt.Errorf("cohere encode request: %w", err)
	}
	if int64(len(payload)) > p.maxRequestBytes {
		return rerank.Response{}, fmt.Errorf("cohere encoded request is %d bytes, limit is %d: %w",
			len(payload), p.maxRequestBytes, rerank.ErrRequestTooLarge)
	}

	httpReq, err := p.connection.NewRequest(ctx, http.MethodPost, "rerank", bytes.NewReader(payload))
	if err != nil {
		return rerank.Response{}, fmt.Errorf("cohere could not create provider request: %w", rerank.ErrUnavailable)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	httpReq.Header.Set("X-Client-Name", clientName)

	httpResp, err := p.connection.Do(httpReq)
	if err != nil {
		return rerank.Response{}, redactTransportError(err)
	}
	defer func() { _ = httpResp.Body.Close() }()

	if httpResp.StatusCode < http.StatusOK || httpResp.StatusCode >= http.StatusMultipleChoices {
		drainBounded(httpResp.Body, p.maxResponseBytes)
		return rerank.Response{}, fmt.Errorf("cohere endpoint returned status %d: %w",
			httpResp.StatusCode, rerank.ErrUnavailable)
	}

	raw, tooLarge, err := readAtMost(httpResp.Body, p.maxResponseBytes)
	if err != nil {
		return rerank.Response{}, fmt.Errorf("cohere could not read provider response: %w", rerank.ErrUnavailable)
	}
	if tooLarge {
		return rerank.Response{}, fmt.Errorf("cohere response body exceeds %d bytes: %w",
			p.maxResponseBytes, rerank.ErrResponseTooLarge)
	}

	wire, err := decodeStrict(raw)
	if err != nil {
		return rerank.Response{}, fmt.Errorf("cohere decode response: %w: %w", err, rerank.ErrInvalidResponse)
	}

	results, err := rerank.ValidateAndMapResults(in.Documents, in.TopN, toRawResults(wire.Results))
	if err != nil {
		return rerank.Response{}, err
	}

	durationMs := time.Since(started).Milliseconds()

	return rerank.Response{
		Provider: ProviderName,
		Model:    effectiveModel,
		Results:  results,
		// Cohere bills rerank in search units, not tokens; Usage stays nil to
		// preserve the token semantics of the core Usage struct (DR-3).
		Usage:      nil,
		Cost:       computeSearchCost(wire.Meta, p.costPerSearch),
		RequestID:  wire.ID,
		DurationMs: &durationMs,
	}, nil
}

// redactTransportError intentionally discards the original transport error.
// net/url errors can contain a redirect target, proxy URL, userinfo, or query
// parameters. The stable sentinel is sufficient for callers to classify the
// failure without serializing protected operational data.
func redactTransportError(_ error) error {
	return fmt.Errorf("cohere provider transport failed: %w", rerank.ErrUnavailable)
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

// computeSearchCost computes the call cost when a per-search rate is
// configured and the provider reported billed search units. Returns nil
// (unknown cost) when either is absent, preserving the nil-vs-zero
// distinction (DR-3).
func computeSearchCost(m *meta, costPerSearch *float64) *float64 {
	if m == nil || m.BilledUnits == nil || costPerSearch == nil {
		return nil
	}
	cost := *costPerSearch * float64(m.BilledUnits.SearchUnits)
	return &cost
}

// Routes declares the hosted reranking operation.
func Routes() connection.Routes { return connection.Routes{"rerank": connection.Relative(rerankPath)} }

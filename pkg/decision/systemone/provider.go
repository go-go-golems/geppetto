// Package systemone is the decision.Provider for LunaRoute's System One
// endpoint (POST {base}/systemone), which serves decision models such as djev.
package systemone

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/go-go-golems/geppetto/pkg/decision"
	"github.com/go-go-golems/geppetto/pkg/security"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/credentials"
)

const (
	// ProviderName is reported in decision.Model and decision.Response.
	ProviderName = "systemone"
	// DefaultBaseURL is LunaRoute's API root (the OpenAI-style base URL Pi
	// stores); the endpoint is DefaultBaseURL + "/systemone".
	DefaultBaseURL = "https://gw.lunaroute.com/v1"
	// DefaultMaxRequestBytes matches the gateway's 8 MiB route cap.
	DefaultMaxRequestBytes int64 = 8 << 20
	// DefaultMaxResponseBytes bounds the response body.
	DefaultMaxResponseBytes int64 = 1 << 20

	endpointPath = "systemone"
)

// NormalizeBaseURL is the normalization New applies to Options.BaseURL
// (surrounding space and trailing slashes removed). An empty result means
// New will use DefaultBaseURL, so callers that decide whether the default is
// acceptable must check the normalized value, not the raw one.
func NormalizeBaseURL(s string) string {
	return strings.TrimRight(strings.TrimSpace(s), "/")
}

// Options configure a Provider. Exactly one of APIKey and TokenSource is
// required.
type Options struct {
	BaseURL     string
	APIKey      string
	TokenSource credentials.BearerTokenSource
	Model       string
	HTTPClient  *http.Client
	OutboundURL security.OutboundURLOptions
	// MaxRequestBytes and MaxResponseBytes default when 0.
	MaxRequestBytes  int64
	MaxResponseBytes int64
	// CostPerMInputTokens prices a call (System One bills input tokens only);
	// nil leaves Response.Cost unknown.
	CostPerMInputTokens *float64
}

// Provider calls the System One endpoint.
type Provider struct {
	baseURL          string
	endpoint         string
	tokens           credentials.BearerTokenSource
	model            string
	client           *http.Client
	maxRequestBytes  int64
	maxResponseBytes int64
	costPerMInput    *float64
}

var _ decision.Provider = (*Provider)(nil)

// New validates options and builds a Provider.
func New(o Options) (*Provider, error) {
	invalid := func(msg string) error { return fmt.Errorf("systemone %s: %w", msg, decision.ErrInvalidRequest) }
	model := strings.TrimSpace(o.Model)
	if model == "" {
		return nil, invalid("model is required")
	}
	tokens := o.TokenSource
	switch key := strings.TrimSpace(o.APIKey); {
	case key != "" && tokens != nil:
		return nil, invalid("takes an API key or a token source, not both")
	case key != "":
		src, err := credentials.NewStaticBearerTokenSource(key)
		if err != nil {
			return nil, invalid("api key is empty")
		}
		tokens = src
	case tokens == nil:
		return nil, invalid("api key or token source is required")
	}
	baseURL := NormalizeBaseURL(o.BaseURL)
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return nil, invalid("base URL is malformed")
	}
	if err := validateBaseURL(parsed); err != nil {
		return nil, err
	}
	endpoint, err := url.JoinPath(baseURL, endpointPath)
	if err != nil {
		return nil, invalid("endpoint construction failed")
	}
	if err := security.ValidateOutboundURL(endpoint, o.OutboundURL); err != nil {
		return nil, fmt.Errorf("systemone endpoint rejected by outbound URL policy: %w: %w", err, decision.ErrInvalidRequest)
	}
	maxReq, maxResp := o.MaxRequestBytes, o.MaxResponseBytes
	if maxReq == 0 {
		maxReq = DefaultMaxRequestBytes
	}
	if maxResp == 0 {
		maxResp = DefaultMaxResponseBytes
	}
	if maxReq < 1 || maxResp < 1 {
		return nil, invalid("byte limits must be positive")
	}
	return &Provider{
		baseURL:          baseURL,
		endpoint:         endpoint,
		tokens:           tokens,
		model:            model,
		client:           rejectRedirects(o.HTTPClient),
		maxRequestBytes:  maxReq,
		maxResponseBytes: maxResp,
		costPerMInput:    o.CostPerMInputTokens,
	}, nil
}

func validateBaseURL(u *url.URL) error {
	bad := func(msg string) error {
		return fmt.Errorf("systemone base URL %s: %w", msg, decision.ErrInvalidRequest)
	}
	switch {
	case u.Scheme != "http" && u.Scheme != "https":
		return bad("scheme must be http or https")
	case u.Host == "":
		return bad("host is required")
	case u.User != nil:
		return bad("must not contain userinfo")
	case u.RawQuery != "" || u.Fragment != "":
		return bad("must not contain a query or fragment")
	case u.RawPath != "" || strings.Contains(u.Path, "//"):
		return bad("path must be unambiguous")
	}
	for _, seg := range strings.Split(u.Path, "/") {
		if seg == "." || seg == ".." {
			return bad("path must not contain dot segments")
		}
	}
	return nil
}

// Model reports the configured model.
func (p *Provider) Model() decision.Model {
	return decision.Model{Provider: ProviderName, Name: p.model}
}

// Decide sends one request and returns validated answers.
func (p *Provider) Decide(ctx context.Context, in decision.Request) (decision.Response, error) {
	started := time.Now()
	if err := decision.ValidateRequest(in, p.Model()); err != nil {
		return decision.Response{}, err
	}
	model := decision.ResolveModel(in, p.Model())
	body, err := encodeRequest(model, in)
	if err != nil {
		return decision.Response{}, err
	}
	if int64(len(body)) > p.maxRequestBytes {
		return decision.Response{}, fmt.Errorf("systemone request is %d bytes, limit %d: %w", len(body), p.maxRequestBytes, decision.ErrRequestTooLarge)
	}
	token, err := p.tokens.BearerToken(ctx, credentials.Request{Provider: ProviderName, BaseURL: p.baseURL})
	if err != nil {
		if ctx.Err() != nil {
			return decision.Response{}, ctx.Err()
		}
		return decision.Response{}, fmt.Errorf("systemone credential unavailable: %w", decision.ErrNotGranted)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.endpoint, bytes.NewReader(body))
	if err != nil {
		return decision.Response{}, fmt.Errorf("systemone could not create the request: %w", decision.ErrUnavailable)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := p.client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return decision.Response{}, ctx.Err()
		}
		// The transport error can carry the URL; report a generic failure.
		return decision.Response{}, fmt.Errorf("systemone transport failed: %w", decision.ErrUnavailable)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, tooLarge, err := readAtMost(resp.Body, p.maxResponseBytes)
	if err != nil {
		if ctx.Err() != nil {
			return decision.Response{}, ctx.Err()
		}
		return decision.Response{}, fmt.Errorf("systemone could not read the response: %w", decision.ErrUnavailable)
	}
	requestID := resp.Header.Get("X-Request-Id")
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return decision.Response{}, providerError(resp, raw, requestID)
	}
	if tooLarge {
		return decision.Response{}, fmt.Errorf("systemone response exceeds %d bytes: %w", p.maxResponseBytes, decision.ErrResponseTooLarge)
	}
	var w wireResponse
	if err := decodeOne(raw, &w); err != nil {
		return decision.Response{}, fmt.Errorf("systemone response is not one JSON object: %w", decision.ErrInvalidResponse)
	}
	answers, err := mapAnswers(in, w.Answers)
	if err != nil {
		return decision.Response{}, err
	}
	if err := decision.ValidateResponse(in, answers); err != nil {
		return decision.Response{}, err
	}
	out := decision.Response{Provider: ProviderName, Model: model, Answers: answers, RequestID: requestID}
	if w.Model != "" {
		out.Model = w.Model
	}
	if w.Usage != nil {
		out.Usage = &decision.Usage{InputTokens: w.Usage.InputTokens, OutputTokens: w.Usage.OutputTokens}
		if p.costPerMInput != nil {
			c := float64(w.Usage.InputTokens) * *p.costPerMInput / 1e6
			out.Cost = &c
		}
	}
	ms := time.Since(started).Milliseconds()
	out.DurationMs = &ms
	return out, nil
}

// encodeRequest builds the wire body; questions are written in id order and
// a choice's options in slice order.
func encodeRequest(model string, in decision.Request) ([]byte, error) {
	qs := make(map[string]wireQuestion, len(in.Questions))
	for id, q := range in.Questions {
		wq := wireQuestion{Type: string(q.Type), Instructions: q.Instructions}
		switch q.Type {
		case decision.Noul:
			if q.True != "" || q.False != "" {
				c, err := json.Marshal(struct {
					True  string `json:"true,omitempty"`
					False string `json:"false,omitempty"`
				}{q.True, q.False})
				if err != nil {
					return nil, fmt.Errorf("systemone encode criteria: %w", decision.ErrInvalidRequest)
				}
				wq.Criteria = c
			}
		case decision.Choice:
			wq.Criteria = encodeOptions(q.Options)
		case decision.Score:
			c, err := json.Marshal(q.Levels)
			if err != nil {
				return nil, fmt.Errorf("systemone encode criteria: %w", decision.ErrInvalidRequest)
			}
			wq.Criteria = c
		}
		qs[id] = wq
	}
	b, err := json.Marshal(wireRequest{Model: model, State: in.State, Images: in.Images, Questions: qs})
	if err != nil {
		return nil, fmt.Errorf("systemone encode request: %w", decision.ErrInvalidRequest)
	}
	return b, nil
}

// encodeOptions writes {"name":"description",...} in slice order. Decision
// models are sensitive to option order; a Go map would sort the keys.
func encodeOptions(opts []decision.Option) json.RawMessage {
	var b bytes.Buffer
	b.WriteByte('{')
	for i, o := range opts {
		if i > 0 {
			b.WriteByte(',')
		}
		k, _ := json.Marshal(o.Name)
		v, _ := json.Marshal(o.Description)
		b.Write(k)
		b.WriteByte(':')
		b.Write(v)
	}
	b.WriteByte('}')
	return b.Bytes()
}

// mapAnswers converts wire answers. A field the answer's type requires but
// the wire omits is an invalid response, not a zero.
func mapAnswers(in decision.Request, wire map[string]wireAnswer) (map[string]decision.Answer, error) {
	bad := func(id, msg string) error {
		if _, asked := in.Questions[id]; !asked {
			return fmt.Errorf("systemone answer to a question that was not asked: %w", decision.ErrInvalidResponse)
		}
		return fmt.Errorf("systemone answer %q %s: %w", id, msg, decision.ErrInvalidResponse)
	}
	out := make(map[string]decision.Answer, len(wire))
	for _, id := range sortedKeys(wire) {
		w := wire[id]
		a := decision.Answer{Type: decision.Type(w.Type)}
		switch a.Type {
		case decision.Noul:
			if w.Noul == nil {
				return nil, bad(id, "has no noul probability")
			}
			a.Probability = *w.Noul
		case decision.Choice:
			if w.Choice == nil || w.Confidence == nil {
				return nil, bad(id, "lacks choice or confidence")
			}
			probs, err := optionProbabilities(w.Probabilities)
			if err != nil {
				return nil, bad(id, err.Error())
			}
			a.Choice, a.Confidence, a.Probabilities = *w.Choice, *w.Confidence, probs
		case decision.Score:
			if w.Score == nil || w.Confidence == nil {
				return nil, bad(id, "lacks score or confidence")
			}
			a.Score, a.Confidence = *w.Score, *w.Confidence
			q, asked := in.Questions[id]
			if asked && q.Type == decision.Score {
				levels, err := levelProbabilities(w.Probabilities, len(q.Levels))
				if err != nil {
					return nil, bad(id, err.Error())
				}
				a.LevelProbabilities = levels
			}
		}
		out[id] = a
	}
	return out, nil
}

// optionProbabilities copies a choice distribution. A JSON null value is
// rejected: encoding/json would otherwise decode it as 0, which is
// indistinguishable from the model assigning zero probability.
func optionProbabilities(m map[string]*float64) (map[string]float64, error) {
	if m == nil {
		return nil, nil
	}
	out := make(map[string]float64, len(m))
	for k, p := range m {
		if p == nil {
			return nil, fmt.Errorf("has a null option probability")
		}
		out[k] = *p
	}
	return out, nil
}

// levelProbabilities turns {"0": p0, "1": p1, ...} into a slice of n.
// Null values are rejected, as in optionProbabilities.
func levelProbabilities(m map[string]*float64, n int) ([]float64, error) {
	if len(m) != n {
		return nil, fmt.Errorf("has %d level probabilities for %d levels", len(m), n)
	}
	out := make([]float64, n)
	for k, p := range m {
		i, err := strconv.Atoi(k)
		if err != nil || i < 0 || i >= n || strconv.Itoa(i) != k {
			return nil, fmt.Errorf("has a probability for an undeclared level")
		}
		if p == nil {
			return nil, fmt.Errorf("has a null level probability")
		}
		out[i] = *p
	}
	return out, nil
}

func providerError(resp *http.Response, raw []byte, requestID string) error {
	var w wireError
	_ = json.Unmarshal(raw, &w)
	if requestID == "" {
		requestID = w.Error.RequestID
	}
	return decision.NewProviderError(ProviderName, resp.StatusCode, safeCode(w.Error.Code), w.Error.Message, requestID, retryAfter(resp.Header.Get("Retry-After")))
}

// safeCode keeps a provider error code only if it looks like one.
func safeCode(code string) string {
	if len(code) > 64 {
		return ""
	}
	for _, r := range code {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '_' {
			return ""
		}
	}
	return code
}

func retryAfter(v string) time.Duration {
	v = strings.TrimSpace(v)
	if v == "" {
		return 0
	}
	if s, err := strconv.Atoi(v); err == nil && s >= 0 {
		return time.Duration(s) * time.Second
	}
	if t, err := http.ParseTime(v); err == nil {
		if d := time.Until(t); d > 0 {
			return d.Round(time.Second)
		}
	}
	return 0
}

func readAtMost(r io.Reader, limit int64) ([]byte, bool, error) {
	b, err := io.ReadAll(&io.LimitedReader{R: r, N: limit + 1})
	if err != nil {
		return nil, false, err
	}
	if int64(len(b)) > limit {
		return b[:limit], true, nil
	}
	return b, false, nil
}

// decodeOne decodes exactly one JSON value; trailing data is an error.
func decodeOne(raw []byte, v any) error {
	dec := json.NewDecoder(bytes.NewReader(raw))
	if err := dec.Decode(v); err != nil {
		return err
	}
	var extra any
	if err := dec.Decode(&extra); !errors.Is(err, io.EOF) {
		return errors.New("trailing data")
	}
	return nil
}

func rejectRedirects(c *http.Client) *http.Client {
	reject := func(*http.Request, []*http.Request) error { return errors.New("systemone rejects redirects") }
	if c == nil {
		return &http.Client{CheckRedirect: reject, Timeout: 60 * time.Second}
	}
	cloned := *c
	cloned.CheckRedirect = reject
	return &cloned
}

func sortedKeys[V any](m map[string]V) []string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	return ks
}

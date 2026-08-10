package qwen3

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/go-go-golems/geppetto/pkg/rerank"
	"github.com/go-go-golems/geppetto/pkg/rerank/llamacpp/internal/transport"
	"github.com/go-go-golems/geppetto/pkg/security"
	"golang.org/x/sync/errgroup"
)

const (
	// ProviderName distinguishes completion/logprob scoring from llama.cpp's
	// standard /v1/rerank provider in durable experiment observations.
	ProviderName = "llama.cpp-qwen3-completion"

	DefaultMaxConcurrency         = 4
	DefaultMaxRequestBytes  int64 = 2 << 20 // 2 MiB per query/document pair
	DefaultMaxResponseBytes int64 = 1 << 20 // 1 MiB per pair

	completionPath = "completion"

	officialSystemPrompt = `Judge whether the Document meets the requirements based on the Query and the Instruct provided. Note that the answer can only be "yes" or "no".`
	officialInstruction  = "Given a web search query, retrieve relevant passages that answer the query"
)

// Options configures the llama.cpp Qwen3 completion rerank provider. BaseURL
// and Model are required. HTTP and local-network targets are denied unless
// explicitly enabled through OutboundURL. CostPerMTokens is optional and can
// be projected only when every pair response reports token usage.
type Options struct {
	BaseURL          string
	Model            string
	HTTPClient       *http.Client
	OutboundURL      security.OutboundURLOptions
	MaxRequestBytes  int64
	MaxResponseBytes int64
	MaxConcurrency   int
	CostPerMTokens   *float64
}

// Provider scores each query/document pair with Qwen3's yes/no completion
// contract and returns one deterministic ranking.
type Provider struct {
	endpoint         string
	model            string
	client           *http.Client
	maxRequestBytes  int64
	maxResponseBytes int64
	maxConcurrency   int
	costPerMTokens   *float64
}

type pairResult struct {
	score           float64
	tokensEvaluated *int
	tokensPredicted *int
}

var _ rerank.Provider = (*Provider)(nil)

// New constructs a strict Qwen3 completion rerank provider.
func New(options Options) (*Provider, error) {
	baseURL, err := transport.ParseAndValidateBaseURL("qwen3 rerank", options.BaseURL)
	if err != nil {
		return nil, err
	}
	model := strings.TrimSpace(options.Model)
	if model == "" {
		return nil, fmt.Errorf("qwen3 rerank model is required: %w", rerank.ErrInvalidRequest)
	}

	maxRequestBytes := options.MaxRequestBytes
	if maxRequestBytes == 0 {
		maxRequestBytes = DefaultMaxRequestBytes
	}
	if maxRequestBytes < 1 {
		return nil, fmt.Errorf("qwen3 rerank max_request_bytes must be positive: %w", rerank.ErrInvalidRequest)
	}
	maxResponseBytes := options.MaxResponseBytes
	if maxResponseBytes == 0 {
		maxResponseBytes = DefaultMaxResponseBytes
	}
	if maxResponseBytes < 1 {
		return nil, fmt.Errorf("qwen3 rerank max_response_bytes must be positive: %w", rerank.ErrInvalidRequest)
	}
	maxConcurrency := options.MaxConcurrency
	if maxConcurrency == 0 {
		maxConcurrency = DefaultMaxConcurrency
	}
	if maxConcurrency < 1 {
		return nil, fmt.Errorf("qwen3 rerank max_concurrency must be positive: %w", rerank.ErrInvalidRequest)
	}
	if options.CostPerMTokens != nil && (math.IsNaN(*options.CostPerMTokens) || math.IsInf(*options.CostPerMTokens, 0) || *options.CostPerMTokens < 0) {
		return nil, fmt.Errorf("qwen3 rerank cost_per_m_tokens must be finite and non-negative: %w", rerank.ErrInvalidRequest)
	}

	endpoint, err := transport.Endpoint("qwen3 rerank", baseURL, completionPath, options.OutboundURL)
	if err != nil {
		return nil, err
	}
	return &Provider{
		endpoint:         endpoint,
		model:            model,
		client:           transport.CloneClientWithRedirectRejection(options.HTTPClient),
		maxRequestBytes:  maxRequestBytes,
		maxResponseBytes: maxResponseBytes,
		maxConcurrency:   maxConcurrency,
		costPerMTokens:   options.CostPerMTokens,
	}, nil
}

// Model returns the configured transport/protocol and model identity.
func (p *Provider) Model() rerank.Model {
	if p == nil {
		return rerank.Model{}
	}
	return rerank.Model{Provider: ProviderName, Name: p.model}
}

// Rerank scores every document, sorts all scores deterministically, then
// returns TopN results. Scoring every pair preserves correctness when TopN is
// smaller than the input cardinality.
func (p *Provider) Rerank(ctx context.Context, in rerank.Request) (rerank.Response, error) {
	started := time.Now()
	if p == nil {
		return rerank.Response{}, fmt.Errorf("qwen3 rerank provider is unavailable: %w", rerank.ErrUnavailable)
	}
	providerModel := p.Model()
	if err := rerank.ValidateRequest(in, providerModel); err != nil {
		return rerank.Response{}, err
	}

	pairs := make([]pairResult, len(in.Documents))
	group, groupCtx := errgroup.WithContext(ctx)
	group.SetLimit(p.maxConcurrency)
	for i, document := range in.Documents {
		i, document := i, document
		group.Go(func() error {
			result, err := p.scorePair(groupCtx, in.Query, document.Text)
			if err != nil {
				return fmt.Errorf("qwen3 rerank pair %d: %w", i, err)
			}
			pairs[i] = result
			return nil
		})
	}
	if err := group.Wait(); err != nil {
		return rerank.Response{}, err
	}

	results := make([]rerank.Result, len(in.Documents))
	for i, document := range in.Documents {
		results[i] = rerank.Result{DocumentID: document.ID, Index: i, Score: pairs[i].score}
	}
	rerank.SortResults(results)
	results = results[:in.TopN]
	rerank.AssignRanks(results)

	usage := aggregateUsage(pairs)
	return rerank.Response{
		Provider:   ProviderName,
		Model:      rerank.ResolveModel(in, providerModel),
		Results:    results,
		Usage:      usage,
		Cost:       computeCost(usage, p.costPerMTokens),
		DurationMs: durationMilliseconds(started),
	}, nil
}

func (p *Provider) scorePair(ctx context.Context, query, document string) (pairResult, error) {
	result := pairResult{}
	payload, err := json.Marshal(completionRequest{
		Prompt:      prompt(query, document),
		NPredict:    1,
		NProbs:      20,
		Temperature: 0,
	})
	if err != nil {
		return result, fmt.Errorf("encode completion request: %w", err)
	}
	if int64(len(payload)) > p.maxRequestBytes {
		return result, fmt.Errorf(
			"qwen3 rerank encoded pair request is %d bytes, limit is %d: %w",
			len(payload),
			p.maxRequestBytes,
			rerank.ErrRequestTooLarge,
		)
	}

	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, p.endpoint, bytes.NewReader(payload))
	if err != nil {
		return result, fmt.Errorf("qwen3 rerank could not create provider request: %w", rerank.ErrUnavailable)
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	httpResponse, err := p.client.Do(httpRequest)
	if err != nil {
		return result, transport.RedactedTransportError("qwen3 rerank")
	}
	defer func() { _ = httpResponse.Body.Close() }()

	if httpResponse.StatusCode < http.StatusOK || httpResponse.StatusCode >= http.StatusMultipleChoices {
		transport.DrainBounded(httpResponse.Body, p.maxResponseBytes)
		return result, fmt.Errorf(
			"qwen3 rerank endpoint returned status %d: %w",
			httpResponse.StatusCode,
			rerank.ErrUnavailable,
		)
	}
	raw, tooLarge, err := transport.ReadAtMost(httpResponse.Body, p.maxResponseBytes)
	if err != nil {
		return result, fmt.Errorf("qwen3 rerank could not read provider response: %w", rerank.ErrUnavailable)
	}
	if tooLarge {
		return result, fmt.Errorf(
			"qwen3 rerank response body exceeds %d bytes: %w",
			p.maxResponseBytes,
			rerank.ErrResponseTooLarge,
		)
	}
	decoded, err := decodeResponse(raw)
	if err != nil {
		return result, err
	}
	score, err := normalizedYesProbability(decoded.CompletionProbabilities[0].TopLogprobs)
	if err != nil {
		return result, err
	}
	result.score = score
	result.tokensEvaluated = decoded.TokensEvaluated
	result.tokensPredicted = decoded.TokensPredicted
	return result, nil
}

func prompt(query, document string) string {
	return "<|im_start|>system\n" + officialSystemPrompt + "<|im_end|>\n" +
		"<|im_start|>user\n<Instruct>: " + officialInstruction +
		"\n<Query>: " + query +
		"\n<Document>: " + document + "<|im_end|>\n" +
		"<|im_start|>assistant\n<think>\n\n</think>\n\n"
}

func decodeResponse(raw []byte) (*completionResponse, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	var response completionResponse
	if err := decoder.Decode(&response); err != nil {
		return nil, fmt.Errorf("qwen3 rerank decode response: invalid JSON: %w", rerank.ErrInvalidResponse)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("qwen3 rerank response contains trailing data: %w", rerank.ErrInvalidResponse)
	}
	if len(response.CompletionProbabilities) == 0 {
		return nil, fmt.Errorf("qwen3 rerank response has no completion probabilities: %w", rerank.ErrInvalidResponse)
	}
	if response.TokensEvaluated != nil && *response.TokensEvaluated < 0 {
		return nil, fmt.Errorf("qwen3 rerank response has negative tokens_evaluated: %w", rerank.ErrInvalidResponse)
	}
	if response.TokensPredicted != nil && *response.TokensPredicted < 0 {
		return nil, fmt.Errorf("qwen3 rerank response has negative tokens_predicted: %w", rerank.ErrInvalidResponse)
	}
	return &response, nil
}

func normalizedYesProbability(values []tokenLogprob) (float64, error) {
	yes := make([]float64, 0, 2)
	no := make([]float64, 0, 2)
	for _, value := range values {
		token := strings.ToLower(strings.TrimSpace(value.Token))
		if token != "yes" && token != "no" {
			continue
		}
		if value.Logprob == nil || math.IsNaN(*value.Logprob) || math.IsInf(*value.Logprob, 0) {
			return 0, fmt.Errorf("qwen3 rerank response has invalid %s logprob: %w", token, rerank.ErrInvalidResponse)
		}
		if token == "yes" {
			yes = append(yes, *value.Logprob)
		} else {
			no = append(no, *value.Logprob)
		}
	}
	if len(yes) == 0 && len(no) == 0 {
		return 0, nil
	}
	if len(yes) == 0 {
		return 0, nil
	}
	if len(no) == 0 {
		return 1, nil
	}

	yesLogSum := logSumExp(yes)
	noLogSum := logSumExp(no)
	maximum := math.Max(yesLogSum, noLogSum)
	yesWeight := math.Exp(yesLogSum - maximum)
	noWeight := math.Exp(noLogSum - maximum)
	return yesWeight / (yesWeight + noWeight), nil
}

func logSumExp(values []float64) float64 {
	maximum := values[0]
	for _, value := range values[1:] {
		if value > maximum {
			maximum = value
		}
	}
	sum := 0.0
	for _, value := range values {
		sum += math.Exp(value - maximum)
	}
	return maximum + math.Log(sum)
}

func aggregateUsage(pairs []pairResult) *rerank.Usage {
	inputTokens := 0
	totalTokens := 0
	for _, pair := range pairs {
		if pair.tokensEvaluated == nil || pair.tokensPredicted == nil {
			return nil
		}
		inputTokens += *pair.tokensEvaluated
		totalTokens += *pair.tokensEvaluated + *pair.tokensPredicted
	}
	return &rerank.Usage{InputTokens: inputTokens, TotalTokens: totalTokens}
}

func computeCost(usage *rerank.Usage, costPerMTokens *float64) *float64 {
	if usage == nil || costPerMTokens == nil {
		return nil
	}
	cost := *costPerMTokens * float64(usage.InputTokens) / 1_000_000
	return &cost
}

func durationMilliseconds(started time.Time) *int64 {
	duration := time.Since(started).Milliseconds()
	return &duration
}

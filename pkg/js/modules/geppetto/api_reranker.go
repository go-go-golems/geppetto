package geppetto

import (
	"context"
	"fmt"

	"github.com/dop251/goja"
	"github.com/go-go-golems/geppetto/pkg/rerank"
	rerankfactory "github.com/go-go-golems/geppetto/pkg/rerank/factory"
)

// rerankerRef holds a profile-resolved rerank provider behind a hidden Go
// reference, mirroring embeddingsRef. JavaScript never receives the provider
// pointer directly; it only sees the wrapper object's methods.
type rerankerRef struct {
	api      *moduleRuntime
	provider rerank.Provider
}

// rerankerBuilder is the top-level gp.reranker(settings) factory. It accepts
// only a registry-resolved InferenceSettings wrapper and constructs the
// provider through the Go settings factory, so JavaScript cannot supply an
// endpoint, credential, HTTP client, or provider callback.
func (m *moduleRuntime) rerankerBuilder(call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 1 || goja.IsUndefined(call.Arguments[0]) || goja.IsNull(call.Arguments[0]) {
		panic(m.vm.NewTypeError("reranker(settings) requires a registry-resolved InferenceSettings wrapper"))
	}
	settingsRef, err := m.requireInferenceSettingsRef(call.Arguments[0])
	if err != nil {
		panic(m.vm.NewGoError(err))
	}
	if settingsRef.settings == nil {
		panic(m.vm.NewGoError(fmt.Errorf("reranker(settings) requires non-empty inference settings")))
	}
	factory, err := rerankfactory.NewSettingsFactoryFromInferenceSettings(settingsRef.settings)
	if err != nil {
		panic(m.vm.NewGoError(fmt.Errorf("reranker(settings): %w", err)))
	}
	provider, err := factory.NewProvider()
	if err != nil {
		panic(m.vm.NewGoError(fmt.Errorf("reranker(settings): %w", err)))
	}
	return m.newRerankerObject(&rerankerRef{api: m, provider: provider})
}

// newRerankerObject builds the JS wrapper object with hidden provider ref and
// the rerank/rerankAsync/model methods.
func (m *moduleRuntime) newRerankerObject(ref *rerankerRef) *goja.Object {
	if ref == nil {
		ref = &rerankerRef{api: m}
	}
	ref.api = m
	o := m.vm.NewObject()
	m.attachRef(o, ref)
	m.mustSet(o, "rerank", func(query string, documents []map[string]any, options map[string]any) (map[string]any, error) {
		if ref.provider == nil {
			return nil, fmt.Errorf("reranker provider is not initialized")
		}
		req, err := decodeRerankRequest(query, documents, options, ref.provider)
		if err != nil {
			return nil, err
		}
		resp, err := ref.provider.Rerank(m.rerankerContext(), req)
		if err != nil {
			return nil, err
		}
		return rerankResponseToJS(resp), nil
	})
	m.mustSet(o, "rerankAsync", func(call goja.FunctionCall) goja.Value {
		return m.rerankerAsync(call, ref)
	})
	m.mustSet(o, "model", func() map[string]any {
		if ref.provider == nil {
			return map[string]any{}
		}
		model := ref.provider.Model()
		return map[string]any{"provider": model.Provider, "name": model.Name}
	})
	return o
}

// rerankerContext returns the runtime lifetime context so runtime shutdown
// cancels in-flight rerank calls.
func (m *moduleRuntime) rerankerContext() context.Context {
	if m != nil && m.runtimeLifetimeContext != nil {
		return m.runtimeLifetimeContext
	}
	return context.Background()
}

// decodeRerankRequest strictly decodes JS arguments into a rerank.Request.
// It rejects unknown option keys, sparse arrays, non-object documents,
// duplicate IDs, non-string text, and non-integral topN. Documents must be
// {id, text} objects; accepting plain strings would discard durable identity.
func decodeRerankRequest(query string, documents []map[string]any, options map[string]any, provider rerank.Provider) (rerank.Request, error) {
	if query == "" {
		return rerank.Request{}, fmt.Errorf("rerank query is required: %w", rerank.ErrInvalidRequest)
	}
	if len(documents) == 0 {
		return rerank.Request{}, fmt.Errorf("rerank requires at least one document: %w", rerank.ErrInvalidRequest)
	}
	docs := make([]rerank.Document, 0, len(documents))
	for i, raw := range documents {
		if raw == nil {
			return rerank.Request{}, fmt.Errorf("rerank document %d is null: %w", i, rerank.ErrInvalidRequest)
		}
		id, ok := raw["id"].(string)
		if !ok || id == "" {
			return rerank.Request{}, fmt.Errorf("rerank document %d requires a non-empty string id: %w", i, rerank.ErrInvalidRequest)
		}
		text, ok := raw["text"].(string)
		if !ok || text == "" {
			return rerank.Request{}, fmt.Errorf("rerank document %d (%s) requires non-empty string text: %w", i, id, rerank.ErrInvalidRequest)
		}
		docs = append(docs, rerank.Document{ID: id, Text: text})
	}

	topN, model, err := decodeRerankOptions(options, len(docs))
	if err != nil {
		return rerank.Request{}, err
	}

	return rerank.Request{
		Model:     model,
		Query:     query,
		Documents: docs,
		TopN:      topN,
	}, nil
}

// decodeRerankOptions decodes the {topN, model?} options object. topN is
// required and must be an integer in [1, len(documents)]. model is optional;
// when provided it must match the provider model.
func decodeRerankOptions(options map[string]any, docCount int) (int, string, error) {
	if options == nil {
		return 0, "", fmt.Errorf("rerank options are required (expected {topN, model?}): %w", rerank.ErrInvalidRequest)
	}
	// Reject unknown keys to prevent silent typos.
	for k := range options {
		switch k {
		case "topN", "model":
		default:
			return 0, "", fmt.Errorf("rerank options: unknown key %q (expected topN, model): %w", k, rerank.ErrInvalidRequest)
		}
	}

	topNRaw, ok := options["topN"]
	if !ok {
		return 0, "", fmt.Errorf("rerank options.topN is required: %w", rerank.ErrInvalidRequest)
	}
	topN, err := toIntStrict(topNRaw)
	if err != nil {
		return 0, "", fmt.Errorf("rerank options.topN must be an integer: %w: %w", err, rerank.ErrInvalidRequest)
	}
	if topN < 1 || topN > docCount {
		return 0, "", fmt.Errorf("rerank options.topN (%d) must be between 1 and %d: %w", topN, docCount, rerank.ErrInvalidRequest)
	}

	model := ""
	if modelRaw, ok := options["model"]; ok && modelRaw != nil {
		m, ok := modelRaw.(string)
		if !ok {
			return 0, "", fmt.Errorf("rerank options.model must be a string: %w", rerank.ErrInvalidRequest)
		}
		model = m
	}

	return topN, model, nil
}

// toIntStrict converts a JS number to an int, rejecting non-integral values and
// values outside the safe integer range. Unlike toInt, it returns an error
// rather than a default so malformed input fails explicitly.
func toIntStrict(v any) (int, error) {
	switch n := v.(type) {
	case int:
		return n, nil
	case int32:
		return int(n), nil
	case int64:
		return int(n), nil
	case float64:
		if n != float64(int(n)) {
			return 0, fmt.Errorf("value %v is not an integer", n)
		}
		return int(n), nil
	default:
		return 0, fmt.Errorf("value is not a number: %T", v)
	}
}

// rerankResponseToJS converts a rerank.Response into a plain camelCase JS
// object. nil optional fields are omitted (not set), matching the Go JSON
// encoding.
func rerankResponseToJS(resp rerank.Response) map[string]any {
	out := map[string]any{
		"provider": resp.Provider,
		"model":    resp.Model,
	}
	results := make([]map[string]any, 0, len(resp.Results))
	for _, r := range resp.Results {
		results = append(results, map[string]any{
			"documentId": r.DocumentID,
			"index":      r.Index,
			"score":      r.Score,
			"rank":       r.Rank,
		})
	}
	out["results"] = results
	if resp.Usage != nil {
		out["usage"] = map[string]any{
			"inputTokens": resp.Usage.InputTokens,
			"totalTokens": resp.Usage.TotalTokens,
		}
	}
	if resp.Cost != nil {
		out["cost"] = *resp.Cost
	}
	if resp.RequestID != "" {
		out["requestId"] = resp.RequestID
	}
	if resp.DurationMs != nil {
		out["durationMs"] = *resp.DurationMs
	}
	return out
}

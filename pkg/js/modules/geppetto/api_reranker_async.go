package geppetto

import (
	"context"
	"fmt"

	"github.com/dop251/goja"
)

// rerankerAsync implements rerankAsync(query, documents, options) returning a
// {promise, cancel, close} handle. The provider goroutine touches no Goja
// value: the request is fully decoded to plain Go values before launching, and
// Promise settlement + response conversion happen on the runtime owner thread
// via postOnOwner.
func (m *moduleRuntime) rerankerAsync(call goja.FunctionCall, ref *rerankerRef) goja.Value {
	if _, err := m.requireBridge("reranker.rerankAsync"); err != nil {
		panic(m.vm.NewTypeError(err.Error()))
	}
	if ref.provider == nil {
		panic(m.vm.NewGoError(fmt.Errorf("reranker provider is not initialized")))
	}

	if len(call.Arguments) < 3 {
		panic(m.vm.NewTypeError("rerankAsync(query, documents, options) requires three arguments"))
	}

	// Decode and deep-copy the JS request to plain Go values on the owner
	// thread before launching the provider goroutine.
	query, ok := call.Arguments[0].Export().(string)
	if !ok {
		panic(m.vm.NewTypeError("rerankAsync: query must be a string"))
	}
	docsRaw, err := exportDocuments(call.Arguments[1])
	if err != nil {
		panic(m.vm.NewTypeError(err.Error()))
	}
	optsRaw, err := exportOptions(call.Arguments[2])
	if err != nil {
		panic(m.vm.NewTypeError(err.Error()))
	}

	req, err := decodeRerankRequest(query, docsRaw, optsRaw, ref.provider)
	if err != nil {
		panic(m.vm.NewGoError(err))
	}

	promise, resolve, reject := m.vm.NewPromise()
	handleObj := m.vm.NewObject()

	ctx, cancel := context.WithCancel(m.rerankerContext())
	closed := false

	m.mustSet(handleObj, "promise", promise)
	m.mustSet(handleObj, "cancel", func(goja.FunctionCall) goja.Value {
		cancel()
		return goja.Undefined()
	})
	m.mustSet(handleObj, "close", func(goja.FunctionCall) goja.Value {
		if !closed {
			closed = true
			cancel()
		}
		return goja.Undefined()
	})

	provider := ref.provider
	go func() {
		defer cancel()
		resp, runErr := provider.Rerank(ctx, req)

		postErr := m.postOnOwner(m.rerankerContext(), "reranker.rerankAsync.settle", func(_ context.Context) {
			if runErr != nil {
				_ = reject(m.vm.NewGoError(runErr))
				return
			}
			_ = resolve(m.toJSValue(rerankResponseToJS(resp)))
		})
		if postErr != nil {
			m.logger.Error().Err(postErr).Msg("reranker.rerankAsync: failed to settle promise on owner thread")
		}
	}()

	return handleObj
}

// exportDocuments converts a goja value (an array of {id, text} objects) into
// a plain Go []map[string]any on the owner thread, so the provider goroutine
// never touches a goja.Value.
func exportDocuments(v goja.Value) ([]map[string]any, error) {
	if v == nil || goja.IsUndefined(v) || goja.IsNull(v) {
		return nil, fmt.Errorf("documents must be an array of {id, text} objects")
	}
	raw := v.Export()
	arr, ok := raw.([]any)
	if !ok {
		return nil, fmt.Errorf("documents must be an array, got %T", raw)
	}
	out := make([]map[string]any, 0, len(arr))
	for i, item := range arr {
		m, ok := item.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("documents[%d] must be an object, got %T", i, item)
		}
		out = append(out, m)
	}
	return out, nil
}

// exportOptions converts a goja value (the {topN, model?} object) into a plain
// Go map[string]any on the owner thread.
func exportOptions(v goja.Value) (map[string]any, error) {
	if v == nil || goja.IsUndefined(v) || goja.IsNull(v) {
		return nil, fmt.Errorf("options are required (expected {topN, model?})")
	}
	raw := v.Export()
	m, ok := raw.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("options must be an object, got %T", raw)
	}
	return m, nil
}

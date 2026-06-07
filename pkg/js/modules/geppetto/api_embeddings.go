package geppetto

import (
	"context"
	"fmt"

	"github.com/dop251/goja"
	"github.com/go-go-golems/geppetto/pkg/embeddings"
)

type embeddingsRef struct {
	api      *moduleRuntime
	provider embeddings.Provider
}

func (m *moduleRuntime) embeddingsBuilder(call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 1 || goja.IsUndefined(call.Arguments[0]) || goja.IsNull(call.Arguments[0]) {
		panic(m.vm.NewTypeError("embeddings(settings) requires a registry-resolved InferenceSettings wrapper"))
	}
	settingsRef, err := m.requireInferenceSettingsRef(call.Arguments[0])
	if err != nil {
		panic(m.vm.NewGoError(err))
	}
	if settingsRef.settings == nil {
		panic(m.vm.NewGoError(fmt.Errorf("embeddings(settings) requires non-empty inference settings")))
	}
	provider, err := embeddings.NewSettingsFactoryFromInferenceSettings(settingsRef.settings).NewProvider()
	if err != nil {
		panic(m.vm.NewGoError(fmt.Errorf("embeddings(settings): %w", err)))
	}
	return m.newEmbeddingsObject(&embeddingsRef{api: m, provider: provider})
}

func (m *moduleRuntime) newEmbeddingsObject(ref *embeddingsRef) *goja.Object {
	if ref == nil {
		ref = &embeddingsRef{api: m}
	}
	ref.api = m
	o := m.vm.NewObject()
	m.attachRef(o, ref)
	m.mustSet(o, "embed", func(text string) ([]float32, error) {
		if ref.provider == nil {
			return nil, fmt.Errorf("embeddings provider is not initialized")
		}
		return ref.provider.GenerateEmbedding(m.embeddingContext(), text)
	})
	m.mustSet(o, "embedBatch", func(texts []string) ([][]float32, error) {
		if ref.provider == nil {
			return nil, fmt.Errorf("embeddings provider is not initialized")
		}
		return ref.provider.GenerateBatchEmbeddings(m.embeddingContext(), texts)
	})
	m.mustSet(o, "model", func() map[string]any {
		if ref.provider == nil {
			return map[string]any{}
		}
		model := ref.provider.GetModel()
		return map[string]any{"name": model.Name, "dimensions": model.Dimensions}
	})
	return o
}

func (m *moduleRuntime) embeddingContext() context.Context {
	if m != nil && m.runtimeLifetimeContext != nil {
		return m.runtimeLifetimeContext
	}
	return context.Background()
}

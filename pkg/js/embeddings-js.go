package js

import (
	"context"
	"fmt"

	"github.com/dop251/goja"
	"github.com/go-go-golems/geppetto/pkg/embeddings"
)

type JSEmbeddingsWrapper struct {
	runtime  *goja.Runtime
	provider embeddings.Provider
}

// RegisterEmbeddings registers the embeddings functionality in the given Goja runtime
func RegisterEmbeddings(runtime *goja.Runtime, globalName string, provider embeddings.Provider) error {
	wrapper := &JSEmbeddingsWrapper{
		runtime:  runtime,
		provider: provider,
	}

	embeddingsObj := runtime.NewObject()
	if err := embeddingsObj.Set("generateEmbedding", wrapper.generateEmbedding); err != nil {
		return fmt.Errorf("failed to set generateEmbedding: %w", err)
	}

	if err := embeddingsObj.Set("getModel", wrapper.getModel); err != nil {
		return fmt.Errorf("failed to set getModel: %w", err)
	}

	return runtime.Set(globalName, embeddingsObj)
}

func (w *JSEmbeddingsWrapper) generateEmbedding(call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 1 {
		panic(w.runtime.NewTypeError("generateEmbedding requires a text argument"))
	}

	text := call.Arguments[0].String()
	embedding, err := w.provider.GenerateEmbedding(context.Background(), text)
	if err != nil {
		panic(w.runtime.NewGoError(fmt.Errorf("failed to generate embedding: %w", err)))
	}

	// Convert []float32 to []interface{} for Goja
	embeddingInterface := make([]interface{}, len(embedding))
	for i, v := range embedding {
		embeddingInterface[i] = v
	}

	return w.runtime.ToValue(embeddingInterface)
}

func (w *JSEmbeddingsWrapper) getModel(call goja.FunctionCall) goja.Value {
	model := w.provider.GetModel()
	modelObj := w.runtime.NewObject()

	if err := modelObj.Set("name", model.Name); err != nil {
		panic(w.runtime.NewGoError(fmt.Errorf("failed to set model name: %w", err)))
	}
	if err := modelObj.Set("dimensions", model.Dimensions); err != nil {
		panic(w.runtime.NewGoError(fmt.Errorf("failed to set model dimensions: %w", err)))
	}

	return modelObj
}

package js

import (
	"context"
	"fmt"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/eventloop"
	"github.com/go-go-golems/geppetto/pkg/embeddings"
	"github.com/rs/zerolog/log"
)

type JSEmbeddingsWrapper struct {
	runtime  *goja.Runtime
	provider embeddings.Provider
	loop     *eventloop.EventLoop
}

// RegisterEmbeddings registers the embeddings functionality in the given Goja runtime
func RegisterEmbeddings(runtime *goja.Runtime, globalName string, provider embeddings.Provider, loop *eventloop.EventLoop) error {
	wrapper := &JSEmbeddingsWrapper{
		runtime:  runtime,
		provider: provider,
		loop:     loop,
	}

	embeddingsObj := runtime.NewObject()

	// Register sync methods
	if err := embeddingsObj.Set("generateEmbedding", wrapper.generateEmbedding); err != nil {
		return fmt.Errorf("failed to set generateEmbedding: %w", err)
	}
	if err := embeddingsObj.Set("getModel", wrapper.getModel); err != nil {
		return fmt.Errorf("failed to set getModel: %w", err)
	}

	// Register async methods
	if err := embeddingsObj.Set("generateEmbeddingAsync", wrapper.generateEmbeddingAsync); err != nil {
		return fmt.Errorf("failed to set generateEmbeddingAsync: %w", err)
	}
	if err := embeddingsObj.Set("generateEmbeddingWithCallbacks", wrapper.generateEmbeddingWithCallbacks); err != nil {
		return fmt.Errorf("failed to set generateEmbeddingWithCallbacks: %w", err)
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

	return w.runtime.ToValue(embedding)
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

// New async methods
func (w *JSEmbeddingsWrapper) generateEmbeddingAsync(call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 1 {
		panic(w.runtime.NewTypeError("generateEmbeddingAsync requires a text argument"))
	}

	text := call.Arguments[0].String()
	promise, resolve, reject := w.runtime.NewPromise()

	w.loop.RunOnLoop(func(*goja.Runtime) {
		go func() {
			embedding, err := w.provider.GenerateEmbedding(context.Background(), text)
			if err != nil {
				w.loop.RunOnLoop(func(*goja.Runtime) {
					rejectErr := reject(w.runtime.ToValue(err.Error()))
					if rejectErr != nil {
						log.Error().Err(rejectErr).Msg("failed to reject promise")
					}
				})
				return
			}

			w.loop.RunOnLoop(func(*goja.Runtime) {
				resolveErr := resolve(w.runtime.ToValue(embedding))
				if resolveErr != nil {
					log.Error().Err(resolveErr).Msg("failed to resolve promise")
				}
			})
		}()
	})

	return w.runtime.ToValue(promise)
}

func (w *JSEmbeddingsWrapper) generateEmbeddingWithCallbacks(call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 2 {
		panic(w.runtime.NewTypeError("generateEmbeddingWithCallbacks requires text and callbacks arguments"))
	}

	text := call.Arguments[0].String()
	callbacksObj := call.Arguments[1].ToObject(w.runtime)

	var onSuccess, onError goja.Callable
	var ok bool

	if onSuccessVal := callbacksObj.Get("onSuccess"); onSuccessVal != nil {
		if onSuccess, ok = goja.AssertFunction(onSuccessVal); !ok {
			panic(w.runtime.NewTypeError("onSuccess must be a function"))
		}
	}

	if onErrorVal := callbacksObj.Get("onError"); onErrorVal != nil {
		if onError, ok = goja.AssertFunction(onErrorVal); !ok {
			panic(w.runtime.NewTypeError("onError must be a function"))
		}
	}

	ctx, cancel := context.WithCancel(context.Background())

	w.loop.RunOnLoop(func(*goja.Runtime) {
		go func() {
			defer cancel()

			embedding, err := w.provider.GenerateEmbedding(ctx, text)
			if err != nil {
				if onError != nil {
					w.loop.RunOnLoop(func(*goja.Runtime) {
						_, callErr := onError(goja.Undefined(), w.runtime.ToValue(err.Error()))
						if callErr != nil {
							log.Error().Err(callErr).Msg("failed to call onError")
						}
					})
				}
				return
			}

			if onSuccess != nil {
				// Convert []float32 to []interface{} for Goja
				embeddingInterface := make([]interface{}, len(embedding))
				for i, v := range embedding {
					embeddingInterface[i] = v
				}

				w.loop.RunOnLoop(func(*goja.Runtime) {
					_, callErr := onSuccess(goja.Undefined(), w.runtime.ToValue(embeddingInterface))
					if callErr != nil {
						log.Error().Err(callErr).Msg("failed to call onSuccess")
					}
				})
			}
		}()
	})

	// Return cancel function
	return w.runtime.ToValue(func(call goja.FunctionCall) goja.Value {
		cancel()
		return goja.Undefined()
	})
}

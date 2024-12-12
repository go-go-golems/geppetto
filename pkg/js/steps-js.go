package js

import (
	"context"
	"fmt"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/eventloop"
	"github.com/go-go-golems/geppetto/pkg/helpers"
	"github.com/go-go-golems/geppetto/pkg/steps"
	"github.com/go-go-golems/geppetto/pkg/steps/utils"
	"github.com/rs/zerolog/log"
)

type JSStepWrapper[T any, U any] struct {
	runtime *goja.Runtime
	step    steps.Step[T, U]
	loop    *eventloop.EventLoop
}

// CreateStepObject creates a JavaScript object wrapping a Step with Promise, blocking and callback APIs
func CreateStepObject[T any, U any](
	runtime *goja.Runtime,
	loop *eventloop.EventLoop,
	step steps.Step[T, U],
	inputConverter func(goja.Value) T,
	outputConverter func(U) goja.Value,
) (*goja.Object, error) {
	wrapper := &JSStepWrapper[T, U]{
		runtime: runtime,
		step:    step,
		loop:    loop,
	}

	stepObj := runtime.NewObject()
	err := stepObj.Set("startAsync", wrapper.makeStartAsync(inputConverter, outputConverter))
	if err != nil {
		return nil, err
	}
	err = stepObj.Set("startBlocking", wrapper.makeStartBlocking(inputConverter, outputConverter))
	if err != nil {
		return nil, err
	}
	err = stepObj.Set("startWithCallbacks", wrapper.makeStartWithCallbacks(inputConverter, outputConverter))
	if err != nil {
		return nil, err
	}

	return stepObj, nil
}

// RegisterStep is kept for backward compatibility
func RegisterStep[T any, U any](
	runtime *goja.Runtime,
	loop *eventloop.EventLoop,
	name string,
	step steps.Step[T, U],
	inputConverter func(goja.Value) T,
	outputConverter func(U) goja.Value,
) error {
	stepObj, err := CreateStepObject(runtime, loop, step, inputConverter, outputConverter)
	if err != nil {
		return err
	}
	return runtime.Set(name, stepObj)
}

func (w *JSStepWrapper[T, U]) makeStartAsync(
	inputConverter func(goja.Value) T,
	outputConverter func(U) goja.Value,
) func(goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		log.Info().Msg("makeStartAsync called")
		if len(call.Arguments) < 1 {
			return w.runtime.ToValue([]interface{}{nil, fmt.Errorf("startAsync requires an input argument")})
		}

		input := inputConverter(call.Arguments[0])
		ctx := context.Background()

		log.Info().Interface("input", input).Msg("makeStartAsync called")

		promise, resolve, reject := w.runtime.NewPromise()

		w.loop.RunOnLoop(func(*goja.Runtime) {
			go func() {
				log.Info().Msg("Starting step")
				result, err := w.step.Start(ctx, input)
				if err != nil {
					log.Info().Msg("Failed to start step")
					w.loop.RunOnLoop(func(*goja.Runtime) {
						rejectErr := reject(w.runtime.ToValue(fmt.Sprintf("failed to start step: %v", err)))
						if rejectErr != nil {
							log.Error().Err(rejectErr).Msg("failed to reject promise")
						}
					})
					return
				}
				defer result.Cancel()

				results := result.Return()
				if len(results) > 0 {
					log.Info().Interface("result", results[0]).Msg("Result")
				}
				log.Info().Array("results", helpers.ToResultSlice(results)).Msg("Results")

				if len(results) == 0 {
					w.loop.RunOnLoop(func(*goja.Runtime) {
						log.Info().Msg("Resolving promise")
						resolveErr := resolve(w.runtime.ToValue([]interface{}{}))
						if resolveErr != nil {
							log.Error().Err(resolveErr).Msg("failed to resolve promise")
						}
					})
					return
				}

				// Convert results to JS values
				jsResults := make([]goja.Value, len(results))
				var resolveErr error
				for i, r := range results {
					if r.Error() != nil {
						w.loop.RunOnLoop(func(*goja.Runtime) {
							rejectErr := reject(w.runtime.ToValue(r.Error().Error()))
							if rejectErr != nil {
								log.Error().Err(rejectErr).Msg("failed to reject promise")
							}
						})
						return
					}
					jsResults[i] = outputConverter(r.Unwrap())
				}

				// Must resolve on the event loop
				w.loop.RunOnLoop(func(*goja.Runtime) {
					if resolveErr != nil {
						rejectErr := reject(w.runtime.ToValue(resolveErr.Error()))
						if rejectErr != nil {
							log.Error().Err(rejectErr).Msg("failed to reject promise")
						}
						return
					}
					resolveErr = resolve(w.runtime.ToValue(jsResults))
					if resolveErr != nil {
						log.Error().Err(resolveErr).Msg("failed to resolve promise")
					}
				})

				log.Info().Msg("Done")
			}()
		})

		log.Info().Msg("Returning promise")

		return w.runtime.ToValue(promise)
	}

}

func (w *JSStepWrapper[T, U]) makeStartBlocking(
	inputConverter func(goja.Value) T,
	outputConverter func(U) goja.Value,
) func(goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			return w.runtime.ToValue([]interface{}{nil, fmt.Errorf("startBlocking requires an input argument")})
		}

		input := inputConverter(call.Arguments[0])
		ctx := context.Background()
		result, err := w.step.Start(ctx, input)
		if err != nil {
			return w.runtime.ToValue([]interface{}{nil, fmt.Errorf("failed to start step: %w", err)})
		}
		defer result.Cancel()

		results := result.Return()
		if len(results) == 0 {
			return goja.Undefined()
		}

		// Convert results to JS values
		jsResults := make([]goja.Value, len(results))
		for i, r := range results {
			if r.Error() != nil {
				return w.runtime.ToValue([]interface{}{nil, r.Error()})
			}
			jsResults[i] = outputConverter(r.Unwrap())
		}

		// Return the value directly on success
		return w.runtime.ToValue(jsResults)
	}
}

type CallbackFunctions struct {
	OnResult goja.Callable
	OnError  goja.Callable
	OnDone   goja.Callable
	OnCancel goja.Callable
}

func (w *JSStepWrapper[T, U]) makeStartWithCallbacks(
	inputConverter func(goja.Value) T,
	outputConverter func(U) goja.Value,
) func(goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 2 {
			return w.runtime.ToValue([]interface{}{nil, fmt.Errorf("startWithCallbacks requires input and callbacks object")})
		}

		log.Info().Msg("makeStartWithCallbacks called")

		input := inputConverter(call.Arguments[0])

		// Extract callbacks
		callbacks := CallbackFunctions{}
		callbacksObj := call.Arguments[1].ToObject(w.runtime)

		var ok bool
		if onResult := callbacksObj.Get("onResult"); onResult != nil {
			callbacks.OnResult, ok = goja.AssertFunction(onResult)
			if !ok {
				return w.runtime.ToValue([]interface{}{nil, fmt.Errorf("failed to assert onResult")})
			}
		}
		if onError := callbacksObj.Get("onError"); onError != nil {
			callbacks.OnError, ok = goja.AssertFunction(onError)
			if !ok {
				return w.runtime.ToValue([]interface{}{nil, fmt.Errorf("failed to assert onError")})
			}
		}
		if onDone := callbacksObj.Get("onDone"); onDone != nil {
			callbacks.OnDone, ok = goja.AssertFunction(onDone)
			if !ok {
				return w.runtime.ToValue([]interface{}{nil, fmt.Errorf("failed to assert onDone")})
			}
		}
		if onCancel := callbacksObj.Get("onCancel"); onCancel != nil {
			callbacks.OnCancel, ok = goja.AssertFunction(onCancel)
			if !ok {
				return w.runtime.ToValue([]interface{}{nil, fmt.Errorf("failed to assert onCancel")})
			}
		}

		ctx, cancel := context.WithCancel(context.Background())

		log.Info().Msg("Starting step")

		result, err := w.step.Start(ctx, input)
		if err != nil {
			cancel()
			return w.runtime.ToValue([]interface{}{nil, fmt.Errorf("failed to start step: %w", err)})
		}

		log.Info().Msg("Step started")

		// Return a cancel function to JavaScript
		cancelFn := func(call goja.FunctionCall) goja.Value {
			log.Info().Msg("Cancelling step")
			cancel()
			result.Cancel()
			if callbacks.OnCancel != nil {
				log.Info().Msg("Calling onCancel")
				w.loop.RunOnLoop(func(*goja.Runtime) {
					log.Info().Msg("Calling onCancel")
					_, err := callbacks.OnCancel(goja.Undefined())
					if err != nil {
						log.Error().Err(err).Msg("failed to call onCancel")
					}
					log.Info().Msg("onCancel called")
				})
			}
			log.Info().Msg("Done")
			return goja.Undefined()
		}

		w.loop.RunOnLoop(func(*goja.Runtime) {
			go func() {
				defer func() {
					log.Info().Msg("StepsJS Done")
				}()

				defer func() {
					log.Info().Msg("Done")
					if callbacks.OnDone != nil {
						w.loop.RunOnLoop(func(*goja.Runtime) {
							log.Info().Msg("Calling onDone")
							_, err := callbacks.OnDone(goja.Undefined())
							if err != nil {
								log.Error().Err(err).Msg("failed to call onDone")
							}
						})
					}
				}()

				for {
					select {
					case <-ctx.Done():
						log.Info().Msg("StepsJS Context done")
						return
					case r, ok := <-result.GetChannel():
						if !ok {
							return
						}
						if r.Error() != nil {
							log.Info().Msg("Error")
							if callbacks.OnError != nil {
								w.loop.RunOnLoop(func(*goja.Runtime) {
									log.Info().Msg("Calling onError")
									_, err := callbacks.OnError(goja.Undefined(), w.runtime.ToValue(r.Error().Error()))
									if err != nil {
										log.Error().Err(err).Msg("failed to call onError")
									}
								})
							}
							continue
						}
						if callbacks.OnResult != nil {
							w.loop.RunOnLoop(func(*goja.Runtime) {
								log.Info().Msg("Calling onResult")
								_, err := callbacks.OnResult(goja.Undefined(), outputConverter(r.Unwrap()))
								if err != nil {
									log.Error().Err(err).Msg("failed to call onResult")
								}
							})
						}
					}
				}

			}()
		})

		return w.runtime.ToValue(cancelFn)
	}
}

// Example usage with embeddings step
func CreateEmbeddingsStepObject(runtime *goja.Runtime, loop *eventloop.EventLoop, provider steps.Step[string, []float32]) (*goja.Object, error) {
	inputConverter := func(v goja.Value) string {
		return v.String()
	}

	outputConverter := func(embedding []float32) goja.Value {
		embeddingInterface := make([]interface{}, len(embedding))
		for i, v := range embedding {
			embeddingInterface[i] = v
		}
		return runtime.ToValue(embeddingInterface)
	}

	return CreateStepObject(runtime, loop, provider, inputConverter, outputConverter)
}

// Helper to create a LambdaStep from a JavaScript function
func NewJSLambdaStep[T any, U any](
	runtime *goja.Runtime,
	fn goja.Callable,
	inputConverter func(T) goja.Value,
	outputConverter func(goja.Value) (U, error),
) steps.Step[T, U] {
	return &utils.LambdaStep[T, U]{
		Function: func(input T) helpers.Result[U] {
			jsInput := inputConverter(input)
			jsResult, err := fn(goja.Undefined(), jsInput)
			if err != nil {
				return helpers.NewErrorResult[U](err)
			}

			result, err := outputConverter(jsResult)
			if err != nil {
				return helpers.NewErrorResult[U](err)
			}

			return helpers.NewValueResult(result)
		},
	}
}

package js

import (
	"context"
	"fmt"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/dop251/goja"
	"github.com/go-go-golems/geppetto/pkg/helpers"
	"github.com/go-go-golems/geppetto/pkg/steps"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// CreateWatermillStepObject creates a step object that uses the watermill-based RuntimeEngine
// This returns a function that creates the step object when called within the VM context
func CreateWatermillStepObject[T any, U any](
	engine *RuntimeEngine,
	step steps.Step[T, U],
	inputConverter func(goja.Value) T,
	outputConverter func(U) goja.Value,
) func(vm *goja.Runtime) *goja.Object {
	stepID := uuid.NewString()
	logger := log.With().
		Str("stepType", fmt.Sprintf("%T", step)).
		Str("stepID", stepID).
		Logger()

	logger.Debug().Msg("Creating watermill step object factory")

	return func(vm *goja.Runtime) *goja.Object {
		logger.Debug().Msg("Creating step object in VM context")
		stepObj := vm.NewObject()

		// Add the runAsync method
		err := stepObj.Set("runAsync", func(call goja.FunctionCall) goja.Value {
			logger.Debug().Msg("runAsync called")

			if len(call.Arguments) < 1 {
				logger.Error().Msg("runAsync requires at least an input argument")
				panic(vm.NewTypeError("runAsync requires input"))
			}

			// Convert input
			logger.Debug().Interface("args", call.Arguments).Msg("runAsync called")
			input := inputConverter(call.Arguments[0])
			logger.Debug().Interface("input", input).Msg("input converted")

			// Optional onEvent callback (may be undefined/null)
			var userOnEvent goja.Callable
			if len(call.Arguments) >= 2 && !goja.IsUndefined(call.Arguments[1]) && !goja.IsNull(call.Arguments[1]) {
				callable, ok := goja.AssertFunction(call.Arguments[1])
				if !ok {
					logger.Error().Msg("onEvent must be a function")
					panic(vm.NewTypeError("onEvent must be a function"))
				}
				userOnEvent = callable
			} else {
				userOnEvent = goja.Callable(func(this goja.Value, args ...goja.Value) (goja.Value, error) {
					logger.Debug().Interface("args", args).Msg("userOnEvent called")
					return goja.Undefined(), nil
				})
			}

			// Create promise
			promise, resolve, reject := vm.NewPromise()
			logger.Debug().Msg("promise created")

			// Start the step with internal callback; obtain stepID
			logger.Debug().Str("stepID", stepID).Msg("stepID created")
			stepResult, err := StartTypedStep(engine, step, stepID, input, userOnEvent)
			logger.Debug().Str("stepID", stepID).Msg("stepResult created")
			if err != nil {
				logger.Error().Err(err).Msg("error starting step")
				panic(vm.NewGoError(err))
			}

			go func() {
				logger.Debug().Msg("Waiting for step to finish")
				// Wait for the step to finish
				r := stepResult.Return()
				logger.Debug().Interface("r", r).Msg("stepResult returned")
				engine.Loop.RunOnLoop(func(vm *goja.Runtime) {
					if len(r) == 0 {
						logger.Error().Msg("no result")
						_ = reject(vm.NewGoError(fmt.Errorf("no result")))
						return
					}
					result := r[0]
					if result.Error() != nil {
						logger.Error().Err(result.Error()).Msg("error in result")
						_ = reject(vm.NewGoError(result.Error()))
						return
					}
					logger.Debug().Interface("result", result.Unwrap()).Msg("result")
					_ = resolve(result.Unwrap())
				})
			}()

			// Build return object { stepID, promise }
			logger.Debug().Msg("Building return object")
			retObj := vm.NewObject()
			_ = retObj.Set("stepID", stepID)
			_ = retObj.Set("promise", promise)

			return retObj
		})
		if err != nil {
			logger.Error().Err(err).Msg("Failed to set runAsync method")
			panic(vm.NewGoError(err))
		}

		// Add the run (blocking) method
		err = stepObj.Set("run", func(call goja.FunctionCall) goja.Value {
			logger.Debug().Msg("run (blocking) called")

			if len(call.Arguments) < 1 {
				logger.Error().Msg("run requires at least an input argument")
				panic(vm.NewTypeError("run requires input"))
			}

			logger.Debug().Interface("args", call.Arguments).Msg("run called")
			input := inputConverter(call.Arguments[0])
			logger.Debug().Interface("input", input).Msg("input converted")

			var userOnEvent goja.Callable
			if len(call.Arguments) >= 2 && !goja.IsUndefined(call.Arguments[1]) && !goja.IsNull(call.Arguments[1]) {
				callable, ok := goja.AssertFunction(call.Arguments[1])
				if !ok {
					panic(vm.NewTypeError("onEvent must be a function"))
				}
				userOnEvent = callable
			} else {
				userOnEvent = goja.Callable(func(this goja.Value, args ...goja.Value) (goja.Value, error) {
					logger.Debug().Interface("args", args).Msg("userOnEvent called")
					return goja.Undefined(), nil
				})
			}

			// Start the step
			logger.Debug().Str("stepID", stepID).Msg("Starting step")
			stepResult, err := StartTypedStep(engine, step, stepID, input, userOnEvent)
			if err != nil {
				panic(vm.NewGoError(err))
			}

			logger.Debug().Msg("Step started")
			r := stepResult.Return()
			if len(r) == 0 {
				logger.Error().Msg("no result")
				panic(vm.NewGoError(fmt.Errorf("no result")))
			}
			result := r[0]
			if result.Error() != nil {
				logger.Error().Err(result.Error()).Msg("error in result")
				panic(vm.NewGoError(result.Error()))
			}

			logger.Debug().Interface("result", result.Unwrap()).Msg("result")
			return outputConverter(result.Unwrap())
		})
		if err != nil {
			logger.Error().Err(err).Msg("Failed to set run method")
			panic(vm.NewGoError(err))
		}

		logger.Debug().Msg("Watermill step object created successfully (simplified API)")
		return stepObj
	}
}

// StartTypedStep is a helper function that allows running a typed step through RuntimeEngine
// It performs the necessary type conversion inline without wrapper classes
func StartTypedStep[T, U any](
	engine *RuntimeEngine,
	step steps.Step[T, U],
	stepID string,
	input T,
	onEvent goja.Callable,
) (steps.StepResult[U], error) {
	// Create an inline adapter that converts the typed step to work with RunStep
	genericStep := genericStepAdapter[T, U]{step: step}
	result, err := engine.RunStep(genericStep, stepID, input, onEvent)
	if err != nil {
		return nil, err
	}

	// Cast the generic result back to the typed result
	typedResult, ok := result.(steps.StepResult[U])
	if !ok {
		return nil, fmt.Errorf("result type mismatch: expected StepResult[%T], got %T", *new(U), result)
	}

	return typedResult, nil
}

// genericStepAdapter is a minimal adapter that implements steps.Step[any, any]
type genericStepAdapter[T, U any] struct {
	step steps.Step[T, U]
}

func (a genericStepAdapter[T, U]) Start(ctx context.Context, input any) (steps.StepResult[any], error) {
	typedInput, ok := input.(T)
	if !ok {
		return nil, fmt.Errorf("input type mismatch: expected %T, got %T", *new(T), input)
	}

	result, err := a.step.Start(ctx, typedInput)
	if err != nil {
		return nil, err
	}

	return genericStepResultAdapter[U]{result: result}, nil
}

func (a genericStepAdapter[T, U]) AddPublishedTopic(publisher message.Publisher, topic string) error {
	return a.step.AddPublishedTopic(publisher, topic)
}

// genericStepResultAdapter converts typed results to work with any type
type genericStepResultAdapter[U any] struct {
	result steps.StepResult[U]
}

func (a genericStepResultAdapter[U]) Return() []helpers.Result[any] {
	typedResults := a.result.Return()
	genericResults := make([]helpers.Result[any], len(typedResults))
	for i, tr := range typedResults {
		if tr.Error() != nil {
			genericResults[i] = helpers.NewErrorResult[any](tr.Error())
		} else {
			genericResults[i] = helpers.NewValueResult[any](tr.Unwrap())
		}
	}
	return genericResults
}

func (a genericStepResultAdapter[U]) GetChannel() <-chan helpers.Result[any] {
	typedCh := a.result.GetChannel()
	genericCh := make(chan helpers.Result[any])

	go func() {
		defer close(genericCh)
		for tr := range typedCh {
			if tr.Error() != nil {
				genericCh <- helpers.NewErrorResult[any](tr.Error())
			} else {
				genericCh <- helpers.NewValueResult[any](tr.Unwrap())
			}
		}
	}()

	return genericCh
}

func (a genericStepResultAdapter[U]) Cancel() {
	a.result.Cancel()
}

func (a genericStepResultAdapter[U]) GetMetadata() *steps.StepMetadata {
	return a.result.GetMetadata()
}

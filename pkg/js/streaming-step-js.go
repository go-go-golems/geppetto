package js

import (
	"context"
	"fmt"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/dop251/goja"
	"github.com/go-go-golems/geppetto/pkg/helpers"
	"github.com/go-go-golems/geppetto/pkg/steps"
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
	log.Debug().Str("stepType", fmt.Sprintf("%T", step)).Msg("Creating watermill step object factory")

	return func(vm *goja.Runtime) *goja.Object {
		log.Debug().Msg("Creating step object in VM context")
		stepObj := vm.NewObject()

		// Add the watermill-based streaming method
		log.Debug().Msg("Adding runWithEvents method to step object")
		err := stepObj.Set("runWithEvents", func(call goja.FunctionCall) goja.Value {
			log.Debug().Msg("runWithEvents called")
			if len(call.Arguments) < 2 {
				log.Error().Msg("runWithEvents requires input and onEvent callback")
				panic(vm.NewTypeError("runWithEvents requires input and onEvent callback"))
			}

			input := inputConverter(call.Arguments[0])
			log.Debug().Interface("input", input).Msg("Input converted")

			onEvent, ok := goja.AssertFunction(call.Arguments[1])
			if !ok {
				log.Error().Msg("onEvent must be a function")
				panic(vm.NewTypeError("onEvent must be a function"))
			}
			log.Debug().Msg("onEvent callback validated")

			// Create a generic step that can be used with RunStep
			log.Debug().Msg("Creating generic step wrapper")
			genericStep := &GenericStepWrapper[T, U]{step: step}

			log.Debug().Msg("Calling engine.RunStep")
			stepID := engine.RunStep(genericStep, input, onEvent)
			log.Debug().Str("stepID", stepID).Msg("RunStep completed")

			return vm.ToValue(stepID)
		})
		if err != nil {
			log.Error().Err(err).Msg("Failed to set runWithEvents method")
			panic(vm.NewGoError(err))
		}

		log.Debug().Msg("Watermill step object created successfully")
		return stepObj
	}
}

// GenericStepWrapper wraps a typed step to make it compatible with RunStep's any type
type GenericStepWrapper[T any, U any] struct {
	step steps.Step[T, U]
}

func (w *GenericStepWrapper[T, U]) Start(ctx context.Context, input any) (steps.StepResult[any], error) {
	typedInput, ok := input.(T)
	if !ok {
		return nil, fmt.Errorf("input type mismatch: expected %T, got %T", *new(T), input)
	}

	result, err := w.step.Start(ctx, typedInput)
	if err != nil {
		return nil, err
	}

	return &GenericStepResultWrapper[U]{result: result}, nil
}

func (w *GenericStepWrapper[T, U]) AddPublishedTopic(publisher message.Publisher, topic string) error {
	return w.step.AddPublishedTopic(publisher, topic)
}

// GenericStepResultWrapper wraps a typed step result to make it compatible with any type
type GenericStepResultWrapper[U any] struct {
	result steps.StepResult[U]
}

func (w *GenericStepResultWrapper[U]) Return() []helpers.Result[any] {
	typedResults := w.result.Return()
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

func (w *GenericStepResultWrapper[U]) GetChannel() <-chan helpers.Result[any] {
	typedCh := w.result.GetChannel()
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

func (w *GenericStepResultWrapper[U]) Cancel() {
	w.result.Cancel()
}

func (w *GenericStepResultWrapper[U]) GetMetadata() *steps.StepMetadata {
	return w.result.GetMetadata()
}

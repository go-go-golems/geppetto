package steps

import (
	"context"
	"github.com/go-go-golems/geppetto/pkg/helpers"
	"github.com/rs/zerolog/log"
)

type StepResult[T any] struct {
	value <-chan helpers.Result[T]
	// Additional monads:
	// - metrics
	// - logs (? maybe doing imperative logging is better,
	//   although they should definitely be collected as part of plunger)
}

func NewStepResult[T any](value <-chan helpers.Result[T]) *StepResult[T] {
	return &StepResult[T]{value: value}
}

func (m *StepResult[T]) Return() []helpers.Result[T] {
	res := []helpers.Result[T]{}
	for r := range m.value {
		res = append(res, r)
	}
	return res
}

func (m *StepResult[T]) GetChannel() <-chan helpers.Result[T] {
	return m.value
}

func Resolve[T any](value T) *StepResult[T] {
	c := make(chan helpers.Result[T], 1)
	c <- helpers.NewValueResult[T](value)
	close(c)
	return &StepResult[T]{
		value: c,
	}
}

func ResolveNone[T any]() *StepResult[T] {
	c := make(chan helpers.Result[T], 1)
	close(c)
	return &StepResult[T]{
		value: c,
	}
}

func Reject[T any](err error) *StepResult[T] {
	c := make(chan helpers.Result[T], 1)
	c <- helpers.NewErrorResult[T](err)
	close(c)
	return &StepResult[T]{
		value: c,
	}
}

// Step is the generalization of a lambda function, with cancellation and closing
// to allow it to own resources.
type Step[T any, U any] interface {
	// Start gets called multiple times for the same Step, once per incoming value,
	// since StepResult is also the list monad (ie., supports multiple values)
	Start(ctx context.Context, input T) (*StepResult[U], error)
	Close(ctx context.Context) error
}

// Bind is the monadic bind operator for StepResult.
// It takes a step result, a step (which is just a lambda turned into a struct)
// iterates over the results in the StepResult, and starts the Step for each
// value.
func Bind[T any, U any](
	ctx context.Context,
	m *StepResult[T],
	step Step[T, U],
) *StepResult[U] {
	return &StepResult[U]{
		value: func() <-chan helpers.Result[U] {
			c := make(chan helpers.Result[U])
			go func() {
				defer close(c)
				defer func(transformer Step[T, U], ctx context.Context) {
					err := transformer.Close(ctx)
					if err != nil {
						log.Error().Err(err).Msg("error closing step")
					}
				}(step, ctx)
				for {
					select {
					case r, ok := <-m.value:
						if !ok {
							return
						}
						if r.Error() != nil {
							// we do need to drain m here
							c <- helpers.NewErrorResult[U](r.Error())
							continue
						}

						c_, err := step.Start(ctx, r.Unwrap())
						if err != nil {
							c <- helpers.NewErrorResult[U](err)
							return
						}
						for u := range c_.value {
							c <- u
						}
					case <-ctx.Done():
						return
					}
				}
			}()
			return c
		}(),
	}
}

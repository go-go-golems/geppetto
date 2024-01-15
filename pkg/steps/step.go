package steps

import (
	"context"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/go-go-golems/geppetto/pkg/helpers"
	"github.com/google/uuid"
)

type StepResult[T any] interface {
	Return() []helpers.Result[T]
	GetChannel() <-chan helpers.Result[T]
	// Cancel can't fail
	Cancel()
	GetMetadata() *StepMetadata
}

type StepMetadata struct {
	StepID     uuid.UUID `json:"step_id"`
	Type       string    `json:"type"`
	InputType  string    `json:"input_type"`
	OutputType string    `json:"output_type"`

	Metadata map[string]interface{} `json:"meta"`
}

const MetadataSettingsSlug = "settings"

type StepResultImpl[T any] struct {
	value        <-chan helpers.Result[T]
	cancel       func()
	metadata     *StepMetadata
	metadataFunc func() *StepMetadata
	// Additional monads:
	// - metrics
	// - logs (? maybe doing imperative logging is better,
	//   although they should definitely be collected as part of plunger)
}

var _ StepResult[string] = &StepResultImpl[string]{}

type StepResultOption[T any] func(*StepResultImpl[T])

func WithCancel[T any](cancel func()) StepResultOption[T] {
	return func(s *StepResultImpl[T]) {
		s.cancel = cancel
	}
}

func WithMetadata[T any](metadata *StepMetadata) StepResultOption[T] {
	return func(s *StepResultImpl[T]) {
		s.metadata = metadata
	}
}

func WithMetadataFunc[T any](metadataFunc func() *StepMetadata) StepResultOption[T] {
	return func(s *StepResultImpl[T]) {
		s.metadataFunc = metadataFunc
	}
}

func NewStepResult[T any](
	value <-chan helpers.Result[T],
	options ...StepResultOption[T],
) *StepResultImpl[T] {
	ret := &StepResultImpl[T]{value: value}

	for _, option := range options {
		option(ret)
	}

	if ret.metadata != nil && ret.metadata.StepID == uuid.Nil {
		ret.metadata.StepID = uuid.New()
	}

	return ret
}

func (m *StepResultImpl[T]) Return() []helpers.Result[T] {
	res := []helpers.Result[T]{}
	for r := range m.value {
		res = append(res, r)
	}
	return res
}

func (m *StepResultImpl[T]) Cancel() {
	if m.cancel != nil {
		m.cancel()
	}
}

func (m *StepResultImpl[T]) GetChannel() <-chan helpers.Result[T] {
	return m.value
}

func (m *StepResultImpl[T]) GetMetadata() *StepMetadata {
	if m.metadataFunc != nil {
		return m.metadataFunc()
	}

	return m.metadata
}

func Resolve[T any](value T, options ...StepResultOption[T]) *StepResultImpl[T] {
	c := make(chan helpers.Result[T], 1)
	c <- helpers.NewValueResult[T](value)
	close(c)
	return NewStepResult[T](c, options...)
}

func ResolveNone[T any](options ...StepResultOption[T]) *StepResultImpl[T] {
	c := make(chan helpers.Result[T], 1)
	close(c)
	return NewStepResult[T](c, options...)
}

func Reject[T any](err error, options ...StepResultOption[T]) *StepResultImpl[T] {
	c := make(chan helpers.Result[T], 1)
	c <- helpers.NewErrorResult[T](err)
	close(c)
	return NewStepResult[T](c, options...)
}

type StepFactory[T any, U any] interface {
	NewStep() (Step[T, U], error)
}

type NewStepFunc[T any, U any] func() (Step[T, U], error)

func (f NewStepFunc[T, U]) NewStep() (Step[T, U], error) {
	return f()
}

// Step is the generalization of a lambda function, with cancellation and closing
// to allow it to own resources.
type Step[T any, U any] interface {
	// Start gets called multiple times for the same Step, once per incoming value,
	// since StepResult is also the list monad (ie., supports multiple values)
	Start(ctx context.Context, input T) (StepResult[U], error)
	AddPublishedTopic(publisher message.Publisher, topic string) error
}

// Bind is the monadic bind operator for StepResult.
// It takes a step result, a step (which is just a lambda turned into a struct)
// iterates over the results in the StepResult, and starts the Step for each
// value.
func Bind[T any, U any](
	ctx context.Context,
	m StepResult[T],
	step Step[T, U],
) StepResult[U] {
	ctx, cancel := context.WithCancel(ctx)
	return NewStepResult[U](
		func() <-chan helpers.Result[U] {
			c := make(chan helpers.Result[U])
			go func() {
				defer close(c)
				for {
					// TODO(manuel, 2023-12-06) The way we handle streaming by calling step.Start on each partial result
					// without eeven telling the step that this is a partial result is not great.
					select {
					case r, ok := <-m.GetChannel():
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
						for u := range c_.GetChannel() {
							c <- u
						}
					case <-ctx.Done():
						return
					}
				}
			}()
			return c
		}(),
		WithCancel[U](cancel))
}

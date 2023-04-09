package steps

import (
	"context"
	"fmt"
	"github.com/go-go-golems/geppetto/pkg/helpers"
	"golang.org/x/sync/errgroup"
	"gopkg.in/errgo.v2/fmt/errors"
)

// Step represents one step in a generic pipeline
type Step[A, B any] interface {
	// Run starts the step and blocks until the end
	Run(ctx context.Context, a A) error
	GetOutput() <-chan helpers.Result[B]
	GetState() interface{}
	IsFinished() bool
}

type GenericStepFactory interface {
	// TODO(manuel, 2023-02-27) This is probably updateFromLayers, and each factory should be able to register its own layers
	UpdateFromParameters(ps map[string]interface{}) error
}

type StepFactory[A, B any] interface {
	NewStep() (Step[A, B], error)
}

type SimpleStepState int

const (
	SimpleStepNotStarted SimpleStepState = iota
	SimpleStepRunning
	SimpleStepFinished
	SimpleStepClosed
)

type SimpleStep[A, B any] struct {
	stepFunction func(A) helpers.Result[B]
	output       chan helpers.Result[B]
	state        SimpleStepState
}

func (s *SimpleStep[A, B]) Run(_ context.Context, a A) error {
	if s.state != SimpleStepNotStarted {
		return errors.Newf("step already started")
	}
	s.state = SimpleStepRunning

	v := s.stepFunction(a)
	s.state = SimpleStepFinished
	s.output <- v
	defer func() {
		s.state = SimpleStepClosed
		close(s.output)
	}()

	return nil
}

func (s *SimpleStep[A, B]) GetOutput() <-chan helpers.Result[B] {
	return s.output
}

func (s *SimpleStep[A, B]) GetState() interface{} {
	return s.state
}

func (s *SimpleStep[A, B]) IsFinished() bool {
	return s.state == SimpleStepFinished
}

func NewSimpleStep[A any, B any](f func(A) B) Step[A, B] {
	s := &SimpleStep[A, B]{
		stepFunction: func(a A) helpers.Result[B] {
			return helpers.NewValueResult(f(a))
		},
		output: make(chan helpers.Result[B]),
		state:  SimpleStepNotStarted,
	}
	return s
}

type PipeStepState int

const (
	PipeStepNotStarted   PipeStepState = iota
	PipeStepRunningStep1               // step 1 is running
	PipeStepRunningStep2               // step 2 is running
	PipeStepFinished
	PipeStepError
	PipeStepClosed
)

type PipeStep[A, B, C any] struct {
	state  PipeStepState
	step1  Step[A, B]
	step2  Step[B, C]
	output chan helpers.Result[C]
}

// TODO(manuel, 2023-02-04) The pipe step should actually take a factory for the second step
// Other wise it's just a simple functional pipe

func (s *PipeStep[A, B, C]) Run(ctx context.Context, a A) error {
	if s.state != PipeStepNotStarted {
		return errors.Newf("step already started")
	}

	s.state = PipeStepRunningStep1

	eg, ctx2 := errgroup.WithContext(ctx)

	eg.Go(func() error {
		return s.step1.Run(ctx2, a)
	})

	// NOTE(manuel, 2023-02-04) This can probably be done more elegantly
	eg.Go(func() error {
		defer func() {
			s.state = PipeStepClosed
			close(s.output)
		}()
		v_, ok := <-s.step1.GetOutput()
		if !ok {
			s.state = PipeStepError
			s.output <- helpers.NewErrorResult[C](errors.Newf("step 1 closed output channel"))
			return nil
		}
		v, err := v_.Value()
		if err != nil {
			s.state = PipeStepError
			s.output <- helpers.NewErrorResult[C](err)
			return nil
		}

		if ctx.Err() != nil {
			s.state = PipeStepError
			s.output <- helpers.NewErrorResult[C](ctx.Err())
			return nil
		}

		s.state = PipeStepRunningStep2
		fmt.Println("Starting step2")

		eg2, ctx3 := errgroup.WithContext(ctx2)
		eg2.Go(func() error {
			return s.step2.Run(ctx3, v)
		})

		// NOTE(manuel, 2023-02-04) Maybe this error handling can be done more elegantly by handling the result
		// of eg2.Wait()
		eg2.Go(func() error {
			select {
			case <-ctx3.Done():
				s.state = PipeStepError
				s.output <- helpers.NewErrorResult[C](ctx3.Err())
				return nil
			case v2_, ok := <-s.step2.GetOutput():
				if !ok {
					s.state = PipeStepError
					s.output <- helpers.NewErrorResult[C](errors.Newf("step 2 closed output channel"))
					return nil
				}
				v2, err := v2_.Value()
				if err != nil {
					s.state = PipeStepError
					s.output <- helpers.NewErrorResult[C](err)
					return nil
				}

				s.state = PipeStepFinished
				s.output <- helpers.NewValueResult(v2)
			}

			return nil
		})

		return eg2.Wait()
	})

	return eg.Wait()
}

func (s *PipeStep[A, B, C]) GetOutput() <-chan helpers.Result[C] {
	return s.output
}

func (s *PipeStep[A, B, C]) GetState() interface{} {
	return s.state
}

func (s *PipeStep[A, B, C]) IsFinished() bool {
	return s.state == PipeStepFinished
}

func NewPipeStep[A, B, C any](step1 Step[A, B], step2 Step[B, C]) Step[A, C] {
	s := &PipeStep[A, B, C]{
		state:  PipeStepNotStarted,
		step1:  step1,
		step2:  step2,
		output: make(chan helpers.Result[C]),
	}
	return s
}

var ErrMissingClientSettings = errors.Newf("missing client settings")

var ErrMissingClientAPIKey = errors.Newf("missing client settings api key")

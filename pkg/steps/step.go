package steps

import (
	"context"
	"github.com/wesen/geppetto/pkg/helpers"
	"gopkg.in/errgo.v2/fmt/errors"
)

// Step represents one step in a geppetto pipeline
type Step[A, B any] interface {
	Start(ctx context.Context, a A) error
	GetOutput() <-chan helpers.Result[B]
	GetState() interface{}
	IsFinished() bool
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

func (s *SimpleStep[A, B]) Start(_ context.Context, a A) error {
	if s.state != SimpleStepNotStarted {
		return errors.Newf("step already started")
	}
	s.output = make(chan helpers.Result[B])
	s.state = SimpleStepRunning

	go func() {
		v := s.stepFunction(a)
		s.state = SimpleStepFinished
		s.output <- v
		defer func() {
			s.state = SimpleStepClosed
			close(s.output)
		}()
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
		output: nil,
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
	PipeStepClosed
)

type PipeStep[A, B, C any] struct {
	state  PipeStepState
	step1  Step[A, B]
	step2  Step[B, C]
	output chan helpers.Result[C]
}

func (s *PipeStep[A, B, C]) Start(ctx context.Context, a A) error {
	if s.state != PipeStepNotStarted {
		return errors.Newf("step already started")
	}

	// TODO(manuel, 2023-01-25) Not sure if this shouldn't just be created in the constructor for wiring...
	s.output = make(chan helpers.Result[C])
	s.state = PipeStepRunningStep1

	err := s.step1.Start(ctx, a)
	if err != nil {
		return err
	}
	go func() {
		defer func() {
			s.state = PipeStepClosed
			close(s.output)
		}()
		v_, ok := <-s.step1.GetOutput()
		if !ok {
			s.output <- helpers.NewErrorResult[C](errors.Newf("step 1 closed output channel"))
			return
		}
		v, err := v_.Value()
		if err != nil {
			s.output <- helpers.NewErrorResult[C](err)
			return
		}

		if ctx.Err() != nil {
			s.output <- helpers.NewErrorResult[C](ctx.Err())
			return
		}

		s.state = PipeStepRunningStep2
		err = s.step2.Start(ctx, v)
		if err != nil {
			s.output <- helpers.NewErrorResult[C](err)
			return
		}
		v2_, ok := <-s.step2.GetOutput()
		if !ok {
			s.output <- helpers.NewErrorResult[C](errors.Newf("step 2 closed output channel"))
			return
		}
		v2, err := v2_.Value()
		if err != nil {
			s.output <- helpers.NewErrorResult[C](err)
			return
		}

		s.state = PipeStepFinished
		s.output <- helpers.NewValueResult(v2)
	}()
	return nil
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
		output: nil,
	}
	return s
}

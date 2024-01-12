package utils

import (
	"context"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/go-go-golems/geppetto/pkg/helpers"
	"github.com/go-go-golems/geppetto/pkg/steps"
	"sync"
)

// TODO(manuel, 2024-01-12) Handle functions taking ctx as first argument

// LambdaStep is a struct that wraps a function to be used as a step in a pipeline.
// The function takes an input and returns a Result.
type LambdaStep[Input any, Output any] struct {
	Function func(Input) helpers.Result[Output]
}

var _ steps.Step[string, float64] = &LambdaStep[string, float64]{}

// BackgroundLambdaStep is a struct that wraps a function to be used as a step in a pipeline.
// The function takes a context and an input, and returns a Result. The function is executed in a separate goroutine.
type BackgroundLambdaStep[Input any, Output any] struct {
	Function func(context.Context, Input) helpers.Result[Output]
	wg       sync.WaitGroup
	c        chan helpers.Result[Output]
}

var _ steps.Step[string, float64] = &BackgroundLambdaStep[string, float64]{}

// MapLambdaStep is a struct that wraps a function to be used as a step in a pipeline.
// The function takes an input and returns a Result. The function is applied to each element of an input slice.
type MapLambdaStep[Input any, Output any] struct {
	Function func(Input) helpers.Result[Output]
}

var _ steps.Step[[]string, float64] = &MapLambdaStep[string, float64]{}

// BackgroundMapLambdaStep is a struct that wraps a function to be used as a step in a pipeline.
// The function takes a context and an input, and returns a Result. The function is applied to each element of an input slice in a separate goroutine.
type BackgroundMapLambdaStep[Input any, Output any] struct {
	Function func(context.Context, Input) helpers.Result[Output]
	wg       sync.WaitGroup
	c        chan helpers.Result[Output]
}

var _ steps.Step[[]string, float64] = &BackgroundMapLambdaStep[string, float64]{}

func (l *LambdaStep[Input, Output]) Start(ctx context.Context, input Input) (steps.StepResult[Output], error) {
	c := make(chan helpers.Result[Output], 1)
	defer close(c)

	c <- l.Function(input)
	return steps.NewStepResult[Output](c), nil
}

func (r *LambdaStep[Input, Output]) AddPublishedTopic(publisher message.Publisher, topic string) error {
	return nil
}

func (l *BackgroundLambdaStep[Input, Output]) Start(ctx context.Context, input Input) (steps.StepResult[Output], error) {
	l.c = make(chan helpers.Result[Output], 1)

	ctx, cancel := context.WithCancel(ctx)

	l.wg.Add(1)
	go func() {
		defer l.wg.Done()
		l.c <- l.Function(ctx, input)
	}()

	return steps.NewStepResult[Output](
		l.c,
		steps.WithCancel[Output](cancel),
	), nil
}

func (r *BackgroundLambdaStep[Input, Output]) AddPublishedTopic(publisher message.Publisher, topic string) error {
	return nil
}
func (l *BackgroundLambdaStep[Input, Output]) Close(ctx context.Context) error {
	defer close(l.c)
	l.wg.Wait()
	return nil
}

func (l *MapLambdaStep[Input, Output]) Start(ctx context.Context, input []Input) (steps.StepResult[Output], error) {
	c := make(chan helpers.Result[Output], len(input))
	defer close(c)

	for _, in := range input {
		o := l.Function(in)
		c <- o

	}

	return steps.NewStepResult[Output](c), nil
}

func (r *MapLambdaStep[Input, Output]) AddPublishedTopic(publisher message.Publisher, topic string) error {
	return nil
}

func (l *BackgroundMapLambdaStep[Input, Output]) Start(ctx context.Context, input []Input) (steps.StepResult[Output], error) {
	l.c = make(chan helpers.Result[Output], len(input))

	ctx, cancel := context.WithCancel(ctx)

	l.wg.Add(1)
	go func() {
		defer close(l.c)
		defer l.wg.Done()
		for _, in := range input {
			l.c <- l.Function(ctx, in)
		}
	}()

	return steps.NewStepResult[Output](
		l.c,
		steps.WithCancel[Output](cancel),
	), nil
}

func (r *BackgroundMapLambdaStep[Input, Output]) AddPublishedTopic(publisher message.Publisher, topic string) error {
	return nil
}

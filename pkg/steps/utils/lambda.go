package utils

import (
	"context"
	"github.com/go-go-golems/geppetto/pkg/helpers"
	"github.com/go-go-golems/geppetto/pkg/steps"
	"sync"
)

type LambdaStep[Input any, Output any] struct {
	Function func(Input) helpers.Result[Output]
}

func (l *LambdaStep[Input, Output]) Start(ctx context.Context, input Input) (*steps.StepResult[Output], error) {
	c := make(chan helpers.Result[Output], 1)
	defer close(c)

	c <- l.Function(input)
	return steps.NewStepResult[Output](c), nil
}

func (l *LambdaStep[Input, Output]) Close(ctx context.Context) error {
	return nil
}

type BackgroundLambdaStep[Input any, Output any] struct {
	Function func(context.Context, Input) helpers.Result[Output]
	wg       sync.WaitGroup
	c        chan helpers.Result[Output]
}

func (l *BackgroundLambdaStep[Input, Output]) Start(ctx context.Context, input Input) (*steps.StepResult[Output], error) {
	l.c = make(chan helpers.Result[Output], 1)

	l.wg.Add(1)
	go func() {
		defer l.wg.Done()
		l.c <- l.Function(ctx, input)
	}()

	return steps.NewStepResult[Output](l.c), nil
}

func (l *BackgroundLambdaStep[Input, Output]) Close(ctx context.Context) error {
	defer close(l.c)
	l.wg.Wait()
	return nil
}

type MapLambdaStep[Input any, Output any] struct {
	Function func(Input) helpers.Result[Output]
}

func (l *MapLambdaStep[Input, Output]) Start(ctx context.Context, input []Input) (*steps.StepResult[Output], error) {
	c := make(chan helpers.Result[Output], len(input))
	defer close(c)

	for _, in := range input {
		o := l.Function(in)
		c <- o

	}

	return steps.NewStepResult[Output](c), nil
}

func (l *MapLambdaStep[Input, Output]) Close(ctx context.Context) error {
	return nil
}

type BackgroundMapLambdaStep[Input any, Output any] struct {
	Function func(context.Context, Input) helpers.Result[Output]
	wg       sync.WaitGroup
	c        chan helpers.Result[Output]
}

func (l *BackgroundMapLambdaStep[Input, Output]) Start(ctx context.Context, input []Input) (*steps.StepResult[Output], error) {
	l.c = make(chan helpers.Result[Output], len(input))

	l.wg.Add(1)
	go func() {
		defer close(l.c)
		defer l.wg.Done()
		for _, in := range input {
			l.c <- l.Function(ctx, in)
		}
	}()

	return steps.NewStepResult[Output](l.c), nil
}

func (l *BackgroundMapLambdaStep[Input, Output]) Close(ctx context.Context) error {
	l.wg.Wait()
	return nil
}

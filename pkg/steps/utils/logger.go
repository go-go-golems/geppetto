package utils

import (
	"context"
	"fmt"
	"github.com/go-go-golems/geppetto/pkg/steps"
)

// LoggerStep is a step that logs its input to stdout.
type LoggerStep[T any] struct{}

var _ steps.Step[string, string] = &LoggerStep[string]{}

// Start implements the Step interface for LoggerStep.
// It prints the input to stdout and returns a StepResult with the same value.
func (ls *LoggerStep[T]) Start(ctx context.Context, input T) (steps.StepResult[T], error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		fmt.Println(input)               // Print the input to stdout
		return steps.Resolve(input), nil // Return the input as a successful result
	}
}

//type CollectingStep[T any] struct {
//	merge     func(T, T) (T, error)
//	onPartial func(T) error
//	collect   T
//}
//
//var _ steps.Step[string, string] = &CollectingStep[string]{}
//
//func NewCollectingStep[T any](merge func(T, T) (T, error), onPartial func(T) error, initial T) *CollectingStep[T] {
//	return &CollectingStep[T]{
//		merge:     merge,
//		onPartial: onPartial,
//		collect:   initial,
//	}
//}
//
//func (c *CollectingStep[T]) Start(ctx context.Context, input T) (steps.StepResult[T], error) {
//}

package utils

import (
	"context"
	"github.com/go-go-golems/geppetto/pkg/helpers"
	"github.com/go-go-golems/geppetto/pkg/steps"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestLambdaStep(t *testing.T) {
	input := "testing-input"
	expected := "testing-result"

	// Test the basic step execution.
	step := &LambdaStep[string, string]{
		Function: func(input string) helpers.Result[string] { return helpers.NewValueResult(expected) },
	}
	result, err := step.Start(context.Background(), input)
	assert.NoError(t, err)

	resValues := result.Return()
	assert.Len(t, resValues, 1) // make sure there is only one value
	assert.Equal(t, expected, resValues[0].Unwrap())

	// Test with the function execution error
	stepErr := &LambdaStep[string, string]{
		Function: func(input string) helpers.Result[string] {
			return helpers.NewErrorResult[string](errors.New("Test Error"))
		},
	}
	resultErr, err := stepErr.Start(context.Background(), input)
	assert.NoError(t, err)

	resErrValues := resultErr.Return()
	assert.Len(t, resErrValues, 1) // make sure there is only one value
	assert.Error(t, resErrValues[0].Error())
}

func TestBackgroundLambdaStep(t *testing.T) {
	t.Run("it should run lambda in the background", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		// make a step that waits 300 ms before returning

		step := &BackgroundLambdaStep[string, string]{
			Function: func(ctx context.Context, input string) helpers.Result[string] {
				// sleep for 300 ms with cancellation
				select {
				case <-time.After(time.Millisecond * 300):
					return helpers.NewValueResult("Hello, " + input)
				case <-ctx.Done():
					return helpers.NewErrorResult[string](ctx.Err())
				}
			},
		}

		result, err := step.Start(ctx, "world")
		assert.NoError(t, err)
		assert.NotNil(t, result)

		// let some time pass to allow context cancellation,
		// verification of background execution of the lambda.
		select {
		case <-time.After(time.Millisecond * 100):
			// No interaction with the result channel until now.
			assert.Equal(t, 0, len(result.GetChannel()))
		case <-ctx.Done():
			t.Fatal("unexpected background lambda completion")
		}

		// read the Result after context cancellation
		select {
		case res := <-result.GetChannel():
			assert.Equal(t, helpers.NewValueResult[string]("Hello, world"), res)
		case <-ctx.Done():
			t.Error("unable to read the result after context cancellation")
		}
	})

	t.Run("it should return an error if lambda returns an error result", func(t *testing.T) {
		// use a step with a function that returns an error
		stepErr := &BackgroundLambdaStep[string, string]{
			Function: func(ctx context.Context, input string) helpers.Result[string] {
				return helpers.NewErrorResult[string](errors.New("dummy error"))
			},
		}

		result, err := stepErr.Start(context.Background(), "irrelevant")
		assert.NoError(t, err)
		assert.NotNil(t, result)

		// read the Result
		res := <-result.GetChannel()
		assert.Error(t, res.Error())

		err = stepErr.Close(context.Background())
		assert.NoError(t, err)
	})

	t.Run("it should wait for all background goroutines to complete in Close method", func(t *testing.T) {
		step := &BackgroundLambdaStep[string, string]{
			Function: func(ctx context.Context, input string) helpers.Result[string] {
				time.Sleep(500 * time.Millisecond)
				return helpers.NewValueResult("Hello, " + input)
			},
		}

		_, err := step.Start(context.Background(), "world")
		assert.NoError(t, err)

		t0 := time.Now()
		_ = step.Close(context.Background())
		t1 := time.Now()
		// should wait at least 500ms
		assert.True(t, t1.Sub(t0) >= 500*time.Millisecond)
	})

	t.Run("it should return an error if context is cancelled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		step := &BackgroundLambdaStep[string, string]{
			Function: func(ctx context.Context, input string) helpers.Result[string] {
				time.Sleep(500 * time.Millisecond)
				return helpers.NewValueResult("Hello, " + input)
			},
		}

		_, err := step.Start(ctx, "world")
		assert.NoError(t, err)

		cancel()
		_ = step.Close(context.Background())
	})
}

func TestMapLambdaStep(t *testing.T) {
	// Test setup
	inputs := []string{"input1", "input2", "input3"}
	expectedResults := []string{"result1", "result2", "result3"}

	step := &MapLambdaStep[string, string]{
		Function: func(in string) helpers.Result[string] {
			for i, input := range inputs {
				if in == input {
					return helpers.NewValueResult(expectedResults[i])
				}
			}
			return helpers.NewErrorResult[string](errors.New("Error in function"))
		},
	}

	// Test with proper function execution
	result, err := step.Start(context.Background(), inputs)
	assert.NoError(t, err)

	resValues := result.Return()
	assert.Len(t, resValues, len(inputs)) // make sure all results are there
	for i, val := range resValues {
		assert.Equal(t, expectedResults[i], val.Unwrap())
	}

	// Test with empty input slice
	result, err = step.Start(context.Background(), []string{})
	assert.NoError(t, err)

	resValues = result.Return()
	assert.Empty(t, resValues) // make sure no result is returned

	// Test with function error
	stepErr := &MapLambdaStep[string, string]{
		Function: func(input string) helpers.Result[string] {
			return helpers.NewErrorResult[string](errors.New("Test Error"))
		},
	}
	resultErr, err := stepErr.Start(context.Background(), inputs)
	assert.NoError(t, err)

	resErrValues := resultErr.Return()
	for _, val := range resErrValues {
		assert.Error(t, val.Error())
		assert.Contains(t, val.Error().Error(), "Test Error")
	}
}

func TestBackgroundMapLambdaStep(t *testing.T) {
	t.Run("Start method correctly starts a new goroutine for each input value and sends the result of the Function to the channel", func(t *testing.T) {
		step := &BackgroundMapLambdaStep[int, int]{
			Function: func(ctx context.Context, input int) helpers.Result[int] {
				return helpers.NewValueResult(input * 2)
			},
		}

		result, err := step.Start(context.Background(), []int{1, 2, 3})
		assert.NoError(t, err)
		assert.NotNil(t, result)

		results := result.Return()
		assert.Equal(t, 3, len(results))
		assert.Equal(t, 2, results[0].Unwrap())
		assert.Equal(t, 4, results[1].Unwrap())
		assert.Equal(t, 6, results[2].Unwrap())
	})

	t.Run("Start method correctly returns a pointer to a StepResult with the channel", func(t *testing.T) {
		step := &BackgroundMapLambdaStep[int, int]{
			Function: func(ctx context.Context, input int) helpers.Result[int] {
				return helpers.NewValueResult(input * 2)
			},
		}

		result, err := step.Start(context.Background(), []int{1, 2, 3})
		assert.NoError(t, err)
		assert.NotNil(t, result)

		assert.Equal(t, 3, len(result.Return()))
	})

	t.Run("Start method returns an error if the Function returns an error", func(t *testing.T) {
		step := &BackgroundMapLambdaStep[int, int]{
			Function: func(ctx context.Context, input int) helpers.Result[int] {
				return helpers.NewErrorResult[int](errors.New("dummy error"))
			},
		}

		result, err := step.Start(context.Background(), []int{1, 2, 3})
		assert.NoError(t, err)
		assert.NotNil(t, result)

		results := result.Return()
		assert.Equal(t, 3, len(results))
		assert.Error(t, results[0].Error())
		assert.Error(t, results[1].Error())
		assert.Error(t, results[2].Error())
	})

	t.Run("Function is correctly called with the context and input value", func(t *testing.T) {
		var receivedInput int
		step := &BackgroundMapLambdaStep[int, int]{
			Function: func(ctx context.Context, input int) helpers.Result[int] {
				receivedInput = input
				return helpers.NewValueResult(input * 2)
			},
		}

		res, err := step.Start(context.Background(), []int{1})
		require.NoError(t, err)

		_ = res.Return()

		assert.Equal(t, 1, receivedInput)
	})

	t.Run("Function correctly returns a Result with the value and error", func(t *testing.T) {
		step := &BackgroundMapLambdaStep[int, int]{
			Function: func(ctx context.Context, input int) helpers.Result[int] {
				if input == 1 {
					return helpers.NewValueResult(input * 2)
				} else {
					return helpers.NewErrorResult[int](errors.New("dummy error"))
				}
			},
		}

		result, err := step.Start(context.Background(), []int{1, 2})
		assert.NoError(t, err)
		assert.NotNil(t, result)

		results := result.Return()
		assert.Equal(t, 2, len(results))
		assert.Equal(t, 2, results[0].Unwrap())
		assert.Error(t, results[1].Error())
	})

	t.Run("Function correctly handles a cancelled context", func(t *testing.T) {
		step := &BackgroundMapLambdaStep[int, int]{
			Function: func(ctx context.Context, input int) helpers.Result[int] {
				select {
				case <-ctx.Done():
					return helpers.NewErrorResult[int](ctx.Err())
				default:
					return helpers.NewValueResult(input * 2)
				}
			},
		}

		ctx, cancel := context.WithCancel(context.Background())
		result, err := step.Start(ctx, []int{1})
		assert.NoError(t, err)
		assert.NotNil(t, result)

		cancel()
		results := result.Return()
		assert.Equal(t, 1, len(results))
		assert.Error(t, results[0].Error())
	})
}

func TestBindLambdas(t *testing.T) {
	t.Run("it should chain two steps together", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		step1 := &LambdaStep[int, int]{
			Function: func(input int) helpers.Result[int] {
				return helpers.NewValueResult(input * 2)
			},
		}
		step2 := &LambdaStep[int, int]{
			Function: func(input int) helpers.Result[int] {
				return helpers.NewValueResult(input + 1)
			},
		}

		result1, _ := step1.Start(ctx, 2)
		result2 := steps.Bind[int, int](ctx, result1, step2)

		res := <-result2.GetChannel()
		assert.Equal(t, 5, res.Unwrap())
	})

}

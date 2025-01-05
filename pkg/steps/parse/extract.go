package parse

import (
	"context"
	"github.com/go-go-golems/geppetto/pkg/helpers"
	"github.com/go-go-golems/geppetto/pkg/steps"
	"github.com/go-go-golems/glazed/pkg/helpers/json"
	"github.com/go-go-golems/glazed/pkg/helpers/markdown"
)

// ExtractJSONStep is a step that extracts JSON blocks from the given string.
type ExtractJSONStep struct{}

// chat.Step : conversation -> message
// json.Step : string -> []string
// json.validator: []string -> (error, []string)

// bind(chat, json, validate) : conversation -> (error, []string)

func (e *ExtractJSONStep) Start(ctx context.Context, input string) (steps.StepResult[[]string], error) {
	c := make(chan helpers.Result[[]string], 1)
	defer close(c)
	jsonBlocks := json.ExtractJSON(input)
	c <- helpers.NewValueResult[[]string](jsonBlocks)

	return steps.NewStepResult[[]string](c), nil
}

type ExtractCodeBlocksStep struct{}

func (e *ExtractCodeBlocksStep) Start(ctx context.Context, input string) (steps.StepResult[[]string], error) {
	c := make(chan helpers.Result[[]string], 1)
	defer close(c)
	codeBlocks := markdown.ExtractQuotedBlocks(input, false)
	c <- helpers.NewValueResult[[]string](codeBlocks)

	return steps.NewStepResult[[]string](c), nil
}

package steps

import (
	"context"
	"github.com/PullRequestInc/go-gpt3"
	"gopkg.in/errgo.v2/fmt/errors"
)

type OpenAICompletionStepState int

const (
	OpenAICompletionStepNotStarted OpenAICompletionStepState = iota
	OpenAICompletionStepRunning
	OpenAICompletionStepFinished
	OpenAICompletionStepClosed
)

type OpenAICompletionStep struct {
	output chan Result[string]
	state  OpenAICompletionStepState
	apiKey string
}

func NewOpenAICompletionStep(apiKey string) *OpenAICompletionStep {
	return &OpenAICompletionStep{
		output: nil,
		apiKey: apiKey,
		state:  OpenAICompletionStepNotStarted,
	}
}

func (o *OpenAICompletionStep) Start(ctx context.Context, prompt string) error {
	o.output = make(chan Result[string])

	o.state = OpenAICompletionStepRunning
	go func() {
		defer func() {
			o.state = OpenAICompletionStepClosed
			close(o.output)
		}()

		client := gpt3.NewClient(o.apiKey)
		completion, err := client.Completion(ctx, gpt3.CompletionRequest{
			Prompt: []string{prompt},
		})
		o.state = OpenAICompletionStepFinished

		if err != nil {
			o.output <- Result[string]{err: err}
			return
		}

		if len(completion.Choices) == 0 {
			o.output <- Result[string]{err: errors.Newf("no choices returned from OpenAI")}
			return
		}

		o.output <- Result[string]{value: completion.Choices[0].Text}
	}()

	return nil
}

func (o *OpenAICompletionStep) GetOutput() <-chan Result[string] {
	return o.output
}

func (o *OpenAICompletionStep) GetState() interface{} {
	return o.state
}

func (o *OpenAICompletionStep) IsFinished() bool {
	return o.state == OpenAICompletionStepFinished
}

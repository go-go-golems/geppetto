package openai

import (
	"context"
	"github.com/PullRequestInc/go-gpt3"
	"github.com/rs/zerolog/log"
	"github.com/wesen/geppetto/pkg/helpers"
	"gopkg.in/errgo.v2/fmt/errors"
)

type CompletionStepState int

const (
	CompletionStepNotStarted CompletionStepState = iota
	CompletionStepRunning
	CompletionStepFinished
	CompletionStepClosed
)

var ErrMissingClientSettings = errors.Newf("missing client settings")

var ErrMissingClientAPIKey = errors.Newf("missing client settings api key")

type CompletionStep struct {
	output   chan helpers.Result[string]
	state    CompletionStepState
	settings *CompletionStepSettings
}

func NewCompletionStep(settings *CompletionStepSettings) *CompletionStep {
	return &CompletionStep{
		output:   nil,
		settings: settings,
		state:    CompletionStepNotStarted,
	}
}

func (o *CompletionStep) Start(ctx context.Context, prompt string) error {
	o.output = make(chan helpers.Result[string])

	o.state = CompletionStepRunning
	go func() {
		defer func() {
			o.state = CompletionStepClosed
			close(o.output)
		}()

		clientSettings := o.settings.ClientSettings
		if clientSettings == nil {
			o.output <- helpers.NewErrorResult[string](ErrMissingClientSettings)
			return
		}

		if clientSettings.APIKey == nil {
			o.output <- helpers.NewErrorResult[string](ErrMissingClientAPIKey)
			return
		}

		client, err := clientSettings.CreateClient()
		if err != nil {
			o.output <- helpers.NewErrorResult[string](err)
			return
		}

		engine := ""
		if o.settings.Engine != nil {
			engine = *o.settings.Engine
		} else if clientSettings.DefaultEngine != nil {
			engine = *clientSettings.DefaultEngine
		} else {
			o.output <- helpers.NewErrorResult[string](errors.Newf("no engine specified"))
			return
		}

		prompts := []string{prompt}

		evt := log.Debug()
		evt = evt.Str("engine", engine)
		if o.settings.MaxResponseTokens != nil {
			evt = evt.Int("max_response_tokens", *o.settings.MaxResponseTokens)
		}
		if o.settings.Temperature != nil {
			evt = evt.Float32("temperature", *o.settings.Temperature)
		}
		if o.settings.TopP != nil {
			evt = evt.Float32("top_p", *o.settings.TopP)
		}
		if o.settings.N != nil {
			evt = evt.Int("n", *o.settings.N)
		}
		if o.settings.LogProbs != nil {
			evt = evt.Int("log_probs", *o.settings.LogProbs)
		}
		if o.settings.Stop != nil {
			evt = evt.Strs("stop", o.settings.Stop)
		}
		evt.Strs("prompts", prompts)
		evt.Msg("sending completion request")

		// TODO(manuel, 2023-01-28) - handle multiple values
		if o.settings.N != nil && *o.settings.N != 1 {
			o.output <- helpers.NewErrorResult[string](errors.Newf("N > 1 is not supported yet"))
			return
		}

		completion := ""

		onData := func(resp *gpt3.CompletionResponse) {
			data := resp.Choices[0].Text
			//fmt.Println("object: %v, choices: %v\n", resp.Object, resp.Choices)
			// TODO(manuel, 2023-02-02) Add stream mode
			// See https://github.com/wesen/geppetto/issues/10
			//
			//fmt.Print(string(data))
			completion += string(data)
		}

		// TODO(manuel, 2023-01-27) This is where we would emit progress status and do some logging
		err = client.CompletionStreamWithEngine(ctx, engine, gpt3.CompletionRequest{
			Prompt:      prompts,
			MaxTokens:   o.settings.MaxResponseTokens,
			Temperature: o.settings.Temperature,
			TopP:        o.settings.TopP,
			N:           o.settings.N,
			LogProbs:    o.settings.LogProbs,
			Echo:        false,
			Stop:        o.settings.Stop,
		}, onData)
		o.state = CompletionStepFinished

		if err != nil {
			o.output <- helpers.NewErrorResult[string](err)
			return
		}

		// TODO(manuel, 2023-02-04) Handle multiple outputs
		// See https://github.com/wesen/geppetto/issues/23

		//if len(completion.Choices) == 0 {
		//	o.output <- helpers.NewErrorResult[string](errors.Newf("no choices returned from OpenAI"))
		//	return
		//}

		o.output <- helpers.NewValueResult(completion)
	}()

	return nil
}

func (o *CompletionStep) GetOutput() <-chan helpers.Result[string] {
	return o.output
}

func (o *CompletionStep) GetState() interface{} {
	return o.state
}

func (o *CompletionStep) IsFinished() bool {
	return o.state == CompletionStepFinished
}

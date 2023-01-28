package openai

import (
	"context"
	"github.com/PullRequestInc/go-gpt3"
	"github.com/rs/zerolog/log"
	"github.com/wesen/geppetto/pkg/helpers"
	"gopkg.in/errgo.v2/fmt/errors"
	"time"
)

type CompletionStepState int

const (
	CompletionStepNotStarted CompletionStepState = iota
	CompletionStepRunning
	CompletionStepFinished
	CompletionStepClosed
)

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

		evt := log.Info()
		if clientSettings.BaseURL != nil {
			evt = evt.Str("base_url", *clientSettings.BaseURL)
		}
		if clientSettings.DefaultEngine != nil {
			evt = evt.Str("default_engine", *clientSettings.DefaultEngine)
		}
		if clientSettings.Organization != nil {
			evt = evt.Str("organization", *clientSettings.Organization)
		}
		if clientSettings.Timeout != nil {
			// convert timeout to seconds
			timeout := *clientSettings.Timeout / time.Second
			evt = evt.Dur("timeout", timeout)
		}
		if clientSettings.UserAgent != nil {
			evt = evt.Str("user_agent", *clientSettings.UserAgent)
		}
		evt.Msg("creating openai client")

		options := clientSettings.ToOptions()
		client := gpt3.NewClient(*clientSettings.APIKey, options...)

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

		evt = log.Info()
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

		// TODO(manuel, 2023-01-27) This is where we would emit progress status and do some logging
		completion, err := client.CompletionWithEngine(ctx, engine, gpt3.CompletionRequest{
			Prompt:      prompts,
			MaxTokens:   o.settings.MaxResponseTokens,
			Temperature: o.settings.Temperature,
			TopP:        o.settings.TopP,
			N:           o.settings.N,
			LogProbs:    o.settings.LogProbs,
			Echo:        false,
			Stop:        o.settings.Stop,
		})
		o.state = CompletionStepFinished

		if err != nil {
			o.output <- helpers.NewErrorResult[string](err)
			return
		}

		if len(completion.Choices) == 0 {
			o.output <- helpers.NewErrorResult[string](errors.Newf("no choices returned from OpenAI"))
			return
		}

		o.output <- helpers.NewValueResult(completion.Choices[0].Text)
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

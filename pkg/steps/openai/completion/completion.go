package completion

import (
	"context"
	"github.com/PullRequestInc/go-gpt3"
	"github.com/go-go-golems/geppetto/pkg/helpers"
	"github.com/go-go-golems/geppetto/pkg/steps"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
	"gopkg.in/errgo.v2/fmt/errors"
)

type StepState int

const (
	StepNotStarted StepState = iota
	StepRunning
	StepFinished
	StepClosed
)

type Step struct {
	output   chan helpers.Result[string]
	state    StepState
	settings *StepSettings
}

func NewStep(settings *StepSettings) *Step {
	return &Step{
		output:   make(chan helpers.Result[string]),
		settings: settings,
		state:    StepNotStarted,
	}
}

func (s *Step) Run(ctx context.Context, prompt string) error {
	s.state = StepRunning

	defer func() {
		s.state = StepClosed
		close(s.output)
	}()

	clientSettings := s.settings.ClientSettings
	if clientSettings == nil {
		s.output <- helpers.NewErrorResult[string](steps.ErrMissingClientSettings)
		return nil
	}

	if clientSettings.APIKey == nil {
		s.output <- helpers.NewErrorResult[string](steps.ErrMissingClientAPIKey)
		return nil
	}

	client, err := clientSettings.CreateClient()
	if err != nil {
		s.output <- helpers.NewErrorResult[string](err)
		return nil
	}

	engine := ""
	if s.settings.Engine != nil {
		engine = *s.settings.Engine
	} else if clientSettings.DefaultEngine != nil {
		engine = *clientSettings.DefaultEngine
	} else {
		s.output <- helpers.NewErrorResult[string](errors.Newf("no engine specified"))
		return nil
	}

	prompts := []string{prompt}

	evt := log.Debug()
	evt = evt.Str("engine", engine)
	if s.settings.MaxResponseTokens != nil {
		evt = evt.Int("max_response_tokens", *s.settings.MaxResponseTokens)
	}
	if s.settings.Temperature != nil {
		evt = evt.Float32("temperature", *s.settings.Temperature)
	}
	if s.settings.TopP != nil {
		evt = evt.Float32("top_p", *s.settings.TopP)
	}
	if s.settings.N != nil {
		evt = evt.Int("n", *s.settings.N)
	}
	if s.settings.LogProbs != nil {
		evt = evt.Int("log_probs", *s.settings.LogProbs)
	}
	if s.settings.Stop != nil {
		evt = evt.Strs("stop", s.settings.Stop)
	}
	evt.Strs("prompts", prompts)
	evt.Msg("sending completion request")

	// TODO(manuel, 2023-01-28) - handle multiple values
	if s.settings.N != nil && *s.settings.N != 1 {
		s.output <- helpers.NewErrorResult[string](errors.Newf("N > 1 is not supported yet"))
		return nil
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
		MaxTokens:   s.settings.MaxResponseTokens,
		Temperature: s.settings.Temperature,
		TopP:        s.settings.TopP,
		N:           s.settings.N,
		LogProbs:    s.settings.LogProbs,
		Echo:        false,
		Stop:        s.settings.Stop,
	}, onData)
	s.state = StepFinished

	if err != nil {
		s.output <- helpers.NewErrorResult[string](err)
		return nil
	}

	// TODO(manuel, 2023-02-04) Handle multiple outputs
	// See https://github.com/wesen/geppetto/issues/23

	// TODO(manuel, 2023-03-38) Count usage
	// See https://github.com/go-go-golems/geppetto/issues/46

	//if len(completion.Choices) == 0 {
	//	s.output <- helpers.NewErrorResult[string](errors.Newf("no choices returned from OpenAI"))
	//	return
	//}

	s.output <- helpers.NewValueResult(completion)

	return nil
}

func (s *Step) GetOutput() <-chan helpers.Result[string] {
	return s.output
}

func (s *Step) GetState() interface{} {
	return s.state
}

func (s *Step) IsFinished() bool {
	return s.state == StepFinished || s.state == StepClosed
}

// TODO(manuel, 2023-02-04) This could be generic, and take a factory

// MultiCompletionStep runs multiple completion steps in parallel
type MultiCompletionStep struct {
	output   chan helpers.Result[[]string]
	state    StepState
	settings *StepSettings
}

func NewMultiCompletionStep(settings *StepSettings) *MultiCompletionStep {
	return &MultiCompletionStep{
		output:   make(chan helpers.Result[[]string]),
		settings: settings,
		state:    StepNotStarted,
	}
}

func (mc *MultiCompletionStep) Run(ctx context.Context, prompts []string) error {
	completionSteps := make([]*Step, len(prompts))
	chans := make([]<-chan helpers.Result[string], len(prompts))
	for i := range prompts {
		completionSteps[i] = NewStep(mc.settings)
		chans[i] = completionSteps[i].GetOutput()
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	eg, ctx2 := errgroup.WithContext(ctx)

	results := []string{}
	eg.Go(func() error {
		mergeChannel := helpers.MergeChannels(chans...)
		for {
			select {
			case <-ctx2.Done():
				return ctx2.Err()
			case result := <-mergeChannel:
				v, err := result.Value()
				// if we have an error, just store the "" string
				if err != nil {
					v = ""
				}
				results = append(results, v)
			}
		}
	})

	for i, prompt := range prompts {
		j := i
		prompt_ := prompt
		eg.Go(func() error {
			return completionSteps[j].Run(ctx2, prompt_)
		})
	}

	return eg.Wait()
}

func (mc *MultiCompletionStep) GetOutput() <-chan helpers.Result[[]string] {
	return mc.output
}

func (mc *MultiCompletionStep) GetState() interface{} {
	return mc.state
}

func (mc *MultiCompletionStep) IsFinished() bool {
	return mc.state == StepFinished
}

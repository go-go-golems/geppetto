package chat

import (
	"context"
	"github.com/go-go-golems/geppetto/pkg/helpers"
	"github.com/go-go-golems/geppetto/pkg/steps"
	"github.com/pkg/errors"
	"github.com/sashabaranov/go-openai"
	"io"
)

type StepState int

const (
	StepNotStarted StepState = iota
	StepRunning
	StepFinished
	StepClosed
)

type Message struct {
	Role    string `json:"role" yaml:"role"`
	Content string `json:"text" yaml:"text"`
}

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

func (s *Step) Run(ctx context.Context, messages []Message) error {
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

	client := openai.NewClient(*clientSettings.APIKey)

	engine := ""
	if s.settings.Engine != nil {
		engine = *s.settings.Engine
	} else if clientSettings.DefaultEngine != nil {
		engine = *clientSettings.DefaultEngine
	} else {
		s.output <- helpers.NewErrorResult[string](errors.New("no engine specified"))
		return nil
	}

	msgs_ := []openai.ChatCompletionMessage{}
	for _, msg := range messages {
		msgs_ = append(msgs_, openai.ChatCompletionMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	temperature := 0.0
	if s.settings.Temperature != nil {
		temperature = *s.settings.Temperature
	}
	topP := 0.0
	if s.settings.TopP != nil {
		topP = *s.settings.TopP
	}
	maxTokens := 32
	if s.settings.MaxResponseTokens != nil {
		maxTokens = *s.settings.MaxResponseTokens
	}
	n := 1
	if s.settings.N != nil {
		n = *s.settings.N
	}
	stream := s.settings.Stream
	stop := s.settings.Stop
	presencePenalty := 0.0
	if s.settings.PresencePenalty != nil {
		presencePenalty = *s.settings.PresencePenalty
	}
	frequencyPenalty := 0.0
	if s.settings.FrequencyPenalty != nil {
		frequencyPenalty = *s.settings.FrequencyPenalty
	}

	req := openai.ChatCompletionRequest{
		Model:            engine,
		Messages:         msgs_,
		MaxTokens:        maxTokens,
		Temperature:      float32(temperature),
		TopP:             float32(topP),
		N:                n,
		Stream:           stream,
		Stop:             stop,
		PresencePenalty:  float32(presencePenalty),
		FrequencyPenalty: float32(frequencyPenalty),
		// TODO(manuel, 2023-03-28) Properly load logit bias
		// See https://github.com/go-go-golems/geppetto/issues/48
		LogitBias: nil,
	}

	if stream {
		stream, err := client.CreateChatCompletionStream(ctx, req)
		if err != nil {
			s.output <- helpers.NewErrorResult[string](err)
			return nil
		}
		defer stream.Close()

		for {
			response, err := stream.Recv()
			if errors.Is(err, io.EOF) {
				s.state = StepFinished
				s.output <- helpers.NewValueResult[string]("")
				return nil
			}
			if err != nil {
				s.output <- helpers.NewErrorResult[string](err)
				return nil
			}

			s.output <- helpers.NewPartialResult[string](response.Choices[0].Delta.Content)
		}
	} else {
		resp, err := client.CreateChatCompletion(ctx, req)
		s.state = StepFinished

		if err != nil {
			s.output <- helpers.NewErrorResult[string](err)
			return nil
		}

		// TODO(manuel, 2023-03-28) Properly handle message formats
		s.output <- helpers.NewValueResult[string](resp.Choices[0].Message.Content)

	}

	return nil
}

func (s *Step) GetOutput() <-chan helpers.Result[string] {
	return s.output
}

func (s *Step) GetState() interface{} {
	return s.state
}

func (s *Step) IsFinished() bool {
	return s.state == StepClosed || s.state == StepFinished
}

package chat

import (
	"context"
	geppetto_context "github.com/go-go-golems/geppetto/pkg/context"
	"github.com/go-go-golems/geppetto/pkg/helpers"
	"github.com/go-go-golems/geppetto/pkg/steps"
	"github.com/go-go-golems/geppetto/pkg/steps/openai"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
	"github.com/pkg/errors"
	go_openai "github.com/sashabaranov/go-openai"
	"gopkg.in/yaml.v3"
	"io"
)

type Transformer struct {
	ClientSettings *openai.ClientSettings `yaml:"client,omitempty"`
	StepSettings   *Settings              `yaml:"completion,omitempty"`
}

// factoryConfigFileWrapper is a helper to help us parse the YAML config file in the format:
// factories:
//
//			  openai:
//			    client_settings:
//	           api_key: SECRETSECRET
//			      timeout: 10s
//				     organization: "org"
//			    completion_settings:
//			      max_total_tokens: 100
//		       ...
//
// TODO(manuel, 2023-01-27) Maybe look into better YAML handling using UnmarshalYAML overloading
type factoryConfigFileWrapper struct {
	Factories struct {
		OpenAI *Transformer `yaml:"openai"`
	} `yaml:"factories"`
}

func NewTransformerFromYAML(s io.Reader) (*Transformer, error) {
	var settings factoryConfigFileWrapper
	if err := yaml.NewDecoder(s).Decode(&settings); err != nil {
		return nil, err
	}

	// NOTE(manuel, 2023-08-11) Unsure what this null check is for, honestly
	if settings.Factories.OpenAI == nil {
		settings.Factories.OpenAI = &Transformer{
			StepSettings:   &Settings{},
			ClientSettings: openai.NewClientSettings(),
		}
	}

	return &Transformer{
		StepSettings:   settings.Factories.OpenAI.StepSettings,
		ClientSettings: settings.Factories.OpenAI.ClientSettings,
	}, nil
}

func (csf *Transformer) UpdateFromParameters(ps map[string]interface{}) error {
	err := parameters.InitializeStructFromParameters(csf.StepSettings, ps)
	if err != nil {
		return err
	}

	err = parameters.InitializeStructFromParameters(csf.ClientSettings, ps)
	if err != nil {
		return err
	}

	return nil
}

func (csf *Transformer) Start(
	ctx context.Context,
	messages []*geppetto_context.Message,
) (*steps.Monad[string], error) {
	// I think in Start, a Transformer is not allowed to modify its own state,
	// everything is now encapsulated in the monad, which can run in the background.
	stepSettings := csf.StepSettings.Clone()
	clientSettings := stepSettings.ClientSettings
	if clientSettings == nil {
		clientSettings = csf.ClientSettings.Clone()
	}

	if clientSettings == nil {
		return nil, steps.ErrMissingClientSettings
	}

	if clientSettings.APIKey == nil {
		return nil, steps.ErrMissingClientAPIKey
	}

	client := go_openai.NewClient(*clientSettings.APIKey)

	engine := ""
	if stepSettings.Engine != nil {
		engine = *stepSettings.Engine
	} else if clientSettings.DefaultEngine != nil {
		engine = *clientSettings.DefaultEngine
	} else {
		return nil, errors.New("no engine specified")
	}

	msgs_ := []go_openai.ChatCompletionMessage{}
	for _, msg := range messages {
		msgs_ = append(msgs_, go_openai.ChatCompletionMessage{
			Role:    msg.Role,
			Content: msg.Text,
		})
	}

	temperature := 0.0
	if stepSettings.Temperature != nil {
		temperature = *stepSettings.Temperature
	}
	topP := 0.0
	if stepSettings.TopP != nil {
		topP = *stepSettings.TopP
	}
	maxTokens := 32
	if stepSettings.MaxResponseTokens != nil {
		maxTokens = *stepSettings.MaxResponseTokens
	}
	n := 1
	if stepSettings.N != nil {
		n = *stepSettings.N
	}
	stream := stepSettings.Stream
	stop := stepSettings.Stop
	presencePenalty := 0.0
	if stepSettings.PresencePenalty != nil {
		presencePenalty = *stepSettings.PresencePenalty
	}
	frequencyPenalty := 0.0
	if stepSettings.FrequencyPenalty != nil {
		frequencyPenalty = *stepSettings.FrequencyPenalty
	}

	req := go_openai.ChatCompletionRequest{
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
			return steps.Reject[string](err), nil
		}
		c := make(chan helpers.Result[string])
		ret := steps.NewMonad[string](c)

		go func() {
			defer close(c)
			defer stream.Close()
			for {
				select {
				case <-ctx.Done():
					return
				default:
					response, err := stream.Recv()
					if errors.Is(err, io.EOF) {
						c <- helpers.NewValueResult[string]("")
						return
					}
					if err != nil {
						c <- helpers.NewErrorResult[string](err)
						return
					}

					c <- helpers.NewPartialResult[string](response.Choices[0].Delta.Content)
				}
			}
		}()

		return ret, nil
	} else {
		resp, err := client.CreateChatCompletion(ctx, req)

		if err != nil {
			return steps.Reject[string](err), nil
		}

		return steps.Resolve(string(resp.Choices[0].Message.Content)), nil
	}
}

// Close is only called after the returned monad has been entirely consumed
func (csf *Transformer) Close(ctx context.Context) error {
	return nil
}

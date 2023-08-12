package chat

import (
	_ "embed"
	"github.com/go-go-golems/geppetto/pkg/steps/openai"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
)

type Settings struct {
	ClientSettings *openai.ClientSettings `yaml:"client,omitempty"`

	Engine *string `yaml:"engine,omitempty" glazed.parameter:"openai-engine"`

	MaxResponseTokens *int `yaml:"max_response_tokens,omitempty" glazed.parameter:"openai-max-response-tokens"`

	// TopP to use
	TopP *float64 `yaml:"top_p,omitempty" glazed.parameter:"openai-top-p"`
	// Sampling temperature to use
	Temperature *float64 `yaml:"temperature,omitempty" glazed.parameter:"openai-temperature"`
	// How many choice to create for each prompt
	N *int `yaml:"n" glazed.parameter:"openai-n"`
	// Up to 4 sequences where the API will stop generating tokens. Response will not contain the stop sequence.
	Stop []string `yaml:"stop,omitempty" glazed.parameter:"openai-stop"`
	// PresencePenalty to use
	PresencePenalty *float64 `yaml:"presence_penalty,omitempty" glazed.parameter:"openai-presence-penalty"`
	// FrequencyPenalty to use
	FrequencyPenalty *float64 `yaml:"frequency_penalty,omitempty" glazed.parameter:"openai-frequency-penalty"`
	// LogitBias to use
	// TODO(manuel, 2023-03-28) Properly load logit bias
	// See https://github.com/go-go-golems/geppetto/issues/48
	LogitBias map[string]string `yaml:"logit_bias,omitempty" glazed.parameter:"openai-logit-bias"`

	Stream bool `yaml:"stream,omitempty" glazed.parameter:"openai-stream"`
}

func NewSettings() *Settings {
	return &Settings{
		ClientSettings:    openai.NewClientSettings(),
		Engine:            nil,
		MaxResponseTokens: nil,
		TopP:              nil,
		Temperature:       nil,
		N:                 nil,
		Stop:              []string{},
		PresencePenalty:   nil,
		FrequencyPenalty:  nil,
		LogitBias:         map[string]string{},
		Stream:            false,
	}
}

func NewStepSettingsFromParameters(ps map[string]interface{}) (*Settings, error) {
	ret := NewSettings()
	// TODO(manuel, 2023-03-28) map[string]int will probably clash with map[string]string for the logit-bias
	err := parameters.InitializeStructFromParameters(ret, ps)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (s *Settings) Clone() *Settings {
	var clientSettings *openai.ClientSettings = nil
	if s.ClientSettings != nil {
		clientSettings = s.ClientSettings.Clone()
	}
	return &Settings{
		ClientSettings:    clientSettings,
		Engine:            s.Engine,
		MaxResponseTokens: s.MaxResponseTokens,
		TopP:              s.TopP,
		Temperature:       s.Temperature,
		N:                 s.N,
		Stop:              s.Stop,
		PresencePenalty:   s.PresencePenalty,
		FrequencyPenalty:  s.FrequencyPenalty,
		LogitBias:         s.LogitBias,
		Stream:            s.Stream,
	}
}

//go:embed "chat.yaml"
var settingsYAML []byte

type ParameterLayer struct {
	*layers.ParameterLayerImpl `yaml:",inline"`
}

func NewParameterLayer(options ...layers.ParameterLayerOptions) (*ParameterLayer, error) {
	ret, err := layers.NewParameterLayerFromYAML(settingsYAML, options...)
	if err != nil {
		return nil, err
	}

	return &ParameterLayer{
		ParameterLayerImpl: ret,
	}, nil
}

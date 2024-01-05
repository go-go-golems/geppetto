package openai

import (
	_ "embed"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
)

type Settings struct {
	// How many choice to create for each prompt
	N *int `yaml:"n" glazed.parameter:"openai-n"`
	// PresencePenalty to use
	PresencePenalty *float64 `yaml:"presence_penalty,omitempty" glazed.parameter:"openai-presence-penalty"`
	// FrequencyPenalty to use
	FrequencyPenalty *float64 `yaml:"frequency_penalty,omitempty" glazed.parameter:"openai-frequency-penalty"`
	// LogitBias to use
	// TODO(manuel, 2023-03-28) Properly load logit bias
	// See https://github.com/go-go-golems/geppetto/issues/48
	LogitBias map[string]string `yaml:"logit_bias,omitempty" glazed.parameter:"openai-logit-bias"`
	BaseURL   *string           `yaml:"base_url,omitempty" glazed.parameter:"openai-base-url"`
	APIKey    *string           `yaml:"api_key,omitempty" glazed.parameter:"openai-api-key"`
}

func NewSettings() *Settings {
	return &Settings{
		N:                nil,
		PresencePenalty:  nil,
		FrequencyPenalty: nil,
		LogitBias:        map[string]string{},
		BaseURL:          nil,
		APIKey:           nil,
	}
}

func (s *Settings) Clone() *Settings {
	return &Settings{
		N:                s.N,
		PresencePenalty:  s.PresencePenalty,
		FrequencyPenalty: s.FrequencyPenalty,
		LogitBias:        s.LogitBias,
		APIKey:           s.APIKey,
		BaseURL:          s.BaseURL,
	}
}

//go:embed "chat.yaml"
var settingsYAML []byte

type ParameterLayer struct {
	*layers.ParameterLayerImpl `yaml:",inline"`
}

const OpenAiChatSlug = "openai-chat"

func NewParameterLayer(options ...layers.ParameterLayerOptions) (*ParameterLayer, error) {
	ret, err := layers.NewParameterLayerFromYAML(settingsYAML, options...)
	if err != nil {
		return nil, err
	}

	return &ParameterLayer{
		ParameterLayerImpl: ret,
	}, nil
}

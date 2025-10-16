package openai

import (
	_ "embed"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/huandu/go-clone"
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
    // ReasoningEffort for Responses API (low|medium|high)
    ReasoningEffort *string `yaml:"reasoning_effort,omitempty" glazed.parameter:"openai-reasoning-effort"`
    // ParallelToolCalls is a hint for tool parallelization in Responses
    ParallelToolCalls *bool `yaml:"parallel_tool_calls,omitempty" glazed.parameter:"openai-parallel-tool-calls"`
}

func NewSettings() (*Settings, error) {
	s := &Settings{
		N:                nil,
		PresencePenalty:  nil,
		FrequencyPenalty: nil,
		LogitBias:        map[string]string{},
        ReasoningEffort:  nil,
        ParallelToolCalls: nil,
	}

	p, err := NewParameterLayer()
	if err != nil {
		return nil, err
	}

	err = p.InitializeStructFromParameterDefaults(s)
	if err != nil {
		return nil, err
	}

	return s, nil
}

func (s *Settings) Clone() *Settings {
	return clone.Clone(s).(*Settings)
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

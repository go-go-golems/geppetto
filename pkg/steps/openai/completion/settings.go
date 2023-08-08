package completion

import (
	_ "embed"
	"github.com/go-go-golems/geppetto/pkg/steps/openai"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
	"gopkg.in/yaml.v3"
	"io"
)

type StepSettings struct {
	ClientSettings *openai.ClientSettings `yaml:"client,omitempty"`

	Engine *string `yaml:"engine,omitempty" glazed.parameter:"openai-engine"`

	MaxResponseTokens *int `yaml:"max_response_tokens,omitempty" glazed.parameter:"openai-max-response-tokens"`

	// Sampling temperature to use
	Temperature *float64 `yaml:"temperature,omitempty" glazed.parameter:"openai-temperature"`
	// Alternative to temperature for nucleus sampling
	TopP *float64 `yaml:"top_p,omitempty" glazed.parameter:"openai-top-p"`
	// How many choice to create for each prompt
	N *int `yaml:"n" glazed.parameter:"openai-n"`
	// Include the probabilities of most likely tokens
	LogProbs *int `yaml:"logprobs" glazed.parameter:"openai-logprobs"`
	// Up to 4 sequences where the API will stop generating tokens. Response will not contain the stop sequence.
	Stop []string `yaml:"stop,omitempty" glazed.parameter:"openai-stop"`

	Stream bool `yaml:"stream,omitempty" glazed.parameter:"openai-stream"`

	FrequencyPenalty *float64 `yaml:"frequency_penalty,omitempty" glazed.parameter:"openai-frequency-penalty"`
	PresencePenalty  *float64 `yaml:"presence_penalty,omitempty" glazed.parameter:"openai-presence-penalty"`
	BestOf           *int     `yaml:"best_of,omitempty" glazed.parameter:"openai-best-of"`
	// TODO(manuel, 2023-03-28) Properly load logit bias
	// See https://github.com/go-go-golems/geppetto/issues/48
	LogitBias map[string]string `yaml:"logit_bias,omitempty" glazed.parameter:"openai-logit-bias"`
}

func NewStepSettingsFromParameters(ps map[string]interface{}) (*StepSettings, error) {
	ret := NewStepSettings()
	err := parameters.InitializeStructFromParameters(ret, ps)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (c *StepSettings) Clone() *StepSettings {
	var clientSettings *openai.ClientSettings = nil
	if c.ClientSettings != nil {
		clientSettings = c.ClientSettings.Clone()
	}
	return &StepSettings{
		ClientSettings:    clientSettings,
		Engine:            c.Engine,
		MaxResponseTokens: c.MaxResponseTokens,
		Temperature:       c.Temperature,
		TopP:              c.TopP,
		N:                 c.N,
		LogProbs:          c.LogProbs,
		Stop:              c.Stop,
		Stream:            c.Stream,
	}
}

//go:embed "completion.yaml"
var completionStepSettingsYAML []byte

type ParameterLayer struct {
	*layers.ParameterLayerImpl `yaml:",inline"`
}

func NewParameterLayer(options ...layers.ParameterLayerOptions) (*ParameterLayer, error) {
	ret, err := layers.NewParameterLayerFromYAML(completionStepSettingsYAML, options...)
	if err != nil {
		return nil, err
	}

	return &ParameterLayer{ret}, nil
}

type StepFactory struct {
	ClientSettings *openai.ClientSettings `yaml:"client,omitempty"`
	StepSettings   *StepSettings          `yaml:"completion,omitempty"`
}

func (csf *StepFactory) UpdateFromParameters(ps map[string]interface{}) error {
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

func NewStepFactory(
	settings *StepSettings,
	clientSettings *openai.ClientSettings,
) *StepFactory {
	return &StepFactory{
		StepSettings:   settings,
		ClientSettings: clientSettings,
	}
}

func (csf *StepFactory) NewStep() (CompletionStep, error) {
	stepSettings := csf.StepSettings.Clone()
	if stepSettings.ClientSettings == nil {
		stepSettings.ClientSettings = csf.ClientSettings.Clone()
	}

	return NewStep(stepSettings), nil
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
		OpenAI *StepFactory `yaml:"openai"`
	} `yaml:"factories"`
}

func NewStepFactoryFromYAML(s io.Reader) (*StepFactory, error) {
	var settings factoryConfigFileWrapper
	if err := yaml.NewDecoder(s).Decode(&settings); err != nil {
		return nil, err
	}

	if settings.Factories.OpenAI == nil {
		settings.Factories.OpenAI = NewStepFactory(NewStepSettings(), openai.NewClientSettings())
	}

	return NewStepFactory(
		settings.Factories.OpenAI.StepSettings,
		settings.Factories.OpenAI.ClientSettings,
	), nil
}

func NewStepSettings() *StepSettings {
	return &StepSettings{}
}

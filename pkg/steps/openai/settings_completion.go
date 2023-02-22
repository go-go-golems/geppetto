package openai

import (
	_ "embed"
	"github.com/go-go-golems/geppetto/pkg/steps"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
	"gopkg.in/yaml.v3"
	"io"
)

type CompletionStepSettings struct {
	ClientSettings *ClientSettings `yaml:"client,omitempty"`

	Engine *string `yaml:"engine,omitempty" glazed.parameter:"openai-engine"`

	MaxResponseTokens *int `yaml:"max_response_tokens,omitempty" glazed.parameter:"openai-max-response-tokens"`

	// Sampling temperature to use
	Temperature *float32 `yaml:"temperature,omitempty" glazed.parameter:"openai-temperature"`
	// Alternative to temperature for nucleus sampling
	TopP *float32 `yaml:"top_p,omitempty" glazed.parameter:"openai-top-p"`
	// How many choice to create for each prompt
	N *int `yaml:"n" glazed.parameter:"openai-n"`
	// Include the probabilities of most likely tokens
	LogProbs *int `yaml:"logprobs" glazed.parameter:"openai-logprobs"`
	// Up to 4 sequences where the API will stop generating tokens. Response will not contain the stop sequence.
	Stop []string `yaml:"stop,omitempty" glazed.parameter:"openai-stop"`

	Stream bool `yaml:"stream,omitempty" glazed.parameter:"openai-stream"`
}

func NewCompletionStepSettingsFromParameters(ps map[string]interface{}) (*CompletionStepSettings, error) {
	ret := NewCompletionStepSettings()
	err := parameters.InitializeStructFromParameters(ret, ps)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (c *CompletionStepSettings) Clone() *CompletionStepSettings {
	var clientSettings *ClientSettings = nil
	if c.ClientSettings != nil {
		clientSettings = c.ClientSettings.Clone()
	}
	return &CompletionStepSettings{
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

//go:embed "flags/completion.yaml"
var completionStepSettingsYAML []byte

type CompletionParameterLayer struct {
	layers.ParameterLayerImpl
}

func NewCompletionParameterLayer(defaults *CompletionStepSettings) (*CompletionParameterLayer, error) {
	ret := &CompletionParameterLayer{}
	err := ret.LoadFromYAML(completionStepSettingsYAML)
	if err != nil {
		return nil, err
	}

	err = ret.InitializeParameterDefaultsFromStruct(defaults)
	if err != nil {
		return nil, err
	}

	return ret, nil
}

type CompletionStepFactory struct {
	ClientSettings *ClientSettings         `yaml:"client,omitempty"`
	StepSettings   *CompletionStepSettings `yaml:"completion,omitempty"`
}

func NewCompletionStepFactory(
	settings *CompletionStepSettings,
	clientSettings *ClientSettings,
) *CompletionStepFactory {
	return &CompletionStepFactory{
		StepSettings:   settings,
		ClientSettings: clientSettings,
	}
}

func (csf *CompletionStepFactory) NewStep() (steps.Step[string, string], error) {
	stepSettings := csf.StepSettings.Clone()
	if stepSettings.ClientSettings == nil {
		stepSettings.ClientSettings = csf.ClientSettings.Clone()
	}

	return NewCompletionStep(stepSettings), nil
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
		OpenAI *CompletionStepFactory `yaml:"openai"`
	} `yaml:"factories"`
}

func NewCompletionStepFactoryFromYAML(s io.Reader) (*CompletionStepFactory, error) {
	var settings factoryConfigFileWrapper
	if err := yaml.NewDecoder(s).Decode(&settings); err != nil {
		return nil, err
	}

	if settings.Factories.OpenAI == nil {
		settings.Factories.OpenAI = NewCompletionStepFactory(NewCompletionStepSettings(), NewClientSettings())
	}

	return NewCompletionStepFactory(
		settings.Factories.OpenAI.StepSettings,
		settings.Factories.OpenAI.ClientSettings,
	), nil
}

func NewCompletionStepSettings() *CompletionStepSettings {
	return &CompletionStepSettings{}
}

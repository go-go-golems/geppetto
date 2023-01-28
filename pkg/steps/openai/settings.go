package openai

import (
	"github.com/PullRequestInc/go-gpt3"
	"gopkg.in/yaml.v3"
	"io"
	"net/http"
	"time"
)

type ClientSettings struct {
	APIKey        *string        `yaml:"api_key,omitempty"`
	Timeout       *time.Duration `yaml:"timeout,omitempty"`
	Organization  *string        `yaml:"organization,omitempty"`
	DefaultEngine *string        `yaml:"default_engine,omitempty"`
	UserAgent     *string        `yaml:"user_agent,omitempty"`
	BaseURL       *string        `yaml:"base_url,omitempty"`
	HTTPClient    *http.Client   `yaml:"omitempty"`
}

var ErrMissingAPIKey = &yaml.TypeError{Errors: []string{"missing api key"}}

func (c *ClientSettings) IsValid() error {
	if c.APIKey == nil {
		return ErrMissingAPIKey
	}
	return nil
}

func (c *ClientSettings) Clone() *ClientSettings {
	return &ClientSettings{
		APIKey:        c.APIKey,
		Timeout:       c.Timeout,
		Organization:  c.Organization,
		DefaultEngine: c.DefaultEngine,
		UserAgent:     c.UserAgent,
		BaseURL:       c.BaseURL,
		HTTPClient:    c.HTTPClient,
	}
}

func (c *ClientSettings) ToOptions() []gpt3.ClientOption {
	ret := make([]gpt3.ClientOption, 0)
	if c.Timeout != nil {
		ret = append(ret, gpt3.WithTimeout(*c.Timeout))
	}
	if c.Organization != nil {
		ret = append(ret, gpt3.WithOrg(*c.Organization))
	}
	if c.DefaultEngine != nil {
		ret = append(ret, gpt3.WithDefaultEngine(*c.DefaultEngine))
	}
	if c.UserAgent != nil {
		ret = append(ret, gpt3.WithUserAgent(*c.UserAgent))
	}
	if c.BaseURL != nil {
		ret = append(ret, gpt3.WithBaseURL(*c.BaseURL))
	}
	if c.HTTPClient != nil {
		ret = append(ret, gpt3.WithHTTPClient(c.HTTPClient))
	}
	return ret
}

type CompletionStepSettings struct {
	ClientSettings *ClientSettings `yaml:"client_settings,omitempty"`

	Engine *string `yaml:"engine,omitempty"`

	MaxResponseTokens *int `yaml:"max_response_tokens,omitempty"`

	// Sampling temperature to use
	Temperature *float32 `yaml:"temperature,omitempty"`
	// Alternative to temperature for nucleus sampling
	TopP *float32 `yaml:"top_p,omitempty"`
	// How many choice to create for each prompt
	N *int `yaml:"n"`
	// Include the probabilities of most likely tokens
	LogProbs *int `yaml:"logprobs"`
	// Up to 4 sequences where the API will stop generating tokens. Response will not contain the stop sequence.
	Stop []string `yaml:"stop,omitempty"`

	Stream bool `yaml:"stream,omitempty"`
}

func (c *CompletionStepSettings) Clone() *CompletionStepSettings {
	return &CompletionStepSettings{
		ClientSettings:    c.ClientSettings.Clone(),
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

type CompletionStepFactory struct {
	OpenAICompletionStepSettings *CompletionStepSettings `yaml:"completion_settings,omitempty"`
}

func NewCompletionStepFactory(
	settings *CompletionStepSettings,
) *CompletionStepFactory {
	return &CompletionStepFactory{
		OpenAICompletionStepSettings: settings,
	}
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
	}
}

func NewCompletionStepFactoryFromYAML(s io.Reader) (*CompletionStepFactory, error) {
	var settings factoryConfigFileWrapper
	if err := yaml.NewDecoder(s).Decode(&settings); err != nil {
		return nil, err
	}

	return NewCompletionStepFactory(
		settings.Factories.OpenAI.OpenAICompletionStepSettings,
	), nil
}

func (csf *CompletionStepFactory) CreateCompletionStep() *CompletionStep {
	return NewCompletionStep(csf.OpenAICompletionStepSettings.Clone())
}

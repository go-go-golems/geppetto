package openai

import (
	"github.com/PullRequestInc/go-gpt3"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/wesen/geppetto/pkg/steps"
	"gopkg.in/yaml.v3"
	"io"
	"net/http"
	"time"
)

var ErrMissingYAMLAPIKey = &yaml.TypeError{Errors: []string{"missing api key"}}

type ClientSettings struct {
	APIKey        *string        `yaml:"api_key,omitempty"`
	Timeout       *time.Duration `yaml:"timeout,omitempty"`
	Organization  *string        `yaml:"organization,omitempty"`
	DefaultEngine *string        `yaml:"default_engine,omitempty"`
	UserAgent     *string        `yaml:"user_agent,omitempty"`
	BaseURL       *string        `yaml:"base_url,omitempty"`
	HTTPClient    *http.Client   `yaml:"omitempty"`
}

// UnmarshalYAML overrides YAML parsing to convert time.duration from int
func (c *ClientSettings) UnmarshalYAML(value *yaml.Node) error {
	type Alias ClientSettings
	aux := &struct {
		Timeout *int `yaml:"timeout,omitempty"`
		*Alias
	}{
		Alias: (*Alias)(c),
	}
	if err := value.Decode(aux); err != nil {
		return err
	}
	if aux.Timeout != nil {
		t := time.Duration(*aux.Timeout) * time.Second
		c.Timeout = &t
	}
	return nil
}

func (c *ClientSettings) IsValid() error {
	if c.APIKey == nil {
		return ErrMissingYAMLAPIKey
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
	ClientSettings *ClientSettings `yaml:"client,omitempty"`

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

func (csf *CompletionStepFactory) AddFlags(cmd *cobra.Command) error {
	cmd.PersistentFlags().String("openai-engine", "", "OpenAI engine to use")
	cmd.PersistentFlags().Int("openai-max-response-tokens", 0, "Maximum number of tokens to return")
	cmd.PersistentFlags().Float32("openai-temperature", 0.7, "Sampling temperature to use")
	cmd.PersistentFlags().Float32("openai-top-p", 0.0, "Alternative to temperature for nucleus sampling")
	cmd.PersistentFlags().Int("openai-n", 1, "How many choice to create for each prompt")
	cmd.PersistentFlags().Int("openai-logprobs", 0, "Include the probabilities of most likely tokens")
	cmd.PersistentFlags().StringSlice("openai-stop", []string{}, "Up to 4 sequences where the API will stop generating tokens. Response will not contain the stop sequence.")
	cmd.PersistentFlags().Bool("openai-stream", false, "Stream the response")

	return nil
}

func (csf *CompletionStepFactory) UpdateFromCobra(cmd *cobra.Command) error {
	apiKey := viper.GetString("openai-api-key")
	if apiKey != "" {
		csf.ClientSettings.APIKey = &apiKey
	}

	if cmd.Flags().Changed("openai-engine") {
		engine := cmd.Flag("openai-engine").Value.String()
		csf.StepSettings.Engine = &engine
	}
	if cmd.Flags().Changed("openai-max-response-tokens") {
		maxResponseTokens, err := cmd.PersistentFlags().GetInt("openai-max-response-tokens")
		if err != nil {
			return err
		}
		csf.StepSettings.MaxResponseTokens = &maxResponseTokens
	}
	if cmd.Flags().Changed("openai-temperature") {
		temperature, err := cmd.PersistentFlags().GetFloat32("openai-temperature")
		if err != nil {
			return err
		}
		csf.StepSettings.Temperature = &temperature
	}
	if cmd.Flags().Changed("openai-top-p") {
		topP, err := cmd.PersistentFlags().GetFloat32("openai-top-p")
		if err != nil {
			return err
		}
		csf.StepSettings.TopP = &topP
	}
	if cmd.Flags().Changed("openai-n") {
		n, err := cmd.PersistentFlags().GetInt("openai-n")
		if err != nil {
			return err
		}
		csf.StepSettings.N = &n
	}
	if cmd.Flags().Changed("openai-logprobs") {
		logProbs, err := cmd.PersistentFlags().GetInt("openai-logprobs")
		if err != nil {
			return err
		}
		csf.StepSettings.LogProbs = &logProbs
	}
	if cmd.Flags().Changed("openai-stop") {
		stop, err := cmd.PersistentFlags().GetStringSlice("openai-stop")
		if err != nil {
			return err
		}
		csf.StepSettings.Stop = stop
	}

	return nil
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

func NewClientSettings() *ClientSettings {
	defaultTimeout := 60 * time.Second
	return &ClientSettings{
		Timeout: &defaultTimeout,
	}
}

func (csf *CompletionStepFactory) CreateCompletionStep() *CompletionStep {
	return NewCompletionStep(csf.StepSettings.Clone())
}

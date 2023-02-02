package openai

import (
	"fmt"
	gpt3 "github.com/PullRequestInc/go-gpt3"
	"github.com/rs/zerolog/log"
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

func (c *ClientSettings) CreateClient() (gpt3.Client, error) {
	evt := log.Debug()
	if c.BaseURL != nil {
		evt = evt.Str("base_url", *c.BaseURL)
	}
	if c.DefaultEngine != nil {
		evt = evt.Str("default_engine", *c.DefaultEngine)
	}
	if c.Organization != nil {
		evt = evt.Str("organization", *c.Organization)
	}
	if c.Timeout != nil {
		// convert timeout to seconds
		timeout := *c.Timeout / time.Second
		evt = evt.Dur("timeout", timeout)
	}
	if c.UserAgent != nil {
		evt = evt.Str("user_agent", *c.UserAgent)
	}
	evt.Msg("creating openai client")

	options := c.ToOptions()

	return gpt3.NewClient(*c.APIKey, options...), nil
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
	flagsDefaults  *CompletionStepFactoryFlagsDefaults
	flagsPrefix    string
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

type CompletionStepFactoryFlagsDefaults struct {
	Engine            *string
	MaxResponseTokens *int
	Temperature       *float32
	TopP              *float32
	N                 *int
	LogProbs          *int
	Stop              *[]string
	Stream            *bool
}

func (csf *CompletionStepFactory) AddFlags(cmd *cobra.Command, prefix string, defaults interface{}) error {
	csfDefaults, ok := defaults.(*CompletionStepFactoryFlagsDefaults)
	if !ok || csfDefaults == nil {
		return fmt.Errorf("defaults are not of type *CompletionStepFactoryFlagsDefaults")
	}

	csf.flagsDefaults = csfDefaults

	defaultEngine := ""
	if csfDefaults.Engine != nil {
		defaultEngine = *csfDefaults.Engine
	}
	cmd.PersistentFlags().String(prefix+"engine", defaultEngine, "OpenAI engine to use")

	defaultMaxResponseTokens := 0
	if csfDefaults.MaxResponseTokens != nil {
		defaultMaxResponseTokens = *csfDefaults.MaxResponseTokens
	}
	cmd.PersistentFlags().Int(prefix+"max-response-tokens", defaultMaxResponseTokens, "Maximum number of tokens to return")

	defaultTemperature := float32(0.7)
	if csfDefaults.Temperature != nil {
		defaultTemperature = *csfDefaults.Temperature
	}
	cmd.PersistentFlags().Float32(prefix+"temperature", defaultTemperature, "Sampling temperature to use")

	defaultTopP := float32(0.0)
	if csfDefaults.TopP != nil {
		defaultTopP = *csfDefaults.TopP
	}
	cmd.PersistentFlags().Float32(prefix+"top-p", defaultTopP, "Alternative to temperature for nucleus sampling")

	defaultN := 1
	if csfDefaults.N != nil {
		defaultN = *csfDefaults.N
	}
	cmd.PersistentFlags().Int(prefix+"n", defaultN, "How many choice to create for each prompt")

	defaultLogProbs := 0
	if csfDefaults.LogProbs != nil {
		defaultLogProbs = *csfDefaults.LogProbs
	}
	cmd.PersistentFlags().Int(prefix+"logprobs", defaultLogProbs, "Include the probabilities of most likely tokens")

	defaultStop := []string{}
	if csfDefaults.Stop != nil {
		defaultStop = *csfDefaults.Stop
	}
	cmd.PersistentFlags().StringSlice(prefix+"stop", defaultStop, "Up to 4 sequences where the API will stop generating tokens. Response will not contain the stop sequence.")

	defaultStream := false
	if csfDefaults.Stream != nil {
		defaultStream = *csfDefaults.Stream
	}
	cmd.PersistentFlags().Bool(prefix+"stream", defaultStream, "Stream the response")

	csf.flagsPrefix = prefix

	return nil
}

func (csf *CompletionStepFactory) UpdateFromCobra(cmd *cobra.Command) error {
	prefix := csf.flagsPrefix
	apiKey := viper.GetString(prefix + "api-key")
	if apiKey != "" {
		csf.ClientSettings.APIKey = &apiKey
	}

	if cmd.Flags().Changed(prefix+"engine") || csf.flagsDefaults.Engine != nil {
		engine := cmd.Flag(prefix + "engine").Value.String()
		csf.StepSettings.Engine = &engine
	}
	if cmd.Flags().Changed(prefix+"max-response-tokens") || csf.flagsDefaults.MaxResponseTokens != nil {
		maxResponseTokens, err := cmd.PersistentFlags().GetInt(prefix + "max-response-tokens")
		if err != nil {
			return err
		}
		csf.StepSettings.MaxResponseTokens = &maxResponseTokens
	}
	if cmd.Flags().Changed(prefix+"temperature") || csf.flagsDefaults.Temperature != nil {
		temperature, err := cmd.PersistentFlags().GetFloat32(prefix + "temperature")
		if err != nil {
			return err
		}
		csf.StepSettings.Temperature = &temperature
	}
	if cmd.Flags().Changed(prefix+"top-p") || csf.flagsDefaults.Temperature != nil {
		topP, err := cmd.PersistentFlags().GetFloat32(prefix + "top-p")
		if err != nil {
			return err
		}
		csf.StepSettings.TopP = &topP
	}

	if cmd.Flags().Changed(prefix+"n") || csf.flagsDefaults.Temperature != nil {
		n, err := cmd.PersistentFlags().GetInt(prefix + "n")
		if err != nil {
			return err
		}
		csf.StepSettings.N = &n
	}

	if cmd.Flags().Changed(prefix+"logprobs") || csf.flagsDefaults.Temperature != nil {
		logProbs, err := cmd.PersistentFlags().GetInt(prefix + "logprobs")
		if err != nil {
			return err
		}
		csf.StepSettings.LogProbs = &logProbs
	}
	if cmd.Flags().Changed(prefix+"stop") || csf.flagsDefaults.Temperature != nil {
		stop, err := cmd.PersistentFlags().GetStringSlice(prefix + "stop")
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

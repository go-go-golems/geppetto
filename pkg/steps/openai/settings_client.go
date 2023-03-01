package openai

import (
	_ "embed"
	"github.com/PullRequestInc/go-gpt3"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
	"net/http"
	"time"
)

var ErrMissingYAMLAPIKey = &yaml.TypeError{Errors: []string{"missing api key"}}

type ClientSettings struct {
	APIKey        *string        `yaml:"api_key,omitempty" glazed.parameter:"openai-api-key"`
	Timeout       *time.Duration `yaml:"timeout,omitempty" glazed.parameter:"openai-timeout"`
	Organization  *string        `yaml:"organization,omitempty" glazed.parameter:"openai-organization"`
	DefaultEngine *string        `yaml:"default_engine,omitempty" glazed.parameter:"openai-default-engine"`
	UserAgent     *string        `yaml:"user_agent,omitempty" glazed.parameter:"openai-user-agent"`
	BaseURL       *string        `yaml:"base_url,omitempty" glazed.parameter:"openai-base-url"`
	HTTPClient    *http.Client   `yaml:"omitempty"`
}

//go:embed "flags/client.yaml"
var clientFlagsYAML []byte

type ClientParameterLayer struct {
	*layers.ParameterLayerImpl
}

func NewClientParameterLayer(options ...layers.ParameterLayerOptions) (*ClientParameterLayer, error) {
	ret, err := layers.NewParameterLayerFromYAML(clientFlagsYAML, options...)
	if err != nil {
		return nil, err
	}

	return &ClientParameterLayer{ParameterLayerImpl: ret}, nil
}

func NewClientSettingsFromParameters(ps map[string]interface{}) (*ClientSettings, error) {
	ret := NewClientSettings()
	err := parameters.InitializeStructFromParameters(ret, ps)
	if err != nil {
		return nil, err
	}

	if ret.Timeout != nil {
		duration := *ret.Timeout * time.Second
		ret.Timeout = &duration
	}

	if ret.APIKey == nil {
		return nil, ErrMissingYAMLAPIKey
	}
	return ret, nil
}

func (cp *ClientParameterLayer) ParseFlagsFromCobraCommand(
	cmd *cobra.Command,
) (map[string]interface{}, error) {
	// actually hijack and load everything from viper instead of cobra...
	ps, err := parameters.GatherFlagsFromViper(cp.Flags, false, cp.Prefix)
	if err != nil {
		return nil, err
	}

	// now load from flag overrides
	ps2, err := parameters.GatherFlagsFromCobraCommand(cmd, cp.Flags, true, cp.Prefix)
	if err != nil {
		return nil, err
	}
	for k, v := range ps2 {
		ps[k] = v
	}

	return ps, nil
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
	if c.APIKey == nil {
		return nil, ErrMissingYAMLAPIKey
	}
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

func NewClientSettings() *ClientSettings {
	defaultTimeout := 60 * time.Second
	return &ClientSettings{
		Timeout: &defaultTimeout,
	}
}

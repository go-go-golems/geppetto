package settings

import (
	_ "embed"
	"net/http"
	"time"

	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/huandu/go-clone"
	"gopkg.in/yaml.v3"
)

var ErrMissingYAMLAPIKey = &yaml.TypeError{Errors: []string{"missing api key"}}

type ClientSettings struct {
	Timeout        *time.Duration `yaml:"timeout,omitempty"`
	TimeoutSeconds *int           `yaml:"timeout_second,omitempty" glazed:"timeout"`
	Organization   *string        `yaml:"organization,omitempty" glazed:"organization"`
	UserAgent      *string        `yaml:"user_agent,omitempty" glazed:"user-agent"`
	HTTPClient     *http.Client   `yaml:"-" json:"-"`
}

//go:embed "flags/client.yaml"
var clientFlagsYAML []byte

type ClientParameterLayer struct {
	*schema.SectionImpl `yaml:",inline"`
}

func NewClientParameterLayer(options ...schema.SectionOption) (*ClientParameterLayer, error) {
	ret, err := schema.NewSectionFromYAML(clientFlagsYAML, options...)
	if err != nil {
		return nil, err
	}

	return &ClientParameterLayer{SectionImpl: ret}, nil
}

// UnmarshalYAML overrides YAML parsing to convert time.duration from int
func (cs *ClientSettings) UnmarshalYAML(value *yaml.Node) error {
	type Alias ClientSettings
	aux := &struct {
		Timeout *int `yaml:"timeout,omitempty"`
		*Alias
	}{
		Alias: (*Alias)(cs),
	}
	if err := value.Decode(aux); err != nil {
		return err
	}
	if aux.Timeout != nil {
		t := time.Duration(*aux.Timeout) * time.Second
		cs.Timeout = &t
		cs.TimeoutSeconds = aux.Timeout
	}
	return nil
}

func (cs *ClientSettings) Clone() *ClientSettings {
	return clone.Clone(cs).(*ClientSettings)
}

const AiClientSlug = "ai-client"

func NewClientSettings() *ClientSettings {
	defaultTimeout := 60 * time.Second
	return &ClientSettings{
		Timeout: &defaultTimeout,
		TimeoutSeconds: func() *int {
			i := int(defaultTimeout.Seconds())
			return &i
		}(),
	}
}

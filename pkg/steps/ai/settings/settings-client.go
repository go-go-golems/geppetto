package settings

import (
	_ "embed"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/huandu/go-clone"
	"gopkg.in/yaml.v3"
	"net/http"
	"time"
)

var ErrMissingYAMLAPIKey = &yaml.TypeError{Errors: []string{"missing api key"}}

type ClientSettings struct {
	Timeout        *time.Duration `yaml:"timeout,omitempty"`
	TimeoutSeconds *int           `yaml:"timeout_second,omitempty" glazed.parameter:"timeout"`
	Organization   *string        `yaml:"organization,omitempty" glazed.parameter:"organization"`
	UserAgent      *string        `yaml:"user_agent,omitempty" glazed.parameter:"user-agent"`
	HTTPClient     *http.Client   `yaml:"omitempty"`
}

//go:embed "flags/client.yaml"
var clientFlagsYAML []byte

type ClientParameterLayer struct {
	*layers.ParameterLayerImpl `yaml:",inline"`
}

func NewClientParameterLayer(options ...layers.ParameterLayerOptions) (*ClientParameterLayer, error) {
	ret, err := layers.NewParameterLayerFromYAML(clientFlagsYAML, options...)
	if err != nil {
		return nil, err
	}

	return &ClientParameterLayer{ParameterLayerImpl: ret}, nil
}

func (cs *ClientSettings) UpdateFromParameters(parsedLayers *layers.ParsedLayer) error {
	_, ok := parsedLayers.Layer.(*ClientParameterLayer)
	if !ok {
		return layers.ErrInvalidParameterLayer{}
	}

	err := parsedLayers.InitializeStruct(cs)
	if err != nil {
		return err
	}

	return nil
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

func (cs *ClientSettings) UpdateFromParsedLayer(layer *layers.ParsedLayer) error {
	_, ok := layer.Layer.(*ClientParameterLayer)
	if !ok {
		return layers.ErrInvalidParameterLayer{
			Name:     layer.Layer.GetName(),
			Expected: AiClientSlug,
		}
	}

	err := layer.InitializeStruct(cs)
	return err
}

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

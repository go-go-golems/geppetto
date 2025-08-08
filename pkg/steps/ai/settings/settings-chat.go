package settings

import (
	_ "embed"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/types"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/huandu/go-clone"
)

type ChatSettings struct {
	Engine            *string           `yaml:"engine,omitempty" glazed.parameter:"ai-engine"`
	ApiType           *types.ApiType    `yaml:"api_type,omitempty" glazed.parameter:"ai-api-type"`
	MaxResponseTokens *int              `yaml:"max_response_tokens,omitempty" glazed.parameter:"ai-max-response-tokens"`
	TopP              *float64          `yaml:"top_p,omitempty" glazed.parameter:"ai-top-p"`
	Temperature       *float64          `yaml:"temperature,omitempty" glazed.parameter:"ai-temperature"`
	Stop              []string          `yaml:"stop,omitempty" glazed.parameter:"ai-stop"`
	APIKeys           map[string]string `yaml:"api_keys,omitempty" glazed.parameter:"*-api-key"`
	Stream            bool              `yaml:"stream,omitempty"`

	// Caching settings
	CacheType       string `yaml:"cache_type,omitempty" glazed.parameter:"ai-cache-type"`
	CacheMaxSize    int64  `yaml:"cache_max_size,omitempty" glazed.parameter:"ai-cache-max-size"`
	CacheMaxEntries int    `yaml:"cache_max_entries,omitempty" glazed.parameter:"ai-cache-max-entries"`
	CacheDirectory  string `yaml:"cache_directory,omitempty" glazed.parameter:"ai-cache-directory"`
}

func NewChatSettings() (*ChatSettings, error) {
	s := &ChatSettings{
		Engine:            nil,
		ApiType:           nil,
		MaxResponseTokens: nil,
		TopP:              nil,
		Temperature:       nil,
		Stop:              []string{},
		APIKeys:           map[string]string{},
		Stream:            true, // Always enable streaming
	}

	p, err := NewChatParameterLayer()
	if err != nil {
		return nil, err
	}
	err = p.InitializeStructFromParameterDefaults(s)
	if err != nil {
		return nil, err
	}

	return s, nil
}

func (s *ChatSettings) Clone() *ChatSettings {
	return clone.Clone(s).(*ChatSettings)
}

//go:embed "flags/chat.yaml"
var settingsYAML []byte

type ChatParameterLayer struct {
	*layers.ParameterLayerImpl `yaml:",inline"`
}

const AiChatSlug = "ai-chat"

func NewChatParameterLayer(options ...layers.ParameterLayerOptions) (*ChatParameterLayer, error) {
	ret, err := layers.NewParameterLayerFromYAML(settingsYAML, options...)
	if err != nil {
		return nil, err
	}

	return &ChatParameterLayer{
		ParameterLayerImpl: ret,
	}, nil
}

// WrapWithCache removed with steps API deprecation. Use engine-level caching/middleware instead.

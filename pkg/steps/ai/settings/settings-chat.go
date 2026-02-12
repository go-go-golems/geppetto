package settings

import (
	_ "embed"

	"github.com/go-go-golems/geppetto/pkg/steps/ai/types"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/huandu/go-clone"
)

type ChatSettings struct {
	Engine            *string           `yaml:"engine,omitempty" glazed:"ai-engine"`
	ApiType           *types.ApiType    `yaml:"api_type,omitempty" glazed:"ai-api-type"`
	MaxResponseTokens *int              `yaml:"max_response_tokens,omitempty" glazed:"ai-max-response-tokens"`
	TopP              *float64          `yaml:"top_p,omitempty" glazed:"ai-top-p"`
	Temperature       *float64          `yaml:"temperature,omitempty" glazed:"ai-temperature"`
	Stop              []string          `yaml:"stop,omitempty" glazed:"ai-stop"`
	APIKeys           map[string]string `yaml:"api_keys,omitempty" glazed:"*-api-key"`
	Stream            bool              `yaml:"stream,omitempty"`

	// Caching settings
	CacheType       string `yaml:"cache_type,omitempty" glazed:"ai-cache-type"`
	CacheMaxSize    int64  `yaml:"cache_max_size,omitempty" glazed:"ai-cache-max-size"`
	CacheMaxEntries int    `yaml:"cache_max_entries,omitempty" glazed:"ai-cache-max-entries"`
	CacheDirectory  string `yaml:"cache_directory,omitempty" glazed:"ai-cache-directory"`
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

	p, err := NewChatValueSection()
	if err != nil {
		return nil, err
	}
	err = p.InitializeStructFromFieldDefaults(s)
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

type ChatValueSection struct {
	*schema.SectionImpl `yaml:",inline"`
}

const AiChatSlug = "ai-chat"

func NewChatValueSection(options ...schema.SectionOption) (*ChatValueSection, error) {
	ret, err := schema.NewSectionFromYAML(settingsYAML, options...)
	if err != nil {
		return nil, err
	}

	return &ChatValueSection{
		SectionImpl: ret,
	}, nil
}

// WrapWithCache removed with steps API deprecation. Use engine-level caching/middleware instead.

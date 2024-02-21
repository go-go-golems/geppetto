package settings

import (
	_ "embed"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/huandu/go-clone"
)

type ApiType string

const (
	ApiTypeOpenAI    ApiType = "openai"
	ApiTypeAnyScale  ApiType = "anyscale"
	ApiTypeFireworks ApiType = "fireworks"
	ApiTypeClaude    ApiType = "claude"
	// not implemented from here on down
	ApiTypeOllama     ApiType = "ollama"
	ApiTypeMistral    ApiType = "mistral"
	ApiTypePerplexity ApiType = "perplexity"
	// Cohere has connectors
	ApiTypeCohere ApiType = "cohere"
)

type ChatSettings struct {
	Engine            *string           `yaml:"engine,omitempty" glazed.parameter:"ai-engine"`
	ApiType           *ApiType          `yaml:"api_type,omitempty" glazed.parameter:"ai-api-type"`
	MaxResponseTokens *int              `yaml:"max_response_tokens,omitempty" glazed.parameter:"ai-max-response-tokens"`
	TopP              *float64          `yaml:"top_p,omitempty" glazed.parameter:"ai-top-p"`
	Temperature       *float64          `yaml:"temperature,omitempty" glazed.parameter:"ai-temperature"`
	Stop              []string          `yaml:"stop,omitempty" glazed.parameter:"ai-stop"`
	Stream            bool              `yaml:"stream,omitempty" glazed.parameter:"ai-stream"`
	APIKeys           map[string]string `yaml:"api_keys,omitempty" glazed.parameter:"*-api-key"`
}

func NewChatSettings() *ChatSettings {
	return &ChatSettings{
		Engine:            nil,
		ApiType:           nil,
		MaxResponseTokens: nil,
		TopP:              nil,
		Temperature:       nil,
		Stop:              []string{},
		Stream:            false,
		APIKeys:           map[string]string{},
	}
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

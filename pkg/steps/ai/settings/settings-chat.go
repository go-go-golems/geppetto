package settings

import (
	_ "embed"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
)

type ChatSettings struct {
	Engine            *string  `yaml:"engine,omitempty" glazed.parameter:"ai-engine"`
	MaxResponseTokens *int     `yaml:"max_response_tokens,omitempty" glazed.parameter:"ai-max-response-tokens"`
	TopP              *float64 `yaml:"top_p,omitempty" glazed.parameter:"ai-top-p"`
	Temperature       *float64 `yaml:"temperature,omitempty" glazed.parameter:"ai-temperature"`
	Stop              []string `yaml:"stop,omitempty" glazed.parameter:"ai-stop"`
	Stream            bool     `yaml:"stream,omitempty" glazed.parameter:"ai-stream"`
}

func NewChatSettings() *ChatSettings {
	return &ChatSettings{
		Engine:            nil,
		MaxResponseTokens: nil,
		TopP:              nil,
		Temperature:       nil,
		Stop:              []string{},
		Stream:            false,
	}
}

func (s *ChatSettings) Clone() *ChatSettings {
	return &ChatSettings{
		Engine:            s.Engine,
		MaxResponseTokens: s.MaxResponseTokens,
		TopP:              s.TopP,
		Temperature:       s.Temperature,
		Stop:              s.Stop,
		Stream:            s.Stream,
	}
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

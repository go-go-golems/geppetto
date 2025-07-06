package gemini

import (
	_ "embed"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/huandu/go-clone"
)

type Settings struct{}

func NewSettings() (*Settings, error) {
	s := &Settings{}
	p, err := NewParameterLayer()
	if err != nil {
		return nil, err
	}
	if err := p.InitializeStructFromParameterDefaults(s); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Settings) Clone() *Settings {
	return clone.Clone(s).(*Settings)
}

const GeminiChatSlug = "gemini-chat"

//go:embed "gemini.yaml"
var settingsYAML []byte

type ParameterLayer struct {
	*layers.ParameterLayerImpl `yaml:",inline"`
}

func NewParameterLayer(options ...layers.ParameterLayerOptions) (*ParameterLayer, error) {
	ret, err := layers.NewParameterLayerFromYAML(settingsYAML, options...)
	if err != nil {
		return nil, err
	}
	return &ParameterLayer{ParameterLayerImpl: ret}, nil
}

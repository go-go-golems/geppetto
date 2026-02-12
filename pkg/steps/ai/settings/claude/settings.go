package claude

import (
	_ "embed"

	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/huandu/go-clone"
)

type Settings struct {
	TopK   *int    `yaml:"top_k,omitempty" glazed:"claude-top-k"`
	UserID *string `yaml:"user_id,omitempty" glazed:"claude-user-id"`
}

func NewSettings() (*Settings, error) {
	s := &Settings{
		TopK:   nil,
		UserID: nil,
	}

	p, err := NewParameterLayer()
	if err != nil {
		return nil, err
	}

	err = p.InitializeStructFromFieldDefaults(s)
	if err != nil {
		return nil, err
	}

	return s, nil
}

func (s *Settings) Clone() *Settings {
	return clone.Clone(s).(*Settings)
}

const ClaudeChatSlug = "claude-chat"

//go:embed "claude.yaml"
var settingsYAML []byte

type ParameterLayer struct {
	*schema.SectionImpl `yaml:",inline"`
}

func NewParameterLayer(options ...schema.SectionOption) (*ParameterLayer, error) {
	ret, err := schema.NewSectionFromYAML(settingsYAML, options...)
	if err != nil {
		return nil, err
	}

	return &ParameterLayer{
		SectionImpl: ret,
	}, nil
}

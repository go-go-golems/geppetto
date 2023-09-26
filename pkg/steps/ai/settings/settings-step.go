package settings

import (
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings/claude"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings/openai"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"gopkg.in/yaml.v3"
	"io"
)

type factoryConfigFileWrapper struct {
	Factories *StepSettings
}

type StepSettings struct {
	Chat   *ChatSettings    `yaml:"chat,omitempty"`
	OpenAI *openai.Settings `yaml:"openai,omitempty"`
	Client *ClientSettings  `yaml:"client,omitempty"`
	Claude *claude.Settings `yaml:"claude,omitempty"`
}

func NewStepSettingsFromYAML(s io.Reader) (*StepSettings, error) {
	settings_ := factoryConfigFileWrapper{
		Factories: &StepSettings{
			Chat:   NewChatSettings(),
			OpenAI: openai.NewSettings(),
			Client: NewClientSettings(),
			Claude: claude.NewSettings(),
		},
	}
	if err := yaml.NewDecoder(s).Decode(&settings_); err != nil {
		return nil, err
	}

	return settings_.Factories, nil
}

func (s *StepSettings) UpdateFromParsedLayers(parsedLayers map[string]*layers.ParsedParameterLayer) error {
	if parsedLayers["ai-chat"] != nil {
		err := s.Chat.UpdateFromParsedLayer(parsedLayers["ai-chat"])
		if err != nil {
			return err
		}
	}

	if parsedLayers["openai-chat"] != nil {
		err := s.OpenAI.UpdateFromParsedLayer(parsedLayers["openai-chat"])
		if err != nil {
			return err
		}
	}

	if parsedLayers["claude-chat"] != nil {
		err := s.Claude.UpdateFromParsedLayer(parsedLayers["claude-chat"])
		if err != nil {
			return err
		}
	}

	if parsedLayers["ai-client"] != nil {
		err := s.Client.UpdateFromParsedLayer(parsedLayers["ai-client"])
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *StepSettings) Clone() *StepSettings {
	return &StepSettings{
		Chat:   s.Chat.Clone(),
		OpenAI: s.OpenAI.Clone(),
		Client: s.Client.Clone(),
		Claude: s.Claude.Clone(),
	}
}

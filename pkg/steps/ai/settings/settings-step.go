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

func NewStepSettings() *StepSettings {
	return &StepSettings{
		Chat:   NewChatSettings(),
		OpenAI: openai.NewSettings(),
		Client: NewClientSettings(),
		Claude: claude.NewSettings(),
	}
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

func (s *StepSettings) UpdateFromParsedLayers(parsedLayers *layers.ParsedLayers) error {
	aiChatLayer, ok := parsedLayers.Get("ai-chat")
	if ok {
		err := s.Chat.UpdateFromParsedLayer(aiChatLayer)
		if err != nil {
			return err
		}
	}

	openaiLayer, ok := parsedLayers.Get("openai-chat")
	if ok {
		err := s.OpenAI.UpdateFromParsedLayer(openaiLayer)
		if err != nil {
			return err
		}
	}

	claudeChatLayer, ok := parsedLayers.Get("claude-chat")
	if ok {
		err := s.Claude.UpdateFromParsedLayer(claudeChatLayer)
		if err != nil {
			return err
		}
	}

	aiClientLayer, ok := parsedLayers.Get("ai-client")
	if ok {
		err := s.Client.UpdateFromParsedLayer(aiClientLayer)
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

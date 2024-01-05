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
	Chat   *ChatSettings    `yaml:"chat,omitempty" glazed.layer:"ai-chat"`
	OpenAI *openai.Settings `yaml:"openai,omitempty" glazed.layer:"openai-chat"`
	Client *ClientSettings  `yaml:"client,omitempty" glazed.layer:"ai-client"`
	Claude *claude.Settings `yaml:"claude,omitempty" glazed.layer:"claude-chat"`
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

// UpdateFromParsedLayers updates the settings of a chat step from the parsedLayers of a glazed command.
// TODO(manuel, 2024-01-05) Not sure how this relates to InitializeStruct
func (s *StepSettings) UpdateFromParsedLayers(parsedLayers *layers.ParsedLayers) error {
	err := parsedLayers.InitializeStruct(AiClientSlug, s.Client)
	if err != nil {
		return err
	}

	err = parsedLayers.InitializeStruct(AiChatSlug, s.Chat)
	if err != nil {
		return err
	}

	err = parsedLayers.InitializeStruct(openai.OpenAiChatSlug, s.OpenAI)
	if err != nil {
		return err
	}

	err = parsedLayers.InitializeStruct(claude.ClaudeChatSlug, s.Claude)
	if err != nil {
		return err
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

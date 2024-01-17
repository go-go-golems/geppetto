package settings

import (
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings/claude"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings/ollama"
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
	Ollama *ollama.Settings `yaml:"ollama,omitempty" glazed.layer:"ollama-chat"`
}

func NewStepSettings() *StepSettings {
	return &StepSettings{
		Chat:   NewChatSettings(),
		OpenAI: openai.NewSettings(),
		Client: NewClientSettings(),
		Claude: claude.NewSettings(),
		Ollama: ollama.NewSettings(),
	}
}

func NewStepSettingsFromYAML(s io.Reader) (*StepSettings, error) {
	settings_ := factoryConfigFileWrapper{
		Factories: &StepSettings{
			Chat:   NewChatSettings(),
			OpenAI: openai.NewSettings(),
			Client: NewClientSettings(),
			Claude: claude.NewSettings(),
			Ollama: ollama.NewSettings(),
		},
	}
	if err := yaml.NewDecoder(s).Decode(&settings_); err != nil {
		return nil, err
	}

	return settings_.Factories, nil
}

func (ss *StepSettings) GetMetadata() map[string]interface{} {
	metadata := make(map[string]interface{})

	if ss.Chat != nil {
		if ss.Chat.Engine != nil {
			metadata["ai-engine"] = *ss.Chat.Engine
		}
		if ss.Chat.MaxResponseTokens != nil {
			metadata["ai-max-response-tokens"] = *ss.Chat.MaxResponseTokens
		}
		if ss.Chat.TopP != nil && *ss.Chat.TopP != 1 {
			metadata["ai-top-p"] = *ss.Chat.TopP
		}
		if ss.Chat.Temperature != nil {
			metadata["ai-temperature"] = *ss.Chat.Temperature
		}
		if len(ss.Chat.Stop) > 0 {
			metadata["ai-stop"] = ss.Chat.Stop
		}
		metadata["ai-stream"] = ss.Chat.Stream
	}

	if ss.OpenAI != nil {
		if ss.OpenAI.N != nil && *ss.OpenAI.N != 1 {
			metadata["openai-n"] = *ss.OpenAI.N
		}
		if ss.OpenAI.PresencePenalty != nil && *ss.OpenAI.PresencePenalty != 0 {
			metadata["openai-presence-penalty"] = *ss.OpenAI.PresencePenalty
		}
		if ss.OpenAI.FrequencyPenalty != nil && *ss.OpenAI.FrequencyPenalty != 0 {
			metadata["openai-frequency-penalty"] = *ss.OpenAI.FrequencyPenalty
		}
		if len(ss.OpenAI.LogitBias) > 0 {
			metadata["openai-logit-bias"] = ss.OpenAI.LogitBias
		}
		if ss.OpenAI.BaseURL != nil {
			metadata["openai-base-url"] = *ss.OpenAI.BaseURL
		}
	}

	if ss.Client != nil {
		if ss.Client.Timeout != nil {
			metadata["timeout"] = ss.Client.Timeout.String()
		}
		if ss.Client.TimeoutSeconds != nil {
			metadata["timeout_second"] = *ss.Client.TimeoutSeconds
		}
		if ss.Client.Organization != nil && *ss.Client.Organization != "" {
			metadata["organization"] = *ss.Client.Organization
		}
		if ss.Client.UserAgent != nil {
			metadata["user-agent"] = *ss.Client.UserAgent
		}
		// Note: HTTPClient is not included as it's not a simple scalar value
	}

	if ss.Claude != nil {
		if ss.Claude.TopK != nil && *ss.Claude.TopK != 1 {
			metadata["claude-top-k"] = *ss.Claude.TopK
		}
		if ss.Claude.UserID != nil && *ss.Claude.UserID != "" {
			metadata["claude-user-id"] = *ss.Claude.UserID
		}
		if ss.Claude.BaseURL != nil {
			metadata["claude-base-url"] = *ss.Claude.BaseURL
		}
	}

	if ss.Ollama != nil {
		if ss.Ollama.Temperature != nil && *ss.Ollama.Temperature != 0 {
			metadata["ollama-temperature"] = *ss.Ollama.Temperature
		}
		if ss.Ollama.Seed != nil && *ss.Ollama.Seed != 0 {
			metadata["ollama-seed"] = *ss.Ollama.Seed
		}

		if ss.Ollama.Stop != nil && *ss.Ollama.Stop != "" {
			metadata["ollama-stop"] = *ss.Ollama.Stop
		}

		if ss.Ollama.TopK != nil && *ss.Ollama.TopK != 40 {
			metadata["ollama-top-k"] = *ss.Ollama.TopK
		}

		if ss.Ollama.TopP != nil && *ss.Ollama.TopP != 0.9 {
			metadata["ollama-top-p"] = *ss.Ollama.TopP
		}
	}

	return metadata
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

	err = parsedLayers.InitializeStruct(ollama.OllamaChatSlug, s.Ollama)
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
		Ollama: s.Ollama.Clone(),
	}
}

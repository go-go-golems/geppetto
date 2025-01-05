package settings

import (
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings/claude"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings/ollama"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings/openai"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/huandu/go-clone"
	"gopkg.in/yaml.v3"
	"io"
)

type factoryConfigFileWrapper struct {
	Factories *StepSettings
}

type APISettings struct {
	APIKeys  map[ApiType]string `yaml:"api_keys,omitempty" glazed.parameter:"*-api-key"`
	BaseUrls map[ApiType]string `yaml:"base_urls,omitempty" glazed.parameter:"*-base-url"`
}

func NewAPISettings() *APISettings {
	return &APISettings{
		APIKeys:  map[ApiType]string{},
		BaseUrls: map[ApiType]string{},
	}
}

func (s *APISettings) Clone() *APISettings {
	return clone.Clone(s).(*APISettings)
}

type StepSettings struct {
	API    *APISettings     `yaml:"api_keys,omitempty"`
	Chat   *ChatSettings    `yaml:"chat,omitempty" glazed.layer:"ai-chat"`
	OpenAI *openai.Settings `yaml:"openai,omitempty" glazed.layer:"openai-chat"`
	Client *ClientSettings  `yaml:"client,omitempty" glazed.layer:"ai-client"`
	Claude *claude.Settings `yaml:"claude,omitempty" glazed.layer:"claude-chat"`
	Ollama *ollama.Settings `yaml:"ollama,omitempty" glazed.layer:"ollama-chat"`
	// NOTE: Maybe we should separate the StepSettings struct into:
	// - Provider settings (API, OpenAI, Claude, Ollama)
	// - Chat settings (Chat, OpenAI, Claude, Ollama)
	// - Embeddings settings (Embeddings)
	Embeddings *EmbeddingsSettings `yaml:"embeddings,omitempty" glazed.layer:"embeddings"`
}

func NewStepSettings() (*StepSettings, error) {
	chatSettings, err := NewChatSettings()
	if err != nil {
		return nil, err
	}
	openaiSettings, err := openai.NewSettings()
	if err != nil {
		return nil, err
	}
	claudeSettings, err := claude.NewSettings()
	if err != nil {
		return nil, err
	}
	ollamaSettings, err := ollama.NewSettings()
	if err != nil {
		return nil, err
	}
	embeddingsSettings, err := NewEmbeddingsSettings()
	if err != nil {
		return nil, err
	}

	return &StepSettings{
		Chat:       chatSettings,
		OpenAI:     openaiSettings,
		Client:     NewClientSettings(),
		Claude:     claudeSettings,
		Ollama:     ollamaSettings,
		API:        NewAPISettings(),
		Embeddings: embeddingsSettings,
	}, nil
}

func NewStepSettingsFromYAML(s io.Reader) (*StepSettings, error) {
	stepSettings, err := NewStepSettings()
	if err != nil {
		return nil, err
	}

	settings_ := factoryConfigFileWrapper{
		Factories: stepSettings,
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
		if ss.Chat.ApiType != nil {
			metadata["ai-api-type"] = *ss.Chat.ApiType

			baseUrl, ok := ss.API.BaseUrls[*ss.Chat.ApiType]
			if ok {
				metadata["ai-base-url"] = baseUrl
			}
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
	}

	if ss.Claude != nil {
		if ss.Claude.TopK != nil && *ss.Claude.TopK != 1 {
			metadata["claude-top-k"] = *ss.Claude.TopK
		}
		if ss.Claude.UserID != nil && *ss.Claude.UserID != "" {
			metadata["claude-user-id"] = *ss.Claude.UserID
		}
	}

	if ss.Ollama != nil {
		if ss.Ollama.Temperature != nil && *ss.Ollama.Temperature != 0 {
			metadata["ollama-temperature"] = *ss.Ollama.Temperature
		}
		if ss.Ollama.Seed != nil && *ss.Ollama.Seed != 0 {
			metadata["ollama-seed"] = *ss.Ollama.Seed
		}
		if len(ss.Ollama.Stop) > 0 {
			metadata["ollama-stop"] = ss.Ollama.Stop
		}
		if ss.Ollama.TopK != nil && *ss.Ollama.TopK != 40 {
			metadata["ollama-top-k"] = *ss.Ollama.TopK
		}
		if ss.Ollama.TopP != nil && *ss.Ollama.TopP != 0.9 {
			metadata["ollama-top-p"] = *ss.Ollama.TopP
		}
	}

	if ss.Embeddings != nil {
		if ss.Embeddings.Engine != nil {
			metadata["embeddings-engine"] = *ss.Embeddings.Engine
		}
		if ss.Embeddings.Type != nil {
			metadata["embeddings-type"] = *ss.Embeddings.Type
		}
		if ss.Embeddings.Dimensions != nil {
			metadata["embeddings-dimensions"] = *ss.Embeddings.Dimensions
		}
	}

	return metadata
}

func (s *StepSettings) Clone() *StepSettings {
	return &StepSettings{
		API:        s.API.Clone(),
		Chat:       s.Chat.Clone(),
		OpenAI:     s.OpenAI.Clone(),
		Client:     s.Client.Clone(),
		Claude:     s.Claude.Clone(),
		Ollama:     s.Ollama.Clone(),
		Embeddings: s.Embeddings.Clone(),
	}
}

func (ss *StepSettings) UpdateFromParsedLayers(parsedLayers *layers.ParsedLayers) error {
	err := parsedLayers.InitializeStruct(AiClientSlug, ss.Client)
	if err != nil {
		return err
	}

	err = parsedLayers.InitializeStruct(AiChatSlug, ss.Chat)
	if err != nil {
		return err
	}

	err = parsedLayers.InitializeStruct(openai.OpenAiChatSlug, ss.OpenAI)
	if err != nil {
		return err
	}

	err = parsedLayers.InitializeStruct(claude.ClaudeChatSlug, ss.Claude)
	if err != nil {
		return err
	}

	err = parsedLayers.InitializeStruct(EmbeddingsSlug, ss.Embeddings)
	if err != nil {
		return err
	}

	apiSlugs := []string{
		openai.OpenAiChatSlug,
		claude.ClaudeChatSlug,
		EmbeddingsSlug,
	}
	for _, slug := range apiSlugs {
		err = parsedLayers.InitializeStruct(slug, ss.API)
		if err != nil {
			return err
		}
	}

	return nil
}

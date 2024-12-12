package embeddings

import (
	"fmt"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/sashabaranov/go-openai"
)

// SettingsFactory creates embedding providers based on chat settings
type SettingsFactory struct {
	stepSettings *settings.StepSettings
}

// NewSettingsFactory creates a new factory that uses step settings
func NewSettingsFactory(stepSettings *settings.StepSettings) *SettingsFactory {
	return &SettingsFactory{
		stepSettings: stepSettings,
	}
}

// NewProvider creates a new embedding provider based on the step settings
func (f *SettingsFactory) NewProvider() (Provider, error) {
	if f.stepSettings == nil {
		return nil, fmt.Errorf("no settings provided")
	}

	if f.stepSettings.Embeddings == nil || f.stepSettings.Embeddings.Type == nil {
		return nil, fmt.Errorf("no embeddings type specified in settings")
	}

	if f.stepSettings.Embeddings.Engine == nil {
		return nil, fmt.Errorf("no embeddings model specified in settings")
	}

	dimensions := 1536 // Default for OpenAI
	if f.stepSettings.Embeddings.Dimensions != nil {
		dimensions = *f.stepSettings.Embeddings.Dimensions
	}

	switch *f.stepSettings.Embeddings.Type {
	case "ollama":
		baseURL := "http://localhost:11434"
		if urls := f.stepSettings.API.BaseUrls; urls != nil {
			if url, ok := urls[settings.ApiTypeOllama]; ok {
				baseURL = url
			}
		}
		return NewOllamaProvider(baseURL, *f.stepSettings.Embeddings.Engine, dimensions), nil

	case "openai":
		apiKey := ""
		if keys := f.stepSettings.API.APIKeys; keys != nil {
			if key, ok := keys[settings.ApiTypeOpenAI]; ok {
				apiKey = key
			}
		}
		if apiKey == "" {
			return nil, fmt.Errorf("no API key provided for OpenAI")
		}

		return NewOpenAIProvider(apiKey, openai.EmbeddingModel(*f.stepSettings.Embeddings.Engine), dimensions), nil

	default:
		return nil, fmt.Errorf("unsupported provider type for embeddings: %s", *f.stepSettings.Embeddings.Type)
	}
}

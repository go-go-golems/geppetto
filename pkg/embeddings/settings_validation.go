package embeddings

import (
	"fmt"
	"strings"

	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
)

const (
	openAIEmbeddingProvider = "openai"
	ollamaEmbeddingProvider = "ollama"
)

// ValidateInferenceSettingsForEmbeddings verifies that final, already-merged
// inference settings can construct an embedding provider.
//
// Profile-backed callers should run this after engine profile stack resolution
// and before provider construction so users get profile-oriented diagnostics
// instead of low-level provider errors such as "no API key provided".
func ValidateInferenceSettingsForEmbeddings(s *settings.InferenceSettings) error {
	if s == nil {
		return fmt.Errorf("selected profile is not embedding-capable: inference settings are required")
	}
	if s.Embeddings == nil {
		return fmt.Errorf("selected profile is not embedding-capable: missing inference_settings.embeddings")
	}

	providerType := strings.TrimSpace(s.Embeddings.Type)
	if providerType == "" {
		return fmt.Errorf("selected profile is not embedding-capable: missing inference_settings.embeddings.type")
	}

	if strings.TrimSpace(s.Embeddings.Engine) == "" {
		return fmt.Errorf("selected profile is not embedding-capable: missing inference_settings.embeddings.engine")
	}

	switch providerType {
	case openAIEmbeddingProvider:
		if s.API == nil || strings.TrimSpace(s.API.APIKeys["openai-api-key"]) == "" {
			return fmt.Errorf("selected OpenAI embedding profile has no openai-api-key; stack an OpenAI base profile or set inference_settings.api.api_keys.openai-api-key")
		}
	case ollamaEmbeddingProvider:
		if s.Embeddings.Dimensions == 0 {
			return fmt.Errorf("selected Ollama embedding profile must set inference_settings.embeddings.dimensions")
		}
	default:
		return fmt.Errorf("unsupported embeddings provider type %q; supported values are openai and ollama", providerType)
	}

	return nil
}

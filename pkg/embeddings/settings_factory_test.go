package embeddings

import (
	"fmt"
	"testing"

	"github.com/go-go-golems/geppetto/pkg/embeddings/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSettingsFactory_NewProvider(t *testing.T) {
	tests := []struct {
		name          string
		config        *config.EmbeddingsConfig
		opts          []ProviderOption
		expectedError string
		expectedType  string
	}{
		{
			name: "openai provider",
			config: &config.EmbeddingsConfig{
				Type:       "openai",
				Engine:     "text-embedding-3-small",
				Dimensions: 1536,
				APIKeys: map[string]string{
					"openai-api-key": "test-key",
				},
			},
			expectedType: "*embeddings.OpenAIProvider",
		},
		{
			name: "ollama provider",
			config: &config.EmbeddingsConfig{
				Type:       "ollama",
				Engine:     "all-minilm",
				Dimensions: 384,
				BaseURLs: map[string]string{
					"ollama-base-url": "http://localhost:11434",
				},
			},
			expectedType: "*embeddings.OllamaProvider",
		},
		{
			name: "cohere provider",
			config: &config.EmbeddingsConfig{
				Type:       "cohere",
				Engine:     "embed-v4.0",
				Dimensions: 1024,
				APIKeys: map[string]string{
					"cohere-api-key": "test-cohere-key",
				},
			},
			expectedType: "*embeddings.CohereProvider",
		},
		{
			name: "cached provider",
			config: &config.EmbeddingsConfig{
				Type:            "openai",
				Engine:          "text-embedding-3-small",
				Dimensions:      1536,
				CacheType:       "memory",
				CacheMaxEntries: 100,
				APIKeys: map[string]string{
					"openai-api-key": "test-key",
				},
			},
			expectedType: "*embeddings.CachedProvider",
		},
		{
			name: "missing api key",
			config: &config.EmbeddingsConfig{
				Type:       "cohere",
				Engine:     "embed-v4.0",
				Dimensions: 1024,
			},
			expectedError: "no API key provided for Cohere",
		},
		{
			name: "override with options",
			config: &config.EmbeddingsConfig{
				Type:       "openai",
				Engine:     "text-embedding-3-small",
				Dimensions: 1536,
				APIKeys: map[string]string{
					"openai-api-key": "test-key",
				},
			},
			opts: []ProviderOption{
				WithType("cohere"),
				WithEngine("embed-v4.0"),
				WithDimensions(1024),
				WithAPIKey("override-key"),
			},
			expectedType: "*embeddings.CohereProvider",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			factory := NewSettingsFactory(tt.config)
			provider, err := factory.NewProvider(tt.opts...)

			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, provider)

			// Check the type of provider
			assert.Equal(t, tt.expectedType, TypeName(provider))

			// Test basic functionality
			model := provider.GetModel()
			assert.NotEmpty(t, model.Name)
			assert.Greater(t, model.Dimensions, 0)
		})
	}
}

// Helper function to get the type name as a string
func TypeName(v interface{}) string {
	return formatTypeName(v)
}

// formatTypeName formats the type name
func formatTypeName(v interface{}) string {
	return fmt.Sprintf("%T", v)
}

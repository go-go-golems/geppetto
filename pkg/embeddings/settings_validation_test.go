package embeddings

import (
	"strings"
	"testing"

	"github.com/go-go-golems/geppetto/pkg/embeddings/config"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
)

func TestValidateInferenceSettingsForEmbeddings(t *testing.T) {
	tests := []struct {
		name    string
		in      *settings.InferenceSettings
		wantErr string
	}{
		{
			name:    "nil settings",
			in:      nil,
			wantErr: "inference settings are required",
		},
		{
			name:    "chat only profile",
			in:      &settings.InferenceSettings{},
			wantErr: "missing inference_settings.embeddings",
		},
		{
			name: "missing embeddings type",
			in: &settings.InferenceSettings{
				Embeddings: &config.EmbeddingsConfig{Engine: "text-embedding-3-small", Dimensions: 1536},
			},
			wantErr: "missing inference_settings.embeddings.type",
		},
		{
			name: "missing embeddings engine",
			in: &settings.InferenceSettings{
				Embeddings: &config.EmbeddingsConfig{Type: "openai", Dimensions: 1536},
			},
			wantErr: "missing inference_settings.embeddings.engine",
		},
		{
			name: "openai missing key",
			in: &settings.InferenceSettings{
				API:        &settings.APISettings{APIKeys: map[string]string{}},
				Embeddings: &config.EmbeddingsConfig{Type: "openai", Engine: "text-embedding-3-small", Dimensions: 1536},
			},
			wantErr: "has no openai-api-key",
		},
		{
			name: "unsupported provider",
			in: &settings.InferenceSettings{
				Embeddings: &config.EmbeddingsConfig{Type: "cohere", Engine: "embed-english-v3", Dimensions: 1024},
			},
			wantErr: "unsupported embeddings provider type",
		},
		{
			name: "ollama missing dimensions",
			in: &settings.InferenceSettings{
				Embeddings: &config.EmbeddingsConfig{Type: "ollama", Engine: "nomic-embed-text"},
			},
			wantErr: "must set inference_settings.embeddings.dimensions",
		},
		{
			name: "openai complete",
			in: &settings.InferenceSettings{
				API:        &settings.APISettings{APIKeys: map[string]string{"openai-api-key": "test-key"}},
				Embeddings: &config.EmbeddingsConfig{Type: "openai", Engine: "text-embedding-3-small", Dimensions: 1536},
			},
		},
		{
			name: "ollama complete",
			in: &settings.InferenceSettings{
				Embeddings: &config.EmbeddingsConfig{Type: "ollama", Engine: "nomic-embed-text", Dimensions: 768},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateInferenceSettingsForEmbeddings(tt.in)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("ValidateInferenceSettingsForEmbeddings returned error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("ValidateInferenceSettingsForEmbeddings returned nil error, want %q", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("ValidateInferenceSettingsForEmbeddings error = %q, want substring %q", err.Error(), tt.wantErr)
			}
		})
	}
}

package engineprofiles

import (
	"context"
	"testing"

	"github.com/go-go-golems/geppetto/pkg/embeddings/config"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
)

func TestResolveEngineProfile_EmbeddingProfileStacksOpenAIBase(t *testing.T) {
	registry := mustNewStackTestRegistry(t, &EngineProfileRegistry{
		Slug:                     MustRegistrySlug("default"),
		DefaultEngineProfileSlug: MustEngineProfileSlug("openai-embedding-small"),
		Profiles: map[EngineProfileSlug]*EngineProfile{
			MustEngineProfileSlug("openai-base"): {
				Slug: MustEngineProfileSlug("openai-base"),
				InferenceSettings: &settings.InferenceSettings{
					API: &settings.APISettings{
						APIKeys: map[string]string{"openai-api-key": "test-openai-key"},
					},
				},
			},
			MustEngineProfileSlug("openai-embedding-small"): {
				Slug: MustEngineProfileSlug("openai-embedding-small"),
				Stack: []EngineProfileRef{
					{EngineProfileSlug: MustEngineProfileSlug("openai-base")},
				},
				InferenceSettings: &settings.InferenceSettings{
					Embeddings: &config.EmbeddingsConfig{
						Type:           "openai",
						Engine:         "text-embedding-3-small",
						Dimensions:     1536,
						CacheType:      "file",
						CacheDirectory: "./.geppetto/embeddings-cache/openai-text-embedding-3-small",
					},
				},
			},
		},
	})

	resolved, err := registry.ResolveEngineProfile(context.Background(), ResolveInput{EngineProfileSlug: MustEngineProfileSlug("openai-embedding-small")})
	if err != nil {
		t.Fatalf("ResolveEngineProfile failed: %v", err)
	}

	if resolved.InferenceSettings == nil || resolved.InferenceSettings.API == nil {
		t.Fatalf("resolved inference settings did not include API settings: %#v", resolved.InferenceSettings)
	}
	if got := resolved.InferenceSettings.API.APIKeys["openai-api-key"]; got != "test-openai-key" {
		t.Fatalf("openai key mismatch: got=%q", got)
	}
	if resolved.InferenceSettings.Embeddings == nil {
		t.Fatalf("resolved inference settings did not include embeddings settings")
	}
	if got := resolved.InferenceSettings.Embeddings.Type; got != "openai" {
		t.Fatalf("embedding type mismatch: got=%q", got)
	}
	if got := resolved.InferenceSettings.Embeddings.Engine; got != "text-embedding-3-small" {
		t.Fatalf("embedding engine mismatch: got=%q", got)
	}
	if got := resolved.InferenceSettings.Embeddings.Dimensions; got != 1536 {
		t.Fatalf("embedding dimensions mismatch: got=%d", got)
	}
}

func TestResolveEngineProfile_EmbeddingProfileCanConfigureOllamaBaseURL(t *testing.T) {
	registry := mustNewStackTestRegistry(t, &EngineProfileRegistry{
		Slug:                     MustRegistrySlug("default"),
		DefaultEngineProfileSlug: MustEngineProfileSlug("ollama-nomic-embedding"),
		Profiles: map[EngineProfileSlug]*EngineProfile{
			MustEngineProfileSlug("ollama-base"): {
				Slug: MustEngineProfileSlug("ollama-base"),
				InferenceSettings: &settings.InferenceSettings{
					API: &settings.APISettings{
						BaseUrls: map[string]string{"ollama-base-url": "http://localhost:11434"},
					},
				},
			},
			MustEngineProfileSlug("ollama-nomic-embedding"): {
				Slug: MustEngineProfileSlug("ollama-nomic-embedding"),
				Stack: []EngineProfileRef{
					{EngineProfileSlug: MustEngineProfileSlug("ollama-base")},
				},
				InferenceSettings: &settings.InferenceSettings{
					Embeddings: &config.EmbeddingsConfig{
						Type:           "ollama",
						Engine:         "nomic-embed-text",
						Dimensions:     768,
						CacheType:      "file",
						CacheDirectory: "./.geppetto/embeddings-cache/ollama-nomic-embed-text",
					},
				},
			},
		},
	})

	resolved, err := registry.ResolveEngineProfile(context.Background(), ResolveInput{EngineProfileSlug: MustEngineProfileSlug("ollama-nomic-embedding")})
	if err != nil {
		t.Fatalf("ResolveEngineProfile failed: %v", err)
	}

	if resolved.InferenceSettings == nil || resolved.InferenceSettings.API == nil {
		t.Fatalf("resolved inference settings did not include API settings: %#v", resolved.InferenceSettings)
	}
	if got := resolved.InferenceSettings.API.BaseUrls["ollama-base-url"]; got != "http://localhost:11434" {
		t.Fatalf("ollama base URL mismatch: got=%q", got)
	}
	if resolved.InferenceSettings.Embeddings == nil {
		t.Fatalf("resolved inference settings did not include embeddings settings")
	}
	if got := resolved.InferenceSettings.Embeddings.Type; got != "ollama" {
		t.Fatalf("embedding type mismatch: got=%q", got)
	}
	if got := resolved.InferenceSettings.Embeddings.Engine; got != "nomic-embed-text" {
		t.Fatalf("embedding engine mismatch: got=%q", got)
	}
	if got := resolved.InferenceSettings.Embeddings.Dimensions; got != 768 {
		t.Fatalf("embedding dimensions mismatch: got=%d", got)
	}
}

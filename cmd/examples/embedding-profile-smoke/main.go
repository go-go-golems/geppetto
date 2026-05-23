package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-go-golems/geppetto/pkg/embeddings"
	"github.com/go-go-golems/geppetto/pkg/embeddings/config"
	profiles "github.com/go-go-golems/geppetto/pkg/engineprofiles"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
)

type output struct {
	Profile           string    `json:"profile"`
	ProfileRegistries string    `json:"profile_registries"`
	ProviderType      string    `json:"provider_type"`
	Model             string    `json:"model"`
	ConfiguredDims    int       `json:"configured_dimensions"`
	ActualDims        int       `json:"actual_dimensions"`
	KeyConfigured     bool      `json:"key_configured,omitempty"`
	BaseURLConfigured bool      `json:"base_url_configured,omitempty"`
	Preview           []float32 `json:"preview,omitempty"`
}

func main() {
	var (
		profileRegistries = flag.String("profile-registries", defaultPinocchioProfilesPath(), "comma-separated profile registry paths/DSNs")
		profile           = flag.String("profile", "", "embedding-capable profile to resolve; if empty, stack --base-profile with embedding flags")
		baseProfile       = flag.String("base-profile", "openai-responses-base", "base profile to stack when --profile is empty")
		embeddingType     = flag.String("embeddings-type", "openai", "embedding provider type: openai or ollama")
		embeddingEngine   = flag.String("embeddings-engine", "text-embedding-3-small", "embedding model/engine")
		dimensions        = flag.Int("embeddings-dimensions", 1536, "expected embedding vector dimensions")
		text              = flag.String("text", "hello profile-backed embeddings", "text to embed")
		jsonOutput        = flag.Bool("json", false, "print JSON output")
		preview           = flag.Int("preview", 5, "number of vector dimensions to print in preview")
		timeout           = flag.Duration("timeout", 30*time.Second, "embedding request timeout")
	)
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	resolved, closeFn, effectiveProfile, err := resolveSettings(ctx, *profileRegistries, *profile, *baseProfile, *embeddingType, *embeddingEngine, *dimensions)
	if err != nil {
		fatal(err)
	}
	if closeFn != nil {
		defer func() { _ = closeFn() }()
	}

	if err := embeddings.ValidateInferenceSettingsForEmbeddings(resolved); err != nil {
		fatal(err)
	}

	provider, err := embeddings.NewSettingsFactoryFromInferenceSettings(resolved).NewProvider()
	if err != nil {
		fatal(err)
	}

	vector, err := provider.GenerateEmbedding(ctx, *text)
	if err != nil {
		fatal(err)
	}

	model := provider.GetModel()
	if len(vector) != model.Dimensions {
		fatal(fmt.Errorf("dimension mismatch: configured=%d actual=%d", model.Dimensions, len(vector)))
	}

	out := output{
		Profile:           effectiveProfile,
		ProfileRegistries: *profileRegistries,
		ProviderType:      resolved.Embeddings.Type,
		Model:             model.Name,
		ConfiguredDims:    model.Dimensions,
		ActualDims:        len(vector),
		KeyConfigured:     resolved.API != nil && strings.TrimSpace(resolved.API.APIKeys["openai-api-key"]) != "",
		BaseURLConfigured: resolved.API != nil && (strings.TrimSpace(resolved.API.BaseUrls["ollama-base-url"]) != "" || strings.TrimSpace(resolved.API.BaseUrls["openai-base-url"]) != ""),
		Preview:           vectorPreview(vector, *preview),
	}

	if *jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(out); err != nil {
			fatal(err)
		}
		return
	}

	fmt.Printf("profile: %s\n", out.Profile)
	fmt.Printf("registries: %s\n", out.ProfileRegistries)
	fmt.Printf("provider: %s\n", out.ProviderType)
	fmt.Printf("model: %s\n", out.Model)
	fmt.Printf("dimensions: configured=%d actual=%d\n", out.ConfiguredDims, out.ActualDims)
	fmt.Printf("openai-key-configured: %t\n", out.KeyConfigured)
	fmt.Printf("base-url-configured: %t\n", out.BaseURLConfigured)
	fmt.Printf("preview: %v\n", out.Preview)
}

func resolveSettings(ctx context.Context, registryEntries string, profile string, baseProfile string, embeddingType string, embeddingEngine string, dimensions int) (*settings.InferenceSettings, func() error, string, error) {
	entries := splitRegistryEntries(registryEntries)
	specs, err := profiles.ParseRegistrySourceSpecs(entries)
	if err != nil {
		return nil, nil, "", err
	}
	chain, err := profiles.NewChainedRegistryFromSourceSpecs(ctx, specs)
	if err != nil {
		return nil, nil, "", err
	}

	if strings.TrimSpace(profile) != "" {
		profileSlug, err := profiles.ParseEngineProfileSlug(profile)
		if err != nil {
			_ = chain.Close()
			return nil, nil, "", err
		}
		resolved, err := chain.ResolveEngineProfile(ctx, profiles.ResolveInput{EngineProfileSlug: profileSlug})
		if err != nil {
			_ = chain.Close()
			return nil, nil, "", err
		}
		return resolved.InferenceSettings, chain.Close, profile, nil
	}

	baseSlug, err := profiles.ParseEngineProfileSlug(baseProfile)
	if err != nil {
		_ = chain.Close()
		return nil, nil, "", err
	}
	baseResolved, err := chain.ResolveEngineProfile(ctx, profiles.ResolveInput{EngineProfileSlug: baseSlug})
	if err != nil {
		_ = chain.Close()
		return nil, nil, "", err
	}

	overlay := &settings.InferenceSettings{
		Embeddings: &config.EmbeddingsConfig{
			Type:       embeddingType,
			Engine:     embeddingEngine,
			Dimensions: dimensions,
		},
	}
	merged, err := profiles.MergeInferenceSettings(baseResolved.InferenceSettings, overlay)
	if err != nil {
		_ = chain.Close()
		return nil, nil, "", err
	}
	return merged, chain.Close, fmt.Sprintf("%s + embeddings(%s/%s)", baseProfile, embeddingType, embeddingEngine), nil
}

func defaultPinocchioProfilesPath() string {
	home, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		return filepath.Join(".config", "pinocchio", "profiles.yaml")
	}
	return filepath.Join(home, ".config", "pinocchio", "profiles.yaml")
}

func splitRegistryEntries(raw string) []string {
	parts := strings.Split(raw, ",")
	entries := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			entries = append(entries, part)
		}
	}
	return entries
}

func vectorPreview(vector []float32, n int) []float32 {
	if n <= 0 || len(vector) == 0 {
		return nil
	}
	if n > len(vector) {
		n = len(vector)
	}
	return vector[:n]
}

func fatal(err error) {
	fmt.Fprintf(os.Stderr, "error: %v\n", err)
	os.Exit(1)
}

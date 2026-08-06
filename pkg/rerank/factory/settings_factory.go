// Package factory constructs rerank providers from RerankConfig or
// InferenceSettings, breaking the import cycle between pkg/rerank (the core
// types) and the provider adapters (pkg/rerank/llamacpp, pkg/rerank/cohere).
package factory

import (
	"fmt"
	"strings"

	"github.com/go-go-golems/geppetto/pkg/rerank"
	"github.com/go-go-golems/geppetto/pkg/rerank/cohere"
	"github.com/go-go-golems/geppetto/pkg/rerank/config"
	"github.com/go-go-golems/geppetto/pkg/rerank/llamacpp"
	"github.com/go-go-golems/geppetto/pkg/security"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
)

const (
	rerankProviderLlamaCpp = "llamacpp"
	rerankProviderCohere   = "cohere"
)

// ProviderFactory constructs rerank providers from configuration.
type ProviderFactory interface {
	// NewProvider constructs the provider described by the resolved settings.
	// Endpoint and credential capability remains in host-owned settings; there
	// are intentionally no ad-hoc caller overrides.
	NewProvider() (rerank.Provider, error)
	SupportedProviders() []string
}

// SettingsFactory creates rerank providers based on a RerankConfig resolved from
// direct configuration or InferenceSettings.
type SettingsFactory struct {
	config *config.RerankConfig
	api    *settings.APISettings
	client *settings.ClientSettings
	model  *settings.ModelInfo
}

var _ ProviderFactory = &SettingsFactory{}

// NewSettingsFactory creates a new factory from a RerankConfig and the
// surrounding API/client/model-info settings. Any nil section is tolerated and
// resolved to defaults where possible; provider construction fails explicitly
// when required values are missing.
func NewSettingsFactory(cfg *config.RerankConfig, api *settings.APISettings, client *settings.ClientSettings, modelInfo *settings.ModelInfo) *SettingsFactory {
	return &SettingsFactory{
		config: cfg,
		api:    api,
		client: client,
		model:  modelInfo,
	}
}

// NewSettingsFactoryFromInferenceSettings creates a factory from final,
// already-merged InferenceSettings. This is the profile-resolved entry point
// used by the Goja wrapper and by hosts that resolve engine profiles.
func NewSettingsFactoryFromInferenceSettings(s *settings.InferenceSettings) (*SettingsFactory, error) {
	if err := ValidateInferenceSettingsForRerank(s); err != nil {
		return nil, err
	}
	return NewSettingsFactory(s.Rerank, s.API, s.Client, s.ModelInfo), nil
}

// SupportedProviders returns the provider types this factory can construct.
func (f *SettingsFactory) SupportedProviders() []string {
	return []string{rerankProviderLlamaCpp, rerankProviderCohere}
}

// NewProvider creates a rerank provider from the resolved configuration.
func (f *SettingsFactory) NewProvider() (rerank.Provider, error) {
	if f == nil || f.config == nil {
		return nil, fmt.Errorf("no rerank configuration provided: %w", rerank.ErrInvalidRequest)
	}

	providerType := strings.TrimSpace(f.config.Type)
	engine := strings.TrimSpace(f.config.Engine)
	if providerType == "" {
		return nil, fmt.Errorf("no rerank type specified: %w", rerank.ErrInvalidRequest)
	}
	if engine == "" {
		return nil, fmt.Errorf("no rerank model specified: %w", rerank.ErrInvalidRequest)
	}

	switch providerType {
	case rerankProviderLlamaCpp:
		baseURL, err := f.resolveBaseURL()
		if err != nil {
			return nil, err
		}
		outbound := f.resolveOutboundURLOptions()
		httpClient, err := settings.EnsureHTTPClient(f.client)
		if err != nil {
			return nil, fmt.Errorf("rerank http client: %w", err)
		}
		return llamacpp.New(llamacpp.Options{
			BaseURL:          baseURL,
			Model:            engine,
			HTTPClient:       httpClient,
			OutboundURL:      outbound,
			MaxRequestBytes:  f.config.MaxRequestBytes,
			MaxResponseBytes: f.config.MaxResponseBytes,
			CostPerMTokens:   f.resolveInputCostPerMTokens(),
		})
	case rerankProviderCohere:
		apiKey, err := f.resolveCohereAPIKey()
		if err != nil {
			return nil, err
		}
		// The base URL override is optional; the adapter defaults to the
		// canonical hosted endpoint when empty (DR-5).
		baseURL := ""
		if f.api != nil {
			baseURL = strings.TrimSpace(f.api.BaseUrls["cohere-base-url"])
		}
		outbound := f.resolveOutboundURLOptions()
		httpClient, err := settings.EnsureHTTPClient(f.client)
		if err != nil {
			return nil, fmt.Errorf("rerank http client: %w", err)
		}
		return cohere.New(cohere.Options{
			APIKey:           apiKey,
			BaseURL:          baseURL,
			Model:            engine,
			HTTPClient:       httpClient,
			OutboundURL:      outbound,
			MaxRequestBytes:  f.config.MaxRequestBytes,
			MaxResponseBytes: f.config.MaxResponseBytes,
			// CostPerSearch stays nil: ModelInfo has no per-search rate field
			// today, so hosted rerank cost is unknown rather than zero (DR-3).
		})
	default:
		return nil, fmt.Errorf("unsupported rerank provider type %q; supported values are %v: %w",
			providerType, f.SupportedProviders(), rerank.ErrInvalidRequest)
	}
}

// resolveBaseURL resolves the rerank base URL from the API BaseUrls map under
// the "rerank-base-url" key.
func (f *SettingsFactory) resolveBaseURL() (string, error) {
	if f.api == nil {
		return "", fmt.Errorf("rerank base URL is required but API settings are absent: %w", rerank.ErrInvalidRequest)
	}
	baseURL := strings.TrimSpace(f.api.BaseUrls["rerank-base-url"])
	if baseURL == "" {
		return "", fmt.Errorf("rerank base URL is required; set inference_settings.api.base_urls.rerank-base-url: %w", rerank.ErrInvalidRequest)
	}
	return baseURL, nil
}

// resolveCohereAPIKey resolves the Cohere API key from the API APIKeys map
// under the "cohere-api-key" key (the same key name the embeddings factory
// uses for Cohere).
func (f *SettingsFactory) resolveCohereAPIKey() (string, error) {
	if f.api == nil {
		return "", fmt.Errorf("cohere rerank api key is required but API settings are absent: %w", rerank.ErrInvalidRequest)
	}
	apiKey := strings.TrimSpace(f.api.APIKeys["cohere-api-key"])
	if apiKey == "" {
		return "", fmt.Errorf("selected rerank profile has no cohere-api-key; set inference_settings.api.api_keys.cohere-api-key: %w", rerank.ErrInvalidRequest)
	}
	return apiKey, nil
}

// resolveOutboundURLOptions builds the outbound URL policy from the API
// AllowHTTP and AllowLocalNetworks maps under the "rerank" key.
func (f *SettingsFactory) resolveOutboundURLOptions() security.OutboundURLOptions {
	opts := security.OutboundURLOptions{}
	if f.api == nil {
		return opts
	}
	if f.api.AllowHTTP != nil {
		opts.AllowHTTP = f.api.AllowHTTP["rerank"]
	}
	if f.api.AllowLocalNetworks != nil {
		opts.AllowLocalNetworks = f.api.AllowLocalNetworks["rerank"]
	}
	return opts
}

// resolveInputCostPerMTokens returns the per-million-token input cost from
// ModelInfo when available, so the response Cost can be computed. Returns nil
// (unknown cost) when ModelInfo or its Cost is absent.
func (f *SettingsFactory) resolveInputCostPerMTokens() *float64 {
	if f.model == nil || f.model.Cost == nil {
		return nil
	}
	rate := f.model.Cost.Input
	return &rate
}

// ValidateInferenceSettingsForRerank verifies that final, already-merged
// inference settings can construct a rerank provider. Profile-backed callers
// should run this after engine profile stack resolution and before provider
// construction so users get profile-oriented diagnostics.
func ValidateInferenceSettingsForRerank(s *settings.InferenceSettings) error {
	if s == nil {
		return fmt.Errorf("selected profile is not rerank-capable: inference settings are required: %w", rerank.ErrInvalidRequest)
	}
	if s.Rerank == nil {
		return fmt.Errorf("selected profile is not rerank-capable: missing inference_settings.rerank: %w", rerank.ErrInvalidRequest)
	}
	providerType := strings.TrimSpace(s.Rerank.Type)
	if providerType == "" {
		return fmt.Errorf("selected profile is not rerank-capable: missing inference_settings.rerank.type: %w", rerank.ErrInvalidRequest)
	}
	if strings.TrimSpace(s.Rerank.Engine) == "" {
		return fmt.Errorf("selected profile is not rerank-capable: missing inference_settings.rerank.engine: %w", rerank.ErrInvalidRequest)
	}
	switch providerType {
	case rerankProviderLlamaCpp:
		if s.API == nil || strings.TrimSpace(s.API.BaseUrls["rerank-base-url"]) == "" {
			return fmt.Errorf("selected rerank profile has no rerank-base-url; set inference_settings.api.base_urls.rerank-base-url: %w", rerank.ErrInvalidRequest)
		}
	case rerankProviderCohere:
		if s.API == nil || strings.TrimSpace(s.API.APIKeys["cohere-api-key"]) == "" {
			return fmt.Errorf("selected rerank profile has no cohere-api-key; set inference_settings.api.api_keys.cohere-api-key: %w", rerank.ErrInvalidRequest)
		}
	default:
		return fmt.Errorf("unsupported rerank provider type %q; supported values are %v: %w",
			providerType, []string{rerankProviderLlamaCpp, rerankProviderCohere}, rerank.ErrInvalidRequest)
	}
	return nil
}

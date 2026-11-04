// Package factory constructs decision providers from DecisionConfig or
// InferenceSettings, breaking the import cycle between pkg/decision (core
// types) and the provider adapters (pkg/decision/systemone).
package factory

import (
	"fmt"
	"strings"

	"github.com/go-go-golems/geppetto/pkg/decision"
	"github.com/go-go-golems/geppetto/pkg/decision/config"
	"github.com/go-go-golems/geppetto/pkg/decision/systemone"
	"github.com/go-go-golems/geppetto/pkg/security"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
)

const (
	providerSystemOne = "systemone"

	// APIKeyName and BaseURLName are the APISettings keys of the System One adapter.
	APIKeyName  = "systemone-api-key"
	BaseURLName = "systemone-base-url"
	// PolicyName keys AllowHTTP / AllowLocalNetworks for decision providers.
	PolicyName = "decision"
)

// SettingsFactory creates decision providers from resolved settings.
type SettingsFactory struct {
	config *config.DecisionConfig
	api    *settings.APISettings
	client *settings.ClientSettings
	model  *settings.ModelInfo
}

// NewSettingsFactory creates a factory from a DecisionConfig and the
// surrounding API, client and model-info settings (each may be nil).
func NewSettingsFactory(cfg *config.DecisionConfig, api *settings.APISettings, client *settings.ClientSettings, modelInfo *settings.ModelInfo) *SettingsFactory {
	return &SettingsFactory{config: cfg, api: api, client: client, model: modelInfo}
}

// NewSettingsFactoryFromInferenceSettings is the profile-resolved entry point.
func NewSettingsFactoryFromInferenceSettings(s *settings.InferenceSettings) (*SettingsFactory, error) {
	if err := ValidateInferenceSettingsForDecision(s); err != nil {
		return nil, err
	}
	return NewSettingsFactory(s.Decision, s.API, s.Client, s.ModelInfo), nil
}

// SupportedProviders lists the provider types this factory builds.
func (f *SettingsFactory) SupportedProviders() []string { return []string{providerSystemOne} }

// NewProvider builds the configured provider.
func (f *SettingsFactory) NewProvider() (decision.Provider, error) {
	if f == nil || f.config == nil {
		return nil, fmt.Errorf("no decision configuration provided: %w", decision.ErrInvalidRequest)
	}
	typ, engine := strings.TrimSpace(f.config.Type), strings.TrimSpace(f.config.Engine)
	if typ == "" {
		return nil, fmt.Errorf("no decision type specified: %w", decision.ErrInvalidRequest)
	}
	if engine == "" {
		return nil, fmt.Errorf("no decision model specified: %w", decision.ErrInvalidRequest)
	}
	switch typ {
	case providerSystemOne:
		key := f.apiKey()
		if key == "" {
			return nil, fmt.Errorf("decision profile has no %s; set inference_settings.api.api_keys.%s: %w", APIKeyName, APIKeyName, decision.ErrInvalidRequest)
		}
		client, err := settings.EnsureHTTPClient(f.client)
		if err != nil {
			return nil, fmt.Errorf("decision http client: %w", err)
		}
		return systemone.New(systemone.Options{
			BaseURL:             f.baseURL(),
			APIKey:              key,
			Model:               engine,
			HTTPClient:          client,
			OutboundURL:         f.outbound(),
			MaxRequestBytes:     f.config.MaxRequestBytes,
			MaxResponseBytes:    f.config.MaxResponseBytes,
			CostPerMInputTokens: f.inputCost(),
		})
	default:
		return nil, fmt.Errorf("unsupported decision provider type %q; supported values are %v: %w", typ, f.SupportedProviders(), decision.ErrInvalidRequest)
	}
}

func (f *SettingsFactory) apiKey() string {
	if f.api == nil {
		return ""
	}
	return strings.TrimSpace(f.api.APIKeys[APIKeyName])
}

func (f *SettingsFactory) baseURL() string {
	if f.api == nil {
		return ""
	}
	return strings.TrimSpace(f.api.BaseUrls[BaseURLName])
}

func (f *SettingsFactory) outbound() security.OutboundURLOptions {
	o := security.OutboundURLOptions{}
	if f.api == nil {
		return o
	}
	o.AllowHTTP = f.api.AllowHTTP[PolicyName]
	o.AllowLocalNetworks = f.api.AllowLocalNetworks[PolicyName]
	return o
}

func (f *SettingsFactory) inputCost() *float64 {
	if f.model == nil || f.model.Cost == nil {
		return nil
	}
	rate := f.model.Cost.Input
	return &rate
}

// ValidateInferenceSettingsForDecision verifies that final, merged settings
// can build a decision provider, with profile-oriented diagnostics.
func ValidateInferenceSettingsForDecision(s *settings.InferenceSettings) error {
	notCapable := func(msg string) error {
		return fmt.Errorf("selected profile is not decision-capable: %s: %w", msg, decision.ErrInvalidRequest)
	}
	switch {
	case s == nil:
		return notCapable("inference settings are required")
	case s.Decision == nil:
		return notCapable("missing inference_settings.decision")
	case strings.TrimSpace(s.Decision.Type) == "":
		return notCapable("missing inference_settings.decision.type")
	case strings.TrimSpace(s.Decision.Engine) == "":
		return notCapable("missing inference_settings.decision.engine")
	}
	switch t := strings.TrimSpace(s.Decision.Type); t {
	case providerSystemOne:
		if s.API == nil || strings.TrimSpace(s.API.APIKeys[APIKeyName]) == "" {
			return fmt.Errorf("selected decision profile has no %s; set inference_settings.api.api_keys.%s: %w", APIKeyName, APIKeyName, decision.ErrInvalidRequest)
		}
	default:
		return fmt.Errorf("unsupported decision provider type %q; supported values are [%s]: %w", t, providerSystemOne, decision.ErrInvalidRequest)
	}
	return nil
}

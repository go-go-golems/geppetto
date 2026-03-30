package engineprofiles

import (
	"context"

	aistepssettings "github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
)

// RegistrySummary gives lightweight metadata for list endpoints.
type RegistrySummary struct {
	Slug                     RegistrySlug      `json:"slug" yaml:"slug"`
	DisplayName              string            `json:"display_name,omitempty" yaml:"display_name,omitempty"`
	DefaultEngineProfileSlug EngineProfileSlug `json:"default_profile_slug,omitempty" yaml:"default_profile_slug,omitempty"`
	EngineProfileCount       int               `json:"profile_count" yaml:"profile_count"`
}

// ResolveInput contains all inputs needed to compute an effective engine profile.
type ResolveInput struct {
	RegistrySlug      RegistrySlug
	EngineProfileSlug EngineProfileSlug
}

// ResolvedProfileStackEntry captures one base->leaf stack entry in typed form.
type ResolvedProfileStackEntry struct {
	RegistrySlug      RegistrySlug      `json:"registry_slug" yaml:"registry_slug"`
	EngineProfileSlug EngineProfileSlug `json:"profile_slug" yaml:"profile_slug"`
	Version           uint64            `json:"version,omitempty" yaml:"version,omitempty"`
	Source            string            `json:"source,omitempty" yaml:"source,omitempty"`
}

// ResolvedEngineProfile is the canonical result of engine profile resolution.
type ResolvedEngineProfile struct {
	RegistrySlug      RegistrySlug
	EngineProfileSlug EngineProfileSlug
	InferenceSettings *aistepssettings.InferenceSettings
	StackLineage      []ResolvedProfileStackEntry
	Metadata          map[string]any
}

// RegistryReader provides read/query operations for profile registry services.
type RegistryReader interface {
	ListRegistries(ctx context.Context) ([]RegistrySummary, error)
	GetRegistry(ctx context.Context, registrySlug RegistrySlug) (*EngineProfileRegistry, error)
	ListEngineProfiles(ctx context.Context, registrySlug RegistrySlug) ([]*EngineProfile, error)
	GetEngineProfile(ctx context.Context, registrySlug RegistrySlug, profileSlug EngineProfileSlug) (*EngineProfile, error)
	ResolveEngineProfile(ctx context.Context, in ResolveInput) (*ResolvedEngineProfile, error)
}

// Registry is the unified profile registry service abstraction.
type Registry interface {
	RegistryReader
}

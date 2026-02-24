package profiles

import (
	"context"

	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
)

// RegistrySummary gives lightweight metadata for list endpoints.
type RegistrySummary struct {
	Slug               RegistrySlug `json:"slug" yaml:"slug"`
	DisplayName        string       `json:"display_name,omitempty" yaml:"display_name,omitempty"`
	DefaultProfileSlug ProfileSlug  `json:"default_profile_slug,omitempty" yaml:"default_profile_slug,omitempty"`
	ProfileCount       int          `json:"profile_count" yaml:"profile_count"`
}

// ProfilePatch describes partial updates to profile fields.
type ProfilePatch struct {
	DisplayName *string          `json:"display_name,omitempty" yaml:"display_name,omitempty"`
	Description *string          `json:"description,omitempty" yaml:"description,omitempty"`
	Runtime     *RuntimeSpec     `json:"runtime,omitempty" yaml:"runtime,omitempty"`
	Policy      *PolicySpec      `json:"policy,omitempty" yaml:"policy,omitempty"`
	Metadata    *ProfileMetadata `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	Extensions  *map[string]any  `json:"extensions,omitempty" yaml:"extensions,omitempty"`
}

// WriteOptions carries mutation context for registry service APIs.
type WriteOptions struct {
	ExpectedVersion uint64
	Actor           string
	Source          string
}

// ResolveInput contains all inputs needed to compute an effective runtime profile.
type ResolveInput struct {
	RegistrySlug       RegistrySlug
	ProfileSlug        ProfileSlug
	RuntimeKeyFallback RuntimeKey

	BaseStepSettings *settings.StepSettings
	RequestOverrides map[string]any
}

// ResolvedProfile is the canonical result of profile resolution.
type ResolvedProfile struct {
	RegistrySlug RegistrySlug
	ProfileSlug  ProfileSlug
	RuntimeKey   RuntimeKey

	EffectiveStepSettings *settings.StepSettings
	EffectiveRuntime      RuntimeSpec
	RuntimeFingerprint    string

	Metadata map[string]any
}

// RegistryReader provides read/query operations for profile registry services.
type RegistryReader interface {
	ListRegistries(ctx context.Context) ([]RegistrySummary, error)
	GetRegistry(ctx context.Context, registrySlug RegistrySlug) (*ProfileRegistry, error)
	ListProfiles(ctx context.Context, registrySlug RegistrySlug) ([]*Profile, error)
	GetProfile(ctx context.Context, registrySlug RegistrySlug, profileSlug ProfileSlug) (*Profile, error)
	ResolveEffectiveProfile(ctx context.Context, in ResolveInput) (*ResolvedProfile, error)
}

// RegistryWriter provides mutation operations for profile registry services.
type RegistryWriter interface {
	CreateProfile(ctx context.Context, registrySlug RegistrySlug, profile *Profile, opts WriteOptions) (*Profile, error)
	UpdateProfile(ctx context.Context, registrySlug RegistrySlug, profileSlug ProfileSlug, patch ProfilePatch, opts WriteOptions) (*Profile, error)
	DeleteProfile(ctx context.Context, registrySlug RegistrySlug, profileSlug ProfileSlug, opts WriteOptions) error
	SetDefaultProfile(ctx context.Context, registrySlug RegistrySlug, profileSlug ProfileSlug, opts WriteOptions) error
}

// Registry is the unified profile registry service abstraction.
type Registry interface {
	RegistryReader
	RegistryWriter
}

package profiles

import "context"

// SaveOptions carries write context for profile/registry persistence.
type SaveOptions struct {
	ExpectedVersion uint64
	Actor           string
	Source          string
}

// ProfileStoreReader provides read operations over profile registries.
type ProfileStoreReader interface {
	ListRegistries(ctx context.Context) ([]*ProfileRegistry, error)
	GetRegistry(ctx context.Context, registrySlug RegistrySlug) (*ProfileRegistry, bool, error)
	ListProfiles(ctx context.Context, registrySlug RegistrySlug) ([]*Profile, error)
	GetProfile(ctx context.Context, registrySlug RegistrySlug, profileSlug ProfileSlug) (*Profile, bool, error)
}

// ProfileStoreWriter provides write operations over profile registries.
type ProfileStoreWriter interface {
	UpsertRegistry(ctx context.Context, registry *ProfileRegistry, opts SaveOptions) error
	DeleteRegistry(ctx context.Context, registrySlug RegistrySlug, opts SaveOptions) error
	UpsertProfile(ctx context.Context, registrySlug RegistrySlug, profile *Profile, opts SaveOptions) error
	DeleteProfile(ctx context.Context, registrySlug RegistrySlug, profileSlug ProfileSlug, opts SaveOptions) error
	SetDefaultProfile(ctx context.Context, registrySlug RegistrySlug, profileSlug ProfileSlug, opts SaveOptions) error
	Close() error
}

// ProfileStore is the primary persistence abstraction used by profile registry services.
type ProfileStore interface {
	ProfileStoreReader
	ProfileStoreWriter
}

package engineprofiles

import "context"

// SaveOptions carries write context for profile/registry persistence.
type SaveOptions struct {
	ExpectedVersion uint64
	Actor           string
	Source          string
}

// EngineProfileStoreReader provides read operations over profile registries.
type EngineProfileStoreReader interface {
	ListRegistries(ctx context.Context) ([]*EngineProfileRegistry, error)
	GetRegistry(ctx context.Context, registrySlug RegistrySlug) (*EngineProfileRegistry, bool, error)
	ListEngineProfiles(ctx context.Context, registrySlug RegistrySlug) ([]*EngineProfile, error)
	GetEngineProfile(ctx context.Context, registrySlug RegistrySlug, profileSlug EngineProfileSlug) (*EngineProfile, bool, error)
}

// EngineProfileStoreWriter provides write operations over profile registries.
type EngineProfileStoreWriter interface {
	UpsertRegistry(ctx context.Context, registry *EngineProfileRegistry, opts SaveOptions) error
	DeleteRegistry(ctx context.Context, registrySlug RegistrySlug, opts SaveOptions) error
	UpsertEngineProfile(ctx context.Context, registrySlug RegistrySlug, profile *EngineProfile, opts SaveOptions) error
	Close() error
}

// EngineProfileStore is the primary persistence abstraction used by profile registry services.
type EngineProfileStore interface {
	EngineProfileStoreReader
	EngineProfileStoreWriter
}

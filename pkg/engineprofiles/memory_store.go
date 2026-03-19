package engineprofiles

import (
	"context"
	"fmt"
	"sort"
	"sync"
)

// InMemoryEngineProfileStore is a thread-safe EngineProfileStore implementation.
type InMemoryEngineProfileStore struct {
	mu         sync.RWMutex
	registries map[RegistrySlug]*EngineProfileRegistry
	closed     bool
}

func NewInMemoryEngineProfileStore() *InMemoryEngineProfileStore {
	return &InMemoryEngineProfileStore{
		registries: map[RegistrySlug]*EngineProfileRegistry{},
	}
}

func (s *InMemoryEngineProfileStore) ListRegistries(_ context.Context) ([]*EngineProfileRegistry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if err := s.ensureOpen(); err != nil {
		return nil, err
	}

	slugs := make([]RegistrySlug, 0, len(s.registries))
	for slug := range s.registries {
		slugs = append(slugs, slug)
	}
	sort.Slice(slugs, func(i, j int) bool { return slugs[i] < slugs[j] })

	out := make([]*EngineProfileRegistry, 0, len(slugs))
	for _, slug := range slugs {
		out = append(out, s.registries[slug].Clone())
	}
	return out, nil
}

func (s *InMemoryEngineProfileStore) GetRegistry(_ context.Context, registrySlug RegistrySlug) (*EngineProfileRegistry, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if err := s.ensureOpen(); err != nil {
		return nil, false, err
	}

	reg, ok := s.registries[registrySlug]
	if !ok || reg == nil {
		return nil, false, nil
	}
	return reg.Clone(), true, nil
}

func (s *InMemoryEngineProfileStore) ListEngineProfiles(ctx context.Context, registrySlug RegistrySlug) ([]*EngineProfile, error) {
	reg, ok, err := s.GetRegistry(ctx, registrySlug)
	if err != nil {
		return nil, err
	}
	if !ok || reg == nil {
		return nil, nil
	}

	slugs := make([]EngineProfileSlug, 0, len(reg.Profiles))
	for slug := range reg.Profiles {
		slugs = append(slugs, slug)
	}
	sort.Slice(slugs, func(i, j int) bool { return slugs[i] < slugs[j] })

	out := make([]*EngineProfile, 0, len(slugs))
	for _, slug := range slugs {
		out = append(out, reg.Profiles[slug].Clone())
	}
	return out, nil
}

func (s *InMemoryEngineProfileStore) GetEngineProfile(ctx context.Context, registrySlug RegistrySlug, profileSlug EngineProfileSlug) (*EngineProfile, bool, error) {
	reg, ok, err := s.GetRegistry(ctx, registrySlug)
	if err != nil {
		return nil, false, err
	}
	if !ok || reg == nil {
		return nil, false, nil
	}
	profile, ok := reg.Profiles[profileSlug]
	if !ok || profile == nil {
		return nil, false, nil
	}
	return profile.Clone(), true, nil
}

func (s *InMemoryEngineProfileStore) UpsertRegistry(_ context.Context, registry *EngineProfileRegistry, opts SaveOptions) error {
	if err := ValidateRegistry(registry); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensureOpen(); err != nil {
		return err
	}

	clone := registry.Clone()
	if clone.Profiles == nil {
		clone.Profiles = map[EngineProfileSlug]*EngineProfile{}
	}

	existing, ok := s.registries[clone.Slug]
	if ok && existing != nil {
		if err := assertExpectedVersion("registry", clone.Slug.String(), opts.ExpectedVersion, existing.Metadata.Version); err != nil {
			return err
		}
		clone.Metadata = existing.Metadata
		if clone.DefaultEngineProfileSlug.IsZero() {
			clone.DefaultEngineProfileSlug = existing.DefaultEngineProfileSlug
		}
	} else {
		if err := assertExpectedVersion("registry", clone.Slug.String(), opts.ExpectedVersion, 0); err != nil {
			return err
		}
	}
	TouchRegistryMetadata(&clone.Metadata, opts, 0)
	s.registries[clone.Slug] = clone
	return nil
}

func (s *InMemoryEngineProfileStore) DeleteRegistry(_ context.Context, registrySlug RegistrySlug, opts SaveOptions) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensureOpen(); err != nil {
		return err
	}

	existing, ok := s.registries[registrySlug]
	if !ok || existing == nil {
		return nil
	}
	if err := assertExpectedVersion("registry", registrySlug.String(), opts.ExpectedVersion, existing.Metadata.Version); err != nil {
		return err
	}
	delete(s.registries, registrySlug)
	return nil
}

func (s *InMemoryEngineProfileStore) UpsertEngineProfile(_ context.Context, registrySlug RegistrySlug, profile *EngineProfile, opts SaveOptions) error {
	if err := ValidateEngineProfile(profile); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensureOpen(); err != nil {
		return err
	}

	registry, ok := s.registries[registrySlug]
	if !ok || registry == nil {
		return ErrRegistryNotFound
	}
	if registry.Profiles == nil {
		registry.Profiles = map[EngineProfileSlug]*EngineProfile{}
	}

	clone := profile.Clone()
	existing, ok := registry.Profiles[clone.Slug]
	if ok && existing != nil {
		if err := assertExpectedVersion("profile", clone.Slug.String(), opts.ExpectedVersion, existing.Metadata.Version); err != nil {
			return err
		}
		clone.Metadata = existing.Metadata
	} else {
		if err := assertExpectedVersion("profile", clone.Slug.String(), opts.ExpectedVersion, 0); err != nil {
			return err
		}
	}
	TouchEngineProfileMetadata(&clone.Metadata, opts, 0)
	registry.Profiles[clone.Slug] = clone
	TouchRegistryMetadata(&registry.Metadata, opts, 0)
	return nil
}

func (s *InMemoryEngineProfileStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
	return nil
}

func (s *InMemoryEngineProfileStore) ensureOpen() error {
	if s.closed {
		return fmt.Errorf("in-memory profile store closed")
	}
	return nil
}

func assertExpectedVersion(resource, slug string, expected, actual uint64) error {
	if expected == 0 {
		return nil
	}
	if expected == actual {
		return nil
	}
	return &VersionConflictError{
		Resource: resource,
		Slug:     slug,
		Expected: expected,
		Actual:   actual,
	}
}

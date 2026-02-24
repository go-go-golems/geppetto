package profiles

import (
	"context"
	"fmt"
	"sort"
	"sync"
)

// InMemoryProfileStore is a thread-safe ProfileStore implementation.
type InMemoryProfileStore struct {
	mu         sync.RWMutex
	registries map[RegistrySlug]*ProfileRegistry
	closed     bool
}

func NewInMemoryProfileStore() *InMemoryProfileStore {
	return &InMemoryProfileStore{
		registries: map[RegistrySlug]*ProfileRegistry{},
	}
}

func (s *InMemoryProfileStore) ListRegistries(_ context.Context) ([]*ProfileRegistry, error) {
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

	out := make([]*ProfileRegistry, 0, len(slugs))
	for _, slug := range slugs {
		out = append(out, s.registries[slug].Clone())
	}
	return out, nil
}

func (s *InMemoryProfileStore) GetRegistry(_ context.Context, registrySlug RegistrySlug) (*ProfileRegistry, bool, error) {
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

func (s *InMemoryProfileStore) ListProfiles(ctx context.Context, registrySlug RegistrySlug) ([]*Profile, error) {
	reg, ok, err := s.GetRegistry(ctx, registrySlug)
	if err != nil {
		return nil, err
	}
	if !ok || reg == nil {
		return nil, nil
	}

	slugs := make([]ProfileSlug, 0, len(reg.Profiles))
	for slug := range reg.Profiles {
		slugs = append(slugs, slug)
	}
	sort.Slice(slugs, func(i, j int) bool { return slugs[i] < slugs[j] })

	out := make([]*Profile, 0, len(slugs))
	for _, slug := range slugs {
		out = append(out, reg.Profiles[slug].Clone())
	}
	return out, nil
}

func (s *InMemoryProfileStore) GetProfile(ctx context.Context, registrySlug RegistrySlug, profileSlug ProfileSlug) (*Profile, bool, error) {
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

func (s *InMemoryProfileStore) UpsertRegistry(_ context.Context, registry *ProfileRegistry, opts SaveOptions) error {
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
		clone.Profiles = map[ProfileSlug]*Profile{}
	}

	existing, ok := s.registries[clone.Slug]
	if ok && existing != nil {
		if err := assertExpectedVersion("registry", clone.Slug.String(), opts.ExpectedVersion, existing.Metadata.Version); err != nil {
			return err
		}
		clone.Metadata = existing.Metadata
		if clone.DefaultProfileSlug.IsZero() {
			clone.DefaultProfileSlug = existing.DefaultProfileSlug
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

func (s *InMemoryProfileStore) DeleteRegistry(_ context.Context, registrySlug RegistrySlug, opts SaveOptions) error {
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

func (s *InMemoryProfileStore) UpsertProfile(_ context.Context, registrySlug RegistrySlug, profile *Profile, opts SaveOptions) error {
	if err := ValidateProfile(profile); err != nil {
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
		registry.Profiles = map[ProfileSlug]*Profile{}
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
	TouchProfileMetadata(&clone.Metadata, opts, 0)
	registry.Profiles[clone.Slug] = clone
	TouchRegistryMetadata(&registry.Metadata, opts, 0)
	return nil
}

func (s *InMemoryProfileStore) DeleteProfile(_ context.Context, registrySlug RegistrySlug, profileSlug ProfileSlug, opts SaveOptions) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensureOpen(); err != nil {
		return err
	}

	registry, ok := s.registries[registrySlug]
	if !ok || registry == nil {
		return ErrRegistryNotFound
	}
	profile, ok := registry.Profiles[profileSlug]
	if !ok || profile == nil {
		return nil
	}
	if err := assertExpectedVersion("profile", profileSlug.String(), opts.ExpectedVersion, profile.Metadata.Version); err != nil {
		return err
	}

	delete(registry.Profiles, profileSlug)
	if registry.DefaultProfileSlug == profileSlug {
		registry.DefaultProfileSlug = ""
	}
	TouchRegistryMetadata(&registry.Metadata, opts, 0)
	return nil
}

func (s *InMemoryProfileStore) SetDefaultProfile(_ context.Context, registrySlug RegistrySlug, profileSlug ProfileSlug, opts SaveOptions) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensureOpen(); err != nil {
		return err
	}

	registry, ok := s.registries[registrySlug]
	if !ok || registry == nil {
		return ErrRegistryNotFound
	}
	if _, ok := registry.Profiles[profileSlug]; !ok {
		return ErrProfileNotFound
	}
	if err := assertExpectedVersion("registry", registrySlug.String(), opts.ExpectedVersion, registry.Metadata.Version); err != nil {
		return err
	}
	registry.DefaultProfileSlug = profileSlug
	TouchRegistryMetadata(&registry.Metadata, opts, 0)
	return nil
}

func (s *InMemoryProfileStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
	return nil
}

func (s *InMemoryProfileStore) ensureOpen() error {
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

package profiles

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// YAMLFileProfileStore persists registries as YAML on disk.
type YAMLFileProfileStore struct {
	mu                  sync.RWMutex
	path                string
	defaultRegistrySlug RegistrySlug
	store               *InMemoryProfileStore
	closed              bool
}

func NewYAMLFileProfileStore(path string, defaultRegistrySlug RegistrySlug) (*YAMLFileProfileStore, error) {
	if path == "" {
		return nil, fmt.Errorf("yaml profile store path is required")
	}
	if defaultRegistrySlug.IsZero() {
		defaultRegistrySlug = MustRegistrySlug("default")
	}

	s := &YAMLFileProfileStore{
		path:                path,
		defaultRegistrySlug: defaultRegistrySlug,
		store:               NewInMemoryProfileStore(),
	}
	if err := s.loadFromDisk(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *YAMLFileProfileStore) ListRegistries(ctx context.Context) ([]*ProfileRegistry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if err := s.ensureOpen(); err != nil {
		return nil, err
	}
	return s.store.ListRegistries(ctx)
}

func (s *YAMLFileProfileStore) GetRegistry(ctx context.Context, registrySlug RegistrySlug) (*ProfileRegistry, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if err := s.ensureOpen(); err != nil {
		return nil, false, err
	}
	return s.store.GetRegistry(ctx, registrySlug)
}

func (s *YAMLFileProfileStore) ListProfiles(ctx context.Context, registrySlug RegistrySlug) ([]*Profile, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if err := s.ensureOpen(); err != nil {
		return nil, err
	}
	return s.store.ListProfiles(ctx, registrySlug)
}

func (s *YAMLFileProfileStore) GetProfile(ctx context.Context, registrySlug RegistrySlug, profileSlug ProfileSlug) (*Profile, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if err := s.ensureOpen(); err != nil {
		return nil, false, err
	}
	return s.store.GetProfile(ctx, registrySlug, profileSlug)
}

func (s *YAMLFileProfileStore) UpsertRegistry(ctx context.Context, registry *ProfileRegistry, opts SaveOptions) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensureOpen(); err != nil {
		return err
	}
	if err := s.store.UpsertRegistry(ctx, registry, opts); err != nil {
		return err
	}
	return s.persistLocked(ctx)
}

func (s *YAMLFileProfileStore) DeleteRegistry(ctx context.Context, registrySlug RegistrySlug, opts SaveOptions) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensureOpen(); err != nil {
		return err
	}
	if err := s.store.DeleteRegistry(ctx, registrySlug, opts); err != nil {
		return err
	}
	return s.persistLocked(ctx)
}

func (s *YAMLFileProfileStore) UpsertProfile(ctx context.Context, registrySlug RegistrySlug, profile *Profile, opts SaveOptions) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensureOpen(); err != nil {
		return err
	}
	if err := s.store.UpsertProfile(ctx, registrySlug, profile, opts); err != nil {
		return err
	}
	return s.persistLocked(ctx)
}

func (s *YAMLFileProfileStore) DeleteProfile(ctx context.Context, registrySlug RegistrySlug, profileSlug ProfileSlug, opts SaveOptions) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensureOpen(); err != nil {
		return err
	}
	if err := s.store.DeleteProfile(ctx, registrySlug, profileSlug, opts); err != nil {
		return err
	}
	return s.persistLocked(ctx)
}

func (s *YAMLFileProfileStore) SetDefaultProfile(ctx context.Context, registrySlug RegistrySlug, profileSlug ProfileSlug, opts SaveOptions) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensureOpen(); err != nil {
		return err
	}
	if err := s.store.SetDefaultProfile(ctx, registrySlug, profileSlug, opts); err != nil {
		return err
	}
	return s.persistLocked(ctx)
}

func (s *YAMLFileProfileStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
	return nil
}

func (s *YAMLFileProfileStore) loadFromDisk() error {
	b, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	registries, err := DecodeYAMLRegistries(b, s.defaultRegistrySlug)
	if err != nil {
		return err
	}

	s.store = NewInMemoryProfileStore()
	s.store.registries = map[RegistrySlug]*ProfileRegistry{}
	for _, registry := range registries {
		if registry == nil {
			continue
		}
		s.store.registries[registry.Slug] = registry.Clone()
	}
	return nil
}

func (s *YAMLFileProfileStore) persistLocked(ctx context.Context) error {
	registries, err := s.store.ListRegistries(ctx)
	if err != nil {
		return err
	}
	b, err := EncodeYAMLRegistries(registries)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	tmpPath := s.path + ".tmp"
	if err := os.WriteFile(tmpPath, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmpPath, s.path)
}

func (s *YAMLFileProfileStore) ensureOpen() error {
	if s.closed {
		return fmt.Errorf("yaml profile store closed")
	}
	return nil
}

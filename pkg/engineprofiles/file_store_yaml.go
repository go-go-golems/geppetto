package engineprofiles

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// YAMLFileEngineProfileStore persists registries as YAML on disk.
type YAMLFileEngineProfileStore struct {
	mu                  sync.RWMutex
	path                string
	defaultRegistrySlug RegistrySlug
	store               *InMemoryEngineProfileStore
	closed              bool
}

func NewYAMLFileEngineProfileStore(path string, defaultRegistrySlug RegistrySlug) (*YAMLFileEngineProfileStore, error) {
	if path == "" {
		return nil, fmt.Errorf("yaml profile store path is required")
	}
	if defaultRegistrySlug.IsZero() {
		defaultRegistrySlug = MustRegistrySlug("default")
	}

	s := &YAMLFileEngineProfileStore{
		path:                path,
		defaultRegistrySlug: defaultRegistrySlug,
		store:               NewInMemoryEngineProfileStore(),
	}
	if err := s.loadFromDisk(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *YAMLFileEngineProfileStore) ListRegistries(ctx context.Context) ([]*EngineProfileRegistry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if err := s.ensureOpen(); err != nil {
		return nil, err
	}
	return s.store.ListRegistries(ctx)
}

func (s *YAMLFileEngineProfileStore) GetRegistry(ctx context.Context, registrySlug RegistrySlug) (*EngineProfileRegistry, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if err := s.ensureOpen(); err != nil {
		return nil, false, err
	}
	resolved, err := s.resolveRegistrySlug(registrySlug)
	if err != nil {
		return nil, false, err
	}
	return s.store.GetRegistry(ctx, resolved)
}

func (s *YAMLFileEngineProfileStore) ListEngineProfiles(ctx context.Context, registrySlug RegistrySlug) ([]*EngineProfile, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if err := s.ensureOpen(); err != nil {
		return nil, err
	}
	resolved, err := s.resolveRegistrySlug(registrySlug)
	if err != nil {
		return nil, err
	}
	return s.store.ListEngineProfiles(ctx, resolved)
}

func (s *YAMLFileEngineProfileStore) GetEngineProfile(ctx context.Context, registrySlug RegistrySlug, profileSlug EngineProfileSlug) (*EngineProfile, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if err := s.ensureOpen(); err != nil {
		return nil, false, err
	}
	resolved, err := s.resolveRegistrySlug(registrySlug)
	if err != nil {
		return nil, false, err
	}
	return s.store.GetEngineProfile(ctx, resolved, profileSlug)
}

func (s *YAMLFileEngineProfileStore) UpsertRegistry(ctx context.Context, registry *EngineProfileRegistry, opts SaveOptions) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensureOpen(); err != nil {
		return err
	}
	if registry == nil {
		return fmt.Errorf("registry is required")
	}
	resolved, err := s.resolveRegistrySlug(registry.Slug)
	if err != nil {
		return err
	}
	cloned := registry.Clone()
	cloned.Slug = resolved
	if err := s.store.UpsertRegistry(ctx, cloned, opts); err != nil {
		return err
	}
	return s.persistLocked(ctx)
}

func (s *YAMLFileEngineProfileStore) DeleteRegistry(ctx context.Context, registrySlug RegistrySlug, opts SaveOptions) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensureOpen(); err != nil {
		return err
	}
	resolved, err := s.resolveRegistrySlug(registrySlug)
	if err != nil {
		return err
	}
	if err := s.store.DeleteRegistry(ctx, resolved, opts); err != nil {
		return err
	}
	return s.persistLocked(ctx)
}

func (s *YAMLFileEngineProfileStore) UpsertEngineProfile(ctx context.Context, registrySlug RegistrySlug, profile *EngineProfile, opts SaveOptions) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensureOpen(); err != nil {
		return err
	}
	resolved, err := s.resolveRegistrySlug(registrySlug)
	if err != nil {
		return err
	}
	if err := s.store.UpsertEngineProfile(ctx, resolved, profile, opts); err != nil {
		return err
	}
	return s.persistLocked(ctx)
}

func (s *YAMLFileEngineProfileStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
	return nil
}

func (s *YAMLFileEngineProfileStore) loadFromDisk() error {
	b, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	registry, err := DecodeEngineProfileYAMLSingleRegistry(b)
	if err != nil {
		return err
	}

	s.store = NewInMemoryEngineProfileStore()
	s.store.registries = map[RegistrySlug]*EngineProfileRegistry{}
	if registry != nil {
		s.defaultRegistrySlug = registry.Slug
		s.store.registries[registry.Slug] = registry.Clone()
	}
	return nil
}

func (s *YAMLFileEngineProfileStore) persistLocked(ctx context.Context) error {
	registry, ok, err := s.store.GetRegistry(ctx, s.defaultRegistrySlug)
	if err != nil {
		return err
	}
	if !ok || registry == nil {
		if err := os.Remove(s.path); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	}
	b, err := EncodeEngineProfileYAMLSingleRegistry(registry)
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

func (s *YAMLFileEngineProfileStore) ensureOpen() error {
	if s.closed {
		return fmt.Errorf("yaml profile store closed")
	}
	return nil
}

func (s *YAMLFileEngineProfileStore) resolveRegistrySlug(registrySlug RegistrySlug) (RegistrySlug, error) {
	resolved := registrySlug
	if resolved.IsZero() {
		resolved = s.defaultRegistrySlug
	}
	if resolved.IsZero() {
		return "", fmt.Errorf("yaml profile store default registry slug is not set")
	}
	if resolved != s.defaultRegistrySlug {
		return "", fmt.Errorf("yaml profile store %q supports only registry %q (got %q)", s.path, s.defaultRegistrySlug, resolved)
	}
	return resolved, nil
}

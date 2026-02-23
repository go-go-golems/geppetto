package profiles

import (
	"context"
	"fmt"
	"sort"
)

// OverlayStore merges profile registries from multiple read sources.
// Later sources override earlier sources for conflicting fields/profile slugs.
type OverlayStore struct {
	readers []ProfileStoreReader
	writer  ProfileStoreWriter
}

// NewOverlayStore constructs an overlay using the given stores as both readers and writers.
// The first store is used as the write target.
func NewOverlayStore(stores ...ProfileStore) *OverlayStore {
	if len(stores) == 0 {
		return &OverlayStore{}
	}

	readers := make([]ProfileStoreReader, 0, len(stores))
	for _, store := range stores {
		if store == nil {
			continue
		}
		readers = append(readers, store)
	}

	var writer ProfileStoreWriter
	if len(stores) > 0 {
		writer = stores[0]
	}

	return &OverlayStore{readers: readers, writer: writer}
}

// NewOverlayStoreWithWriter constructs an overlay with an explicit write target.
func NewOverlayStoreWithWriter(writer ProfileStoreWriter, readers ...ProfileStoreReader) *OverlayStore {
	cleanReaders := make([]ProfileStoreReader, 0, len(readers))
	for _, reader := range readers {
		if reader == nil {
			continue
		}
		cleanReaders = append(cleanReaders, reader)
	}
	return &OverlayStore{readers: cleanReaders, writer: writer}
}

func (s *OverlayStore) ListRegistries(ctx context.Context) ([]*ProfileRegistry, error) {
	merged := map[RegistrySlug]*ProfileRegistry{}
	for _, reader := range s.readers {
		registries, err := reader.ListRegistries(ctx)
		if err != nil {
			return nil, err
		}
		for _, registry := range registries {
			if registry == nil || registry.Slug.IsZero() {
				continue
			}
			current := merged[registry.Slug]
			merged[registry.Slug] = mergeRegistry(current, registry)
		}
	}

	slugs := make([]RegistrySlug, 0, len(merged))
	for slug := range merged {
		slugs = append(slugs, slug)
	}
	sort.Slice(slugs, func(i, j int) bool { return slugs[i] < slugs[j] })

	out := make([]*ProfileRegistry, 0, len(slugs))
	for _, slug := range slugs {
		out = append(out, merged[slug].Clone())
	}
	return out, nil
}

func (s *OverlayStore) GetRegistry(ctx context.Context, registrySlug RegistrySlug) (*ProfileRegistry, bool, error) {
	if registrySlug.IsZero() {
		return nil, false, nil
	}
	var merged *ProfileRegistry
	for _, reader := range s.readers {
		registry, ok, err := reader.GetRegistry(ctx, registrySlug)
		if err != nil {
			return nil, false, err
		}
		if !ok || registry == nil {
			continue
		}
		merged = mergeRegistry(merged, registry)
	}
	if merged == nil {
		return nil, false, nil
	}
	return merged.Clone(), true, nil
}

func (s *OverlayStore) ListProfiles(ctx context.Context, registrySlug RegistrySlug) ([]*Profile, error) {
	registry, ok, err := s.GetRegistry(ctx, registrySlug)
	if err != nil {
		return nil, err
	}
	if !ok || registry == nil {
		return nil, nil
	}

	slugs := make([]ProfileSlug, 0, len(registry.Profiles))
	for slug := range registry.Profiles {
		slugs = append(slugs, slug)
	}
	sort.Slice(slugs, func(i, j int) bool { return slugs[i] < slugs[j] })

	out := make([]*Profile, 0, len(slugs))
	for _, slug := range slugs {
		out = append(out, registry.Profiles[slug].Clone())
	}
	return out, nil
}

func (s *OverlayStore) GetProfile(ctx context.Context, registrySlug RegistrySlug, profileSlug ProfileSlug) (*Profile, bool, error) {
	registry, ok, err := s.GetRegistry(ctx, registrySlug)
	if err != nil {
		return nil, false, err
	}
	if !ok || registry == nil {
		return nil, false, nil
	}
	profile, ok := registry.Profiles[profileSlug]
	if !ok || profile == nil {
		return nil, false, nil
	}
	return profile.Clone(), true, nil
}

func (s *OverlayStore) UpsertRegistry(ctx context.Context, registry *ProfileRegistry, opts SaveOptions) error {
	if s.writer == nil {
		return ErrReadOnlyStore
	}
	return s.writer.UpsertRegistry(ctx, registry, opts)
}

func (s *OverlayStore) DeleteRegistry(ctx context.Context, registrySlug RegistrySlug, opts SaveOptions) error {
	if s.writer == nil {
		return ErrReadOnlyStore
	}
	return s.writer.DeleteRegistry(ctx, registrySlug, opts)
}

func (s *OverlayStore) UpsertProfile(ctx context.Context, registrySlug RegistrySlug, profile *Profile, opts SaveOptions) error {
	if s.writer == nil {
		return ErrReadOnlyStore
	}
	return s.writer.UpsertProfile(ctx, registrySlug, profile, opts)
}

func (s *OverlayStore) DeleteProfile(ctx context.Context, registrySlug RegistrySlug, profileSlug ProfileSlug, opts SaveOptions) error {
	if s.writer == nil {
		return ErrReadOnlyStore
	}
	return s.writer.DeleteProfile(ctx, registrySlug, profileSlug, opts)
}

func (s *OverlayStore) SetDefaultProfile(ctx context.Context, registrySlug RegistrySlug, profileSlug ProfileSlug, opts SaveOptions) error {
	if s.writer == nil {
		return ErrReadOnlyStore
	}
	return s.writer.SetDefaultProfile(ctx, registrySlug, profileSlug, opts)
}

func (s *OverlayStore) Close() error {
	var errs []error
	closed := map[string]struct{}{}

	if s.writer != nil {
		id := fmt.Sprintf("%p", s.writer)
		closed[id] = struct{}{}
		if err := s.writer.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	for _, reader := range s.readers {
		closer, ok := reader.(interface{ Close() error })
		if !ok {
			continue
		}
		id := fmt.Sprintf("%p", closer)
		if _, ok := closed[id]; ok {
			continue
		}
		closed[id] = struct{}{}
		if err := closer.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) == 0 {
		return nil
	}
	if len(errs) == 1 {
		return errs[0]
	}
	return fmt.Errorf("overlay close failed with %d errors", len(errs))
}

func mergeRegistry(base *ProfileRegistry, overlay *ProfileRegistry) *ProfileRegistry {
	if base == nil {
		return overlay.Clone()
	}
	if overlay == nil {
		return base.Clone()
	}

	ret := base.Clone()
	if overlay.DisplayName != "" {
		ret.DisplayName = overlay.DisplayName
	}
	if overlay.Description != "" {
		ret.Description = overlay.Description
	}
	if !overlay.DefaultProfileSlug.IsZero() {
		ret.DefaultProfileSlug = overlay.DefaultProfileSlug
	}
	if overlay.Metadata.Source != "" || overlay.Metadata.Version > 0 {
		ret.Metadata = overlay.Metadata
	}
	if ret.Profiles == nil {
		ret.Profiles = map[ProfileSlug]*Profile{}
	}
	for slug, profile := range overlay.Profiles {
		ret.Profiles[slug] = profile.Clone()
	}
	return ret
}

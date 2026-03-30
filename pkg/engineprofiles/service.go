package engineprofiles

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

var _ Registry = (*StoreRegistry)(nil)

// StoreRegistry is the default Registry implementation backed by an EngineProfileStore.
type StoreRegistry struct {
	store               EngineProfileStore
	defaultRegistrySlug RegistrySlug
}

type StoreRegistryOption func(*StoreRegistry) error

func NewStoreRegistry(store EngineProfileStore, defaultRegistrySlug RegistrySlug, options ...StoreRegistryOption) (*StoreRegistry, error) {
	if store == nil {
		return nil, fmt.Errorf("engine profile store is required")
	}
	if defaultRegistrySlug.IsZero() {
		defaultRegistrySlug = MustRegistrySlug("default")
	}
	ret := &StoreRegistry{store: store, defaultRegistrySlug: defaultRegistrySlug}
	for _, opt := range options {
		if opt == nil {
			continue
		}
		if err := opt(ret); err != nil {
			return nil, err
		}
	}
	return ret, nil
}

func (r *StoreRegistry) ListRegistries(ctx context.Context) ([]RegistrySummary, error) {
	registries, err := r.store.ListRegistries(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]RegistrySummary, 0, len(registries))
	for _, reg := range registries {
		if reg == nil {
			continue
		}
		out = append(out, RegistrySummary{
			Slug:                     reg.Slug,
			DisplayName:              reg.DisplayName,
			DefaultEngineProfileSlug: reg.DefaultEngineProfileSlug,
			EngineProfileCount:       len(reg.Profiles),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Slug < out[j].Slug })
	return out, nil
}

func (r *StoreRegistry) GetRegistry(ctx context.Context, registrySlug RegistrySlug) (*EngineProfileRegistry, error) {
	resolvedRegistrySlug := r.resolveRegistrySlug(registrySlug)
	reg, ok, err := r.store.GetRegistry(ctx, resolvedRegistrySlug)
	if err != nil {
		return nil, err
	}
	if !ok || reg == nil {
		return nil, ErrRegistryNotFound
	}
	return reg, nil
}

func (r *StoreRegistry) ListEngineProfiles(ctx context.Context, registrySlug RegistrySlug) ([]*EngineProfile, error) {
	resolvedRegistrySlug := r.resolveRegistrySlug(registrySlug)
	if _, err := r.GetRegistry(ctx, resolvedRegistrySlug); err != nil {
		return nil, err
	}
	profiles, err := r.store.ListEngineProfiles(ctx, resolvedRegistrySlug)
	if err != nil {
		return nil, err
	}
	return profiles, nil
}

func (r *StoreRegistry) GetEngineProfile(ctx context.Context, registrySlug RegistrySlug, profileSlug EngineProfileSlug) (*EngineProfile, error) {
	resolvedRegistrySlug := r.resolveRegistrySlug(registrySlug)
	resolvedEngineProfileSlug, err := resolveEngineProfileSlugInput(profileSlug)
	if err != nil {
		return nil, err
	}

	if _, err := r.GetRegistry(ctx, resolvedRegistrySlug); err != nil {
		return nil, err
	}

	profile, ok, err := r.store.GetEngineProfile(ctx, resolvedRegistrySlug, resolvedEngineProfileSlug)
	if err != nil {
		return nil, err
	}
	if !ok || profile == nil {
		return nil, ErrProfileNotFound
	}
	return profile, nil
}

func (r *StoreRegistry) ResolveEngineProfile(ctx context.Context, in ResolveInput) (*ResolvedEngineProfile, error) {
	registrySlug := r.resolveRegistrySlug(in.RegistrySlug)
	registry, err := r.GetRegistry(ctx, registrySlug)
	if err != nil {
		return nil, err
	}

	profileSlug, err := r.resolveEngineProfileSlugForRegistry(in.EngineProfileSlug, registry)
	if err != nil {
		return nil, err
	}

	stackLayers, err := r.ExpandEngineProfileStack(ctx, registrySlug, profileSlug, StackResolverOptions{})
	if err != nil {
		return nil, err
	}
	if len(stackLayers) == 0 {
		return nil, ErrProfileNotFound
	}
	rootLayer := stackLayers[len(stackLayers)-1]
	profile := rootLayer.EngineProfile

	stackMerge, err := MergeEngineProfileStackLayers(stackLayers)
	if err != nil {
		return nil, err
	}

	lineage := resolvedProfileStackLineage(stackLayers)
	metadata := map[string]any{
		"profile.registry":      registrySlug.String(),
		"profile.slug":          profileSlug.String(),
		"profile.version":       profile.Metadata.Version,
		"profile.source":        profileMetadataSource(profile, registry),
		"profile.stack.lineage": resolvedProfileStackLineageMetadata(lineage),
	}

	return &ResolvedEngineProfile{
		RegistrySlug:      registrySlug,
		EngineProfileSlug: profileSlug,
		InferenceSettings: cloneInferenceSettings(stackMerge.InferenceSettings),
		StackLineage:      lineage,
		Metadata:          metadata,
	}, nil
}

func (r *StoreRegistry) resolveRegistrySlug(slug RegistrySlug) RegistrySlug {
	if !slug.IsZero() {
		return slug
	}
	return r.defaultRegistrySlug
}

func (r *StoreRegistry) resolveEngineProfileSlugForRegistry(input EngineProfileSlug, registry *EngineProfileRegistry) (EngineProfileSlug, error) {
	if registry == nil {
		return "", ErrRegistryNotFound
	}
	if !input.IsZero() {
		return resolveEngineProfileSlugInput(input)
	}
	if !registry.DefaultEngineProfileSlug.IsZero() {
		return registry.DefaultEngineProfileSlug, nil
	}
	if _, ok := registry.Profiles[MustEngineProfileSlug("default")]; ok {
		return MustEngineProfileSlug("default"), nil
	}
	return "", &ValidationError{Field: "profile.slug", Reason: "profile slug is required and registry has no default"}
}

func resolveEngineProfileSlugInput(slug EngineProfileSlug) (EngineProfileSlug, error) {
	if slug.IsZero() {
		return "", &ValidationError{Field: "profile.slug", Reason: "must not be empty"}
	}
	parsed, err := ParseEngineProfileSlug(slug.String())
	if err != nil {
		return "", &ValidationError{Field: "profile.slug", Reason: err.Error()}
	}
	return parsed, nil
}

func resolvedProfileStackLineage(stackLayers []EngineProfileStackLayer) []ResolvedProfileStackEntry {
	lineage := make([]ResolvedProfileStackEntry, 0, len(stackLayers))
	for _, layer := range stackLayers {
		version := uint64(0)
		source := ""
		if layer.EngineProfile != nil {
			version = layer.EngineProfile.Metadata.Version
			source = strings.TrimSpace(layer.EngineProfile.Metadata.Source)
		}
		lineage = append(lineage, ResolvedProfileStackEntry{
			RegistrySlug:      layer.RegistrySlug,
			EngineProfileSlug: layer.EngineProfileSlug,
			Version:           version,
			Source:            source,
		})
	}
	return lineage
}

func resolvedProfileStackLineageMetadata(lineage []ResolvedProfileStackEntry) []map[string]any {
	if len(lineage) == 0 {
		return nil
	}
	ret := make([]map[string]any, 0, len(lineage))
	for _, entry := range lineage {
		ret = append(ret, map[string]any{
			"registry_slug": entry.RegistrySlug.String(),
			"profile_slug":  entry.EngineProfileSlug.String(),
			"version":       entry.Version,
			"source":        entry.Source,
		})
	}
	return ret
}

func profileMetadataSource(profile *EngineProfile, registry *EngineProfileRegistry) string {
	if profile != nil && strings.TrimSpace(profile.Metadata.Source) != "" {
		return profile.Metadata.Source
	}
	if registry != nil && strings.TrimSpace(registry.Metadata.Source) != "" {
		return registry.Metadata.Source
	}
	return ""
}

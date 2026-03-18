package engineprofiles

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

var _ Registry = (*StoreRegistry)(nil)

// StoreRegistry is the default Registry implementation backed by a ProfileStore.
type StoreRegistry struct {
	store               ProfileStore
	defaultRegistrySlug RegistrySlug
}

type StoreRegistryOption func(*StoreRegistry) error

func NewStoreRegistry(store ProfileStore, defaultRegistrySlug RegistrySlug, options ...StoreRegistryOption) (*StoreRegistry, error) {
	if store == nil {
		return nil, fmt.Errorf("profile store is required")
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
			Slug:               reg.Slug,
			DisplayName:        reg.DisplayName,
			DefaultProfileSlug: reg.DefaultProfileSlug,
			ProfileCount:       len(reg.Profiles),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Slug < out[j].Slug })
	return out, nil
}

func (r *StoreRegistry) GetRegistry(ctx context.Context, registrySlug RegistrySlug) (*ProfileRegistry, error) {
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

func (r *StoreRegistry) ListProfiles(ctx context.Context, registrySlug RegistrySlug) ([]*Profile, error) {
	resolvedRegistrySlug := r.resolveRegistrySlug(registrySlug)
	if _, err := r.GetRegistry(ctx, resolvedRegistrySlug); err != nil {
		return nil, err
	}
	profiles, err := r.store.ListProfiles(ctx, resolvedRegistrySlug)
	if err != nil {
		return nil, err
	}
	return profiles, nil
}

func (r *StoreRegistry) GetProfile(ctx context.Context, registrySlug RegistrySlug, profileSlug ProfileSlug) (*Profile, error) {
	resolvedRegistrySlug := r.resolveRegistrySlug(registrySlug)
	resolvedProfileSlug, err := resolveProfileSlugInput(profileSlug)
	if err != nil {
		return nil, err
	}

	if _, err := r.GetRegistry(ctx, resolvedRegistrySlug); err != nil {
		return nil, err
	}

	profile, ok, err := r.store.GetProfile(ctx, resolvedRegistrySlug, resolvedProfileSlug)
	if err != nil {
		return nil, err
	}
	if !ok || profile == nil {
		return nil, ErrProfileNotFound
	}
	return profile, nil
}

func (r *StoreRegistry) ResolveEffectiveProfile(ctx context.Context, in ResolveInput) (*ResolvedProfile, error) {
	registrySlug := r.resolveRegistrySlug(in.RegistrySlug)
	registry, err := r.GetRegistry(ctx, registrySlug)
	if err != nil {
		return nil, err
	}

	profileSlug, err := r.resolveProfileSlugForRegistry(in.ProfileSlug, registry)
	if err != nil {
		return nil, err
	}

	stackLayers, err := r.ExpandProfileStack(ctx, registrySlug, profileSlug, StackResolverOptions{})
	if err != nil {
		return nil, err
	}
	if len(stackLayers) == 0 {
		return nil, ErrProfileNotFound
	}
	rootLayer := stackLayers[len(stackLayers)-1]
	profile := rootLayer.Profile

	stackMerge, stackTrace, err := MergeProfileStackLayersWithTrace(stackLayers)
	if err != nil {
		return nil, err
	}

	effectiveRuntime, err := resolveRuntimeSpec(stackMerge.Runtime)
	if err != nil {
		return nil, err
	}

	runtimeKey, err := ParseRuntimeKey(profileSlug.String())
	if err != nil {
		return nil, err
	}

	metadata := map[string]any{
		"profile.registry":      registrySlug.String(),
		"profile.slug":          profileSlug.String(),
		"profile.version":       profile.Metadata.Version,
		"profile.source":        profileMetadataSource(profile, registry),
		"profile.stack.lineage": stackLayerLineage(stackLayers),
		"profile.stack.trace":   stackTrace.BuildDebugPayload(),
	}

	return &ResolvedProfile{
		RegistrySlug:       registrySlug,
		ProfileSlug:        profileSlug,
		RuntimeKey:         runtimeKey,
		EffectiveRuntime:   effectiveRuntime,
		RuntimeFingerprint: runtimeFingerprint(registrySlug, profile, stackLayers, effectiveRuntime),
		Metadata:           metadata,
	}, nil
}

func (r *StoreRegistry) resolveRegistrySlug(slug RegistrySlug) RegistrySlug {
	if !slug.IsZero() {
		return slug
	}
	return r.defaultRegistrySlug
}

func (r *StoreRegistry) resolveProfileSlugForRegistry(input ProfileSlug, registry *ProfileRegistry) (ProfileSlug, error) {
	if registry == nil {
		return "", ErrRegistryNotFound
	}
	if !input.IsZero() {
		return resolveProfileSlugInput(input)
	}
	if !registry.DefaultProfileSlug.IsZero() {
		return registry.DefaultProfileSlug, nil
	}
	if _, ok := registry.Profiles[MustProfileSlug("default")]; ok {
		return MustProfileSlug("default"), nil
	}
	return "", &ValidationError{Field: "profile.slug", Reason: "profile slug is required and registry has no default"}
}

func resolveProfileSlugInput(slug ProfileSlug) (ProfileSlug, error) {
	if slug.IsZero() {
		return "", &ValidationError{Field: "profile.slug", Reason: "must not be empty"}
	}
	parsed, err := ParseProfileSlug(slug.String())
	if err != nil {
		return "", &ValidationError{Field: "profile.slug", Reason: err.Error()}
	}
	return parsed, nil
}

func resolveRuntimeSpec(base RuntimeSpec) (RuntimeSpec, error) {
	return cloneRuntimeSpec(base), nil
}

func runtimeFingerprint(registrySlug RegistrySlug, profile *Profile, stackLayers []ProfileStackLayer, runtime RuntimeSpec) string {
	payload := map[string]any{
		"registry_slug":   registrySlug.String(),
		"profile_slug":    profile.Slug.String(),
		"profile_version": profile.Metadata.Version,
		"stack_lineage":   stackLayerLineage(stackLayers),
		"runtime": map[string]any{
			"system_prompt": runtime.SystemPrompt,
			"middlewares":   cloneMiddlewares(runtime.Middlewares),
			"tools":         append([]string(nil), runtime.Tools...),
		},
	}

	b, err := json.Marshal(payload)
	if err != nil {
		return profile.Slug.String()
	}
	sum := sha256.Sum256(b)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func stackLayerLineage(stackLayers []ProfileStackLayer) []map[string]any {
	lineage := make([]map[string]any, 0, len(stackLayers))
	for _, layer := range stackLayers {
		version := uint64(0)
		source := ""
		if layer.Profile != nil {
			version = layer.Profile.Metadata.Version
			source = strings.TrimSpace(layer.Profile.Metadata.Source)
		}
		lineage = append(lineage, map[string]any{
			"registry_slug": layer.RegistrySlug.String(),
			"profile_slug":  layer.ProfileSlug.String(),
			"version":       version,
			"source":        source,
		})
	}
	return lineage
}

func cloneMiddlewares(in []MiddlewareUse) []MiddlewareUse {
	if len(in) == 0 {
		return nil
	}
	ret := make([]MiddlewareUse, 0, len(in))
	for _, mw := range in {
		ret = append(ret, MiddlewareUse{
			Name:    strings.TrimSpace(mw.Name),
			ID:      strings.TrimSpace(mw.ID),
			Enabled: cloneBoolPtr(mw.Enabled),
			Config:  deepCopyAny(mw.Config),
		})
	}
	return ret
}

func cloneRuntimeSpec(in RuntimeSpec) RuntimeSpec {
	return RuntimeSpec{
		SystemPrompt: in.SystemPrompt,
		Middlewares:  cloneMiddlewares(in.Middlewares),
		Tools:        append([]string(nil), in.Tools...),
	}
}

func profileMetadataSource(profile *Profile, registry *ProfileRegistry) string {
	if profile != nil && strings.TrimSpace(profile.Metadata.Source) != "" {
		return profile.Metadata.Source
	}
	if registry != nil && strings.TrimSpace(registry.Metadata.Source) != "" {
		return registry.Metadata.Source
	}
	return ""
}

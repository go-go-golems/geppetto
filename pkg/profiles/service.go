package profiles

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
)

const (
	overrideKeySystemPrompt      = "system_prompt"
	overrideKeyMiddlewares       = "middlewares"
	overrideKeyTools             = "tools"
	overrideKeyStepSettingsPatch = "step_settings_patch"
)

var _ Registry = (*StoreRegistry)(nil)

// StoreRegistry is the default Registry implementation backed by a ProfileStore.
type StoreRegistry struct {
	store               ProfileStore
	defaultRegistrySlug RegistrySlug
	extensionCodecs     ExtensionCodecRegistry
}

type StoreRegistryOption func(*StoreRegistry) error

func WithExtensionCodecRegistry(registry ExtensionCodecRegistry) StoreRegistryOption {
	return func(sr *StoreRegistry) error {
		if sr == nil {
			return fmt.Errorf("store registry is nil")
		}
		sr.extensionCodecs = registry
		return nil
	}
}

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
	profile, err := r.GetProfile(ctx, registrySlug, profileSlug)
	if err != nil {
		return nil, err
	}

	effectiveRuntime, err := resolveRuntimeSpec(profile.Runtime, profile.Policy, in.RequestOverrides)
	if err != nil {
		return nil, err
	}

	effectiveStepSettings, err := ApplyStepSettingsPatch(in.BaseStepSettings, effectiveRuntime.StepSettingsPatch)
	if err != nil {
		return nil, err
	}

	runtimeKey := in.RuntimeKeyFallback
	if runtimeKey.IsZero() {
		runtimeKey, err = ParseRuntimeKey(profileSlug.String())
		if err != nil {
			return nil, err
		}
	}

	metadata := map[string]any{
		"profile.registry": registrySlug.String(),
		"profile.slug":     profileSlug.String(),
		"profile.version":  profile.Metadata.Version,
		"profile.source":   profileMetadataSource(profile, registry),
	}

	return &ResolvedProfile{
		RegistrySlug:          registrySlug,
		ProfileSlug:           profileSlug,
		RuntimeKey:            runtimeKey,
		EffectiveStepSettings: effectiveStepSettings,
		EffectiveRuntime:      effectiveRuntime,
		RuntimeFingerprint:    runtimeFingerprint(registrySlug, profile, effectiveRuntime, effectiveStepSettings),
		Metadata:              metadata,
	}, nil
}

func (r *StoreRegistry) CreateProfile(ctx context.Context, registrySlug RegistrySlug, profile *Profile, opts WriteOptions) (*Profile, error) {
	resolvedRegistrySlug := r.resolveRegistrySlug(registrySlug)
	if _, err := r.GetRegistry(ctx, resolvedRegistrySlug); err != nil {
		return nil, err
	}

	if err := ValidateProfile(profile); err != nil {
		return nil, err
	}
	if _, ok, err := r.store.GetProfile(ctx, resolvedRegistrySlug, profile.Slug); err != nil {
		return nil, err
	} else if ok {
		return nil, &VersionConflictError{
			Resource: "profile",
			Slug:     profile.Slug.String(),
			Expected: 0,
			Actual:   profile.Metadata.Version,
		}
	}

	if err := r.store.UpsertProfile(ctx, resolvedRegistrySlug, profile, toSaveOptions(opts)); err != nil {
		return nil, err
	}
	return r.GetProfile(ctx, resolvedRegistrySlug, profile.Slug)
}

func (r *StoreRegistry) UpdateProfile(ctx context.Context, registrySlug RegistrySlug, profileSlug ProfileSlug, patch ProfilePatch, opts WriteOptions) (*Profile, error) {
	resolvedRegistrySlug := r.resolveRegistrySlug(registrySlug)
	resolvedProfileSlug, err := resolveProfileSlugInput(profileSlug)
	if err != nil {
		return nil, err
	}

	current, err := r.GetProfile(ctx, resolvedRegistrySlug, resolvedProfileSlug)
	if err != nil {
		return nil, err
	}
	if current.Policy.ReadOnly {
		return nil, &PolicyViolationError{ProfileSlug: resolvedProfileSlug, Reason: "profile is read-only"}
	}

	next := current.Clone()
	if patch.DisplayName != nil {
		next.DisplayName = strings.TrimSpace(*patch.DisplayName)
	}
	if patch.Description != nil {
		next.Description = strings.TrimSpace(*patch.Description)
	}
	if patch.Runtime != nil {
		next.Runtime = cloneRuntimeSpec(*patch.Runtime)
	}
	if patch.Policy != nil {
		next.Policy = clonePolicySpec(*patch.Policy)
	}
	if patch.Metadata != nil {
		next.Metadata = mergeProfileMetadata(next.Metadata, *patch.Metadata)
	}

	if err := ValidateProfile(next); err != nil {
		return nil, err
	}
	if err := r.store.UpsertProfile(ctx, resolvedRegistrySlug, next, toSaveOptions(opts)); err != nil {
		return nil, err
	}
	return r.GetProfile(ctx, resolvedRegistrySlug, resolvedProfileSlug)
}

func (r *StoreRegistry) DeleteProfile(ctx context.Context, registrySlug RegistrySlug, profileSlug ProfileSlug, opts WriteOptions) error {
	resolvedRegistrySlug := r.resolveRegistrySlug(registrySlug)
	resolvedProfileSlug, err := resolveProfileSlugInput(profileSlug)
	if err != nil {
		return err
	}

	current, err := r.GetProfile(ctx, resolvedRegistrySlug, resolvedProfileSlug)
	if err != nil {
		return err
	}
	if current.Policy.ReadOnly {
		return &PolicyViolationError{ProfileSlug: resolvedProfileSlug, Reason: "profile is read-only"}
	}

	return r.store.DeleteProfile(ctx, resolvedRegistrySlug, resolvedProfileSlug, toSaveOptions(opts))
}

func (r *StoreRegistry) SetDefaultProfile(ctx context.Context, registrySlug RegistrySlug, profileSlug ProfileSlug, opts WriteOptions) error {
	resolvedRegistrySlug := r.resolveRegistrySlug(registrySlug)
	resolvedProfileSlug, err := resolveProfileSlugInput(profileSlug)
	if err != nil {
		return err
	}
	if _, err := r.GetProfile(ctx, resolvedRegistrySlug, resolvedProfileSlug); err != nil {
		return err
	}
	return r.store.SetDefaultProfile(ctx, resolvedRegistrySlug, resolvedProfileSlug, toSaveOptions(opts))
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

func resolveRuntimeSpec(base RuntimeSpec, policy PolicySpec, requestOverrides map[string]any) (RuntimeSpec, error) {
	ret := cloneRuntimeSpec(base)
	if len(requestOverrides) == 0 {
		return ret, nil
	}

	normalized, err := normalizeOverrideMap(requestOverrides)
	if err != nil {
		return RuntimeSpec{}, err
	}
	if err := enforceOverridePolicy(policy, normalized); err != nil {
		return RuntimeSpec{}, err
	}

	for key, value := range normalized {
		switch key {
		case overrideKeySystemPrompt:
			s, ok := value.(string)
			if !ok {
				return RuntimeSpec{}, &ValidationError{Field: "request_overrides.system_prompt", Reason: "must be a string"}
			}
			ret.SystemPrompt = s
		case overrideKeyMiddlewares:
			mws, err := parseMiddlewareOverrideValue(value)
			if err != nil {
				return RuntimeSpec{}, err
			}
			ret.Middlewares = mws
		case overrideKeyTools:
			tools, err := parseToolOverrideValue(value)
			if err != nil {
				return RuntimeSpec{}, err
			}
			ret.Tools = tools
		case overrideKeyStepSettingsPatch:
			overlayPatch, err := parseStepSettingsPatchOverrideValue(value)
			if err != nil {
				return RuntimeSpec{}, err
			}
			merged, err := MergeStepSettingsPatches(ret.StepSettingsPatch, overlayPatch)
			if err != nil {
				return RuntimeSpec{}, err
			}
			ret.StepSettingsPatch = merged
		default:
			return RuntimeSpec{}, &ValidationError{Field: fmt.Sprintf("request_overrides.%s", key), Reason: "unsupported override key"}
		}
	}

	return ret, nil
}

func normalizeOverrideMap(overrides map[string]any) (map[string]any, error) {
	if len(overrides) == 0 {
		return nil, nil
	}
	out := map[string]any{}
	for rawKey, value := range overrides {
		key := canonicalOverrideKey(rawKey)
		if key == "" {
			return nil, &ValidationError{Field: "request_overrides", Reason: "override keys must not be empty"}
		}
		out[key] = value
	}
	return out, nil
}

func enforceOverridePolicy(policy PolicySpec, overrides map[string]any) error {
	if len(overrides) == 0 {
		return nil
	}
	if !policy.AllowOverrides {
		return &PolicyViolationError{Reason: "request overrides are disabled for this profile"}
	}

	allowed := map[string]struct{}{}
	for _, key := range policy.AllowedOverrideKeys {
		allowed[canonicalOverrideKey(key)] = struct{}{}
	}
	denied := map[string]struct{}{}
	for _, key := range policy.DeniedOverrideKeys {
		denied[canonicalOverrideKey(key)] = struct{}{}
	}

	for key := range overrides {
		if _, deniedKey := denied[key]; deniedKey {
			return &PolicyViolationError{Reason: fmt.Sprintf("override key %q is denied", key)}
		}
		if len(allowed) > 0 {
			if _, allowedKey := allowed[key]; !allowedKey {
				return &PolicyViolationError{Reason: fmt.Sprintf("override key %q is not allowed", key)}
			}
		}
	}
	return nil
}

func canonicalOverrideKey(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	trimmed = camelToSnake(trimmed)
	trimmed = strings.ToLower(trimmed)
	trimmed = strings.ReplaceAll(trimmed, "-", "_")
	return trimmed
}

func camelToSnake(s string) string {
	if s == "" {
		return s
	}
	out := make([]rune, 0, len(s)+4)
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			out = append(out, '_')
		}
		out = append(out, r)
	}
	return string(out)
}

func parseMiddlewareOverrideValue(v any) ([]MiddlewareUse, error) {
	switch typed := v.(type) {
	case []MiddlewareUse:
		ret := make([]MiddlewareUse, 0, len(typed))
		for i := range typed {
			name := strings.TrimSpace(typed[i].Name)
			if name == "" {
				return nil, &ValidationError{Field: fmt.Sprintf("request_overrides.middlewares[%d].name", i), Reason: "must not be empty"}
			}
			ret = append(ret, MiddlewareUse{Name: name, Config: deepCopyAny(typed[i].Config)})
		}
		return ret, nil
	case []any:
		ret := make([]MiddlewareUse, 0, len(typed))
		for i, raw := range typed {
			obj, ok := toStringAnyMap(raw)
			if !ok {
				return nil, &ValidationError{Field: fmt.Sprintf("request_overrides.middlewares[%d]", i), Reason: "must be an object"}
			}
			name, _ := obj["name"].(string)
			name = strings.TrimSpace(name)
			if name == "" {
				return nil, &ValidationError{Field: fmt.Sprintf("request_overrides.middlewares[%d].name", i), Reason: "must not be empty"}
			}
			ret = append(ret, MiddlewareUse{Name: name, Config: deepCopyAny(obj["config"])})
		}
		return ret, nil
	default:
		return nil, &ValidationError{Field: "request_overrides.middlewares", Reason: "must be an array"}
	}
}

func parseToolOverrideValue(v any) ([]string, error) {
	switch typed := v.(type) {
	case []string:
		ret := make([]string, 0, len(typed))
		for i, tool := range typed {
			normalized := strings.TrimSpace(tool)
			if normalized == "" {
				return nil, &ValidationError{Field: fmt.Sprintf("request_overrides.tools[%d]", i), Reason: "must not be empty"}
			}
			ret = append(ret, normalized)
		}
		return ret, nil
	case []any:
		ret := make([]string, 0, len(typed))
		for i, raw := range typed {
			tool, ok := raw.(string)
			if !ok {
				return nil, &ValidationError{Field: fmt.Sprintf("request_overrides.tools[%d]", i), Reason: "must be a string"}
			}
			normalized := strings.TrimSpace(tool)
			if normalized == "" {
				return nil, &ValidationError{Field: fmt.Sprintf("request_overrides.tools[%d]", i), Reason: "must not be empty"}
			}
			ret = append(ret, normalized)
		}
		return ret, nil
	default:
		return nil, &ValidationError{Field: "request_overrides.tools", Reason: "must be an array"}
	}
}

func parseStepSettingsPatchOverrideValue(v any) (map[string]any, error) {
	obj, ok := toStringAnyMap(v)
	if !ok {
		return nil, &ValidationError{Field: "request_overrides.step_settings_patch", Reason: "must be an object"}
	}
	out := map[string]any{}
	for sectionSlug, rawSection := range obj {
		sectionMap, ok := toStringAnyMap(rawSection)
		if !ok {
			return nil, &ValidationError{Field: fmt.Sprintf("request_overrides.step_settings_patch.%s", sectionSlug), Reason: "must be an object"}
		}
		out[sectionSlug] = deepCopyStringAnyMap(sectionMap)
	}
	return out, nil
}

func runtimeFingerprint(registrySlug RegistrySlug, profile *Profile, runtime RuntimeSpec, stepSettings *settings.StepSettings) string {
	payload := map[string]any{
		"registry_slug":   registrySlug.String(),
		"profile_slug":    profile.Slug.String(),
		"profile_version": profile.Metadata.Version,
		"runtime": map[string]any{
			"step_settings_patch": deepCopyStringAnyMap(runtime.StepSettingsPatch),
			"system_prompt":       runtime.SystemPrompt,
			"middlewares":         cloneMiddlewares(runtime.Middlewares),
			"tools":               append([]string(nil), runtime.Tools...),
		},
	}
	if stepSettings != nil {
		payload["step_metadata"] = stepSettings.GetMetadata()
	}

	b, err := json.Marshal(payload)
	if err != nil {
		return profile.Slug.String()
	}
	sum := sha256.Sum256(b)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func cloneMiddlewares(in []MiddlewareUse) []MiddlewareUse {
	if len(in) == 0 {
		return nil
	}
	ret := make([]MiddlewareUse, 0, len(in))
	for _, mw := range in {
		ret = append(ret, MiddlewareUse{Name: strings.TrimSpace(mw.Name), Config: deepCopyAny(mw.Config)})
	}
	return ret
}

func cloneRuntimeSpec(in RuntimeSpec) RuntimeSpec {
	return RuntimeSpec{
		StepSettingsPatch: deepCopyStringAnyMap(in.StepSettingsPatch),
		SystemPrompt:      in.SystemPrompt,
		Middlewares:       cloneMiddlewares(in.Middlewares),
		Tools:             append([]string(nil), in.Tools...),
	}
}

func clonePolicySpec(in PolicySpec) PolicySpec {
	return PolicySpec{
		AllowOverrides:      in.AllowOverrides,
		AllowedOverrideKeys: append([]string(nil), in.AllowedOverrideKeys...),
		DeniedOverrideKeys:  append([]string(nil), in.DeniedOverrideKeys...),
		ReadOnly:            in.ReadOnly,
	}
}

func mergeProfileMetadata(base ProfileMetadata, patch ProfileMetadata) ProfileMetadata {
	ret := base
	if patch.Source != "" {
		ret.Source = patch.Source
	}
	if patch.CreatedBy != "" {
		ret.CreatedBy = patch.CreatedBy
	}
	if patch.UpdatedBy != "" {
		ret.UpdatedBy = patch.UpdatedBy
	}
	if patch.CreatedAtMs != 0 {
		ret.CreatedAtMs = patch.CreatedAtMs
	}
	if patch.UpdatedAtMs != 0 {
		ret.UpdatedAtMs = patch.UpdatedAtMs
	}
	if patch.Version != 0 {
		ret.Version = patch.Version
	}
	if patch.Tags != nil {
		ret.Tags = append([]string(nil), patch.Tags...)
	}
	return ret
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

func toSaveOptions(opts WriteOptions) SaveOptions {
	return SaveOptions(opts)
}

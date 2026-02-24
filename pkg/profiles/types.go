package profiles

import "strings"

// MiddlewareUse describes a named middleware and optional config payload.
type MiddlewareUse struct {
	Name    string `json:"name" yaml:"name"`
	ID      string `json:"id,omitempty" yaml:"id,omitempty"`
	Enabled *bool  `json:"enabled,omitempty" yaml:"enabled,omitempty"`
	Config  any    `json:"config,omitempty" yaml:"config,omitempty"`
}

// RuntimeSpec describes runtime-level defaults and patches provided by a profile.
type RuntimeSpec struct {
	StepSettingsPatch map[string]any  `json:"step_settings_patch,omitempty" yaml:"step_settings_patch,omitempty"`
	SystemPrompt      string          `json:"system_prompt,omitempty" yaml:"system_prompt,omitempty"`
	Middlewares       []MiddlewareUse `json:"middlewares,omitempty" yaml:"middlewares,omitempty"`
	Tools             []string        `json:"tools,omitempty" yaml:"tools,omitempty"`
}

// PolicySpec controls mutability and override behavior for a profile.
type PolicySpec struct {
	AllowOverrides      bool     `json:"allow_overrides" yaml:"allow_overrides"`
	AllowedOverrideKeys []string `json:"allowed_override_keys,omitempty" yaml:"allowed_override_keys,omitempty"`
	DeniedOverrideKeys  []string `json:"denied_override_keys,omitempty" yaml:"denied_override_keys,omitempty"`
	ReadOnly            bool     `json:"read_only" yaml:"read_only"`
}

// ProfileMetadata stores provenance and version fields.
type ProfileMetadata struct {
	Source      string   `json:"source,omitempty" yaml:"source,omitempty"`
	Version     uint64   `json:"version,omitempty" yaml:"version,omitempty"`
	CreatedAtMs int64    `json:"created_at_ms,omitempty" yaml:"created_at_ms,omitempty"`
	UpdatedAtMs int64    `json:"updated_at_ms,omitempty" yaml:"updated_at_ms,omitempty"`
	CreatedBy   string   `json:"created_by,omitempty" yaml:"created_by,omitempty"`
	UpdatedBy   string   `json:"updated_by,omitempty" yaml:"updated_by,omitempty"`
	Tags        []string `json:"tags,omitempty" yaml:"tags,omitempty"`
}

// RegistryMetadata stores provenance and version fields at registry scope.
type RegistryMetadata struct {
	Source      string   `json:"source,omitempty" yaml:"source,omitempty"`
	Version     uint64   `json:"version,omitempty" yaml:"version,omitempty"`
	CreatedAtMs int64    `json:"created_at_ms,omitempty" yaml:"created_at_ms,omitempty"`
	UpdatedAtMs int64    `json:"updated_at_ms,omitempty" yaml:"updated_at_ms,omitempty"`
	CreatedBy   string   `json:"created_by,omitempty" yaml:"created_by,omitempty"`
	UpdatedBy   string   `json:"updated_by,omitempty" yaml:"updated_by,omitempty"`
	Tags        []string `json:"tags,omitempty" yaml:"tags,omitempty"`
}

// Profile is a named runtime preset with policy and metadata.
type Profile struct {
	Slug        ProfileSlug     `json:"slug" yaml:"slug"`
	DisplayName string          `json:"display_name,omitempty" yaml:"display_name,omitempty"`
	Description string          `json:"description,omitempty" yaml:"description,omitempty"`
	Runtime     RuntimeSpec     `json:"runtime,omitempty" yaml:"runtime,omitempty"`
	Policy      PolicySpec      `json:"policy,omitempty" yaml:"policy,omitempty"`
	Metadata    ProfileMetadata `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	Extensions  map[string]any  `json:"extensions,omitempty" yaml:"extensions,omitempty"`
}

// ProfileRegistry is a set of profiles with a default profile selector.
type ProfileRegistry struct {
	Slug               RegistrySlug             `json:"slug" yaml:"slug"`
	DisplayName        string                   `json:"display_name,omitempty" yaml:"display_name,omitempty"`
	Description        string                   `json:"description,omitempty" yaml:"description,omitempty"`
	DefaultProfileSlug ProfileSlug              `json:"default_profile_slug,omitempty" yaml:"default_profile_slug,omitempty"`
	Profiles           map[ProfileSlug]*Profile `json:"profiles,omitempty" yaml:"profiles,omitempty"`
	Metadata           RegistryMetadata         `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}

func (p *Profile) Clone() *Profile {
	if p == nil {
		return nil
	}

	ret := &Profile{
		Slug:        p.Slug,
		DisplayName: p.DisplayName,
		Description: p.Description,
		Runtime: RuntimeSpec{
			StepSettingsPatch: deepCopyStringAnyMap(p.Runtime.StepSettingsPatch),
			SystemPrompt:      p.Runtime.SystemPrompt,
			Middlewares:       append([]MiddlewareUse(nil), p.Runtime.Middlewares...),
			Tools:             append([]string(nil), p.Runtime.Tools...),
		},
		Policy: PolicySpec{
			AllowOverrides:      p.Policy.AllowOverrides,
			AllowedOverrideKeys: append([]string(nil), p.Policy.AllowedOverrideKeys...),
			DeniedOverrideKeys:  append([]string(nil), p.Policy.DeniedOverrideKeys...),
			ReadOnly:            p.Policy.ReadOnly,
		},
		Metadata: ProfileMetadata{
			Source:      p.Metadata.Source,
			Version:     p.Metadata.Version,
			CreatedAtMs: p.Metadata.CreatedAtMs,
			UpdatedAtMs: p.Metadata.UpdatedAtMs,
			CreatedBy:   p.Metadata.CreatedBy,
			UpdatedBy:   p.Metadata.UpdatedBy,
			Tags:        append([]string(nil), p.Metadata.Tags...),
		},
		Extensions: deepCopyStringAnyMap(p.Extensions),
	}
	for i := range ret.Runtime.Middlewares {
		ret.Runtime.Middlewares[i].Name = strings.TrimSpace(ret.Runtime.Middlewares[i].Name)
		ret.Runtime.Middlewares[i].ID = strings.TrimSpace(ret.Runtime.Middlewares[i].ID)
		ret.Runtime.Middlewares[i].Enabled = cloneBoolPtr(ret.Runtime.Middlewares[i].Enabled)
		ret.Runtime.Middlewares[i].Config = deepCopyAny(ret.Runtime.Middlewares[i].Config)
	}

	return ret
}

func (r *ProfileRegistry) Clone() *ProfileRegistry {
	if r == nil {
		return nil
	}

	ret := &ProfileRegistry{
		Slug:               r.Slug,
		DisplayName:        r.DisplayName,
		Description:        r.Description,
		DefaultProfileSlug: r.DefaultProfileSlug,
		Profiles:           make(map[ProfileSlug]*Profile, len(r.Profiles)),
		Metadata: RegistryMetadata{
			Source:      r.Metadata.Source,
			Version:     r.Metadata.Version,
			CreatedAtMs: r.Metadata.CreatedAtMs,
			UpdatedAtMs: r.Metadata.UpdatedAtMs,
			CreatedBy:   r.Metadata.CreatedBy,
			UpdatedBy:   r.Metadata.UpdatedBy,
			Tags:        append([]string(nil), r.Metadata.Tags...),
		},
	}
	for slug, p := range r.Profiles {
		ret.Profiles[slug] = p.Clone()
	}

	return ret
}

func deepCopyStringAnyMap(in map[string]any) map[string]any {
	if len(in) == 0 {
		return nil
	}
	ret := make(map[string]any, len(in))
	for k, v := range in {
		ret[k] = deepCopyAny(v)
	}
	return ret
}

func deepCopyAny(in any) any {
	switch v := in.(type) {
	case map[string]any:
		return deepCopyStringAnyMap(v)
	case []any:
		ret := make([]any, 0, len(v))
		for _, item := range v {
			ret = append(ret, deepCopyAny(item))
		}
		return ret
	default:
		return in
	}
}

func cloneBoolPtr(in *bool) *bool {
	if in == nil {
		return nil
	}
	v := *in
	return &v
}

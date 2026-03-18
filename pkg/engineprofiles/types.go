package engineprofiles

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
	SystemPrompt string          `json:"system_prompt,omitempty" yaml:"system_prompt,omitempty"`
	Middlewares  []MiddlewareUse `json:"middlewares,omitempty" yaml:"middlewares,omitempty"`
	Tools        []string        `json:"tools,omitempty" yaml:"tools,omitempty"`
}

// EngineProfileRef identifies a profile that can be layered via stack composition.
// Empty RegistrySlug means "same registry as the referencing profile".
type EngineProfileRef struct {
	RegistrySlug      RegistrySlug      `json:"registry_slug,omitempty" yaml:"registry_slug,omitempty"`
	EngineProfileSlug EngineProfileSlug `json:"profile_slug" yaml:"profile_slug"`
}

// EngineProfileMetadata stores provenance and version fields.
type EngineProfileMetadata struct {
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

// EngineProfile is a named runtime preset with runtime settings and metadata.
type EngineProfile struct {
	Slug        EngineProfileSlug     `json:"slug" yaml:"slug"`
	DisplayName string                `json:"display_name,omitempty" yaml:"display_name,omitempty"`
	Description string                `json:"description,omitempty" yaml:"description,omitempty"`
	Stack       []EngineProfileRef    `json:"stack,omitempty" yaml:"stack,omitempty"`
	Runtime     RuntimeSpec           `json:"runtime,omitempty" yaml:"runtime,omitempty"`
	Metadata    EngineProfileMetadata `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	Extensions  map[string]any        `json:"extensions,omitempty" yaml:"extensions,omitempty"`
}

// EngineProfileRegistry is a set of profiles with a default profile selector.
type EngineProfileRegistry struct {
	Slug                     RegistrySlug                         `json:"slug" yaml:"slug"`
	DisplayName              string                               `json:"display_name,omitempty" yaml:"display_name,omitempty"`
	Description              string                               `json:"description,omitempty" yaml:"description,omitempty"`
	DefaultEngineProfileSlug EngineProfileSlug                    `json:"default_profile_slug,omitempty" yaml:"default_profile_slug,omitempty"`
	Profiles                 map[EngineProfileSlug]*EngineProfile `json:"profiles,omitempty" yaml:"profiles,omitempty"`
	Metadata                 RegistryMetadata                     `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}

func (p *EngineProfile) Clone() *EngineProfile {
	if p == nil {
		return nil
	}

	ret := &EngineProfile{
		Slug:        p.Slug,
		DisplayName: p.DisplayName,
		Description: p.Description,
		Stack:       append([]EngineProfileRef(nil), p.Stack...),
		Runtime: RuntimeSpec{
			SystemPrompt: p.Runtime.SystemPrompt,
			Middlewares:  append([]MiddlewareUse(nil), p.Runtime.Middlewares...),
			Tools:        append([]string(nil), p.Runtime.Tools...),
		},
		Metadata: EngineProfileMetadata{
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

func (r *EngineProfileRegistry) Clone() *EngineProfileRegistry {
	if r == nil {
		return nil
	}

	ret := &EngineProfileRegistry{
		Slug:                     r.Slug,
		DisplayName:              r.DisplayName,
		Description:              r.Description,
		DefaultEngineProfileSlug: r.DefaultEngineProfileSlug,
		Profiles:                 make(map[EngineProfileSlug]*EngineProfile, len(r.Profiles)),
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

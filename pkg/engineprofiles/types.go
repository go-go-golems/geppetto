package engineprofiles

import aistepssettings "github.com/go-go-golems/geppetto/pkg/steps/ai/settings"

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

// EngineProfile is a named engine preset with inference settings and metadata.
type EngineProfile struct {
	Slug              EngineProfileSlug                  `json:"slug" yaml:"slug"`
	DisplayName       string                             `json:"display_name,omitempty" yaml:"display_name,omitempty"`
	Description       string                             `json:"description,omitempty" yaml:"description,omitempty"`
	Stack             []EngineProfileRef                 `json:"stack,omitempty" yaml:"stack,omitempty"`
	InferenceSettings *aistepssettings.InferenceSettings `json:"inference_settings,omitempty" yaml:"inference_settings,omitempty"`
	Metadata          EngineProfileMetadata              `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	Extensions        map[string]any                     `json:"extensions,omitempty" yaml:"extensions,omitempty"`
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
		Slug:              p.Slug,
		DisplayName:       p.DisplayName,
		Description:       p.Description,
		Stack:             append([]EngineProfileRef(nil), p.Stack...),
		InferenceSettings: cloneInferenceSettings(p.InferenceSettings),
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

package profiles

import (
	"fmt"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// DecodeRuntimeYAMLSingleRegistry decodes the runtime YAML source format.
// Runtime YAML is hard-cut to one-file-one-registry and rejects both bundle and legacy formats.
func DecodeRuntimeYAMLSingleRegistry(data []byte) (*ProfileRegistry, error) {
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" {
		return nil, nil
	}

	var raw map[string]any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	if len(raw) == 0 {
		return nil, nil
	}

	if _, hasBundle := raw["registries"]; hasBundle {
		return nil, &ValidationError{Field: "registry", Reason: "runtime YAML must define a single registry document (top-level registries is not supported)"}
	}
	if _, hasDefault := raw["default_profile_slug"]; hasDefault {
		return nil, &ValidationError{Field: "registry.default_profile_slug", Reason: "runtime YAML does not support default_profile_slug; use profile slug \"default\""}
	}
	if !looksLikeSingleRegistry(raw) {
		return nil, &ValidationError{Field: "registry", Reason: "runtime YAML must be a single registry document (legacy profile-map format is not supported)"}
	}

	reg, err := decodeSingleRegistry(raw)
	if err != nil {
		return nil, err
	}
	if reg.Slug.IsZero() {
		return nil, &ValidationError{Field: "registry.slug", Reason: "must be set for runtime YAML sources"}
	}
	if len(reg.Profiles) > 0 && reg.DefaultProfileSlug.IsZero() {
		if _, ok := reg.Profiles[MustProfileSlug("default")]; ok {
			reg.DefaultProfileSlug = MustProfileSlug("default")
		} else {
			slugs := make([]ProfileSlug, 0, len(reg.Profiles))
			for slug := range reg.Profiles {
				slugs = append(slugs, slug)
			}
			sort.Slice(slugs, func(i, j int) bool { return slugs[i] < slugs[j] })
			if len(slugs) > 0 {
				reg.DefaultProfileSlug = slugs[0]
			}
		}
	}
	if err := ValidateRegistry(reg); err != nil {
		return nil, fmt.Errorf("runtime YAML registry validation failed: %w", err)
	}
	return reg, nil
}

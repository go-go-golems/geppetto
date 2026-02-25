package profiles

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// DecodeYAMLRegistries is a compatibility wrapper around runtime YAML decoding.
// Hard-cut behavior: only one-file-one-registry runtime YAML is accepted.
func DecodeYAMLRegistries(data []byte, _ RegistrySlug) ([]*ProfileRegistry, error) {
	reg, err := DecodeRuntimeYAMLSingleRegistry(data)
	if err != nil {
		return nil, err
	}
	if reg == nil {
		return nil, nil
	}
	return []*ProfileRegistry{reg}, nil
}

func decodeSingleRegistry(raw map[string]any) (*ProfileRegistry, error) {
	b, err := yaml.Marshal(raw)
	if err != nil {
		return nil, err
	}
	reg := &ProfileRegistry{}
	if err := yaml.Unmarshal(b, reg); err != nil {
		return nil, err
	}
	if reg.Profiles == nil {
		reg.Profiles = map[ProfileSlug]*Profile{}
	}
	for slug, profile := range reg.Profiles {
		if profile == nil {
			continue
		}
		if profile.Slug.IsZero() {
			profile.Slug = slug
		}
	}
	return reg, nil
}

// EncodeYAMLRegistries is a compatibility wrapper around runtime YAML encoding.
// Hard-cut behavior: exactly one non-nil registry may be encoded.
func EncodeYAMLRegistries(registries []*ProfileRegistry) ([]byte, error) {
	nonNil := make([]*ProfileRegistry, 0, len(registries))
	for _, reg := range registries {
		if reg == nil {
			continue
		}
		nonNil = append(nonNil, reg)
	}
	switch len(nonNil) {
	case 0:
		return []byte{}, nil
	case 1:
		return EncodeRuntimeYAMLSingleRegistry(nonNil[0])
	default:
		return nil, fmt.Errorf("runtime YAML supports exactly one registry per file; got %d", len(nonNil))
	}
}

func looksLikeSingleRegistry(raw map[string]any) bool {
	_, hasProfiles := raw["profiles"]
	_, hasSlug := raw["slug"]
	_, hasDefault := raw["default_profile_slug"]
	return hasProfiles || hasSlug || hasDefault
}

func toStringAnyMap(v any) (map[string]any, bool) {
	m, ok := v.(map[string]any)
	if ok {
		return m, true
	}
	return nil, false
}

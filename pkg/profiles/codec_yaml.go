package profiles

import (
	"fmt"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// YAMLRegistryBundle is the canonical serialized format for profile registries.
type YAMLRegistryBundle struct {
	Registries map[string]*ProfileRegistry `yaml:"registries,omitempty"`
}

// DecodeYAMLRegistries decodes profile registries from YAML.
// Supported inputs:
// - canonical format with top-level "registries"
// - single-registry document with top-level registry fields (slug/profiles/...)
// - legacy pinocchio profiles map: profileSlug -> step settings patch map
func DecodeYAMLRegistries(data []byte, defaultRegistrySlug RegistrySlug) ([]*ProfileRegistry, error) {
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" {
		return nil, nil
	}
	if defaultRegistrySlug.IsZero() {
		defaultRegistrySlug = MustRegistrySlug("default")
	}

	var raw map[string]any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	if len(raw) == 0 {
		return nil, nil
	}

	if registriesRaw, ok := raw["registries"]; ok {
		return decodeCanonicalRegistries(registriesRaw)
	}

	if looksLikeSingleRegistry(raw) {
		reg, err := decodeSingleRegistry(raw)
		if err != nil {
			return nil, err
		}
		if err := ValidateRegistry(reg); err != nil {
			return nil, err
		}
		return []*ProfileRegistry{reg}, nil
	}

	reg, err := ConvertLegacyProfilesMapToRegistry(raw, defaultRegistrySlug)
	if err != nil {
		return nil, err
	}
	return []*ProfileRegistry{reg}, nil
}

func decodeCanonicalRegistries(raw any) ([]*ProfileRegistry, error) {
	registryMap, ok := toStringAnyMap(raw)
	if !ok {
		return nil, fmt.Errorf("registries must be a mapping")
	}

	slugs := make([]string, 0, len(registryMap))
	for slug := range registryMap {
		slugs = append(slugs, slug)
	}
	sort.Strings(slugs)

	out := make([]*ProfileRegistry, 0, len(slugs))
	for _, slug := range slugs {
		regRaw, ok := toStringAnyMap(registryMap[slug])
		if !ok {
			return nil, fmt.Errorf("registry %q must be a mapping", slug)
		}
		reg, err := decodeSingleRegistry(regRaw)
		if err != nil {
			return nil, err
		}
		if reg.Slug.IsZero() {
			parsed, err := ParseRegistrySlug(slug)
			if err != nil {
				return nil, err
			}
			reg.Slug = parsed
		}
		if err := ValidateRegistry(reg); err != nil {
			return nil, err
		}
		out = append(out, reg)
	}

	return out, nil
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

// ConvertLegacyProfilesMapToRegistry converts legacy profile maps
// (profileSlug -> step settings patch) to a typed registry.
func ConvertLegacyProfilesMapToRegistry(raw map[string]any, registrySlug RegistrySlug) (*ProfileRegistry, error) {
	if registrySlug.IsZero() {
		registrySlug = MustRegistrySlug("default")
	}

	slugs := make([]string, 0, len(raw))
	for k := range raw {
		slugs = append(slugs, k)
	}
	sort.Strings(slugs)

	profiles := map[ProfileSlug]*Profile{}
	for _, slugRaw := range slugs {
		profileSlug, err := ParseProfileSlug(slugRaw)
		if err != nil {
			return nil, err
		}
		patch, ok := toStringAnyMap(raw[slugRaw])
		if !ok {
			return nil, fmt.Errorf("legacy profile %q must map to section settings", slugRaw)
		}
		profiles[profileSlug] = &Profile{
			Slug: profileSlug,
			Runtime: RuntimeSpec{
				StepSettingsPatch: deepCopyStringAnyMap(patch),
			},
		}
	}

	defaultProfile := ProfileSlug("")
	if _, ok := profiles[MustProfileSlug("default")]; ok {
		defaultProfile = MustProfileSlug("default")
	} else if len(slugs) > 0 {
		defaultProfile = MustProfileSlug(slugs[0])
	}

	reg := &ProfileRegistry{
		Slug:               registrySlug,
		DefaultProfileSlug: defaultProfile,
		Profiles:           profiles,
	}
	if err := ValidateRegistry(reg); err != nil {
		return nil, err
	}
	return reg, nil
}

// EncodeYAMLRegistries encodes registries into the canonical YAML format.
func EncodeYAMLRegistries(registries []*ProfileRegistry) ([]byte, error) {
	bundle := YAMLRegistryBundle{Registries: map[string]*ProfileRegistry{}}
	for _, reg := range registries {
		if reg == nil {
			continue
		}
		if err := ValidateRegistry(reg); err != nil {
			return nil, err
		}
		bundle.Registries[reg.Slug.String()] = reg.Clone()
	}
	return yaml.Marshal(bundle)
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

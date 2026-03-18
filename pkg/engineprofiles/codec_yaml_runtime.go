package engineprofiles

import (
	"fmt"
	"sort"
	"strings"

	aistepssettings "github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"gopkg.in/yaml.v3"
)

// DecodeEngineProfileYAMLSingleRegistry decodes the engine profile YAML source format.
// Engine profile YAML is hard-cut to one-file-one-registry and rejects both bundle and legacy formats.
func DecodeEngineProfileYAMLSingleRegistry(data []byte) (*EngineProfileRegistry, error) {
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
		return nil, &ValidationError{Field: "registry", Reason: "engine profile YAML must define a single registry document (top-level registries is not supported)"}
	}
	if _, hasDefault := raw["default_profile_slug"]; hasDefault {
		return nil, &ValidationError{Field: "registry.default_profile_slug", Reason: "engine profile YAML does not support default_profile_slug; use profile slug \"default\""}
	}
	if !looksLikeSingleRegistry(raw) {
		return nil, &ValidationError{Field: "registry", Reason: "engine profile YAML must be a single registry document (legacy profile-map format is not supported)"}
	}

	reg, err := decodeSingleRegistry(raw)
	if err != nil {
		return nil, err
	}
	if reg.Slug.IsZero() {
		return nil, &ValidationError{Field: "registry.slug", Reason: "must be set for engine profile YAML sources"}
	}
	if len(reg.Profiles) > 0 && reg.DefaultEngineProfileSlug.IsZero() {
		if _, ok := reg.Profiles[MustEngineProfileSlug("default")]; ok {
			reg.DefaultEngineProfileSlug = MustEngineProfileSlug("default")
		} else {
			slugs := make([]EngineProfileSlug, 0, len(reg.Profiles))
			for slug := range reg.Profiles {
				slugs = append(slugs, slug)
			}
			sort.Slice(slugs, func(i, j int) bool { return slugs[i] < slugs[j] })
			if len(slugs) > 0 {
				reg.DefaultEngineProfileSlug = slugs[0]
			}
		}
	}
	if err := ValidateRegistry(reg); err != nil {
		return nil, fmt.Errorf("engine profile YAML registry validation failed: %w", err)
	}
	return reg, nil
}

// EncodeEngineProfileYAMLSingleRegistry encodes a single registry as engine profile YAML.
// Hard-cut behavior: default_profile_slug is omitted from the serialized document.
func EncodeEngineProfileYAMLSingleRegistry(registry *EngineProfileRegistry) ([]byte, error) {
	if registry == nil {
		return nil, fmt.Errorf("engine profile registry is nil")
	}
	clone := registry.Clone()
	if clone == nil {
		return nil, fmt.Errorf("engine profile registry clone is nil")
	}
	if err := ValidateRegistry(clone); err != nil {
		return nil, fmt.Errorf("engine profile YAML registry validation failed: %w", err)
	}
	for profileSlug, profile := range clone.Profiles {
		if profile == nil {
			continue
		}
		if profile.Slug.IsZero() {
			profile.Slug = profileSlug
		}
		normalizeInferenceSettingsForYAML(profile.InferenceSettings)
	}
	type engineProfileRegistryYAML struct {
		Slug        RegistrySlug                         `yaml:"slug"`
		DisplayName string                               `yaml:"display_name,omitempty"`
		Description string                               `yaml:"description,omitempty"`
		Profiles    map[EngineProfileSlug]*EngineProfile `yaml:"profiles,omitempty"`
		Metadata    RegistryMetadata                     `yaml:"metadata,omitempty"`
	}
	out := engineProfileRegistryYAML{
		Slug:        clone.Slug,
		DisplayName: clone.DisplayName,
		Description: clone.Description,
		Profiles:    clone.Profiles,
		Metadata:    clone.Metadata,
	}
	return yaml.Marshal(out)
}

func normalizeInferenceSettingsForYAML(in *aistepssettings.InferenceSettings) {
	if in == nil || in.Client == nil {
		return
	}
	if in.Client.TimeoutSeconds != nil {
		in.Client.Timeout = nil
	}
}

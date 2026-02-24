package profiles

import (
	"encoding/json"
	"fmt"
	"strings"
)

func ValidateRegistrySlug(slug RegistrySlug) error {
	if slug.IsZero() {
		return &ValidationError{Field: "registry.slug", Reason: "must not be empty"}
	}
	if _, err := ParseRegistrySlug(slug.String()); err != nil {
		return &ValidationError{Field: "registry.slug", Reason: err.Error()}
	}
	return nil
}

func ValidateProfileSlug(slug ProfileSlug) error {
	if slug.IsZero() {
		return &ValidationError{Field: "profile.slug", Reason: "must not be empty"}
	}
	if _, err := ParseProfileSlug(slug.String()); err != nil {
		return &ValidationError{Field: "profile.slug", Reason: err.Error()}
	}
	return nil
}

func ValidateRuntimeSpec(spec RuntimeSpec) error {
	for i, mw := range spec.Middlewares {
		if strings.TrimSpace(mw.Name) == "" {
			return &ValidationError{Field: fmt.Sprintf("runtime.middlewares[%d].name", i), Reason: "must not be empty"}
		}
	}

	for i, tool := range spec.Tools {
		if strings.TrimSpace(tool) == "" {
			return &ValidationError{Field: fmt.Sprintf("runtime.tools[%d]", i), Reason: "must not be empty"}
		}
	}

	return nil
}

func ValidatePolicySpec(policy PolicySpec) error {
	allow := map[string]struct{}{}
	for i, key := range policy.AllowedOverrideKeys {
		normalized := strings.TrimSpace(key)
		if normalized == "" {
			return &ValidationError{Field: fmt.Sprintf("policy.allowed_override_keys[%d]", i), Reason: "must not be empty"}
		}
		allow[normalized] = struct{}{}
	}

	deny := map[string]struct{}{}
	for i, key := range policy.DeniedOverrideKeys {
		normalized := strings.TrimSpace(key)
		if normalized == "" {
			return &ValidationError{Field: fmt.Sprintf("policy.denied_override_keys[%d]", i), Reason: "must not be empty"}
		}
		if _, ok := allow[normalized]; ok {
			return &ValidationError{Field: "policy.override_keys", Reason: fmt.Sprintf("key %q appears in both allow and deny lists", normalized)}
		}
		deny[normalized] = struct{}{}
	}

	_ = deny
	return nil
}

func ValidateProfile(profile *Profile) error {
	if profile == nil {
		return &ValidationError{Field: "profile", Reason: "must not be nil"}
	}
	if err := ValidateProfileSlug(profile.Slug); err != nil {
		return err
	}
	if err := ValidateRuntimeSpec(profile.Runtime); err != nil {
		return err
	}
	if err := ValidatePolicySpec(profile.Policy); err != nil {
		return err
	}
	if err := ValidateProfileExtensions(profile.Extensions); err != nil {
		return err
	}
	return nil
}

func ValidateProfileExtensions(extensions map[string]any) error {
	for rawKey, rawValue := range extensions {
		key, err := ParseExtensionKey(rawKey)
		if err != nil {
			return &ValidationError{
				Field:  fmt.Sprintf("profile.extensions[%s]", strings.TrimSpace(rawKey)),
				Reason: err.Error(),
			}
		}
		if _, err := json.Marshal(rawValue); err != nil {
			return &ValidationError{
				Field:  fmt.Sprintf("profile.extensions[%s]", key),
				Reason: fmt.Sprintf("payload must be JSON-serializable: %v", err),
			}
		}
	}
	return nil
}

func ValidateRegistry(registry *ProfileRegistry) error {
	if registry == nil {
		return &ValidationError{Field: "registry", Reason: "must not be nil"}
	}
	if err := ValidateRegistrySlug(registry.Slug); err != nil {
		return err
	}

	if len(registry.Profiles) > 0 && registry.DefaultProfileSlug.IsZero() {
		return &ValidationError{Field: "registry.default_profile_slug", Reason: "must be set when profiles are present"}
	}

	for slug, profile := range registry.Profiles {
		if err := ValidateProfileSlug(slug); err != nil {
			return err
		}
		if profile == nil {
			return &ValidationError{Field: fmt.Sprintf("registry.profiles[%s]", slug), Reason: "must not be nil"}
		}
		if err := ValidateProfile(profile); err != nil {
			return err
		}
		if profile.Slug != slug {
			return &ValidationError{Field: fmt.Sprintf("registry.profiles[%s].slug", slug), Reason: "map key and profile slug must match"}
		}
	}

	if !registry.DefaultProfileSlug.IsZero() {
		if err := ValidateProfileSlug(registry.DefaultProfileSlug); err != nil {
			return err
		}
		if len(registry.Profiles) > 0 {
			if _, ok := registry.Profiles[registry.DefaultProfileSlug]; !ok {
				return &ValidationError{Field: "registry.default_profile_slug", Reason: "default profile does not exist in registry"}
			}
		}
	}

	return nil
}

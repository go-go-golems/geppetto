package profiles

import (
	"errors"
	"testing"
)

func TestValidateProfile_OK(t *testing.T) {
	p := &Profile{
		Slug: MustProfileSlug("default"),
		Runtime: RuntimeSpec{
			Middlewares: []MiddlewareUse{{Name: "agentmode"}},
			Tools:       []string{"calculator"},
		},
		Policy: PolicySpec{
			AllowOverrides:      true,
			AllowedOverrideKeys: []string{"system_prompt"},
		},
	}
	if err := ValidateProfile(p); err != nil {
		t.Fatalf("ValidateProfile failed: %v", err)
	}
}

func TestValidatePolicySpec_OverlapFails(t *testing.T) {
	err := ValidatePolicySpec(PolicySpec{
		AllowedOverrideKeys: []string{"system_prompt"},
		DeniedOverrideKeys:  []string{"system_prompt"},
	})
	if err == nil {
		t.Fatalf("expected overlap error")
	}
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("expected validation error, got: %v", err)
	}
}

func TestValidateRegistry_DefaultMustExist(t *testing.T) {
	r := &ProfileRegistry{
		Slug:               MustRegistrySlug("default"),
		DefaultProfileSlug: MustProfileSlug("agent"),
		Profiles: map[ProfileSlug]*Profile{
			MustProfileSlug("default"): {
				Slug: MustProfileSlug("default"),
			},
		},
	}
	err := ValidateRegistry(r)
	if err == nil {
		t.Fatalf("expected default missing error")
	}
	requireValidationField(t, err, "registry.default_profile_slug")
}

func TestValidateRegistry_ProfileMapKeyMustMatch(t *testing.T) {
	r := &ProfileRegistry{
		Slug:               MustRegistrySlug("default"),
		DefaultProfileSlug: MustProfileSlug("default"),
		Profiles: map[ProfileSlug]*Profile{
			MustProfileSlug("default"): {
				Slug: MustProfileSlug("agent"),
			},
		},
	}
	err := ValidateRegistry(r)
	if err == nil {
		t.Fatalf("expected key mismatch error")
	}
	requireValidationField(t, err, "registry.profiles[default].slug")
}

func TestValidateRegistry_EmptySlugFieldPath(t *testing.T) {
	err := ValidateRegistry(&ProfileRegistry{})
	requireValidationField(t, err, "registry.slug")
}

func TestValidateProfile_EmptySlugFieldPath(t *testing.T) {
	err := ValidateProfile(&Profile{})
	requireValidationField(t, err, "profile.slug")
}

func TestValidateRegistry_DefaultProfileRequiredWhenProfilesPresent(t *testing.T) {
	err := ValidateRegistry(&ProfileRegistry{
		Slug: MustRegistrySlug("default"),
		Profiles: map[ProfileSlug]*Profile{
			MustProfileSlug("default"): {Slug: MustProfileSlug("default")},
		},
	})
	requireValidationField(t, err, "registry.default_profile_slug")
}

func TestValidateRegistry_NilProfileEntry(t *testing.T) {
	err := ValidateRegistry(&ProfileRegistry{
		Slug:               MustRegistrySlug("default"),
		DefaultProfileSlug: MustProfileSlug("default"),
		Profiles: map[ProfileSlug]*Profile{
			MustProfileSlug("default"): nil,
		},
	})
	requireValidationField(t, err, "registry.profiles[default]")
}

func TestValidateRuntimeSpec_RejectsWhitespaceMiddlewareName(t *testing.T) {
	err := ValidateRuntimeSpec(RuntimeSpec{
		Middlewares: []MiddlewareUse{{Name: "  "}},
	})
	requireValidationField(t, err, "runtime.middlewares[0].name")
}

func TestValidateRuntimeSpec_RejectsWhitespaceToolName(t *testing.T) {
	err := ValidateRuntimeSpec(RuntimeSpec{
		Tools: []string{"  "},
	})
	requireValidationField(t, err, "runtime.tools[0]")
}

func TestValidateProfile_RejectsInvalidExtensionKeyFieldPath(t *testing.T) {
	err := ValidateProfile(&Profile{
		Slug: MustProfileSlug("default"),
		Extensions: map[string]any{
			"bad key": map[string]any{"foo": "bar"},
		},
	})
	requireValidationField(t, err, "profile.extensions[bad key]")
}

func TestValidateProfile_RejectsNonSerializableExtensionPayload(t *testing.T) {
	err := ValidateProfile(&Profile{
		Slug: MustProfileSlug("default"),
		Extensions: map[string]any{
			"webchat.starter_suggestions@v1": func() {},
		},
	})
	requireValidationField(t, err, "profile.extensions[webchat.starter_suggestions@v1]")
}

func requireValidationField(t *testing.T, err error, field string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected validation error")
	}
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("expected ErrValidation, got %v", err)
	}

	var verr *ValidationError
	if !errors.As(err, &verr) {
		t.Fatalf("expected ValidationError type, got %T", err)
	}
	if verr.Field != field {
		t.Fatalf("validation field mismatch: got=%q want=%q", verr.Field, field)
	}
}

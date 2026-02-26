package profiles

import (
	"errors"
	"strings"
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

func TestValidateRuntimeSpec_RejectsWhitespaceMiddlewareID(t *testing.T) {
	err := ValidateRuntimeSpec(RuntimeSpec{
		Middlewares: []MiddlewareUse{{Name: "agentmode", ID: "  "}},
	})
	requireValidationField(t, err, "runtime.middlewares[0].id")
}

func TestValidateRuntimeSpec_RejectsDuplicateMiddlewareID(t *testing.T) {
	err := ValidateRuntimeSpec(RuntimeSpec{
		Middlewares: []MiddlewareUse{
			{Name: "agentmode", ID: "agent"},
			{Name: "sqlite", ID: "agent"},
		},
	})
	requireValidationField(t, err, "runtime.middlewares[1].id")
}

func TestValidateRuntimeSpec_AllowsRepeatedNamesWithUniqueIDs(t *testing.T) {
	err := ValidateRuntimeSpec(RuntimeSpec{
		Middlewares: []MiddlewareUse{
			{Name: "agentmode", ID: "agent-primary"},
			{Name: "agentmode", ID: "agent-secondary"},
		},
	})
	if err != nil {
		t.Fatalf("expected repeated middleware names with unique IDs to be valid, got %v", err)
	}
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

func TestValidateProfile_RejectsEmptyStackProfileSlug(t *testing.T) {
	err := ValidateProfile(&Profile{
		Slug:  MustProfileSlug("default"),
		Stack: []ProfileRef{{}},
	})
	requireValidationField(t, err, "profile.stack[0].profile_slug")
}

func TestValidateProfile_RejectsInvalidStackRegistrySlug(t *testing.T) {
	err := ValidateProfile(&Profile{
		Slug: MustProfileSlug("default"),
		Stack: []ProfileRef{
			{
				RegistrySlug: RegistrySlug("Invalid Registry"),
				ProfileSlug:  MustProfileSlug("base"),
			},
		},
	})
	requireValidationField(t, err, "profile.stack[0].registry_slug")
}

func TestValidateRegistry_RejectsMissingSameRegistryStackRef(t *testing.T) {
	err := ValidateRegistry(&ProfileRegistry{
		Slug:               MustRegistrySlug("default"),
		DefaultProfileSlug: MustProfileSlug("agent"),
		Profiles: map[ProfileSlug]*Profile{
			MustProfileSlug("agent"): {
				Slug: MustProfileSlug("agent"),
				Stack: []ProfileRef{
					{ProfileSlug: MustProfileSlug("provider-openai")},
				},
			},
		},
	})
	requireValidationField(t, err, "registry.profiles[agent].stack[0]")
}

func TestValidateRegistry_RejectsStackCycle(t *testing.T) {
	err := ValidateRegistry(&ProfileRegistry{
		Slug:               MustRegistrySlug("default"),
		DefaultProfileSlug: MustProfileSlug("a"),
		Profiles: map[ProfileSlug]*Profile{
			MustProfileSlug("a"): {
				Slug: MustProfileSlug("a"),
				Stack: []ProfileRef{
					{ProfileSlug: MustProfileSlug("b")},
				},
			},
			MustProfileSlug("b"): {
				Slug: MustProfileSlug("b"),
				Stack: []ProfileRef{
					{ProfileSlug: MustProfileSlug("a")},
				},
			},
		},
	})
	requireValidationField(t, err, "registry.profiles[b].stack[0]")
	if !strings.Contains(err.Error(), "cycle") {
		t.Fatalf("expected cycle details in error, got %v", err)
	}
}

func TestValidateProfileStackTopology_RejectsDepthOverflow(t *testing.T) {
	reg := &ProfileRegistry{
		Slug:               MustRegistrySlug("default"),
		DefaultProfileSlug: MustProfileSlug("a"),
		Profiles: map[ProfileSlug]*Profile{
			MustProfileSlug("a"): {
				Slug:  MustProfileSlug("a"),
				Stack: []ProfileRef{{ProfileSlug: MustProfileSlug("b")}},
			},
			MustProfileSlug("b"): {
				Slug:  MustProfileSlug("b"),
				Stack: []ProfileRef{{ProfileSlug: MustProfileSlug("c")}},
			},
			MustProfileSlug("c"): {
				Slug: MustProfileSlug("c"),
			},
		},
	}

	err := ValidateProfileStackTopology([]*ProfileRegistry{reg}, StackValidationOptions{MaxDepth: 2})
	requireValidationField(t, err, "registry.profiles[b].stack[0]")
	if !strings.Contains(err.Error(), "max_depth=2") {
		t.Fatalf("expected max_depth details in error, got %v", err)
	}
}

func TestValidateProfileStackTopology_RejectsMissingCrossRegistryRef(t *testing.T) {
	reg := &ProfileRegistry{
		Slug:               MustRegistrySlug("default"),
		DefaultProfileSlug: MustProfileSlug("agent"),
		Profiles: map[ProfileSlug]*Profile{
			MustProfileSlug("agent"): {
				Slug: MustProfileSlug("agent"),
				Stack: []ProfileRef{
					{
						RegistrySlug: MustRegistrySlug("shared"),
						ProfileSlug:  MustProfileSlug("provider-openai"),
					},
				},
			},
		},
	}

	err := ValidateProfileStackTopology([]*ProfileRegistry{reg}, StackValidationOptions{})
	requireValidationField(t, err, "registry.profiles[agent].stack[0]")
}

func TestValidateProfileStackTopology_AllowsUnresolvedCrossRegistryRefWhenConfigured(t *testing.T) {
	reg := &ProfileRegistry{
		Slug:               MustRegistrySlug("default"),
		DefaultProfileSlug: MustProfileSlug("agent"),
		Profiles: map[ProfileSlug]*Profile{
			MustProfileSlug("agent"): {
				Slug: MustProfileSlug("agent"),
				Stack: []ProfileRef{
					{
						RegistrySlug: MustRegistrySlug("shared"),
						ProfileSlug:  MustProfileSlug("provider-openai"),
					},
				},
			},
		},
	}

	err := ValidateProfileStackTopology([]*ProfileRegistry{reg}, StackValidationOptions{
		AllowUnresolvedExternalRefs: true,
	})
	if err != nil {
		t.Fatalf("expected unresolved external ref to be allowed, got %v", err)
	}
}

func TestValidateProfileStackTopology_DoesNotBypassMissingRefInKnownRegistry(t *testing.T) {
	reg := &ProfileRegistry{
		Slug:               MustRegistrySlug("default"),
		DefaultProfileSlug: MustProfileSlug("agent"),
		Profiles: map[ProfileSlug]*Profile{
			MustProfileSlug("agent"): {
				Slug: MustProfileSlug("agent"),
				Stack: []ProfileRef{
					{
						RegistrySlug: MustRegistrySlug("default"),
						ProfileSlug:  MustProfileSlug("missing-local"),
					},
				},
			},
		},
	}

	err := ValidateProfileStackTopology([]*ProfileRegistry{reg}, StackValidationOptions{
		AllowUnresolvedExternalRefs: true,
	})
	requireValidationField(t, err, "registry.profiles[agent].stack[0]")
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

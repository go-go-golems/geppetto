package engineprofiles

import (
	"errors"
	"strings"
	"testing"
)

func TestValidateProfile_OK(t *testing.T) {
	p := &EngineProfile{
		Slug: MustEngineProfileSlug("default"),
		Runtime: RuntimeSpec{
			Middlewares: []MiddlewareUse{{Name: "agentmode"}},
			Tools:       []string{"calculator"},
		},
	}
	if err := ValidateEngineProfile(p); err != nil {
		t.Fatalf("ValidateEngineProfile failed: %v", err)
	}
}

func TestValidateRegistry_DefaultMustExist(t *testing.T) {
	r := &EngineProfileRegistry{
		Slug:                     MustRegistrySlug("default"),
		DefaultEngineProfileSlug: MustEngineProfileSlug("agent"),
		Profiles: map[EngineProfileSlug]*EngineProfile{
			MustEngineProfileSlug("default"): {
				Slug: MustEngineProfileSlug("default"),
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
	r := &EngineProfileRegistry{
		Slug:                     MustRegistrySlug("default"),
		DefaultEngineProfileSlug: MustEngineProfileSlug("default"),
		Profiles: map[EngineProfileSlug]*EngineProfile{
			MustEngineProfileSlug("default"): {
				Slug: MustEngineProfileSlug("agent"),
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
	err := ValidateRegistry(&EngineProfileRegistry{})
	requireValidationField(t, err, "registry.slug")
}

func TestValidateProfile_EmptySlugFieldPath(t *testing.T) {
	err := ValidateEngineProfile(&EngineProfile{})
	requireValidationField(t, err, "profile.slug")
}

func TestValidateRegistry_DefaultProfileRequiredWhenProfilesPresent(t *testing.T) {
	err := ValidateRegistry(&EngineProfileRegistry{
		Slug: MustRegistrySlug("default"),
		Profiles: map[EngineProfileSlug]*EngineProfile{
			MustEngineProfileSlug("default"): {Slug: MustEngineProfileSlug("default")},
		},
	})
	requireValidationField(t, err, "registry.default_profile_slug")
}

func TestValidateRegistry_NilProfileEntry(t *testing.T) {
	err := ValidateRegistry(&EngineProfileRegistry{
		Slug:                     MustRegistrySlug("default"),
		DefaultEngineProfileSlug: MustEngineProfileSlug("default"),
		Profiles: map[EngineProfileSlug]*EngineProfile{
			MustEngineProfileSlug("default"): nil,
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
	err := ValidateEngineProfile(&EngineProfile{
		Slug: MustEngineProfileSlug("default"),
		Extensions: map[string]any{
			"bad key": map[string]any{"foo": "bar"},
		},
	})
	requireValidationField(t, err, "profile.extensions[bad key]")
}

func TestValidateProfile_RejectsNonSerializableExtensionPayload(t *testing.T) {
	err := ValidateEngineProfile(&EngineProfile{
		Slug: MustEngineProfileSlug("default"),
		Extensions: map[string]any{
			"webchat.starter_suggestions@v1": func() {},
		},
	})
	requireValidationField(t, err, "profile.extensions[webchat.starter_suggestions@v1]")
}

func TestValidateProfile_RejectsEmptyStackEngineProfileSlug(t *testing.T) {
	err := ValidateEngineProfile(&EngineProfile{
		Slug:  MustEngineProfileSlug("default"),
		Stack: []EngineProfileRef{{}},
	})
	requireValidationField(t, err, "profile.stack[0].profile_slug")
}

func TestValidateProfile_RejectsInvalidStackRegistrySlug(t *testing.T) {
	err := ValidateEngineProfile(&EngineProfile{
		Slug: MustEngineProfileSlug("default"),
		Stack: []EngineProfileRef{
			{
				RegistrySlug:      RegistrySlug("Invalid Registry"),
				EngineProfileSlug: MustEngineProfileSlug("base"),
			},
		},
	})
	requireValidationField(t, err, "profile.stack[0].registry_slug")
}

func TestValidateRegistry_RejectsMissingSameRegistryStackRef(t *testing.T) {
	err := ValidateRegistry(&EngineProfileRegistry{
		Slug:                     MustRegistrySlug("default"),
		DefaultEngineProfileSlug: MustEngineProfileSlug("agent"),
		Profiles: map[EngineProfileSlug]*EngineProfile{
			MustEngineProfileSlug("agent"): {
				Slug: MustEngineProfileSlug("agent"),
				Stack: []EngineProfileRef{
					{EngineProfileSlug: MustEngineProfileSlug("provider-openai")},
				},
			},
		},
	})
	requireValidationField(t, err, "registry.profiles[agent].stack[0]")
}

func TestValidateRegistry_RejectsStackCycle(t *testing.T) {
	err := ValidateRegistry(&EngineProfileRegistry{
		Slug:                     MustRegistrySlug("default"),
		DefaultEngineProfileSlug: MustEngineProfileSlug("a"),
		Profiles: map[EngineProfileSlug]*EngineProfile{
			MustEngineProfileSlug("a"): {
				Slug: MustEngineProfileSlug("a"),
				Stack: []EngineProfileRef{
					{EngineProfileSlug: MustEngineProfileSlug("b")},
				},
			},
			MustEngineProfileSlug("b"): {
				Slug: MustEngineProfileSlug("b"),
				Stack: []EngineProfileRef{
					{EngineProfileSlug: MustEngineProfileSlug("a")},
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
	reg := &EngineProfileRegistry{
		Slug:                     MustRegistrySlug("default"),
		DefaultEngineProfileSlug: MustEngineProfileSlug("a"),
		Profiles: map[EngineProfileSlug]*EngineProfile{
			MustEngineProfileSlug("a"): {
				Slug:  MustEngineProfileSlug("a"),
				Stack: []EngineProfileRef{{EngineProfileSlug: MustEngineProfileSlug("b")}},
			},
			MustEngineProfileSlug("b"): {
				Slug:  MustEngineProfileSlug("b"),
				Stack: []EngineProfileRef{{EngineProfileSlug: MustEngineProfileSlug("c")}},
			},
			MustEngineProfileSlug("c"): {
				Slug: MustEngineProfileSlug("c"),
			},
		},
	}

	err := ValidateProfileStackTopology([]*EngineProfileRegistry{reg}, StackValidationOptions{MaxDepth: 2})
	requireValidationField(t, err, "registry.profiles[b].stack[0]")
	if !strings.Contains(err.Error(), "max_depth=2") {
		t.Fatalf("expected max_depth details in error, got %v", err)
	}
}

func TestValidateProfileStackTopology_RejectsMissingCrossRegistryRef(t *testing.T) {
	reg := &EngineProfileRegistry{
		Slug:                     MustRegistrySlug("default"),
		DefaultEngineProfileSlug: MustEngineProfileSlug("agent"),
		Profiles: map[EngineProfileSlug]*EngineProfile{
			MustEngineProfileSlug("agent"): {
				Slug: MustEngineProfileSlug("agent"),
				Stack: []EngineProfileRef{
					{
						RegistrySlug:      MustRegistrySlug("shared"),
						EngineProfileSlug: MustEngineProfileSlug("provider-openai"),
					},
				},
			},
		},
	}

	err := ValidateProfileStackTopology([]*EngineProfileRegistry{reg}, StackValidationOptions{})
	requireValidationField(t, err, "registry.profiles[agent].stack[0]")
}

func TestValidateProfileStackTopology_AllowsUnresolvedCrossRegistryRefWhenConfigured(t *testing.T) {
	reg := &EngineProfileRegistry{
		Slug:                     MustRegistrySlug("default"),
		DefaultEngineProfileSlug: MustEngineProfileSlug("agent"),
		Profiles: map[EngineProfileSlug]*EngineProfile{
			MustEngineProfileSlug("agent"): {
				Slug: MustEngineProfileSlug("agent"),
				Stack: []EngineProfileRef{
					{
						RegistrySlug:      MustRegistrySlug("shared"),
						EngineProfileSlug: MustEngineProfileSlug("provider-openai"),
					},
				},
			},
		},
	}

	err := ValidateProfileStackTopology([]*EngineProfileRegistry{reg}, StackValidationOptions{
		AllowUnresolvedExternalRefs: true,
	})
	if err != nil {
		t.Fatalf("expected unresolved external ref to be allowed, got %v", err)
	}
}

func TestValidateProfileStackTopology_DoesNotBypassMissingRefInKnownRegistry(t *testing.T) {
	reg := &EngineProfileRegistry{
		Slug:                     MustRegistrySlug("default"),
		DefaultEngineProfileSlug: MustEngineProfileSlug("agent"),
		Profiles: map[EngineProfileSlug]*EngineProfile{
			MustEngineProfileSlug("agent"): {
				Slug: MustEngineProfileSlug("agent"),
				Stack: []EngineProfileRef{
					{
						RegistrySlug:      MustRegistrySlug("default"),
						EngineProfileSlug: MustEngineProfileSlug("missing-local"),
					},
				},
			},
		},
	}

	err := ValidateProfileStackTopology([]*EngineProfileRegistry{reg}, StackValidationOptions{
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

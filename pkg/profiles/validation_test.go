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
	if err := ValidateRegistry(r); err == nil {
		t.Fatalf("expected default missing error")
	}
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
	if err := ValidateRegistry(r); err == nil {
		t.Fatalf("expected key mismatch error")
	}
}

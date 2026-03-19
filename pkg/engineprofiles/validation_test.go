package engineprofiles

import "testing"

func TestValidateEngineProfile_RejectsEmptySlug(t *testing.T) {
	err := ValidateEngineProfile(&EngineProfile{})
	requireValidationField(t, err, "profile.slug")
}

func TestValidateEngineProfile_RejectsInvalidExtensionKey(t *testing.T) {
	err := ValidateEngineProfile(&EngineProfile{
		Slug: MustEngineProfileSlug("assistant"),
		Extensions: map[string]any{
			"bad key": true,
		},
	})
	requireValidationField(t, err, "profile.extensions[bad key]")
}

func TestValidateRegistry_RejectsMissingDefaultWhenProfilesPresent(t *testing.T) {
	err := ValidateRegistry(&EngineProfileRegistry{
		Slug: MustRegistrySlug("default"),
		Profiles: map[EngineProfileSlug]*EngineProfile{
			MustEngineProfileSlug("assistant"): {
				Slug: MustEngineProfileSlug("assistant"),
			},
		},
	})
	requireValidationField(t, err, "registry.default_profile_slug")
}

func requireValidationField(t *testing.T, err error, field string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected validation error for %s", field)
	}
	verr, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("expected ValidationError, got %T (%v)", err, err)
	}
	if verr.Field != field {
		t.Fatalf("validation field mismatch: got=%q want=%q", verr.Field, field)
	}
}

package profiles

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
)

func TestSQLiteProfileStore_RegistryRoundTrip(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "profiles.db")
	dsn, err := SQLiteProfileDSNForFile(dbPath)
	if err != nil {
		t.Fatalf("SQLiteProfileDSNForFile returned error: %v", err)
	}

	store, err := NewSQLiteProfileStore(dsn, MustRegistrySlug("default"))
	if err != nil {
		t.Fatalf("NewSQLiteProfileStore returned error: %v", err)
	}

	registry := &ProfileRegistry{
		Slug:               MustRegistrySlug("default"),
		DefaultProfileSlug: MustProfileSlug("default"),
		Profiles: map[ProfileSlug]*Profile{
			MustProfileSlug("default"): {
				Slug:        MustProfileSlug("default"),
				DisplayName: "Default",
				Runtime: RuntimeSpec{
					SystemPrompt: "You are default",
					Tools:        []string{"calculator"},
				},
				Policy: PolicySpec{AllowOverrides: true},
			},
		},
	}
	if err := store.UpsertRegistry(ctx, registry, SaveOptions{Actor: "test", Source: "sqlite"}); err != nil {
		t.Fatalf("UpsertRegistry returned error: %v", err)
	}

	got, ok, err := store.GetRegistry(ctx, MustRegistrySlug("default"))
	if err != nil {
		t.Fatalf("GetRegistry returned error: %v", err)
	}
	if !ok || got == nil {
		t.Fatalf("GetRegistry expected registry")
	}
	if got.Metadata.Version == 0 {
		t.Fatalf("expected registry metadata version > 0")
	}
	if got.DefaultProfileSlug != MustProfileSlug("default") {
		t.Fatalf("unexpected default profile slug: %s", got.DefaultProfileSlug)
	}
	if _, ok := got.Profiles[MustProfileSlug("default")]; !ok {
		t.Fatalf("expected default profile in registry")
	}

	if err := store.Close(); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}

	reopened, err := NewSQLiteProfileStore(dsn, MustRegistrySlug("default"))
	if err != nil {
		t.Fatalf("reopen NewSQLiteProfileStore returned error: %v", err)
	}
	defer func() { _ = reopened.Close() }()

	reloaded, ok, err := reopened.GetRegistry(ctx, MustRegistrySlug("default"))
	if err != nil {
		t.Fatalf("reloaded GetRegistry returned error: %v", err)
	}
	if !ok || reloaded == nil {
		t.Fatalf("reloaded registry expected to exist")
	}
	if reloaded.DefaultProfileSlug != MustProfileSlug("default") {
		t.Fatalf("reloaded default profile slug mismatch: %s", reloaded.DefaultProfileSlug)
	}
	if reloaded.Profiles[MustProfileSlug("default")] == nil {
		t.Fatalf("reloaded default profile missing")
	}
}

func TestSQLiteProfileStore_ProfileLifecycleAndVersionConflicts(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "profiles.db")
	dsn, err := SQLiteProfileDSNForFile(dbPath)
	if err != nil {
		t.Fatalf("SQLiteProfileDSNForFile returned error: %v", err)
	}

	store, err := NewSQLiteProfileStore(dsn, MustRegistrySlug("default"))
	if err != nil {
		t.Fatalf("NewSQLiteProfileStore returned error: %v", err)
	}
	defer func() { _ = store.Close() }()

	registry := &ProfileRegistry{
		Slug:               MustRegistrySlug("default"),
		DefaultProfileSlug: MustProfileSlug("default"),
		Profiles: map[ProfileSlug]*Profile{
			MustProfileSlug("default"): {
				Slug:    MustProfileSlug("default"),
				Runtime: RuntimeSpec{SystemPrompt: "You are default"},
			},
		},
	}
	if err := store.UpsertRegistry(ctx, registry, SaveOptions{Actor: "test", Source: "sqlite"}); err != nil {
		t.Fatalf("UpsertRegistry returned error: %v", err)
	}

	if err := store.UpsertProfile(ctx, MustRegistrySlug("default"), &Profile{
		Slug:        MustProfileSlug("analyst"),
		DisplayName: "Analyst",
		Runtime: RuntimeSpec{
			SystemPrompt: "You are analyst",
		},
		Policy: PolicySpec{AllowOverrides: true},
	}, SaveOptions{Actor: "test", Source: "sqlite"}); err != nil {
		t.Fatalf("UpsertProfile create returned error: %v", err)
	}

	analyst, ok, err := store.GetProfile(ctx, MustRegistrySlug("default"), MustProfileSlug("analyst"))
	if err != nil {
		t.Fatalf("GetProfile returned error: %v", err)
	}
	if !ok || analyst == nil {
		t.Fatalf("expected analyst profile")
	}
	if analyst.Metadata.Version != 1 {
		t.Fatalf("expected analyst version=1, got=%d", analyst.Metadata.Version)
	}

	analystUpdate := analyst.Clone()
	analystUpdate.DisplayName = "Analyst v2"
	if err := store.UpsertProfile(ctx, MustRegistrySlug("default"), analystUpdate, SaveOptions{
		ExpectedVersion: analyst.Metadata.Version,
		Actor:           "test",
		Source:          "sqlite",
	}); err != nil {
		t.Fatalf("UpsertProfile update returned error: %v", err)
	}

	analystV2, ok, err := store.GetProfile(ctx, MustRegistrySlug("default"), MustProfileSlug("analyst"))
	if err != nil {
		t.Fatalf("GetProfile after update returned error: %v", err)
	}
	if !ok || analystV2 == nil {
		t.Fatalf("expected analyst profile after update")
	}
	if analystV2.Metadata.Version != 2 {
		t.Fatalf("expected analyst version=2, got=%d", analystV2.Metadata.Version)
	}

	// Stale expected-version should fail.
	if err := store.UpsertProfile(ctx, MustRegistrySlug("default"), analystV2, SaveOptions{
		ExpectedVersion: 1,
		Actor:           "test",
		Source:          "sqlite",
	}); !errors.Is(err, ErrVersionConflict) {
		t.Fatalf("expected ErrVersionConflict, got=%v", err)
	}

	reg, ok, err := store.GetRegistry(ctx, MustRegistrySlug("default"))
	if err != nil {
		t.Fatalf("GetRegistry returned error: %v", err)
	}
	if !ok || reg == nil {
		t.Fatalf("expected registry")
	}

	// Stale expected-version should fail for default-profile update.
	if err := store.SetDefaultProfile(ctx, MustRegistrySlug("default"), MustProfileSlug("analyst"), SaveOptions{
		ExpectedVersion: reg.Metadata.Version + 100,
		Actor:           "test",
		Source:          "sqlite",
	}); !errors.Is(err, ErrVersionConflict) {
		t.Fatalf("expected ErrVersionConflict from SetDefaultProfile, got=%v", err)
	}

	if err := store.SetDefaultProfile(ctx, MustRegistrySlug("default"), MustProfileSlug("analyst"), SaveOptions{
		ExpectedVersion: reg.Metadata.Version,
		Actor:           "test",
		Source:          "sqlite",
	}); err != nil {
		t.Fatalf("SetDefaultProfile returned error: %v", err)
	}

	regAfterDefault, ok, err := store.GetRegistry(ctx, MustRegistrySlug("default"))
	if err != nil {
		t.Fatalf("GetRegistry after SetDefaultProfile returned error: %v", err)
	}
	if !ok || regAfterDefault == nil {
		t.Fatalf("expected registry after SetDefaultProfile")
	}
	if regAfterDefault.DefaultProfileSlug != MustProfileSlug("analyst") {
		t.Fatalf("expected default profile analyst, got=%s", regAfterDefault.DefaultProfileSlug)
	}
	if regAfterDefault.Metadata.Version <= reg.Metadata.Version {
		t.Fatalf("expected registry version increment after SetDefaultProfile")
	}

	if err := store.DeleteProfile(ctx, MustRegistrySlug("default"), MustProfileSlug("analyst"), SaveOptions{
		ExpectedVersion: analystV2.Metadata.Version + 100,
		Actor:           "test",
		Source:          "sqlite",
	}); !errors.Is(err, ErrVersionConflict) {
		t.Fatalf("expected ErrVersionConflict from DeleteProfile, got=%v", err)
	}

	if err := store.DeleteProfile(ctx, MustRegistrySlug("default"), MustProfileSlug("analyst"), SaveOptions{
		ExpectedVersion: analystV2.Metadata.Version,
		Actor:           "test",
		Source:          "sqlite",
	}); err != nil {
		t.Fatalf("DeleteProfile returned error: %v", err)
	}

	_, ok, err = store.GetProfile(ctx, MustRegistrySlug("default"), MustProfileSlug("analyst"))
	if err != nil {
		t.Fatalf("GetProfile after delete returned error: %v", err)
	}
	if ok {
		t.Fatalf("expected analyst profile to be deleted")
	}
}

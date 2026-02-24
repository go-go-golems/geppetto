package profiles

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
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

func TestSQLiteProfileStore_SchemaMigrationIdempotency(t *testing.T) {
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

	if err := store.migrate(); err != nil {
		t.Fatalf("first migrate call failed: %v", err)
	}
	if err := store.migrate(); err != nil {
		t.Fatalf("second migrate call failed: %v", err)
	}
}

func TestSQLiteProfileStore_MalformedPayloadRowHandling(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "profiles.db")
	dsn, err := SQLiteProfileDSNForFile(dbPath)
	if err != nil {
		t.Fatalf("SQLiteProfileDSNForFile returned error: %v", err)
	}

	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		t.Fatalf("sql.Open returned error: %v", err)
	}
	if _, err := db.Exec(sqliteProfilesSchemaV1); err != nil {
		t.Fatalf("create schema failed: %v", err)
	}
	if _, err := db.Exec(
		`INSERT INTO profile_registries (slug, payload_json, updated_at_ms) VALUES (?, ?, ?)`,
		"default",
		`{"slug":"default","profiles":`,
		1,
	); err != nil {
		t.Fatalf("insert malformed payload failed: %v", err)
	}
	_ = db.Close()

	_, err = NewSQLiteProfileStore(dsn, MustRegistrySlug("default"))
	if err == nil {
		t.Fatalf("expected load error for malformed payload JSON")
	}
	errS := strings.ToLower(err.Error())
	if !strings.Contains(errS, "json") && !strings.Contains(errS, "unexpected") && !strings.Contains(errS, "invalid") {
		t.Fatalf("expected invalid payload details, got %v", err)
	}
}

func TestSQLiteProfileStore_RowSlugPayloadSlugMismatch(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "profiles.db")
	dsn, err := SQLiteProfileDSNForFile(dbPath)
	if err != nil {
		t.Fatalf("SQLiteProfileDSNForFile returned error: %v", err)
	}

	registry := &ProfileRegistry{
		Slug:               MustRegistrySlug("other"),
		DefaultProfileSlug: MustProfileSlug("default"),
		Profiles: map[ProfileSlug]*Profile{
			MustProfileSlug("default"): {Slug: MustProfileSlug("default")},
		},
	}
	payload, err := json.Marshal(registry)
	if err != nil {
		t.Fatalf("marshal payload failed: %v", err)
	}

	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		t.Fatalf("sql.Open returned error: %v", err)
	}
	if _, err := db.Exec(sqliteProfilesSchemaV1); err != nil {
		t.Fatalf("create schema failed: %v", err)
	}
	if _, err := db.Exec(
		`INSERT INTO profile_registries (slug, payload_json, updated_at_ms) VALUES (?, ?, ?)`,
		"default",
		string(payload),
		1,
	); err != nil {
		t.Fatalf("insert mismatch payload failed: %v", err)
	}
	_ = db.Close()

	_, err = NewSQLiteProfileStore(dsn, MustRegistrySlug("default"))
	if err == nil {
		t.Fatalf("expected slug mismatch error")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "slug mismatch") {
		t.Fatalf("expected slug mismatch details, got %v", err)
	}
}

func TestSQLiteProfileStore_PersistenceAfterProfileCRUD(t *testing.T) {
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

	if err := store.UpsertRegistry(ctx, &ProfileRegistry{
		Slug:               MustRegistrySlug("default"),
		DefaultProfileSlug: MustProfileSlug("default"),
		Profiles: map[ProfileSlug]*Profile{
			MustProfileSlug("default"): {Slug: MustProfileSlug("default")},
		},
	}, SaveOptions{Actor: "test", Source: "sqlite"}); err != nil {
		t.Fatalf("UpsertRegistry failed: %v", err)
	}
	if err := store.UpsertProfile(ctx, MustRegistrySlug("default"), &Profile{
		Slug:        MustProfileSlug("agent"),
		DisplayName: "Agent v1",
	}, SaveOptions{Actor: "test", Source: "sqlite"}); err != nil {
		t.Fatalf("UpsertProfile create failed: %v", err)
	}
	agent, ok, err := store.GetProfile(ctx, MustRegistrySlug("default"), MustProfileSlug("agent"))
	if err != nil || !ok || agent == nil {
		t.Fatalf("expected created profile, ok=%v err=%v", ok, err)
	}
	updated := agent.Clone()
	updated.DisplayName = "Agent v2"
	if err := store.UpsertProfile(ctx, MustRegistrySlug("default"), updated, SaveOptions{
		ExpectedVersion: agent.Metadata.Version,
		Actor:           "test",
		Source:          "sqlite",
	}); err != nil {
		t.Fatalf("UpsertProfile update failed: %v", err)
	}
	if err := store.SetDefaultProfile(ctx, MustRegistrySlug("default"), MustProfileSlug("agent"), SaveOptions{
		Actor:  "test",
		Source: "sqlite",
	}); err != nil {
		t.Fatalf("SetDefaultProfile failed: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	reloaded, err := NewSQLiteProfileStore(dsn, MustRegistrySlug("default"))
	if err != nil {
		t.Fatalf("reopen store failed: %v", err)
	}
	defer func() { _ = reloaded.Close() }()

	reg, ok, err := reloaded.GetRegistry(ctx, MustRegistrySlug("default"))
	if err != nil || !ok || reg == nil {
		t.Fatalf("expected registry after reopen, ok=%v err=%v", ok, err)
	}
	if got := reg.DefaultProfileSlug; got != MustProfileSlug("agent") {
		t.Fatalf("default profile mismatch after reopen: %q", got)
	}
	reloadedAgent, ok, err := reloaded.GetProfile(ctx, MustRegistrySlug("default"), MustProfileSlug("agent"))
	if err != nil || !ok || reloadedAgent == nil {
		t.Fatalf("expected profile after reopen, ok=%v err=%v", ok, err)
	}
	if got := reloadedAgent.DisplayName; got != "Agent v2" {
		t.Fatalf("updated profile mismatch after reopen: %q", got)
	}
}

func TestSQLiteProfileStore_DeleteProfilePersistsRegistryRowUpdate(t *testing.T) {
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

	if err := store.UpsertRegistry(ctx, &ProfileRegistry{
		Slug:               MustRegistrySlug("default"),
		DefaultProfileSlug: MustProfileSlug("agent"),
		Profiles: map[ProfileSlug]*Profile{
			MustProfileSlug("agent"): {Slug: MustProfileSlug("agent")},
		},
	}, SaveOptions{Actor: "test", Source: "sqlite"}); err != nil {
		t.Fatalf("UpsertRegistry failed: %v", err)
	}

	if err := store.DeleteProfile(ctx, MustRegistrySlug("default"), MustProfileSlug("agent"), SaveOptions{
		Actor:  "test",
		Source: "sqlite",
	}); err != nil {
		t.Fatalf("DeleteProfile failed: %v", err)
	}

	reloaded, err := NewSQLiteProfileStore(dsn, MustRegistrySlug("default"))
	if err != nil {
		t.Fatalf("reopen store failed: %v", err)
	}
	defer func() { _ = reloaded.Close() }()

	reg, ok, err := reloaded.GetRegistry(ctx, MustRegistrySlug("default"))
	if err != nil || !ok || reg == nil {
		t.Fatalf("expected registry after reopen, ok=%v err=%v", ok, err)
	}
	if got := reg.DefaultProfileSlug; got != "" {
		t.Fatalf("expected cleared default profile after deletion, got %q", got)
	}
	_, ok, err = reloaded.GetProfile(ctx, MustRegistrySlug("default"), MustProfileSlug("agent"))
	if err != nil {
		t.Fatalf("GetProfile after reopen failed: %v", err)
	}
	if ok {
		t.Fatalf("expected deleted profile to remain absent after reopen")
	}
}

func TestSQLiteProfileStore_ExtensionsRoundTrip(t *testing.T) {
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
	if err := store.UpsertRegistry(ctx, &ProfileRegistry{
		Slug:               MustRegistrySlug("default"),
		DefaultProfileSlug: MustProfileSlug("default"),
		Profiles: map[ProfileSlug]*Profile{
			MustProfileSlug("default"): {Slug: MustProfileSlug("default")},
		},
	}, SaveOptions{Actor: "bootstrap", Source: "sqlite"}); err != nil {
		t.Fatalf("UpsertRegistry failed: %v", err)
	}
	if err := store.UpsertProfile(ctx, MustRegistrySlug("default"), &Profile{
		Slug: MustProfileSlug("agent"),
		Extensions: map[string]any{
			"app.custom@v1": map[string]any{
				"items": []any{
					map[string]any{"enabled": true},
					"note",
				},
			},
		},
	}, SaveOptions{Actor: "create", Source: "sqlite"}); err != nil {
		t.Fatalf("UpsertProfile failed: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	reloaded, err := NewSQLiteProfileStore(dsn, MustRegistrySlug("default"))
	if err != nil {
		t.Fatalf("reopen store failed: %v", err)
	}
	defer func() { _ = reloaded.Close() }()

	profile, ok, err := reloaded.GetProfile(ctx, MustRegistrySlug("default"), MustProfileSlug("agent"))
	if err != nil || !ok || profile == nil {
		t.Fatalf("expected profile after reopen, ok=%v err=%v", ok, err)
	}
	ext, ok := profile.Extensions["app.custom@v1"]
	if !ok {
		t.Fatalf("expected extension key after sqlite roundtrip")
	}
	items := ext.(map[string]any)["items"].([]any)
	if got, want := len(items), 2; got != want {
		t.Fatalf("extension items length mismatch: got=%d want=%d", got, want)
	}
}

func TestSQLiteProfileStore_UnknownExtensionsPreservedOnServicePartialUpdate(t *testing.T) {
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
	if err := store.UpsertRegistry(ctx, &ProfileRegistry{
		Slug:               MustRegistrySlug("default"),
		DefaultProfileSlug: MustProfileSlug("default"),
		Profiles: map[ProfileSlug]*Profile{
			MustProfileSlug("default"): {Slug: MustProfileSlug("default")},
		},
	}, SaveOptions{Actor: "bootstrap", Source: "sqlite"}); err != nil {
		t.Fatalf("UpsertRegistry failed: %v", err)
	}

	service, err := NewStoreRegistry(store, MustRegistrySlug("default"))
	if err != nil {
		t.Fatalf("NewStoreRegistry failed: %v", err)
	}
	if _, err := service.CreateProfile(ctx, MustRegistrySlug("default"), &Profile{
		Slug: MustProfileSlug("agent"),
		Extensions: map[string]any{
			"Vendor.Custom@V1": map[string]any{
				"flags": []any{map[string]any{"enabled": true}},
			},
		},
	}, WriteOptions{Actor: "create", Source: "sqlite"}); err != nil {
		t.Fatalf("CreateProfile failed: %v", err)
	}

	displayName := "Agent Updated"
	if _, err := service.UpdateProfile(ctx, MustRegistrySlug("default"), MustProfileSlug("agent"), ProfilePatch{
		DisplayName: &displayName,
	}, WriteOptions{Actor: "update", Source: "sqlite"}); err != nil {
		t.Fatalf("UpdateProfile failed: %v", err)
	}

	if err := store.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}
	reloaded, err := NewSQLiteProfileStore(dsn, MustRegistrySlug("default"))
	if err != nil {
		t.Fatalf("reopen store failed: %v", err)
	}
	defer func() { _ = reloaded.Close() }()

	profile, ok, err := reloaded.GetProfile(ctx, MustRegistrySlug("default"), MustProfileSlug("agent"))
	if err != nil || !ok || profile == nil {
		t.Fatalf("expected profile after reopen, ok=%v err=%v", ok, err)
	}
	if got := profile.DisplayName; got != "Agent Updated" {
		t.Fatalf("display name mismatch after partial update: %q", got)
	}
	ext, ok := profile.Extensions["vendor.custom@v1"]
	if !ok {
		t.Fatalf("expected canonical unknown extension key after partial update")
	}
	enabled := ext.(map[string]any)["flags"].([]any)[0].(map[string]any)["enabled"].(bool)
	if !enabled {
		t.Fatalf("expected unknown extension payload preserved after partial update")
	}
}

func TestSQLiteProfileStore_CloseIdempotencyAndPostCloseGuards(t *testing.T) {
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

	if err := store.Close(); err != nil {
		t.Fatalf("first Close failed: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("second Close failed: %v", err)
	}

	assertClosedErr := func(err error) {
		t.Helper()
		if err == nil {
			t.Fatalf("expected closed-store error")
		}
		if !strings.Contains(strings.ToLower(err.Error()), "closed") {
			t.Fatalf("expected closed-store error details, got %v", err)
		}
	}

	_, err = store.ListRegistries(ctx)
	assertClosedErr(err)
	_, _, err = store.GetRegistry(ctx, MustRegistrySlug("default"))
	assertClosedErr(err)
	_, err = store.ListProfiles(ctx, MustRegistrySlug("default"))
	assertClosedErr(err)
	_, _, err = store.GetProfile(ctx, MustRegistrySlug("default"), MustProfileSlug("default"))
	assertClosedErr(err)
	err = store.UpsertRegistry(ctx, &ProfileRegistry{Slug: MustRegistrySlug("default")}, SaveOptions{})
	assertClosedErr(err)
	err = store.DeleteRegistry(ctx, MustRegistrySlug("default"), SaveOptions{})
	assertClosedErr(err)
	err = store.UpsertProfile(ctx, MustRegistrySlug("default"), &Profile{Slug: MustProfileSlug("default")}, SaveOptions{})
	assertClosedErr(err)
	err = store.DeleteProfile(ctx, MustRegistrySlug("default"), MustProfileSlug("default"), SaveOptions{})
	assertClosedErr(err)
	err = store.SetDefaultProfile(ctx, MustRegistrySlug("default"), MustProfileSlug("default"), SaveOptions{})
	assertClosedErr(err)
}

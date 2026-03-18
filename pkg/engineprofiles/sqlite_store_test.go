package engineprofiles

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

func TestSQLiteEngineProfileStore_RegistryRoundTrip(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "profiles.db")
	dsn, err := SQLiteProfileDSNForFile(dbPath)
	if err != nil {
		t.Fatalf("SQLiteProfileDSNForFile returned error: %v", err)
	}

	store, err := NewSQLiteEngineProfileStore(dsn, MustRegistrySlug("default"))
	if err != nil {
		t.Fatalf("NewSQLiteEngineProfileStore returned error: %v", err)
	}

	registry := &EngineProfileRegistry{
		Slug:                     MustRegistrySlug("default"),
		DefaultEngineProfileSlug: MustEngineProfileSlug("default"),
		Profiles: map[EngineProfileSlug]*EngineProfile{
			MustEngineProfileSlug("default"): {
				Slug:        MustEngineProfileSlug("default"),
				DisplayName: "Default",
				Runtime: RuntimeSpec{
					SystemPrompt: "You are default",
					Tools:        []string{"calculator"},
				},
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
	if got.DefaultEngineProfileSlug != MustEngineProfileSlug("default") {
		t.Fatalf("unexpected default profile slug: %s", got.DefaultEngineProfileSlug)
	}
	if _, ok := got.Profiles[MustEngineProfileSlug("default")]; !ok {
		t.Fatalf("expected default profile in registry")
	}

	if err := store.Close(); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}

	reopened, err := NewSQLiteEngineProfileStore(dsn, MustRegistrySlug("default"))
	if err != nil {
		t.Fatalf("reopen NewSQLiteEngineProfileStore returned error: %v", err)
	}
	defer func() { _ = reopened.Close() }()

	reloaded, ok, err := reopened.GetRegistry(ctx, MustRegistrySlug("default"))
	if err != nil {
		t.Fatalf("reloaded GetRegistry returned error: %v", err)
	}
	if !ok || reloaded == nil {
		t.Fatalf("reloaded registry expected to exist")
	}
	if reloaded.DefaultEngineProfileSlug != MustEngineProfileSlug("default") {
		t.Fatalf("reloaded default profile slug mismatch: %s", reloaded.DefaultEngineProfileSlug)
	}
	if reloaded.Profiles[MustEngineProfileSlug("default")] == nil {
		t.Fatalf("reloaded default profile missing")
	}
}

func TestSQLiteEngineProfileStore_RegistryRoundTrip_PreservesStackRefs(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "profiles.db")
	dsn, err := SQLiteProfileDSNForFile(dbPath)
	if err != nil {
		t.Fatalf("SQLiteProfileDSNForFile returned error: %v", err)
	}

	store, err := NewSQLiteEngineProfileStore(dsn, MustRegistrySlug("default"))
	if err != nil {
		t.Fatalf("NewSQLiteEngineProfileStore returned error: %v", err)
	}

	if err := store.UpsertRegistry(ctx, &EngineProfileRegistry{
		Slug:                     MustRegistrySlug("shared"),
		DefaultEngineProfileSlug: MustEngineProfileSlug("mw-observability"),
		Profiles: map[EngineProfileSlug]*EngineProfile{
			MustEngineProfileSlug("mw-observability"): {
				Slug: MustEngineProfileSlug("mw-observability"),
			},
		},
	}, SaveOptions{Actor: "test", Source: "sqlite"}); err != nil {
		t.Fatalf("UpsertRegistry(shared) returned error: %v", err)
	}

	if err := store.UpsertRegistry(ctx, &EngineProfileRegistry{
		Slug:                     MustRegistrySlug("default"),
		DefaultEngineProfileSlug: MustEngineProfileSlug("agent"),
		Profiles: map[EngineProfileSlug]*EngineProfile{
			MustEngineProfileSlug("provider-openai"): {
				Slug: MustEngineProfileSlug("provider-openai"),
			},
			MustEngineProfileSlug("model-gpt4o"): {
				Slug: MustEngineProfileSlug("model-gpt4o"),
				Stack: []EngineProfileRef{
					{EngineProfileSlug: MustEngineProfileSlug("provider-openai")},
				},
			},
			MustEngineProfileSlug("agent"): {
				Slug: MustEngineProfileSlug("agent"),
				Stack: []EngineProfileRef{
					{EngineProfileSlug: MustEngineProfileSlug("model-gpt4o")},
					{RegistrySlug: MustRegistrySlug("shared"), EngineProfileSlug: MustEngineProfileSlug("mw-observability")},
				},
			},
		},
	}, SaveOptions{Actor: "test", Source: "sqlite"}); err != nil {
		t.Fatalf("UpsertRegistry(default) returned error: %v", err)
	}

	if err := store.Close(); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}

	reopened, err := NewSQLiteEngineProfileStore(dsn, MustRegistrySlug("default"))
	if err != nil {
		t.Fatalf("reopen NewSQLiteEngineProfileStore returned error: %v", err)
	}
	defer func() { _ = reopened.Close() }()

	model, ok, err := reopened.GetEngineProfile(ctx, MustRegistrySlug("default"), MustEngineProfileSlug("model-gpt4o"))
	if err != nil {
		t.Fatalf("GetEngineProfile(default/model-gpt4o) returned error: %v", err)
	}
	if !ok || model == nil {
		t.Fatalf("expected default/model-gpt4o profile after reload")
	}
	if got, want := len(model.Stack), 1; got != want {
		t.Fatalf("model stack length mismatch: got=%d want=%d", got, want)
	}
	if got := model.Stack[0].EngineProfileSlug; got != MustEngineProfileSlug("provider-openai") {
		t.Fatalf("model stack profile mismatch: got=%q", got)
	}
	if got := model.Stack[0].RegistrySlug; got != "" {
		t.Fatalf("model stack registry should be empty for same-registry ref, got=%q", got)
	}

	agent, ok, err := reopened.GetEngineProfile(ctx, MustRegistrySlug("default"), MustEngineProfileSlug("agent"))
	if err != nil {
		t.Fatalf("GetEngineProfile(default/agent) returned error: %v", err)
	}
	if !ok || agent == nil {
		t.Fatalf("expected default/agent profile after reload")
	}
	if got, want := len(agent.Stack), 2; got != want {
		t.Fatalf("agent stack length mismatch: got=%d want=%d", got, want)
	}
	if got := agent.Stack[1].RegistrySlug; got != MustRegistrySlug("shared") {
		t.Fatalf("agent stack second registry mismatch: got=%q", got)
	}
	if got := agent.Stack[1].EngineProfileSlug; got != MustEngineProfileSlug("mw-observability") {
		t.Fatalf("agent stack second profile mismatch: got=%q", got)
	}
}

func TestSQLiteEngineProfileStore_ProfileLifecycleAndVersionConflicts(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "profiles.db")
	dsn, err := SQLiteProfileDSNForFile(dbPath)
	if err != nil {
		t.Fatalf("SQLiteProfileDSNForFile returned error: %v", err)
	}

	store, err := NewSQLiteEngineProfileStore(dsn, MustRegistrySlug("default"))
	if err != nil {
		t.Fatalf("NewSQLiteEngineProfileStore returned error: %v", err)
	}
	defer func() { _ = store.Close() }()

	registry := &EngineProfileRegistry{
		Slug:                     MustRegistrySlug("default"),
		DefaultEngineProfileSlug: MustEngineProfileSlug("default"),
		Profiles: map[EngineProfileSlug]*EngineProfile{
			MustEngineProfileSlug("default"): {
				Slug:    MustEngineProfileSlug("default"),
				Runtime: RuntimeSpec{SystemPrompt: "You are default"},
			},
		},
	}
	if err := store.UpsertRegistry(ctx, registry, SaveOptions{Actor: "test", Source: "sqlite"}); err != nil {
		t.Fatalf("UpsertRegistry returned error: %v", err)
	}

	if err := store.UpsertEngineProfile(ctx, MustRegistrySlug("default"), &EngineProfile{
		Slug:        MustEngineProfileSlug("analyst"),
		DisplayName: "Analyst",
		Runtime: RuntimeSpec{
			SystemPrompt: "You are analyst",
		},
	}, SaveOptions{Actor: "test", Source: "sqlite"}); err != nil {
		t.Fatalf("UpsertEngineProfile create returned error: %v", err)
	}

	analyst, ok, err := store.GetEngineProfile(ctx, MustRegistrySlug("default"), MustEngineProfileSlug("analyst"))
	if err != nil {
		t.Fatalf("GetEngineProfile returned error: %v", err)
	}
	if !ok || analyst == nil {
		t.Fatalf("expected analyst profile")
	}
	if analyst.Metadata.Version != 1 {
		t.Fatalf("expected analyst version=1, got=%d", analyst.Metadata.Version)
	}

	analystUpdate := analyst.Clone()
	analystUpdate.DisplayName = "Analyst v2"
	if err := store.UpsertEngineProfile(ctx, MustRegistrySlug("default"), analystUpdate, SaveOptions{
		ExpectedVersion: analyst.Metadata.Version,
		Actor:           "test",
		Source:          "sqlite",
	}); err != nil {
		t.Fatalf("UpsertEngineProfile update returned error: %v", err)
	}

	analystV2, ok, err := store.GetEngineProfile(ctx, MustRegistrySlug("default"), MustEngineProfileSlug("analyst"))
	if err != nil {
		t.Fatalf("GetEngineProfile after update returned error: %v", err)
	}
	if !ok || analystV2 == nil {
		t.Fatalf("expected analyst profile after update")
	}
	if analystV2.Metadata.Version != 2 {
		t.Fatalf("expected analyst version=2, got=%d", analystV2.Metadata.Version)
	}

	// Stale expected-version should fail.
	if err := store.UpsertEngineProfile(ctx, MustRegistrySlug("default"), analystV2, SaveOptions{
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

	regPatch := reg.Clone()
	regPatch.DefaultEngineProfileSlug = MustEngineProfileSlug("analyst")
	if err := store.UpsertRegistry(ctx, regPatch, SaveOptions{
		ExpectedVersion: reg.Metadata.Version + 100,
		Actor:           "test",
		Source:          "sqlite",
	}); !errors.Is(err, ErrVersionConflict) {
		t.Fatalf("expected ErrVersionConflict from UpsertRegistry default-profile update, got=%v", err)
	}

	if err := store.UpsertRegistry(ctx, regPatch, SaveOptions{
		ExpectedVersion: reg.Metadata.Version,
		Actor:           "test",
		Source:          "sqlite",
	}); err != nil {
		t.Fatalf("UpsertRegistry default-profile update returned error: %v", err)
	}

	regAfterDefault, ok, err := store.GetRegistry(ctx, MustRegistrySlug("default"))
	if err != nil {
		t.Fatalf("GetRegistry after default-profile update returned error: %v", err)
	}
	if !ok || regAfterDefault == nil {
		t.Fatalf("expected registry after default-profile update")
	}
	if regAfterDefault.DefaultEngineProfileSlug != MustEngineProfileSlug("analyst") {
		t.Fatalf("expected default profile analyst, got=%s", regAfterDefault.DefaultEngineProfileSlug)
	}
	if regAfterDefault.Metadata.Version <= reg.Metadata.Version {
		t.Fatalf("expected registry version increment after default-profile update")
	}
}

func TestSQLiteEngineProfileStore_SchemaMigrationIdempotency(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "profiles.db")
	dsn, err := SQLiteProfileDSNForFile(dbPath)
	if err != nil {
		t.Fatalf("SQLiteProfileDSNForFile returned error: %v", err)
	}

	store, err := NewSQLiteEngineProfileStore(dsn, MustRegistrySlug("default"))
	if err != nil {
		t.Fatalf("NewSQLiteEngineProfileStore returned error: %v", err)
	}
	defer func() { _ = store.Close() }()

	if err := store.migrate(); err != nil {
		t.Fatalf("first migrate call failed: %v", err)
	}
	if err := store.migrate(); err != nil {
		t.Fatalf("second migrate call failed: %v", err)
	}
}

func TestSQLiteEngineProfileStore_MalformedPayloadRowHandling(t *testing.T) {
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

	_, err = NewSQLiteEngineProfileStore(dsn, MustRegistrySlug("default"))
	if err == nil {
		t.Fatalf("expected load error for malformed payload JSON")
	}
	errS := strings.ToLower(err.Error())
	if !strings.Contains(errS, "json") && !strings.Contains(errS, "unexpected") && !strings.Contains(errS, "invalid") {
		t.Fatalf("expected invalid payload details, got %v", err)
	}
}

func TestSQLiteEngineProfileStore_RowSlugPayloadSlugMismatch(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "profiles.db")
	dsn, err := SQLiteProfileDSNForFile(dbPath)
	if err != nil {
		t.Fatalf("SQLiteProfileDSNForFile returned error: %v", err)
	}

	registry := &EngineProfileRegistry{
		Slug:                     MustRegistrySlug("other"),
		DefaultEngineProfileSlug: MustEngineProfileSlug("default"),
		Profiles: map[EngineProfileSlug]*EngineProfile{
			MustEngineProfileSlug("default"): {Slug: MustEngineProfileSlug("default")},
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

	_, err = NewSQLiteEngineProfileStore(dsn, MustRegistrySlug("default"))
	if err == nil {
		t.Fatalf("expected slug mismatch error")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "slug mismatch") {
		t.Fatalf("expected slug mismatch details, got %v", err)
	}
}

func TestSQLiteEngineProfileStore_PersistenceAfterProfileCRUD(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "profiles.db")
	dsn, err := SQLiteProfileDSNForFile(dbPath)
	if err != nil {
		t.Fatalf("SQLiteProfileDSNForFile returned error: %v", err)
	}

	store, err := NewSQLiteEngineProfileStore(dsn, MustRegistrySlug("default"))
	if err != nil {
		t.Fatalf("NewSQLiteEngineProfileStore returned error: %v", err)
	}

	if err := store.UpsertRegistry(ctx, &EngineProfileRegistry{
		Slug:                     MustRegistrySlug("default"),
		DefaultEngineProfileSlug: MustEngineProfileSlug("default"),
		Profiles: map[EngineProfileSlug]*EngineProfile{
			MustEngineProfileSlug("default"): {Slug: MustEngineProfileSlug("default")},
		},
	}, SaveOptions{Actor: "test", Source: "sqlite"}); err != nil {
		t.Fatalf("UpsertRegistry failed: %v", err)
	}
	if err := store.UpsertEngineProfile(ctx, MustRegistrySlug("default"), &EngineProfile{
		Slug:        MustEngineProfileSlug("agent"),
		DisplayName: "Agent v1",
	}, SaveOptions{Actor: "test", Source: "sqlite"}); err != nil {
		t.Fatalf("UpsertEngineProfile create failed: %v", err)
	}
	agent, ok, err := store.GetEngineProfile(ctx, MustRegistrySlug("default"), MustEngineProfileSlug("agent"))
	if err != nil || !ok || agent == nil {
		t.Fatalf("expected created profile, ok=%v err=%v", ok, err)
	}
	updated := agent.Clone()
	updated.DisplayName = "Agent v2"
	if err := store.UpsertEngineProfile(ctx, MustRegistrySlug("default"), updated, SaveOptions{
		ExpectedVersion: agent.Metadata.Version,
		Actor:           "test",
		Source:          "sqlite",
	}); err != nil {
		t.Fatalf("UpsertEngineProfile update failed: %v", err)
	}
	reg, ok, err := store.GetRegistry(ctx, MustRegistrySlug("default"))
	if err != nil || !ok || reg == nil {
		t.Fatalf("expected registry before default update, ok=%v err=%v", ok, err)
	}
	reg.DefaultEngineProfileSlug = MustEngineProfileSlug("agent")
	if err := store.UpsertRegistry(ctx, reg, SaveOptions{
		ExpectedVersion: reg.Metadata.Version,
		Actor:           "test",
		Source:          "sqlite",
	}); err != nil {
		t.Fatalf("UpsertRegistry default-profile update failed: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	reloaded, err := NewSQLiteEngineProfileStore(dsn, MustRegistrySlug("default"))
	if err != nil {
		t.Fatalf("reopen store failed: %v", err)
	}
	defer func() { _ = reloaded.Close() }()

	reg, ok, err = reloaded.GetRegistry(ctx, MustRegistrySlug("default"))
	if err != nil || !ok || reg == nil {
		t.Fatalf("expected registry after reopen, ok=%v err=%v", ok, err)
	}
	if got := reg.DefaultEngineProfileSlug; got != MustEngineProfileSlug("agent") {
		t.Fatalf("default profile mismatch after reopen: %q", got)
	}
	reloadedAgent, ok, err := reloaded.GetEngineProfile(ctx, MustRegistrySlug("default"), MustEngineProfileSlug("agent"))
	if err != nil || !ok || reloadedAgent == nil {
		t.Fatalf("expected profile after reopen, ok=%v err=%v", ok, err)
	}
	if got := reloadedAgent.DisplayName; got != "Agent v2" {
		t.Fatalf("updated profile mismatch after reopen: %q", got)
	}
}

func TestSQLiteEngineProfileStore_ExtensionsRoundTrip(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "profiles.db")
	dsn, err := SQLiteProfileDSNForFile(dbPath)
	if err != nil {
		t.Fatalf("SQLiteProfileDSNForFile returned error: %v", err)
	}

	store, err := NewSQLiteEngineProfileStore(dsn, MustRegistrySlug("default"))
	if err != nil {
		t.Fatalf("NewSQLiteEngineProfileStore returned error: %v", err)
	}
	if err := store.UpsertRegistry(ctx, &EngineProfileRegistry{
		Slug:                     MustRegistrySlug("default"),
		DefaultEngineProfileSlug: MustEngineProfileSlug("default"),
		Profiles: map[EngineProfileSlug]*EngineProfile{
			MustEngineProfileSlug("default"): {Slug: MustEngineProfileSlug("default")},
		},
	}, SaveOptions{Actor: "bootstrap", Source: "sqlite"}); err != nil {
		t.Fatalf("UpsertRegistry failed: %v", err)
	}
	if err := store.UpsertEngineProfile(ctx, MustRegistrySlug("default"), &EngineProfile{
		Slug: MustEngineProfileSlug("agent"),
		Extensions: map[string]any{
			"app.custom@v1": map[string]any{
				"items": []any{
					map[string]any{"enabled": true},
					"note",
				},
			},
		},
	}, SaveOptions{Actor: "create", Source: "sqlite"}); err != nil {
		t.Fatalf("UpsertEngineProfile failed: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	reloaded, err := NewSQLiteEngineProfileStore(dsn, MustRegistrySlug("default"))
	if err != nil {
		t.Fatalf("reopen store failed: %v", err)
	}
	defer func() { _ = reloaded.Close() }()

	profile, ok, err := reloaded.GetEngineProfile(ctx, MustRegistrySlug("default"), MustEngineProfileSlug("agent"))
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

func TestSQLiteEngineProfileStore_CloseIdempotencyAndPostCloseGuards(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "profiles.db")
	dsn, err := SQLiteProfileDSNForFile(dbPath)
	if err != nil {
		t.Fatalf("SQLiteProfileDSNForFile returned error: %v", err)
	}

	store, err := NewSQLiteEngineProfileStore(dsn, MustRegistrySlug("default"))
	if err != nil {
		t.Fatalf("NewSQLiteEngineProfileStore returned error: %v", err)
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
	_, err = store.ListEngineProfiles(ctx, MustRegistrySlug("default"))
	assertClosedErr(err)
	_, _, err = store.GetEngineProfile(ctx, MustRegistrySlug("default"), MustEngineProfileSlug("default"))
	assertClosedErr(err)
	err = store.UpsertRegistry(ctx, &EngineProfileRegistry{Slug: MustRegistrySlug("default")}, SaveOptions{})
	assertClosedErr(err)
	err = store.DeleteRegistry(ctx, MustRegistrySlug("default"), SaveOptions{})
	assertClosedErr(err)
	err = store.UpsertEngineProfile(ctx, MustRegistrySlug("default"), &EngineProfile{Slug: MustEngineProfileSlug("default")}, SaveOptions{})
	assertClosedErr(err)
}

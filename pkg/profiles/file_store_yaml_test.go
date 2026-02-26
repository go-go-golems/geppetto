package profiles

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestYAMLFileProfileStore_PersistAndReload(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "profiles.yaml")

	store, err := NewYAMLFileProfileStore(path, MustRegistrySlug("default"))
	if err != nil {
		t.Fatalf("NewYAMLFileProfileStore failed: %v", err)
	}
	ctx := context.Background()

	reg := &ProfileRegistry{
		Slug:               MustRegistrySlug("default"),
		DefaultProfileSlug: MustProfileSlug("default"),
		Profiles: map[ProfileSlug]*Profile{
			MustProfileSlug("default"): {Slug: MustProfileSlug("default")},
		},
	}
	if err := store.UpsertRegistry(ctx, reg, SaveOptions{Actor: "test", Source: "file"}); err != nil {
		t.Fatalf("UpsertRegistry failed: %v", err)
	}
	if err := store.UpsertProfile(ctx, MustRegistrySlug("default"), &Profile{Slug: MustProfileSlug("agent")}, SaveOptions{Actor: "test"}); err != nil {
		t.Fatalf("UpsertProfile failed: %v", err)
	}

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected yaml file to exist: %v", err)
	}

	reloaded, err := NewYAMLFileProfileStore(path, MustRegistrySlug("default"))
	if err != nil {
		t.Fatalf("reloaded store init failed: %v", err)
	}
	profiles, err := reloaded.ListProfiles(ctx, MustRegistrySlug("default"))
	if err != nil {
		t.Fatalf("ListProfiles failed: %v", err)
	}
	if len(profiles) != 2 {
		t.Fatalf("expected 2 profiles after reload, got %d", len(profiles))
	}
}

func TestYAMLFileProfileStore_RejectsLegacyFile(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "profiles.yaml")
	legacy := []byte(`default:
  ai-chat:
    ai-engine: gpt-4o-mini
agent:
  ai-chat:
    ai-engine: gpt-4.1
`)
	if err := os.WriteFile(path, legacy, 0o644); err != nil {
		t.Fatalf("write legacy fixture failed: %v", err)
	}

	_, err := NewYAMLFileProfileStore(path, MustRegistrySlug("default"))
	if err == nil {
		t.Fatalf("expected legacy file decode error")
	}
	if !strings.Contains(err.Error(), "legacy profile-map format is not supported") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestYAMLFileProfileStore_MissingFileInitialization(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "missing", "profiles.yaml")
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected file to be missing before init")
	}

	store, err := NewYAMLFileProfileStore(path, MustRegistrySlug("default"))
	if err != nil {
		t.Fatalf("NewYAMLFileProfileStore failed: %v", err)
	}
	ctx := context.Background()

	registries, err := store.ListRegistries(ctx)
	if err != nil {
		t.Fatalf("ListRegistries failed: %v", err)
	}
	if len(registries) != 0 {
		t.Fatalf("expected no registries, got %d", len(registries))
	}

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected file to still be missing until first write")
	}

	reg := &ProfileRegistry{
		Slug:               MustRegistrySlug("default"),
		DefaultProfileSlug: MustProfileSlug("default"),
		Profiles: map[ProfileSlug]*Profile{
			MustProfileSlug("default"): {Slug: MustProfileSlug("default")},
		},
	}
	if err := store.UpsertRegistry(ctx, reg, SaveOptions{Actor: "test", Source: "yaml"}); err != nil {
		t.Fatalf("UpsertRegistry failed: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected file after write: %v", err)
	}
}

func TestYAMLFileProfileStore_ParseFailureSurfacing(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "profiles.yaml")
	if err := os.WriteFile(path, []byte("registries:\n  : bad"), 0o644); err != nil {
		t.Fatalf("write malformed yaml failed: %v", err)
	}

	_, err := NewYAMLFileProfileStore(path, MustRegistrySlug("default"))
	if err == nil {
		t.Fatalf("expected parse error for malformed yaml")
	}
	errS := strings.ToLower(err.Error())
	if !strings.Contains(errS, "yaml") && !strings.Contains(errS, "runtime yaml") {
		t.Fatalf("expected parse/runtime yaml error details, got %v", err)
	}
}

func TestYAMLFileProfileStore_RejectsSecondRegistrySlug(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "profiles.yaml")
	ctx := context.Background()

	store, err := NewYAMLFileProfileStore(path, MustRegistrySlug("default"))
	if err != nil {
		t.Fatalf("NewYAMLFileProfileStore failed: %v", err)
	}

	if err := store.UpsertRegistry(ctx, &ProfileRegistry{
		Slug:               MustRegistrySlug("default"),
		DefaultProfileSlug: MustProfileSlug("default"),
		Profiles: map[ProfileSlug]*Profile{
			MustProfileSlug("default"): {
				Slug: MustProfileSlug("default"),
				Runtime: RuntimeSpec{
					SystemPrompt: "default profile",
					Tools:        []string{"calculator"},
				},
			},
		},
	}, SaveOptions{Actor: "test", Source: "yaml"}); err != nil {
		t.Fatalf("upsert default registry failed: %v", err)
	}

	err = store.UpsertRegistry(ctx, &ProfileRegistry{
		Slug:               MustRegistrySlug("team"),
		DefaultProfileSlug: MustProfileSlug("agent"),
		Profiles: map[ProfileSlug]*Profile{
			MustProfileSlug("agent"): {
				Slug:        MustProfileSlug("agent"),
				DisplayName: "Team Agent",
				Runtime: RuntimeSpec{
					SystemPrompt: "team profile",
					Middlewares: []MiddlewareUse{
						{Name: "agentmode", Config: map[string]any{"mode": "planner"}},
					},
				},
			},
		},
	}, SaveOptions{Actor: "test", Source: "yaml"})
	if err == nil {
		t.Fatalf("expected second registry upsert to fail for single-registry yaml store")
	}
	if !strings.Contains(err.Error(), "supports only registry") {
		t.Fatalf("unexpected second registry upsert error: %v", err)
	}

	reloaded, err := NewYAMLFileProfileStore(path, MustRegistrySlug("default"))
	if err != nil {
		t.Fatalf("reloaded store init failed: %v", err)
	}

	registries, err := reloaded.ListRegistries(ctx)
	if err != nil {
		t.Fatalf("reloaded ListRegistries failed: %v", err)
	}
	if len(registries) != 1 {
		t.Fatalf("expected 1 registry after reload, got %d", len(registries))
	}
	if registries[0].Slug != MustRegistrySlug("default") {
		t.Fatalf("unexpected registry slug after reload: %q", registries[0].Slug)
	}

	_, _, err = reloaded.GetProfile(ctx, MustRegistrySlug("team"), MustProfileSlug("agent"))
	if err == nil {
		t.Fatalf("expected non-default registry lookup to fail")
	}
	if !strings.Contains(err.Error(), "supports only registry") {
		t.Fatalf("unexpected non-default registry lookup error: %v", err)
	}
}

func TestYAMLFileProfileStore_AtomicTempRenameBehavior(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "profiles.yaml")
	ctx := context.Background()

	store, err := NewYAMLFileProfileStore(path, MustRegistrySlug("default"))
	if err != nil {
		t.Fatalf("NewYAMLFileProfileStore failed: %v", err)
	}

	reg := &ProfileRegistry{
		Slug:               MustRegistrySlug("default"),
		DefaultProfileSlug: MustProfileSlug("default"),
		Profiles: map[ProfileSlug]*Profile{
			MustProfileSlug("default"): {Slug: MustProfileSlug("default")},
		},
	}
	if err := store.UpsertRegistry(ctx, reg, SaveOptions{Actor: "test", Source: "yaml"}); err != nil {
		t.Fatalf("UpsertRegistry failed: %v", err)
	}
	if _, err := os.Stat(path + ".tmp"); !os.IsNotExist(err) {
		t.Fatalf("expected no tmp file residue after write")
	}

	if err := store.UpsertProfile(ctx, MustRegistrySlug("default"), &Profile{
		Slug: MustProfileSlug("agent"),
	}, SaveOptions{Actor: "test", Source: "yaml"}); err != nil {
		t.Fatalf("UpsertProfile failed: %v", err)
	}
	if _, err := os.Stat(path + ".tmp"); !os.IsNotExist(err) {
		t.Fatalf("expected no tmp file residue after second write")
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read final yaml failed: %v", err)
	}
	if _, err := DecodeYAMLRegistries(raw, MustRegistrySlug("default")); err != nil {
		t.Fatalf("final yaml decode failed: %v", err)
	}
}

func TestYAMLFileProfileStore_UnknownExtensionsPreservedOnServicePartialUpdate(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "profiles.yaml")

	store, err := NewYAMLFileProfileStore(path, MustRegistrySlug("default"))
	if err != nil {
		t.Fatalf("NewYAMLFileProfileStore failed: %v", err)
	}
	if err := store.UpsertRegistry(ctx, &ProfileRegistry{
		Slug:               MustRegistrySlug("default"),
		DefaultProfileSlug: MustProfileSlug("default"),
		Profiles: map[ProfileSlug]*Profile{
			MustProfileSlug("default"): {Slug: MustProfileSlug("default")},
		},
	}, SaveOptions{Actor: "bootstrap", Source: "yaml"}); err != nil {
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
				"items": []any{map[string]any{"enabled": true}},
			},
		},
	}, WriteOptions{Actor: "create", Source: "yaml"}); err != nil {
		t.Fatalf("CreateProfile failed: %v", err)
	}

	name := "Agent Updated"
	if _, err := service.UpdateProfile(ctx, MustRegistrySlug("default"), MustProfileSlug("agent"), ProfilePatch{
		DisplayName: &name,
	}, WriteOptions{Actor: "update", Source: "yaml"}); err != nil {
		t.Fatalf("UpdateProfile failed: %v", err)
	}

	if err := store.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}
	reloaded, err := NewYAMLFileProfileStore(path, MustRegistrySlug("default"))
	if err != nil {
		t.Fatalf("NewYAMLFileProfileStore reload failed: %v", err)
	}
	defer func() { _ = reloaded.Close() }()

	profile, ok, err := reloaded.GetProfile(ctx, MustRegistrySlug("default"), MustProfileSlug("agent"))
	if err != nil || !ok || profile == nil {
		t.Fatalf("expected profile after reload, ok=%v err=%v", ok, err)
	}
	if got := profile.DisplayName; got != "Agent Updated" {
		t.Fatalf("display name mismatch after partial update: %q", got)
	}
	ext, ok := profile.Extensions["vendor.custom@v1"]
	if !ok {
		t.Fatalf("expected canonical unknown extension key after reload")
	}
	enabled := ext.(map[string]any)["items"].([]any)[0].(map[string]any)["enabled"].(bool)
	if !enabled {
		t.Fatalf("expected unknown extension payload preserved after partial update")
	}
}

func TestYAMLFileProfileStore_CloseStateGuards(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "profiles.yaml")
	ctx := context.Background()

	store, err := NewYAMLFileProfileStore(path, MustRegistrySlug("default"))
	if err != nil {
		t.Fatalf("NewYAMLFileProfileStore failed: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
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

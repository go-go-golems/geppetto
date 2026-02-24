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

func TestYAMLFileProfileStore_LoadLegacyFile(t *testing.T) {
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

	store, err := NewYAMLFileProfileStore(path, MustRegistrySlug("default"))
	if err != nil {
		t.Fatalf("NewYAMLFileProfileStore failed: %v", err)
	}
	profiles, err := store.ListProfiles(context.Background(), MustRegistrySlug("default"))
	if err != nil {
		t.Fatalf("ListProfiles failed: %v", err)
	}
	if len(profiles) != 2 {
		t.Fatalf("expected 2 profiles from legacy file, got %d", len(profiles))
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
	if !strings.Contains(strings.ToLower(err.Error()), "yaml") {
		t.Fatalf("expected yaml error details, got %v", err)
	}
}

func TestYAMLFileProfileStore_WriteThenReloadParity_MultipleRegistries(t *testing.T) {
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

	if err := store.UpsertRegistry(ctx, &ProfileRegistry{
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
	}, SaveOptions{Actor: "test", Source: "yaml"}); err != nil {
		t.Fatalf("upsert team registry failed: %v", err)
	}

	reloaded, err := NewYAMLFileProfileStore(path, MustRegistrySlug("default"))
	if err != nil {
		t.Fatalf("reloaded store init failed: %v", err)
	}

	registries, err := reloaded.ListRegistries(ctx)
	if err != nil {
		t.Fatalf("reloaded ListRegistries failed: %v", err)
	}
	if len(registries) != 2 {
		t.Fatalf("expected 2 registries after reload, got %d", len(registries))
	}
	if registries[0].Slug != MustRegistrySlug("default") || registries[1].Slug != MustRegistrySlug("team") {
		t.Fatalf("unexpected registry ordering after reload: %q, %q", registries[0].Slug, registries[1].Slug)
	}

	teamAgent, ok, err := reloaded.GetProfile(ctx, MustRegistrySlug("team"), MustProfileSlug("agent"))
	if err != nil {
		t.Fatalf("GetProfile(team/agent) failed: %v", err)
	}
	if !ok || teamAgent == nil {
		t.Fatalf("expected team/agent profile after reload")
	}
	if got := teamAgent.DisplayName; got != "Team Agent" {
		t.Fatalf("display name mismatch after reload: %q", got)
	}
	if got := teamAgent.Runtime.Middlewares[0].Config.(map[string]any)["mode"].(string); got != "planner" {
		t.Fatalf("middleware config mismatch after reload: %q", got)
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

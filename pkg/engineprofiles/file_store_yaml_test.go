package engineprofiles

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestYAMLFileEngineProfileStore_PersistAndReload(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "profiles.yaml")

	store, err := NewYAMLFileEngineProfileStore(path, MustRegistrySlug("default"))
	if err != nil {
		t.Fatalf("NewYAMLFileEngineProfileStore failed: %v", err)
	}
	ctx := context.Background()

	reg := &EngineProfileRegistry{
		Slug:                     MustRegistrySlug("default"),
		DefaultEngineProfileSlug: MustEngineProfileSlug("default"),
		Profiles: map[EngineProfileSlug]*EngineProfile{
			MustEngineProfileSlug("default"): {Slug: MustEngineProfileSlug("default")},
		},
	}
	if err := store.UpsertRegistry(ctx, reg, SaveOptions{Actor: "test", Source: "file"}); err != nil {
		t.Fatalf("UpsertRegistry failed: %v", err)
	}
	if err := store.UpsertEngineProfile(ctx, MustRegistrySlug("default"), &EngineProfile{Slug: MustEngineProfileSlug("agent")}, SaveOptions{Actor: "test"}); err != nil {
		t.Fatalf("UpsertEngineProfile failed: %v", err)
	}

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected yaml file to exist: %v", err)
	}

	reloaded, err := NewYAMLFileEngineProfileStore(path, MustRegistrySlug("default"))
	if err != nil {
		t.Fatalf("reloaded store init failed: %v", err)
	}
	profiles, err := reloaded.ListEngineProfiles(ctx, MustRegistrySlug("default"))
	if err != nil {
		t.Fatalf("ListEngineProfiles failed: %v", err)
	}
	if len(profiles) != 2 {
		t.Fatalf("expected 2 profiles after reload, got %d", len(profiles))
	}
}

func TestYAMLFileEngineProfileStore_RejectsLegacyFile(t *testing.T) {
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

	_, err := NewYAMLFileEngineProfileStore(path, MustRegistrySlug("default"))
	if err == nil {
		t.Fatalf("expected legacy file decode error")
	}
	if !strings.Contains(err.Error(), "legacy profile-map format is not supported") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestYAMLFileEngineProfileStore_MissingFileInitialization(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "missing", "profiles.yaml")
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected file to be missing before init")
	}

	store, err := NewYAMLFileEngineProfileStore(path, MustRegistrySlug("default"))
	if err != nil {
		t.Fatalf("NewYAMLFileEngineProfileStore failed: %v", err)
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

	reg := &EngineProfileRegistry{
		Slug:                     MustRegistrySlug("default"),
		DefaultEngineProfileSlug: MustEngineProfileSlug("default"),
		Profiles: map[EngineProfileSlug]*EngineProfile{
			MustEngineProfileSlug("default"): {Slug: MustEngineProfileSlug("default")},
		},
	}
	if err := store.UpsertRegistry(ctx, reg, SaveOptions{Actor: "test", Source: "yaml"}); err != nil {
		t.Fatalf("UpsertRegistry failed: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected file after write: %v", err)
	}
}

func TestYAMLFileEngineProfileStore_ParseFailureSurfacing(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "profiles.yaml")
	if err := os.WriteFile(path, []byte("registries:\n  : bad"), 0o644); err != nil {
		t.Fatalf("write malformed yaml failed: %v", err)
	}

	_, err := NewYAMLFileEngineProfileStore(path, MustRegistrySlug("default"))
	if err == nil {
		t.Fatalf("expected parse error for malformed yaml")
	}
	errS := strings.ToLower(err.Error())
	if !strings.Contains(errS, "yaml") && !strings.Contains(errS, "runtime yaml") {
		t.Fatalf("expected parse/runtime yaml error details, got %v", err)
	}
}

func TestYAMLFileEngineProfileStore_RejectsSecondRegistrySlug(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "profiles.yaml")
	ctx := context.Background()

	store, err := NewYAMLFileEngineProfileStore(path, MustRegistrySlug("default"))
	if err != nil {
		t.Fatalf("NewYAMLFileEngineProfileStore failed: %v", err)
	}

	if err := store.UpsertRegistry(ctx, &EngineProfileRegistry{
		Slug:                     MustRegistrySlug("default"),
		DefaultEngineProfileSlug: MustEngineProfileSlug("default"),
		Profiles: map[EngineProfileSlug]*EngineProfile{
			MustEngineProfileSlug("default"): {
				Slug: MustEngineProfileSlug("default"),
				Runtime: RuntimeSpec{
					SystemPrompt: "default profile",
					Tools:        []string{"calculator"},
				},
			},
		},
	}, SaveOptions{Actor: "test", Source: "yaml"}); err != nil {
		t.Fatalf("upsert default registry failed: %v", err)
	}

	err = store.UpsertRegistry(ctx, &EngineProfileRegistry{
		Slug:                     MustRegistrySlug("team"),
		DefaultEngineProfileSlug: MustEngineProfileSlug("agent"),
		Profiles: map[EngineProfileSlug]*EngineProfile{
			MustEngineProfileSlug("agent"): {
				Slug:        MustEngineProfileSlug("agent"),
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

	reloaded, err := NewYAMLFileEngineProfileStore(path, MustRegistrySlug("default"))
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

	_, _, err = reloaded.GetEngineProfile(ctx, MustRegistrySlug("team"), MustEngineProfileSlug("agent"))
	if err == nil {
		t.Fatalf("expected non-default registry lookup to fail")
	}
	if !strings.Contains(err.Error(), "supports only registry") {
		t.Fatalf("unexpected non-default registry lookup error: %v", err)
	}
}

func TestYAMLFileEngineProfileStore_AtomicTempRenameBehavior(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "profiles.yaml")
	ctx := context.Background()

	store, err := NewYAMLFileEngineProfileStore(path, MustRegistrySlug("default"))
	if err != nil {
		t.Fatalf("NewYAMLFileEngineProfileStore failed: %v", err)
	}

	reg := &EngineProfileRegistry{
		Slug:                     MustRegistrySlug("default"),
		DefaultEngineProfileSlug: MustEngineProfileSlug("default"),
		Profiles: map[EngineProfileSlug]*EngineProfile{
			MustEngineProfileSlug("default"): {Slug: MustEngineProfileSlug("default")},
		},
	}
	if err := store.UpsertRegistry(ctx, reg, SaveOptions{Actor: "test", Source: "yaml"}); err != nil {
		t.Fatalf("UpsertRegistry failed: %v", err)
	}
	if _, err := os.Stat(path + ".tmp"); !os.IsNotExist(err) {
		t.Fatalf("expected no tmp file residue after write")
	}

	if err := store.UpsertEngineProfile(ctx, MustRegistrySlug("default"), &EngineProfile{
		Slug: MustEngineProfileSlug("agent"),
	}, SaveOptions{Actor: "test", Source: "yaml"}); err != nil {
		t.Fatalf("UpsertEngineProfile failed: %v", err)
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

func TestYAMLFileEngineProfileStore_CloseStateGuards(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "profiles.yaml")
	ctx := context.Background()

	store, err := NewYAMLFileEngineProfileStore(path, MustRegistrySlug("default"))
	if err != nil {
		t.Fatalf("NewYAMLFileEngineProfileStore failed: %v", err)
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

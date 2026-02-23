package profiles

import (
	"context"
	"os"
	"path/filepath"
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

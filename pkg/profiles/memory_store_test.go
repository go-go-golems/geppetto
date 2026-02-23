package profiles

import (
	"context"
	"errors"
	"testing"
)

func TestInMemoryProfileStore_RegistryLifecycle(t *testing.T) {
	ctx := context.Background()
	store := NewInMemoryProfileStore()

	registry := &ProfileRegistry{
		Slug:               MustRegistrySlug("default"),
		DefaultProfileSlug: MustProfileSlug("default"),
		Profiles: map[ProfileSlug]*Profile{
			MustProfileSlug("default"): {Slug: MustProfileSlug("default")},
		},
	}
	if err := store.UpsertRegistry(ctx, registry, SaveOptions{Actor: "test", Source: "unit"}); err != nil {
		t.Fatalf("UpsertRegistry failed: %v", err)
	}

	got, ok, err := store.GetRegistry(ctx, MustRegistrySlug("default"))
	if err != nil {
		t.Fatalf("GetRegistry failed: %v", err)
	}
	if !ok {
		t.Fatalf("expected registry to exist")
	}
	if got.Metadata.Version == 0 {
		t.Fatalf("expected metadata version to be incremented")
	}

	got.DisplayName = "mutated"
	again, ok, err := store.GetRegistry(ctx, MustRegistrySlug("default"))
	if err != nil || !ok {
		t.Fatalf("GetRegistry failed: ok=%v err=%v", ok, err)
	}
	if again.DisplayName == "mutated" {
		t.Fatalf("expected clone-on-read semantics")
	}
}

func TestInMemoryProfileStore_ProfileLifecycle(t *testing.T) {
	ctx := context.Background()
	store := NewInMemoryProfileStore()

	registry := &ProfileRegistry{
		Slug:               MustRegistrySlug("default"),
		DefaultProfileSlug: MustProfileSlug("default"),
		Profiles: map[ProfileSlug]*Profile{
			MustProfileSlug("default"): {Slug: MustProfileSlug("default")},
		},
	}
	if err := store.UpsertRegistry(ctx, registry, SaveOptions{}); err != nil {
		t.Fatalf("UpsertRegistry failed: %v", err)
	}

	profile := &Profile{
		Slug:        MustProfileSlug("agent"),
		DisplayName: "Agent",
	}
	if err := store.UpsertProfile(ctx, MustRegistrySlug("default"), profile, SaveOptions{Actor: "test"}); err != nil {
		t.Fatalf("UpsertProfile failed: %v", err)
	}

	list, err := store.ListProfiles(ctx, MustRegistrySlug("default"))
	if err != nil {
		t.Fatalf("ListProfiles failed: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 profiles, got %d", len(list))
	}

	if err := store.SetDefaultProfile(ctx, MustRegistrySlug("default"), MustProfileSlug("agent"), SaveOptions{}); err != nil {
		t.Fatalf("SetDefaultProfile failed: %v", err)
	}
	reg, ok, err := store.GetRegistry(ctx, MustRegistrySlug("default"))
	if err != nil || !ok {
		t.Fatalf("GetRegistry failed: ok=%v err=%v", ok, err)
	}
	if reg.DefaultProfileSlug != MustProfileSlug("agent") {
		t.Fatalf("expected default profile=agent, got %q", reg.DefaultProfileSlug)
	}

	if err := store.DeleteProfile(ctx, MustRegistrySlug("default"), MustProfileSlug("agent"), SaveOptions{}); err != nil {
		t.Fatalf("DeleteProfile failed: %v", err)
	}
	_, ok, err = store.GetProfile(ctx, MustRegistrySlug("default"), MustProfileSlug("agent"))
	if err != nil {
		t.Fatalf("GetProfile failed: %v", err)
	}
	if ok {
		t.Fatalf("expected profile to be deleted")
	}
}

func TestInMemoryProfileStore_VersionConflict(t *testing.T) {
	ctx := context.Background()
	store := NewInMemoryProfileStore()

	registry := &ProfileRegistry{
		Slug:               MustRegistrySlug("default"),
		DefaultProfileSlug: MustProfileSlug("default"),
		Profiles: map[ProfileSlug]*Profile{
			MustProfileSlug("default"): {Slug: MustProfileSlug("default")},
		},
	}
	if err := store.UpsertRegistry(ctx, registry, SaveOptions{Actor: "test"}); err != nil {
		t.Fatalf("UpsertRegistry failed: %v", err)
	}

	err := store.UpsertRegistry(ctx, registry, SaveOptions{ExpectedVersion: 999, Actor: "test"})
	if err == nil {
		t.Fatalf("expected version conflict")
	}
	if !errors.Is(err, ErrVersionConflict) {
		t.Fatalf("expected ErrVersionConflict, got %v", err)
	}
}

func TestInMemoryProfileStore_Close(t *testing.T) {
	ctx := context.Background()
	store := NewInMemoryProfileStore()
	if err := store.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}
	if _, _, err := store.GetRegistry(ctx, MustRegistrySlug("default")); err == nil {
		t.Fatalf("expected error after close")
	}
}

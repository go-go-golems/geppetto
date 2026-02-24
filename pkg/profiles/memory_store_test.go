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

func TestInMemoryProfileStore_RegistryMetadataVersionAndAttribution(t *testing.T) {
	ctx := context.Background()
	store := NewInMemoryProfileStore()

	if err := store.UpsertRegistry(ctx, &ProfileRegistry{
		Slug:               MustRegistrySlug("default"),
		DefaultProfileSlug: MustProfileSlug("default"),
		Profiles: map[ProfileSlug]*Profile{
			MustProfileSlug("default"): {Slug: MustProfileSlug("default")},
			MustProfileSlug("agent"):   {Slug: MustProfileSlug("agent")},
		},
	}, SaveOptions{Actor: "alice", Source: "seed"}); err != nil {
		t.Fatalf("UpsertRegistry(create) failed: %v", err)
	}

	regV1, ok, err := store.GetRegistry(ctx, MustRegistrySlug("default"))
	if err != nil || !ok || regV1 == nil {
		t.Fatalf("GetRegistry(v1) failed: ok=%v err=%v", ok, err)
	}
	if regV1.Metadata.Version != 1 {
		t.Fatalf("expected registry version=1, got %d", regV1.Metadata.Version)
	}
	if regV1.Metadata.CreatedBy != "alice" || regV1.Metadata.UpdatedBy != "alice" {
		t.Fatalf("unexpected created/updated by on v1: %+v", regV1.Metadata)
	}
	if regV1.Metadata.Source != "seed" {
		t.Fatalf("unexpected source on v1: %+v", regV1.Metadata)
	}
	if regV1.Metadata.CreatedAtMs == 0 || regV1.Metadata.UpdatedAtMs == 0 {
		t.Fatalf("expected non-zero timestamps on v1")
	}

	if err := store.SetDefaultProfile(ctx, MustRegistrySlug("default"), MustProfileSlug("agent"), SaveOptions{
		ExpectedVersion: regV1.Metadata.Version,
		Actor:           "bob",
		Source:          "api",
	}); err != nil {
		t.Fatalf("SetDefaultProfile failed: %v", err)
	}
	regV2, ok, err := store.GetRegistry(ctx, MustRegistrySlug("default"))
	if err != nil || !ok || regV2 == nil {
		t.Fatalf("GetRegistry(v2) failed: ok=%v err=%v", ok, err)
	}
	if regV2.Metadata.Version != regV1.Metadata.Version+1 {
		t.Fatalf("expected registry version increment on default set, got v1=%d v2=%d", regV1.Metadata.Version, regV2.Metadata.Version)
	}
	if regV2.Metadata.CreatedAtMs != regV1.Metadata.CreatedAtMs {
		t.Fatalf("expected registry CreatedAtMs to remain immutable")
	}
	if regV2.Metadata.UpdatedAtMs < regV1.Metadata.UpdatedAtMs {
		t.Fatalf("expected registry UpdatedAtMs monotonic, v1=%d v2=%d", regV1.Metadata.UpdatedAtMs, regV2.Metadata.UpdatedAtMs)
	}
	if regV2.Metadata.CreatedBy != "alice" || regV2.Metadata.UpdatedBy != "bob" {
		t.Fatalf("unexpected created/updated by on v2: %+v", regV2.Metadata)
	}
	if regV2.Metadata.Source != "api" {
		t.Fatalf("unexpected source on v2: %+v", regV2.Metadata)
	}

	regPatch := regV2.Clone()
	regPatch.DisplayName = "Registry v3"
	if err := store.UpsertRegistry(ctx, regPatch, SaveOptions{
		ExpectedVersion: regV2.Metadata.Version,
		Actor:           "carol",
		Source:          "cli",
	}); err != nil {
		t.Fatalf("UpsertRegistry(update) failed: %v", err)
	}
	regV3, ok, err := store.GetRegistry(ctx, MustRegistrySlug("default"))
	if err != nil || !ok || regV3 == nil {
		t.Fatalf("GetRegistry(v3) failed: ok=%v err=%v", ok, err)
	}
	if regV3.Metadata.Version != regV2.Metadata.Version+1 {
		t.Fatalf("expected registry version increment on upsert update, got v2=%d v3=%d", regV2.Metadata.Version, regV3.Metadata.Version)
	}
	if regV3.Metadata.CreatedAtMs != regV2.Metadata.CreatedAtMs {
		t.Fatalf("expected registry CreatedAtMs immutable between v2 and v3")
	}
	if regV3.Metadata.UpdatedAtMs < regV2.Metadata.UpdatedAtMs {
		t.Fatalf("expected registry UpdatedAtMs monotonic between v2 and v3")
	}
	if regV3.Metadata.CreatedBy != "alice" || regV3.Metadata.UpdatedBy != "carol" {
		t.Fatalf("unexpected created/updated by on v3: %+v", regV3.Metadata)
	}
	if regV3.Metadata.Source != "cli" {
		t.Fatalf("unexpected source on v3: %+v", regV3.Metadata)
	}
}

func TestInMemoryProfileStore_ProfileMetadataVersionAndAttribution(t *testing.T) {
	ctx := context.Background()
	store := NewInMemoryProfileStore()

	if err := store.UpsertRegistry(ctx, &ProfileRegistry{
		Slug:               MustRegistrySlug("default"),
		DefaultProfileSlug: MustProfileSlug("default"),
		Profiles: map[ProfileSlug]*Profile{
			MustProfileSlug("default"): {Slug: MustProfileSlug("default")},
		},
	}, SaveOptions{Actor: "setup", Source: "seed"}); err != nil {
		t.Fatalf("UpsertRegistry failed: %v", err)
	}

	if err := store.UpsertProfile(ctx, MustRegistrySlug("default"), &Profile{
		Slug:        MustProfileSlug("agent"),
		DisplayName: "Agent v1",
	}, SaveOptions{Actor: "alice", Source: "seed"}); err != nil {
		t.Fatalf("UpsertProfile(create) failed: %v", err)
	}
	pV1, ok, err := store.GetProfile(ctx, MustRegistrySlug("default"), MustProfileSlug("agent"))
	if err != nil || !ok || pV1 == nil {
		t.Fatalf("GetProfile(v1) failed: ok=%v err=%v", ok, err)
	}
	if pV1.Metadata.Version != 1 {
		t.Fatalf("expected profile version=1, got %d", pV1.Metadata.Version)
	}
	if pV1.Metadata.CreatedBy != "alice" || pV1.Metadata.UpdatedBy != "alice" {
		t.Fatalf("unexpected created/updated by on pV1: %+v", pV1.Metadata)
	}
	if pV1.Metadata.Source != "seed" {
		t.Fatalf("unexpected source on pV1: %+v", pV1.Metadata)
	}
	if pV1.Metadata.CreatedAtMs == 0 || pV1.Metadata.UpdatedAtMs == 0 {
		t.Fatalf("expected non-zero timestamps on pV1")
	}

	pPatch := pV1.Clone()
	pPatch.DisplayName = "Agent v2"
	if err := store.UpsertProfile(ctx, MustRegistrySlug("default"), pPatch, SaveOptions{
		ExpectedVersion: pV1.Metadata.Version,
		Actor:           "bob",
		Source:          "api",
	}); err != nil {
		t.Fatalf("UpsertProfile(update1) failed: %v", err)
	}
	pV2, ok, err := store.GetProfile(ctx, MustRegistrySlug("default"), MustProfileSlug("agent"))
	if err != nil || !ok || pV2 == nil {
		t.Fatalf("GetProfile(v2) failed: ok=%v err=%v", ok, err)
	}
	if pV2.Metadata.Version != pV1.Metadata.Version+1 {
		t.Fatalf("expected profile version increment on update1, got v1=%d v2=%d", pV1.Metadata.Version, pV2.Metadata.Version)
	}
	if pV2.Metadata.CreatedAtMs != pV1.Metadata.CreatedAtMs {
		t.Fatalf("expected profile CreatedAtMs immutable between v1 and v2")
	}
	if pV2.Metadata.UpdatedAtMs < pV1.Metadata.UpdatedAtMs {
		t.Fatalf("expected profile UpdatedAtMs monotonic between v1 and v2")
	}
	if pV2.Metadata.CreatedBy != "alice" || pV2.Metadata.UpdatedBy != "bob" {
		t.Fatalf("unexpected created/updated by on pV2: %+v", pV2.Metadata)
	}
	if pV2.Metadata.Source != "api" {
		t.Fatalf("unexpected source on pV2: %+v", pV2.Metadata)
	}

	pPatch2 := pV2.Clone()
	pPatch2.DisplayName = "Agent v3"
	if err := store.UpsertProfile(ctx, MustRegistrySlug("default"), pPatch2, SaveOptions{
		ExpectedVersion: pV2.Metadata.Version,
		Actor:           "carol",
		Source:          "cli",
	}); err != nil {
		t.Fatalf("UpsertProfile(update2) failed: %v", err)
	}
	pV3, ok, err := store.GetProfile(ctx, MustRegistrySlug("default"), MustProfileSlug("agent"))
	if err != nil || !ok || pV3 == nil {
		t.Fatalf("GetProfile(v3) failed: ok=%v err=%v", ok, err)
	}
	if pV3.Metadata.Version != pV2.Metadata.Version+1 {
		t.Fatalf("expected profile version increment on update2, got v2=%d v3=%d", pV2.Metadata.Version, pV3.Metadata.Version)
	}
	if pV3.Metadata.CreatedAtMs != pV2.Metadata.CreatedAtMs {
		t.Fatalf("expected profile CreatedAtMs immutable between v2 and v3")
	}
	if pV3.Metadata.UpdatedAtMs < pV2.Metadata.UpdatedAtMs {
		t.Fatalf("expected profile UpdatedAtMs monotonic between v2 and v3")
	}
	if pV3.Metadata.CreatedBy != "alice" || pV3.Metadata.UpdatedBy != "carol" {
		t.Fatalf("unexpected created/updated by on pV3: %+v", pV3.Metadata)
	}
	if pV3.Metadata.Source != "cli" {
		t.Fatalf("unexpected source on pV3: %+v", pV3.Metadata)
	}
}

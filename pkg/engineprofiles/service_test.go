package engineprofiles

import (
	"context"
	"errors"
	"testing"

	aitypes "github.com/go-go-golems/geppetto/pkg/steps/ai/types"
)

func TestStoreRegistryResolve_DefaultProfileFallbackAndMetadata(t *testing.T) {
	ctx := context.Background()
	store := NewInMemoryEngineProfileStore()
	mustUpsertRegistry(t, store, &EngineProfileRegistry{
		Slug:                     MustRegistrySlug("default"),
		DefaultEngineProfileSlug: MustEngineProfileSlug("agent"),
		Metadata:                 RegistryMetadata{Source: "file"},
		Profiles: map[EngineProfileSlug]*EngineProfile{
			MustEngineProfileSlug("agent"): {
				Slug:              MustEngineProfileSlug("agent"),
				InferenceSettings: mustTestInferenceSettings(t, aitypes.ApiTypeOpenAI, "gpt-4o-mini"),
				Metadata:          EngineProfileMetadata{Version: 3, Source: "db"},
			},
		},
	})

	registry := mustNewStoreRegistry(t, store)
	resolved, err := registry.ResolveEngineProfile(ctx, ResolveInput{})
	if err != nil {
		t.Fatalf("ResolveEngineProfile returned error: %v", err)
	}

	if resolved.EngineProfileSlug != MustEngineProfileSlug("agent") {
		t.Fatalf("expected fallback profile=agent, got %q", resolved.EngineProfileSlug)
	}
	if resolved.Metadata["profile.registry"] != "default" {
		t.Fatalf("metadata profile.registry mismatch: %#v", resolved.Metadata)
	}
	if resolved.Metadata["profile.slug"] != "agent" {
		t.Fatalf("metadata profile.slug mismatch: %#v", resolved.Metadata)
	}
	if resolved.Metadata["profile.version"] != uint64(3) {
		t.Fatalf("metadata profile.version mismatch: %#v", resolved.Metadata)
	}
	if resolved.Metadata["profile.source"] != "db" {
		t.Fatalf("metadata profile.source mismatch: %#v", resolved.Metadata)
	}
	lineageRaw, ok := resolved.Metadata["profile.stack.lineage"]
	if !ok {
		t.Fatalf("expected stack lineage metadata key")
	}
	lineage, ok := lineageRaw.([]map[string]any)
	if !ok {
		t.Fatalf("expected stack lineage metadata type, got %T", lineageRaw)
	}
	if got, want := len(lineage), 1; got != want {
		t.Fatalf("lineage length mismatch: got=%d want=%d", got, want)
	}
	if got := lineage[0]["profile_slug"]; got != "agent" {
		t.Fatalf("lineage profile slug mismatch: %#v", lineage)
	}
	if resolved.InferenceSettings == nil || resolved.InferenceSettings.Chat == nil || resolved.InferenceSettings.Chat.Engine == nil {
		t.Fatalf("expected resolved inference settings")
	}
	if got := *resolved.InferenceSettings.Chat.Engine; got != "gpt-4o-mini" {
		t.Fatalf("expected engine, got %q", got)
	}
}

func TestStoreRegistryResolve_UnknownMapping(t *testing.T) {
	ctx := context.Background()
	store := NewInMemoryEngineProfileStore()
	mustUpsertRegistry(t, store, &EngineProfileRegistry{
		Slug:                     MustRegistrySlug("default"),
		DefaultEngineProfileSlug: MustEngineProfileSlug("default"),
		Profiles: map[EngineProfileSlug]*EngineProfile{
			MustEngineProfileSlug("default"): {Slug: MustEngineProfileSlug("default")},
		},
	})

	registry := mustNewStoreRegistry(t, store)

	_, err := registry.ResolveEngineProfile(ctx, ResolveInput{RegistrySlug: MustRegistrySlug("missing")})
	if !errors.Is(err, ErrRegistryNotFound) {
		t.Fatalf("expected ErrRegistryNotFound, got %v", err)
	}

	_, err = registry.ResolveEngineProfile(ctx, ResolveInput{EngineProfileSlug: MustEngineProfileSlug("missing")})
	if !errors.Is(err, ErrProfileNotFound) {
		t.Fatalf("expected ErrProfileNotFound, got %v", err)
	}
}

func TestStoreRegistryResolve_EmptyProfileFallbackToDefaultSlug(t *testing.T) {
	ctx := context.Background()
	store := NewInMemoryEngineProfileStore()
	store.registries[MustRegistrySlug("default")] = (&EngineProfileRegistry{
		Slug: MustRegistrySlug("default"),
		Profiles: map[EngineProfileSlug]*EngineProfile{
			MustEngineProfileSlug("default"): {
				Slug: MustEngineProfileSlug("default"),
			},
		},
	}).Clone()

	registry := mustNewStoreRegistry(t, store)
	resolved, err := registry.ResolveEngineProfile(ctx, ResolveInput{})
	if err != nil {
		t.Fatalf("ResolveEngineProfile returned error: %v", err)
	}
	if got := resolved.EngineProfileSlug; got != MustEngineProfileSlug("default") {
		t.Fatalf("expected fallback profile slug=default, got %q", got)
	}
}

func TestStoreRegistryResolve_StackMetadataLineageOrder(t *testing.T) {
	ctx := context.Background()
	store := NewInMemoryEngineProfileStore()
	mustUpsertRegistry(t, store, &EngineProfileRegistry{
		Slug:                     MustRegistrySlug("default"),
		DefaultEngineProfileSlug: MustEngineProfileSlug("agent"),
		Profiles: map[EngineProfileSlug]*EngineProfile{
			MustEngineProfileSlug("provider"): {
				Slug:              MustEngineProfileSlug("provider"),
				InferenceSettings: mustTestInferenceSettings(t, aitypes.ApiTypeOpenAI, "gpt-4o-mini"),
				Metadata:          EngineProfileMetadata{Version: 11},
			},
			MustEngineProfileSlug("agent"): {
				Slug: MustEngineProfileSlug("agent"),
				Stack: []EngineProfileRef{
					{EngineProfileSlug: MustEngineProfileSlug("provider")},
				},
				Metadata: EngineProfileMetadata{Version: 12},
			},
		},
	})

	registry := mustNewStoreRegistry(t, store)
	resolved, err := registry.ResolveEngineProfile(ctx, ResolveInput{EngineProfileSlug: MustEngineProfileSlug("agent")})
	if err != nil {
		t.Fatalf("ResolveEngineProfile returned error: %v", err)
	}

	lineageRaw := resolved.Metadata["profile.stack.lineage"]
	lineage, ok := lineageRaw.([]map[string]any)
	if !ok {
		t.Fatalf("expected lineage slice metadata, got %T", lineageRaw)
	}
	if got, want := len(lineage), 2; got != want {
		t.Fatalf("lineage length mismatch: got=%d want=%d", got, want)
	}
	if got := lineage[0]["profile_slug"]; got != "provider" {
		t.Fatalf("expected provider lineage first, got %#v", lineage)
	}
	if got := lineage[1]["profile_slug"]; got != "agent" {
		t.Fatalf("expected agent lineage last, got %#v", lineage)
	}
}

func TestStoreRegistryResolve_StackInferenceSettingsIntegration(t *testing.T) {
	ctx := context.Background()
	store := NewInMemoryEngineProfileStore()
	base := mustTestInferenceSettings(t, aitypes.ApiTypeOpenAI, "gpt-4o-mini")
	leaf := mustTestInferenceSettings(t, aitypes.ApiTypeOpenAI, "gpt-5-mini")
	timeoutSeconds := 60
	leaf.Client.TimeoutSeconds = &timeoutSeconds

	mustUpsertRegistry(t, store, &EngineProfileRegistry{
		Slug:                     MustRegistrySlug("default"),
		DefaultEngineProfileSlug: MustEngineProfileSlug("agent"),
		Profiles: map[EngineProfileSlug]*EngineProfile{
			MustEngineProfileSlug("provider"): {
				Slug:              MustEngineProfileSlug("provider"),
				InferenceSettings: base,
			},
			MustEngineProfileSlug("agent"): {
				Slug: MustEngineProfileSlug("agent"),
				Stack: []EngineProfileRef{
					{EngineProfileSlug: MustEngineProfileSlug("provider")},
				},
				InferenceSettings: leaf,
			},
		},
	})

	registry := mustNewStoreRegistry(t, store)
	resolved, err := registry.ResolveEngineProfile(ctx, ResolveInput{EngineProfileSlug: MustEngineProfileSlug("agent")})
	if err != nil {
		t.Fatalf("ResolveEngineProfile returned error: %v", err)
	}

	if got := *resolved.InferenceSettings.Chat.Engine; got != "gpt-5-mini" {
		t.Fatalf("expected leaf engine override, got %q", got)
	}
	if resolved.InferenceSettings.Chat.ApiType == nil || *resolved.InferenceSettings.Chat.ApiType != aitypes.ApiTypeOpenAI {
		t.Fatalf("expected base api type to persist")
	}
}

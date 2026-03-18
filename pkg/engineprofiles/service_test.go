package engineprofiles

import (
	"context"
	"errors"
	"testing"
)

func TestStoreRegistryResolve_DefaultProfileFallbackAndMetadata(t *testing.T) {
	ctx := context.Background()
	store := NewInMemoryProfileStore()
	mustUpsertRegistry(t, store, &ProfileRegistry{
		Slug:               MustRegistrySlug("default"),
		DefaultProfileSlug: MustProfileSlug("agent"),
		Metadata:           RegistryMetadata{Source: "file"},
		Profiles: map[ProfileSlug]*Profile{
			MustProfileSlug("agent"): {
				Slug: MustProfileSlug("agent"),
				Runtime: RuntimeSpec{
					SystemPrompt: "hello",
				},
				Metadata: ProfileMetadata{Version: 3, Source: "db"},
			},
		},
	})

	registry := mustNewStoreRegistry(t, store)
	resolved, err := registry.ResolveEffectiveProfile(ctx, ResolveInput{})
	if err != nil {
		t.Fatalf("ResolveEffectiveProfile returned error: %v", err)
	}

	if resolved.ProfileSlug != MustProfileSlug("agent") {
		t.Fatalf("expected fallback profile=agent, got %q", resolved.ProfileSlug)
	}
	if resolved.RuntimeKey != MustRuntimeKey("agent") {
		t.Fatalf("expected runtime key=agent, got %q", resolved.RuntimeKey)
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
	traceRaw, ok := resolved.Metadata["profile.stack.trace"]
	if !ok {
		t.Fatalf("expected stack trace metadata key")
	}
	if _, ok := traceRaw.(*ProfileTraceDebugPayload); !ok {
		t.Fatalf("expected stack trace debug payload type, got %T", traceRaw)
	}
	if resolved.RuntimeFingerprint == "" {
		t.Fatalf("runtime fingerprint must be non-empty")
	}
	if got := resolved.EffectiveRuntime.SystemPrompt; got != "hello" {
		t.Fatalf("expected system prompt, got %q", got)
	}
}

func TestStoreRegistryResolve_UnknownMapping(t *testing.T) {
	ctx := context.Background()
	store := NewInMemoryProfileStore()
	mustUpsertRegistry(t, store, &ProfileRegistry{
		Slug:               MustRegistrySlug("default"),
		DefaultProfileSlug: MustProfileSlug("default"),
		Profiles: map[ProfileSlug]*Profile{
			MustProfileSlug("default"): {Slug: MustProfileSlug("default")},
		},
	})

	registry := mustNewStoreRegistry(t, store)

	_, err := registry.ResolveEffectiveProfile(ctx, ResolveInput{RegistrySlug: MustRegistrySlug("missing")})
	if !errors.Is(err, ErrRegistryNotFound) {
		t.Fatalf("expected ErrRegistryNotFound, got %v", err)
	}

	_, err = registry.ResolveEffectiveProfile(ctx, ResolveInput{ProfileSlug: MustProfileSlug("missing")})
	if !errors.Is(err, ErrProfileNotFound) {
		t.Fatalf("expected ErrProfileNotFound, got %v", err)
	}
}

func TestStoreRegistryResolve_EmptyProfileFallbackToDefaultSlug(t *testing.T) {
	ctx := context.Background()
	store := NewInMemoryProfileStore()
	store.registries[MustRegistrySlug("default")] = (&ProfileRegistry{
		Slug: MustRegistrySlug("default"),
		Profiles: map[ProfileSlug]*Profile{
			MustProfileSlug("default"): {
				Slug: MustProfileSlug("default"),
				Runtime: RuntimeSpec{
					SystemPrompt: "fallback profile",
				},
			},
		},
	}).Clone()

	registry := mustNewStoreRegistry(t, store)
	resolved, err := registry.ResolveEffectiveProfile(ctx, ResolveInput{})
	if err != nil {
		t.Fatalf("ResolveEffectiveProfile returned error: %v", err)
	}
	if got := resolved.ProfileSlug; got != MustProfileSlug("default") {
		t.Fatalf("expected fallback profile slug=default, got %q", got)
	}
}

func TestStoreRegistryResolve_EmptyProfileWithoutDefaultReturnsValidation(t *testing.T) {
	ctx := context.Background()
	store := NewInMemoryProfileStore()
	store.registries[MustRegistrySlug("default")] = (&ProfileRegistry{
		Slug: MustRegistrySlug("default"),
		Profiles: map[ProfileSlug]*Profile{
			MustProfileSlug("agent"): {Slug: MustProfileSlug("agent")},
		},
	}).Clone()

	registry := mustNewStoreRegistry(t, store)
	_, err := registry.ResolveEffectiveProfile(ctx, ResolveInput{})
	if err == nil {
		t.Fatalf("expected validation error")
	}
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("expected ErrValidation, got %v", err)
	}
	var verr *ValidationError
	if !errors.As(err, &verr) {
		t.Fatalf("expected ValidationError type, got %T", err)
	}
	if got, want := verr.Field, "profile.slug"; got != want {
		t.Fatalf("validation field mismatch: got=%q want=%q", got, want)
	}
}

func TestStoreRegistryResolve_ProfileRuntimeIntegration(t *testing.T) {
	ctx := context.Background()
	store := NewInMemoryProfileStore()
	mustUpsertRegistry(t, store, &ProfileRegistry{
		Slug:               MustRegistrySlug("default"),
		DefaultProfileSlug: MustProfileSlug("default"),
		Profiles: map[ProfileSlug]*Profile{
			MustProfileSlug("default"): {
				Slug: MustProfileSlug("default"),
				Runtime: RuntimeSpec{
					SystemPrompt: "profile prompt",
					Middlewares:  []MiddlewareUse{{Name: "profile-mw"}},
					Tools:        []string{"profile-tool"},
				},
			},
		},
	})

	registry := mustNewStoreRegistry(t, store)
	resolved, err := registry.ResolveEffectiveProfile(ctx, ResolveInput{})
	if err != nil {
		t.Fatalf("ResolveEffectiveProfile returned error: %v", err)
	}

	if got := resolved.EffectiveRuntime.SystemPrompt; got != "profile prompt" {
		t.Fatalf("expected profile system prompt, got %q", got)
	}
	if got := resolved.EffectiveRuntime.Middlewares[0].Name; got != "profile-mw" {
		t.Fatalf("expected profile middleware, got %q", got)
	}
	if got := resolved.EffectiveRuntime.Tools[0]; got != "profile-tool" {
		t.Fatalf("expected profile tool, got %q", got)
	}
}

func TestStoreRegistryResolve_StackMetadataLineageOrder(t *testing.T) {
	ctx := context.Background()
	store := NewInMemoryProfileStore()
	mustUpsertRegistry(t, store, &ProfileRegistry{
		Slug:               MustRegistrySlug("default"),
		DefaultProfileSlug: MustProfileSlug("agent"),
		Profiles: map[ProfileSlug]*Profile{
			MustProfileSlug("provider"): {
				Slug: MustProfileSlug("provider"),
				Runtime: RuntimeSpec{
					SystemPrompt: "provider",
				},
				Metadata: ProfileMetadata{Version: 11},
			},
			MustProfileSlug("agent"): {
				Slug: MustProfileSlug("agent"),
				Stack: []ProfileRef{
					{ProfileSlug: MustProfileSlug("provider")},
				},
				Metadata: ProfileMetadata{Version: 12},
			},
		},
	})

	registry := mustNewStoreRegistry(t, store)
	resolved, err := registry.ResolveEffectiveProfile(ctx, ResolveInput{ProfileSlug: MustProfileSlug("agent")})
	if err != nil {
		t.Fatalf("ResolveEffectiveProfile returned error: %v", err)
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

func TestStoreRegistryResolve_StackRuntimeIntegration(t *testing.T) {
	ctx := context.Background()
	store := NewInMemoryProfileStore()
	mustUpsertRegistry(t, store, &ProfileRegistry{
		Slug:               MustRegistrySlug("default"),
		DefaultProfileSlug: MustProfileSlug("agent"),
		Profiles: map[ProfileSlug]*Profile{
			MustProfileSlug("provider"): {
				Slug: MustProfileSlug("provider"),
				Runtime: RuntimeSpec{
					SystemPrompt: "provider prompt",
					Middlewares:  []MiddlewareUse{{Name: "provider-mw"}},
				},
			},
			MustProfileSlug("agent"): {
				Slug: MustProfileSlug("agent"),
				Stack: []ProfileRef{
					{ProfileSlug: MustProfileSlug("provider")},
				},
				Runtime: RuntimeSpec{
					Tools: []string{"agent-tool"},
				},
			},
		},
	})

	registry := mustNewStoreRegistry(t, store)
	resolved, err := registry.ResolveEffectiveProfile(ctx, ResolveInput{ProfileSlug: MustProfileSlug("agent")})
	if err != nil {
		t.Fatalf("ResolveEffectiveProfile returned error: %v", err)
	}

	if got := resolved.EffectiveRuntime.SystemPrompt; got != "provider prompt" {
		t.Fatalf("expected stacked system prompt inheritance, got %q", got)
	}
	if got := resolved.EffectiveRuntime.Tools; len(got) != 1 || got[0] != "agent-tool" {
		t.Fatalf("expected leaf tools to win, got %#v", got)
	}
	if got := resolved.EffectiveRuntime.Middlewares[0].Name; got != "provider-mw" {
		t.Fatalf("expected provider middleware inheritance, got %q", got)
	}
}

func TestStoreRegistryListRegistries_DeterministicOrdering(t *testing.T) {
	ctx := context.Background()
	store := NewInMemoryProfileStore()
	mustUpsertRegistry(t, store, &ProfileRegistry{
		Slug:               MustRegistrySlug("zeta"),
		DefaultProfileSlug: MustProfileSlug("default"),
		Profiles: map[ProfileSlug]*Profile{
			MustProfileSlug("default"): {Slug: MustProfileSlug("default")},
		},
	})
	mustUpsertRegistry(t, store, &ProfileRegistry{
		Slug:               MustRegistrySlug("alpha"),
		DefaultProfileSlug: MustProfileSlug("default"),
		Profiles: map[ProfileSlug]*Profile{
			MustProfileSlug("default"): {Slug: MustProfileSlug("default")},
		},
	})

	registry := mustNewStoreRegistry(t, store)
	summaries, err := registry.ListRegistries(ctx)
	if err != nil {
		t.Fatalf("ListRegistries returned error: %v", err)
	}
	if len(summaries) != 2 {
		t.Fatalf("expected 2 summaries, got %d", len(summaries))
	}
	if got, want := summaries[0].Slug, MustRegistrySlug("alpha"); got != want {
		t.Fatalf("summary ordering mismatch at index 0: got=%q want=%q", got, want)
	}
	if got, want := summaries[1].Slug, MustRegistrySlug("zeta"); got != want {
		t.Fatalf("summary ordering mismatch at index 1: got=%q want=%q", got, want)
	}
}

func TestResolveEffectiveProfile_FingerprintChangesOnUpstreamLayerVersion(t *testing.T) {
	ctx := context.Background()
	store := NewInMemoryProfileStore()
	mustUpsertRegistry(t, store, &ProfileRegistry{
		Slug:               MustRegistrySlug("default"),
		DefaultProfileSlug: MustProfileSlug("agent"),
		Profiles: map[ProfileSlug]*Profile{
			MustProfileSlug("provider"): {
				Slug:     MustProfileSlug("provider"),
				Metadata: ProfileMetadata{Version: 1},
			},
			MustProfileSlug("agent"): {
				Slug: MustProfileSlug("agent"),
				Stack: []ProfileRef{
					{ProfileSlug: MustProfileSlug("provider")},
				},
			},
		},
	})
	registry := mustNewStoreRegistry(t, store)

	first, err := registry.ResolveEffectiveProfile(ctx, ResolveInput{ProfileSlug: MustProfileSlug("agent")})
	if err != nil {
		t.Fatalf("ResolveEffectiveProfile first call failed: %v", err)
	}

	provider, err := registry.GetProfile(ctx, MustRegistrySlug("default"), MustProfileSlug("provider"))
	if err != nil {
		t.Fatalf("GetProfile(provider) failed: %v", err)
	}
	provider.Metadata.Version = 2
	if err := store.UpsertProfile(ctx, MustRegistrySlug("default"), provider, SaveOptions{Actor: "test", Source: "test"}); err != nil {
		t.Fatalf("UpsertProfile(provider) failed: %v", err)
	}

	second, err := registry.ResolveEffectiveProfile(ctx, ResolveInput{ProfileSlug: MustProfileSlug("agent")})
	if err != nil {
		t.Fatalf("ResolveEffectiveProfile second call failed: %v", err)
	}
	if first.RuntimeFingerprint == second.RuntimeFingerprint {
		t.Fatalf("expected fingerprint change after upstream layer version change")
	}
}

func TestResolveEffectiveProfile_FingerprintChangesOnLayerOrder(t *testing.T) {
	ctx := context.Background()
	store := NewInMemoryProfileStore()
	mustUpsertRegistry(t, store, &ProfileRegistry{
		Slug:               MustRegistrySlug("default"),
		DefaultProfileSlug: MustProfileSlug("agent"),
		Profiles: map[ProfileSlug]*Profile{
			MustProfileSlug("base-a"): {Slug: MustProfileSlug("base-a"), Metadata: ProfileMetadata{Version: 1}},
			MustProfileSlug("base-b"): {Slug: MustProfileSlug("base-b"), Metadata: ProfileMetadata{Version: 1}},
			MustProfileSlug("agent"): {
				Slug: MustProfileSlug("agent"),
				Stack: []ProfileRef{
					{ProfileSlug: MustProfileSlug("base-a")},
					{ProfileSlug: MustProfileSlug("base-b")},
				},
			},
		},
	})
	registry := mustNewStoreRegistry(t, store)

	first, err := registry.ResolveEffectiveProfile(ctx, ResolveInput{ProfileSlug: MustProfileSlug("agent")})
	if err != nil {
		t.Fatalf("ResolveEffectiveProfile first call failed: %v", err)
	}

	agent, err := registry.GetProfile(ctx, MustRegistrySlug("default"), MustProfileSlug("agent"))
	if err != nil {
		t.Fatalf("GetProfile(agent) failed: %v", err)
	}
	agent.Stack = []ProfileRef{
		{ProfileSlug: MustProfileSlug("base-b")},
		{ProfileSlug: MustProfileSlug("base-a")},
	}
	if err := store.UpsertProfile(ctx, MustRegistrySlug("default"), agent, SaveOptions{Actor: "test", Source: "test"}); err != nil {
		t.Fatalf("UpsertProfile(agent) failed: %v", err)
	}

	second, err := registry.ResolveEffectiveProfile(ctx, ResolveInput{ProfileSlug: MustProfileSlug("agent")})
	if err != nil {
		t.Fatalf("ResolveEffectiveProfile second call failed: %v", err)
	}
	if first.RuntimeFingerprint == second.RuntimeFingerprint {
		t.Fatalf("expected fingerprint change after layer order change")
	}
}

func TestResolveEffectiveProfile_FingerprintStableForNonStackProfile(t *testing.T) {
	ctx := context.Background()
	store := NewInMemoryProfileStore()
	mustUpsertRegistry(t, store, &ProfileRegistry{
		Slug:               MustRegistrySlug("default"),
		DefaultProfileSlug: MustProfileSlug("agent"),
		Profiles: map[ProfileSlug]*Profile{
			MustProfileSlug("agent"): {
				Slug: MustProfileSlug("agent"),
				Runtime: RuntimeSpec{
					SystemPrompt: "stable",
				},
			},
		},
	})
	registry := mustNewStoreRegistry(t, store)

	first, err := registry.ResolveEffectiveProfile(ctx, ResolveInput{ProfileSlug: MustProfileSlug("agent")})
	if err != nil {
		t.Fatalf("ResolveEffectiveProfile first call failed: %v", err)
	}
	second, err := registry.ResolveEffectiveProfile(ctx, ResolveInput{ProfileSlug: MustProfileSlug("agent")})
	if err != nil {
		t.Fatalf("ResolveEffectiveProfile second call failed: %v", err)
	}
	if first.RuntimeFingerprint != second.RuntimeFingerprint {
		t.Fatalf("expected stable fingerprint for identical non-stack resolves")
	}
}

func mustNewStoreRegistry(t *testing.T, store ProfileStore) *StoreRegistry {
	t.Helper()
	registry, err := NewStoreRegistry(store, MustRegistrySlug("default"))
	if err != nil {
		t.Fatalf("NewStoreRegistry returned error: %v", err)
	}
	return registry
}

func mustUpsertRegistry(t *testing.T, store *InMemoryProfileStore, registry *ProfileRegistry) {
	t.Helper()
	if err := store.UpsertRegistry(context.Background(), registry, SaveOptions{Actor: "test", Source: "test"}); err != nil {
		t.Fatalf("UpsertRegistry returned error: %v", err)
	}
}

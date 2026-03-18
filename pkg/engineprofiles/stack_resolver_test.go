package engineprofiles

import (
	"context"
	"strings"
	"testing"
)

func TestExpandEngineProfileStack_LinearStack(t *testing.T) {
	registry := mustNewStackTestRegistry(t, &EngineProfileRegistry{
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
				},
			},
		},
	})

	layers, err := registry.ExpandEngineProfileStack(context.Background(), MustRegistrySlug("default"), MustEngineProfileSlug("agent"), StackResolverOptions{})
	if err != nil {
		t.Fatalf("ExpandEngineProfileStack failed: %v", err)
	}

	assertStackLayerOrder(t, layers,
		"default/provider-openai",
		"default/model-gpt4o",
		"default/agent",
	)
}

func TestExpandEngineProfileStack_FanInDedupesByFirstOccurrence(t *testing.T) {
	registry := mustNewStackTestRegistry(t, &EngineProfileRegistry{
		Slug:                     MustRegistrySlug("default"),
		DefaultEngineProfileSlug: MustEngineProfileSlug("agent"),
		Profiles: map[EngineProfileSlug]*EngineProfile{
			MustEngineProfileSlug("provider-openai"): {
				Slug: MustEngineProfileSlug("provider-openai"),
			},
			MustEngineProfileSlug("model-a"): {
				Slug: MustEngineProfileSlug("model-a"),
				Stack: []EngineProfileRef{
					{EngineProfileSlug: MustEngineProfileSlug("provider-openai")},
				},
			},
			MustEngineProfileSlug("model-b"): {
				Slug: MustEngineProfileSlug("model-b"),
				Stack: []EngineProfileRef{
					{EngineProfileSlug: MustEngineProfileSlug("provider-openai")},
				},
			},
			MustEngineProfileSlug("agent"): {
				Slug: MustEngineProfileSlug("agent"),
				Stack: []EngineProfileRef{
					{EngineProfileSlug: MustEngineProfileSlug("model-a")},
					{EngineProfileSlug: MustEngineProfileSlug("model-b")},
					{EngineProfileSlug: MustEngineProfileSlug("provider-openai")},
				},
			},
		},
	})

	layers, err := registry.ExpandEngineProfileStack(context.Background(), MustRegistrySlug("default"), MustEngineProfileSlug("agent"), StackResolverOptions{})
	if err != nil {
		t.Fatalf("ExpandEngineProfileStack failed: %v", err)
	}

	assertStackLayerOrder(t, layers,
		"default/provider-openai",
		"default/model-a",
		"default/model-b",
		"default/agent",
	)
}

func TestExpandEngineProfileStack_CrossRegistryRefs(t *testing.T) {
	registry := mustNewStackTestRegistry(t,
		&EngineProfileRegistry{
			Slug:                     MustRegistrySlug("shared"),
			DefaultEngineProfileSlug: MustEngineProfileSlug("provider-openai"),
			Profiles: map[EngineProfileSlug]*EngineProfile{
				MustEngineProfileSlug("provider-openai"): {
					Slug: MustEngineProfileSlug("provider-openai"),
				},
			},
		},
		&EngineProfileRegistry{
			Slug:                     MustRegistrySlug("default"),
			DefaultEngineProfileSlug: MustEngineProfileSlug("agent"),
			Profiles: map[EngineProfileSlug]*EngineProfile{
				MustEngineProfileSlug("agent"): {
					Slug: MustEngineProfileSlug("agent"),
					Stack: []EngineProfileRef{
						{RegistrySlug: MustRegistrySlug("shared"), EngineProfileSlug: MustEngineProfileSlug("provider-openai")},
					},
				},
			},
		},
	)

	layers, err := registry.ExpandEngineProfileStack(context.Background(), MustRegistrySlug("default"), MustEngineProfileSlug("agent"), StackResolverOptions{})
	if err != nil {
		t.Fatalf("ExpandEngineProfileStack failed: %v", err)
	}

	assertStackLayerOrder(t, layers,
		"shared/provider-openai",
		"default/agent",
	)
}

func TestExpandEngineProfileStack_RejectsCycleWithExplicitChain(t *testing.T) {
	registry := mustNewStackTestRegistry(t, &EngineProfileRegistry{
		Slug:                     MustRegistrySlug("default"),
		DefaultEngineProfileSlug: MustEngineProfileSlug("a"),
		Profiles: map[EngineProfileSlug]*EngineProfile{
			MustEngineProfileSlug("a"): {
				Slug: MustEngineProfileSlug("a"),
				Stack: []EngineProfileRef{
					{EngineProfileSlug: MustEngineProfileSlug("b")},
				},
			},
			MustEngineProfileSlug("b"): {
				Slug: MustEngineProfileSlug("b"),
				Stack: []EngineProfileRef{
					{EngineProfileSlug: MustEngineProfileSlug("a")},
				},
			},
		},
	})

	_, err := registry.ExpandEngineProfileStack(context.Background(), MustRegistrySlug("default"), MustEngineProfileSlug("a"), StackResolverOptions{})
	requireValidationField(t, err, "registry.profiles[b].stack[0]")
	if !strings.Contains(err.Error(), "default/a -> default/b -> default/a") {
		t.Fatalf("expected explicit cycle chain in error, got %v", err)
	}
}

func TestExpandEngineProfileStack_RejectsMissingRegistry(t *testing.T) {
	registry := mustNewStackTestRegistry(t, &EngineProfileRegistry{
		Slug:                     MustRegistrySlug("default"),
		DefaultEngineProfileSlug: MustEngineProfileSlug("agent"),
		Profiles: map[EngineProfileSlug]*EngineProfile{
			MustEngineProfileSlug("agent"): {
				Slug: MustEngineProfileSlug("agent"),
				Stack: []EngineProfileRef{
					{RegistrySlug: MustRegistrySlug("missing"), EngineProfileSlug: MustEngineProfileSlug("provider-openai")},
				},
			},
		},
	})

	_, err := registry.ExpandEngineProfileStack(context.Background(), MustRegistrySlug("default"), MustEngineProfileSlug("agent"), StackResolverOptions{})
	requireValidationField(t, err, "registry.profiles[agent].stack[0]")
	if !strings.Contains(err.Error(), `referenced registry "missing" not found`) {
		t.Fatalf("expected missing registry details, got %v", err)
	}
}

func TestExpandEngineProfileStack_RejectsMissingProfile(t *testing.T) {
	registry := mustNewStackTestRegistry(t, &EngineProfileRegistry{
		Slug:                     MustRegistrySlug("default"),
		DefaultEngineProfileSlug: MustEngineProfileSlug("agent"),
		Profiles: map[EngineProfileSlug]*EngineProfile{
			MustEngineProfileSlug("agent"): {
				Slug: MustEngineProfileSlug("agent"),
				Stack: []EngineProfileRef{
					{EngineProfileSlug: MustEngineProfileSlug("provider-openai")},
				},
			},
		},
	})

	_, err := registry.ExpandEngineProfileStack(context.Background(), MustRegistrySlug("default"), MustEngineProfileSlug("agent"), StackResolverOptions{})
	requireValidationField(t, err, "registry.profiles[agent].stack[0]")
	if !strings.Contains(err.Error(), `referenced profile "provider-openai" not found in registry "default"`) {
		t.Fatalf("expected missing profile details, got %v", err)
	}
}

func TestExpandEngineProfileStack_RejectsMaxDepthBreach(t *testing.T) {
	registry := mustNewStackTestRegistry(t, &EngineProfileRegistry{
		Slug:                     MustRegistrySlug("default"),
		DefaultEngineProfileSlug: MustEngineProfileSlug("a"),
		Profiles: map[EngineProfileSlug]*EngineProfile{
			MustEngineProfileSlug("a"): {
				Slug:  MustEngineProfileSlug("a"),
				Stack: []EngineProfileRef{{EngineProfileSlug: MustEngineProfileSlug("b")}},
			},
			MustEngineProfileSlug("b"): {
				Slug:  MustEngineProfileSlug("b"),
				Stack: []EngineProfileRef{{EngineProfileSlug: MustEngineProfileSlug("c")}},
			},
			MustEngineProfileSlug("c"): {
				Slug: MustEngineProfileSlug("c"),
			},
		},
	})

	_, err := registry.ExpandEngineProfileStack(context.Background(), MustRegistrySlug("default"), MustEngineProfileSlug("a"), StackResolverOptions{MaxDepth: 2})
	requireValidationField(t, err, "registry.profiles[b].stack[0]")
	if !strings.Contains(err.Error(), "max_depth=2") {
		t.Fatalf("expected max depth details, got %v", err)
	}
}

func mustNewStackTestRegistry(t *testing.T, registries ...*EngineProfileRegistry) *StoreRegistry {
	t.Helper()
	store := NewInMemoryEngineProfileStore()
	store.registries = map[RegistrySlug]*EngineProfileRegistry{}
	for _, registry := range registries {
		store.registries[registry.Slug] = registry.Clone()
	}
	ret, err := NewStoreRegistry(store, MustRegistrySlug("default"))
	if err != nil {
		t.Fatalf("NewStoreRegistry failed: %v", err)
	}
	return ret
}

func assertStackLayerOrder(t *testing.T, layers []EngineProfileStackLayer, want ...string) {
	t.Helper()
	if got, expected := len(layers), len(want); got != expected {
		t.Fatalf("unexpected layer length: got=%d want=%d", got, expected)
	}
	for i, layer := range layers {
		got := layer.RegistrySlug.String() + "/" + layer.EngineProfileSlug.String()
		if got != want[i] {
			t.Fatalf("layer order mismatch at %d: got=%q want=%q", i, got, want[i])
		}
	}
}

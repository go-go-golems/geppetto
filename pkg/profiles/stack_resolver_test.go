package profiles

import (
	"context"
	"strings"
	"testing"
)

func TestExpandProfileStack_LinearStack(t *testing.T) {
	registry := mustNewStackTestRegistry(t, &ProfileRegistry{
		Slug:               MustRegistrySlug("default"),
		DefaultProfileSlug: MustProfileSlug("agent"),
		Profiles: map[ProfileSlug]*Profile{
			MustProfileSlug("provider-openai"): {
				Slug: MustProfileSlug("provider-openai"),
			},
			MustProfileSlug("model-gpt4o"): {
				Slug: MustProfileSlug("model-gpt4o"),
				Stack: []ProfileRef{
					{ProfileSlug: MustProfileSlug("provider-openai")},
				},
			},
			MustProfileSlug("agent"): {
				Slug: MustProfileSlug("agent"),
				Stack: []ProfileRef{
					{ProfileSlug: MustProfileSlug("model-gpt4o")},
				},
			},
		},
	})

	layers, err := registry.ExpandProfileStack(context.Background(), MustRegistrySlug("default"), MustProfileSlug("agent"), StackResolverOptions{})
	if err != nil {
		t.Fatalf("ExpandProfileStack failed: %v", err)
	}

	assertStackLayerOrder(t, layers,
		"default/provider-openai",
		"default/model-gpt4o",
		"default/agent",
	)
}

func TestExpandProfileStack_FanInDedupesByFirstOccurrence(t *testing.T) {
	registry := mustNewStackTestRegistry(t, &ProfileRegistry{
		Slug:               MustRegistrySlug("default"),
		DefaultProfileSlug: MustProfileSlug("agent"),
		Profiles: map[ProfileSlug]*Profile{
			MustProfileSlug("provider-openai"): {
				Slug: MustProfileSlug("provider-openai"),
			},
			MustProfileSlug("model-a"): {
				Slug: MustProfileSlug("model-a"),
				Stack: []ProfileRef{
					{ProfileSlug: MustProfileSlug("provider-openai")},
				},
			},
			MustProfileSlug("model-b"): {
				Slug: MustProfileSlug("model-b"),
				Stack: []ProfileRef{
					{ProfileSlug: MustProfileSlug("provider-openai")},
				},
			},
			MustProfileSlug("agent"): {
				Slug: MustProfileSlug("agent"),
				Stack: []ProfileRef{
					{ProfileSlug: MustProfileSlug("model-a")},
					{ProfileSlug: MustProfileSlug("model-b")},
					{ProfileSlug: MustProfileSlug("provider-openai")},
				},
			},
		},
	})

	layers, err := registry.ExpandProfileStack(context.Background(), MustRegistrySlug("default"), MustProfileSlug("agent"), StackResolverOptions{})
	if err != nil {
		t.Fatalf("ExpandProfileStack failed: %v", err)
	}

	assertStackLayerOrder(t, layers,
		"default/provider-openai",
		"default/model-a",
		"default/model-b",
		"default/agent",
	)
}

func TestExpandProfileStack_CrossRegistryRefs(t *testing.T) {
	registry := mustNewStackTestRegistry(t,
		&ProfileRegistry{
			Slug:               MustRegistrySlug("shared"),
			DefaultProfileSlug: MustProfileSlug("provider-openai"),
			Profiles: map[ProfileSlug]*Profile{
				MustProfileSlug("provider-openai"): {
					Slug: MustProfileSlug("provider-openai"),
				},
			},
		},
		&ProfileRegistry{
			Slug:               MustRegistrySlug("default"),
			DefaultProfileSlug: MustProfileSlug("agent"),
			Profiles: map[ProfileSlug]*Profile{
				MustProfileSlug("agent"): {
					Slug: MustProfileSlug("agent"),
					Stack: []ProfileRef{
						{RegistrySlug: MustRegistrySlug("shared"), ProfileSlug: MustProfileSlug("provider-openai")},
					},
				},
			},
		},
	)

	layers, err := registry.ExpandProfileStack(context.Background(), MustRegistrySlug("default"), MustProfileSlug("agent"), StackResolverOptions{})
	if err != nil {
		t.Fatalf("ExpandProfileStack failed: %v", err)
	}

	assertStackLayerOrder(t, layers,
		"shared/provider-openai",
		"default/agent",
	)
}

func TestExpandProfileStack_RejectsCycleWithExplicitChain(t *testing.T) {
	registry := mustNewStackTestRegistry(t, &ProfileRegistry{
		Slug:               MustRegistrySlug("default"),
		DefaultProfileSlug: MustProfileSlug("a"),
		Profiles: map[ProfileSlug]*Profile{
			MustProfileSlug("a"): {
				Slug: MustProfileSlug("a"),
				Stack: []ProfileRef{
					{ProfileSlug: MustProfileSlug("b")},
				},
			},
			MustProfileSlug("b"): {
				Slug: MustProfileSlug("b"),
				Stack: []ProfileRef{
					{ProfileSlug: MustProfileSlug("a")},
				},
			},
		},
	})

	_, err := registry.ExpandProfileStack(context.Background(), MustRegistrySlug("default"), MustProfileSlug("a"), StackResolverOptions{})
	requireValidationField(t, err, "registry.profiles[b].stack[0]")
	if !strings.Contains(err.Error(), "default/a -> default/b -> default/a") {
		t.Fatalf("expected explicit cycle chain in error, got %v", err)
	}
}

func TestExpandProfileStack_RejectsMissingRegistry(t *testing.T) {
	registry := mustNewStackTestRegistry(t, &ProfileRegistry{
		Slug:               MustRegistrySlug("default"),
		DefaultProfileSlug: MustProfileSlug("agent"),
		Profiles: map[ProfileSlug]*Profile{
			MustProfileSlug("agent"): {
				Slug: MustProfileSlug("agent"),
				Stack: []ProfileRef{
					{RegistrySlug: MustRegistrySlug("missing"), ProfileSlug: MustProfileSlug("provider-openai")},
				},
			},
		},
	})

	_, err := registry.ExpandProfileStack(context.Background(), MustRegistrySlug("default"), MustProfileSlug("agent"), StackResolverOptions{})
	requireValidationField(t, err, "registry.profiles[agent].stack[0]")
	if !strings.Contains(err.Error(), `referenced registry "missing" not found`) {
		t.Fatalf("expected missing registry details, got %v", err)
	}
}

func TestExpandProfileStack_RejectsMissingProfile(t *testing.T) {
	registry := mustNewStackTestRegistry(t, &ProfileRegistry{
		Slug:               MustRegistrySlug("default"),
		DefaultProfileSlug: MustProfileSlug("agent"),
		Profiles: map[ProfileSlug]*Profile{
			MustProfileSlug("agent"): {
				Slug: MustProfileSlug("agent"),
				Stack: []ProfileRef{
					{ProfileSlug: MustProfileSlug("provider-openai")},
				},
			},
		},
	})

	_, err := registry.ExpandProfileStack(context.Background(), MustRegistrySlug("default"), MustProfileSlug("agent"), StackResolverOptions{})
	requireValidationField(t, err, "registry.profiles[agent].stack[0]")
	if !strings.Contains(err.Error(), `referenced profile "provider-openai" not found in registry "default"`) {
		t.Fatalf("expected missing profile details, got %v", err)
	}
}

func TestExpandProfileStack_RejectsMaxDepthBreach(t *testing.T) {
	registry := mustNewStackTestRegistry(t, &ProfileRegistry{
		Slug:               MustRegistrySlug("default"),
		DefaultProfileSlug: MustProfileSlug("a"),
		Profiles: map[ProfileSlug]*Profile{
			MustProfileSlug("a"): {
				Slug:  MustProfileSlug("a"),
				Stack: []ProfileRef{{ProfileSlug: MustProfileSlug("b")}},
			},
			MustProfileSlug("b"): {
				Slug:  MustProfileSlug("b"),
				Stack: []ProfileRef{{ProfileSlug: MustProfileSlug("c")}},
			},
			MustProfileSlug("c"): {
				Slug: MustProfileSlug("c"),
			},
		},
	})

	_, err := registry.ExpandProfileStack(context.Background(), MustRegistrySlug("default"), MustProfileSlug("a"), StackResolverOptions{MaxDepth: 2})
	requireValidationField(t, err, "registry.profiles[b].stack[0]")
	if !strings.Contains(err.Error(), "max_depth=2") {
		t.Fatalf("expected max depth details, got %v", err)
	}
}

func mustNewStackTestRegistry(t *testing.T, registries ...*ProfileRegistry) *StoreRegistry {
	t.Helper()
	store := NewInMemoryProfileStore()
	store.registries = map[RegistrySlug]*ProfileRegistry{}
	for _, registry := range registries {
		store.registries[registry.Slug] = registry.Clone()
	}
	ret, err := NewStoreRegistry(store, MustRegistrySlug("default"))
	if err != nil {
		t.Fatalf("NewStoreRegistry failed: %v", err)
	}
	return ret
}

func assertStackLayerOrder(t *testing.T, layers []ProfileStackLayer, want ...string) {
	t.Helper()
	if got, expected := len(layers), len(want); got != expected {
		t.Fatalf("unexpected layer length: got=%d want=%d", got, expected)
	}
	for i, layer := range layers {
		got := layer.RegistrySlug.String() + "/" + layer.ProfileSlug.String()
		if got != want[i] {
			t.Fatalf("layer order mismatch at %d: got=%q want=%q", i, got, want[i])
		}
	}
}

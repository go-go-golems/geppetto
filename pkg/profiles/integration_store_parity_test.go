package profiles

import (
	"context"
	"path/filepath"
	"testing"
)

func TestStoreRegistryStackRefParityAcrossBackends(t *testing.T) {
	type backendFactory struct {
		name  string
		build func(t *testing.T) ProfileStore
	}

	backends := []backendFactory{
		{
			name: "memory",
			build: func(t *testing.T) ProfileStore {
				t.Helper()
				return NewInMemoryProfileStore()
			},
		},
		{
			name: "sqlite",
			build: func(t *testing.T) ProfileStore {
				t.Helper()
				dsn, err := SQLiteProfileDSNForFile(filepath.Join(t.TempDir(), "profiles.db"))
				if err != nil {
					t.Fatalf("SQLiteProfileDSNForFile failed: %v", err)
				}
				store, err := NewSQLiteProfileStore(dsn, MustRegistrySlug("default"))
				if err != nil {
					t.Fatalf("NewSQLiteProfileStore failed: %v", err)
				}
				return store
			},
		},
	}

	for _, backend := range backends {
		backend := backend
		t.Run(backend.name, func(t *testing.T) {
			ctx := context.Background()
			store := backend.build(t)
			t.Cleanup(func() { _ = store.Close() })

			if err := store.UpsertRegistry(ctx, &ProfileRegistry{
				Slug:               MustRegistrySlug("shared"),
				DefaultProfileSlug: MustProfileSlug("mw-observability"),
				Profiles: map[ProfileSlug]*Profile{
					MustProfileSlug("mw-observability"): {
						Slug: MustProfileSlug("mw-observability"),
					},
				},
			}, SaveOptions{Actor: "bootstrap", Source: backend.name}); err != nil {
				t.Fatalf("bootstrap shared UpsertRegistry failed: %v", err)
			}

			if err := store.UpsertRegistry(ctx, &ProfileRegistry{
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
							{RegistrySlug: MustRegistrySlug("shared"), ProfileSlug: MustProfileSlug("mw-observability")},
						},
					},
				},
			}, SaveOptions{Actor: "bootstrap", Source: backend.name}); err != nil {
				t.Fatalf("bootstrap default UpsertRegistry failed: %v", err)
			}

			model, ok, err := store.GetProfile(ctx, MustRegistrySlug("default"), MustProfileSlug("model-gpt4o"))
			if err != nil {
				t.Fatalf("GetProfile(default/model-gpt4o) failed: %v", err)
			}
			if !ok || model == nil {
				t.Fatalf("expected default/model-gpt4o profile")
			}
			if got, want := len(model.Stack), 1; got != want {
				t.Fatalf("model stack length mismatch: got=%d want=%d", got, want)
			}
			if got := model.Stack[0].ProfileSlug; got != MustProfileSlug("provider-openai") {
				t.Fatalf("model stack profile mismatch: got=%q", got)
			}
			if got := model.Stack[0].RegistrySlug; got != "" {
				t.Fatalf("model stack registry should be empty for same-registry ref, got=%q", got)
			}

			agent, ok, err := store.GetProfile(ctx, MustRegistrySlug("default"), MustProfileSlug("agent"))
			if err != nil {
				t.Fatalf("GetProfile(default/agent) failed: %v", err)
			}
			if !ok || agent == nil {
				t.Fatalf("expected default/agent profile")
			}
			if got, want := len(agent.Stack), 2; got != want {
				t.Fatalf("agent stack length mismatch: got=%d want=%d", got, want)
			}
			if got := agent.Stack[0].ProfileSlug; got != MustProfileSlug("model-gpt4o") {
				t.Fatalf("agent stack first profile mismatch: got=%q", got)
			}
			if got := agent.Stack[1].RegistrySlug; got != MustRegistrySlug("shared") {
				t.Fatalf("agent stack second registry mismatch: got=%q", got)
			}
			if got := agent.Stack[1].ProfileSlug; got != MustProfileSlug("mw-observability") {
				t.Fatalf("agent stack second profile mismatch: got=%q", got)
			}
		})
	}
}

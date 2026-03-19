package engineprofiles

import (
	"context"
	"path/filepath"
	"testing"
)

func TestStoreRegistryStackRefParityAcrossBackends(t *testing.T) {
	type backendFactory struct {
		name  string
		build func(t *testing.T) EngineProfileStore
	}

	backends := []backendFactory{
		{
			name: "memory",
			build: func(t *testing.T) EngineProfileStore {
				t.Helper()
				return NewInMemoryEngineProfileStore()
			},
		},
		{
			name: "sqlite",
			build: func(t *testing.T) EngineProfileStore {
				t.Helper()
				dsn, err := SQLiteProfileDSNForFile(filepath.Join(t.TempDir(), "profiles.db"))
				if err != nil {
					t.Fatalf("SQLiteProfileDSNForFile failed: %v", err)
				}
				store, err := NewSQLiteEngineProfileStore(dsn, MustRegistrySlug("default"))
				if err != nil {
					t.Fatalf("NewSQLiteEngineProfileStore failed: %v", err)
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

			if err := store.UpsertRegistry(ctx, &EngineProfileRegistry{
				Slug:                     MustRegistrySlug("shared"),
				DefaultEngineProfileSlug: MustEngineProfileSlug("mw-observability"),
				Profiles: map[EngineProfileSlug]*EngineProfile{
					MustEngineProfileSlug("mw-observability"): {
						Slug: MustEngineProfileSlug("mw-observability"),
					},
				},
			}, SaveOptions{Actor: "bootstrap", Source: backend.name}); err != nil {
				t.Fatalf("bootstrap shared UpsertRegistry failed: %v", err)
			}

			if err := store.UpsertRegistry(ctx, &EngineProfileRegistry{
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
							{RegistrySlug: MustRegistrySlug("shared"), EngineProfileSlug: MustEngineProfileSlug("mw-observability")},
						},
					},
				},
			}, SaveOptions{Actor: "bootstrap", Source: backend.name}); err != nil {
				t.Fatalf("bootstrap default UpsertRegistry failed: %v", err)
			}

			model, ok, err := store.GetEngineProfile(ctx, MustRegistrySlug("default"), MustEngineProfileSlug("model-gpt4o"))
			if err != nil {
				t.Fatalf("GetEngineProfile(default/model-gpt4o) failed: %v", err)
			}
			if !ok || model == nil {
				t.Fatalf("expected default/model-gpt4o profile")
			}
			if got, want := len(model.Stack), 1; got != want {
				t.Fatalf("model stack length mismatch: got=%d want=%d", got, want)
			}
			if got := model.Stack[0].EngineProfileSlug; got != MustEngineProfileSlug("provider-openai") {
				t.Fatalf("model stack profile mismatch: got=%q", got)
			}
			if got := model.Stack[0].RegistrySlug; got != "" {
				t.Fatalf("model stack registry should be empty for same-registry ref, got=%q", got)
			}

			agent, ok, err := store.GetEngineProfile(ctx, MustRegistrySlug("default"), MustEngineProfileSlug("agent"))
			if err != nil {
				t.Fatalf("GetEngineProfile(default/agent) failed: %v", err)
			}
			if !ok || agent == nil {
				t.Fatalf("expected default/agent profile")
			}
			if got, want := len(agent.Stack), 2; got != want {
				t.Fatalf("agent stack length mismatch: got=%d want=%d", got, want)
			}
			if got := agent.Stack[0].EngineProfileSlug; got != MustEngineProfileSlug("model-gpt4o") {
				t.Fatalf("agent stack first profile mismatch: got=%q", got)
			}
			if got := agent.Stack[1].RegistrySlug; got != MustRegistrySlug("shared") {
				t.Fatalf("agent stack second registry mismatch: got=%q", got)
			}
			if got := agent.Stack[1].EngineProfileSlug; got != MustEngineProfileSlug("mw-observability") {
				t.Fatalf("agent stack second profile mismatch: got=%q", got)
			}
		})
	}
}

package profiles

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
)

func TestStoreRegistryLifecycleParityAcrossBackends(t *testing.T) {
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
			name: "yaml",
			build: func(t *testing.T) ProfileStore {
				t.Helper()
				path := filepath.Join(t.TempDir(), "profiles.yaml")
				store, err := NewYAMLFileProfileStore(path, MustRegistrySlug("default"))
				if err != nil {
					t.Fatalf("NewYAMLFileProfileStore failed: %v", err)
				}
				return store
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

			// Bootstrap a baseline registry so service CRUD operations can run.
			if err := store.UpsertRegistry(ctx, &ProfileRegistry{
				Slug:               MustRegistrySlug("default"),
				DefaultProfileSlug: MustProfileSlug("default"),
				Profiles: map[ProfileSlug]*Profile{
					MustProfileSlug("default"): {Slug: MustProfileSlug("default")},
				},
			}, SaveOptions{Actor: "bootstrap", Source: backend.name}); err != nil {
				t.Fatalf("bootstrap UpsertRegistry failed: %v", err)
			}

			service := mustNewStoreRegistry(t, store)

			created, err := service.CreateProfile(ctx, MustRegistrySlug("default"), &Profile{
				Slug:        MustProfileSlug("agent"),
				DisplayName: "Agent v1",
			}, WriteOptions{Actor: "create", Source: backend.name})
			if err != nil {
				t.Fatalf("CreateProfile failed: %v", err)
			}
			if created.Metadata.Version != 1 {
				t.Fatalf("expected created profile version=1, got %d", created.Metadata.Version)
			}

			displayName := "Agent v2"
			updated, err := service.UpdateProfile(ctx, MustRegistrySlug("default"), MustProfileSlug("agent"), ProfilePatch{
				DisplayName: &displayName,
			}, WriteOptions{
				ExpectedVersion: created.Metadata.Version,
				Actor:           "update",
				Source:          backend.name,
			})
			if err != nil {
				t.Fatalf("UpdateProfile failed: %v", err)
			}
			if updated.Metadata.Version != 2 {
				t.Fatalf("expected updated profile version=2, got %d", updated.Metadata.Version)
			}

			if err := service.SetDefaultProfile(ctx, MustRegistrySlug("default"), MustProfileSlug("agent"), WriteOptions{
				Actor:  "default",
				Source: backend.name,
			}); err != nil {
				t.Fatalf("SetDefaultProfile failed: %v", err)
			}

			defaultProfile, err := service.GetProfile(ctx, MustRegistrySlug("default"), MustProfileSlug("default"))
			if err != nil {
				t.Fatalf("GetProfile(default) failed before delete: %v", err)
			}
			if err := service.DeleteProfile(ctx, MustRegistrySlug("default"), MustProfileSlug("default"), WriteOptions{
				ExpectedVersion: defaultProfile.Metadata.Version,
				Actor:           "delete",
				Source:          backend.name,
			}); err != nil {
				t.Fatalf("DeleteProfile(default) failed: %v", err)
			}

			_, err = service.GetProfile(ctx, MustRegistrySlug("default"), MustProfileSlug("default"))
			if !errors.Is(err, ErrProfileNotFound) {
				t.Fatalf("expected deleted default profile to be missing, got %v", err)
			}

			registry, err := service.GetRegistry(ctx, MustRegistrySlug("default"))
			if err != nil {
				t.Fatalf("GetRegistry failed: %v", err)
			}
			if got, want := registry.DefaultProfileSlug, MustProfileSlug("agent"); got != want {
				t.Fatalf("expected default profile slug=%q after lifecycle, got %q", want, got)
			}

			profiles, err := service.ListProfiles(ctx, MustRegistrySlug("default"))
			if err != nil {
				t.Fatalf("ListProfiles failed: %v", err)
			}
			if len(profiles) != 1 {
				t.Fatalf("expected one remaining profile, got %d", len(profiles))
			}
			if got, want := profiles[0].Slug, MustProfileSlug("agent"); got != want {
				t.Fatalf("expected remaining profile slug=%q, got %q", want, got)
			}
			if got := profiles[0].DisplayName; got != "Agent v2" {
				t.Fatalf("expected updated display name to persist, got %q", got)
			}
		})
	}
}

func TestStoreRegistryExtensionParityAcrossBackends(t *testing.T) {
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
			name: "yaml",
			build: func(t *testing.T) ProfileStore {
				t.Helper()
				path := filepath.Join(t.TempDir(), "profiles.yaml")
				store, err := NewYAMLFileProfileStore(path, MustRegistrySlug("default"))
				if err != nil {
					t.Fatalf("NewYAMLFileProfileStore failed: %v", err)
				}
				return store
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
				Slug:               MustRegistrySlug("default"),
				DefaultProfileSlug: MustProfileSlug("default"),
				Profiles: map[ProfileSlug]*Profile{
					MustProfileSlug("default"): {Slug: MustProfileSlug("default")},
				},
			}, SaveOptions{Actor: "bootstrap", Source: backend.name}); err != nil {
				t.Fatalf("bootstrap UpsertRegistry failed: %v", err)
			}

			service := mustNewStoreRegistry(t, store)
			created, err := service.CreateProfile(ctx, MustRegistrySlug("default"), &Profile{
				Slug: MustProfileSlug("agent"),
				Extensions: map[string]any{
					"Vendor.Custom@V1": map[string]any{
						"flags": []any{map[string]any{"enabled": true}},
					},
				},
			}, WriteOptions{Actor: "create", Source: backend.name})
			if err != nil {
				t.Fatalf("CreateProfile failed: %v", err)
			}
			if _, ok := created.Extensions["vendor.custom@v1"]; !ok {
				t.Fatalf("expected canonical extension key after create")
			}

			displayName := "Agent Updated"
			updated, err := service.UpdateProfile(ctx, MustRegistrySlug("default"), MustProfileSlug("agent"), ProfilePatch{
				DisplayName: &displayName,
			}, WriteOptions{
				ExpectedVersion: created.Metadata.Version,
				Actor:           "update",
				Source:          backend.name,
			})
			if err != nil {
				t.Fatalf("UpdateProfile failed: %v", err)
			}

			ext, ok := updated.Extensions["vendor.custom@v1"]
			if !ok {
				t.Fatalf("expected canonical extension key after update")
			}
			enabled := ext.(map[string]any)["flags"].([]any)[0].(map[string]any)["enabled"].(bool)
			if !enabled {
				t.Fatalf("expected unknown extension payload preserved after update")
			}
		})
	}
}

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
			name: "yaml",
			build: func(t *testing.T) ProfileStore {
				t.Helper()
				path := filepath.Join(t.TempDir(), "profiles.yaml")
				store, err := NewYAMLFileProfileStore(path, MustRegistrySlug("default"))
				if err != nil {
					t.Fatalf("NewYAMLFileProfileStore failed: %v", err)
				}
				return store
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

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

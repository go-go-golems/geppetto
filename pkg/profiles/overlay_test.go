package profiles

import (
	"context"
	"testing"
)

type fakeStore struct {
	registries map[RegistrySlug]*ProfileRegistry
	closeCount int
}

func (s *fakeStore) ListRegistries(_ context.Context) ([]*ProfileRegistry, error) {
	out := make([]*ProfileRegistry, 0, len(s.registries))
	for _, reg := range s.registries {
		out = append(out, reg.Clone())
	}
	return out, nil
}

func (s *fakeStore) GetRegistry(_ context.Context, registrySlug RegistrySlug) (*ProfileRegistry, bool, error) {
	reg, ok := s.registries[registrySlug]
	if !ok {
		return nil, false, nil
	}
	return reg.Clone(), true, nil
}

func (s *fakeStore) ListProfiles(ctx context.Context, registrySlug RegistrySlug) ([]*Profile, error) {
	reg, ok, err := s.GetRegistry(ctx, registrySlug)
	if err != nil || !ok || reg == nil {
		return nil, err
	}
	out := make([]*Profile, 0, len(reg.Profiles))
	for _, p := range reg.Profiles {
		out = append(out, p.Clone())
	}
	return out, nil
}

func (s *fakeStore) GetProfile(ctx context.Context, registrySlug RegistrySlug, profileSlug ProfileSlug) (*Profile, bool, error) {
	reg, ok, err := s.GetRegistry(ctx, registrySlug)
	if err != nil || !ok || reg == nil {
		return nil, false, err
	}
	p, ok := reg.Profiles[profileSlug]
	if !ok {
		return nil, false, nil
	}
	return p.Clone(), true, nil
}

func (s *fakeStore) UpsertRegistry(_ context.Context, registry *ProfileRegistry, _ SaveOptions) error {
	if s.registries == nil {
		s.registries = map[RegistrySlug]*ProfileRegistry{}
	}
	s.registries[registry.Slug] = registry.Clone()
	return nil
}

func (s *fakeStore) DeleteRegistry(_ context.Context, registrySlug RegistrySlug, _ SaveOptions) error {
	delete(s.registries, registrySlug)
	return nil
}

func (s *fakeStore) UpsertProfile(_ context.Context, registrySlug RegistrySlug, profile *Profile, _ SaveOptions) error {
	reg, ok := s.registries[registrySlug]
	if !ok {
		reg = &ProfileRegistry{Slug: registrySlug, Profiles: map[ProfileSlug]*Profile{}}
		s.registries[registrySlug] = reg
	}
	if reg.Profiles == nil {
		reg.Profiles = map[ProfileSlug]*Profile{}
	}
	reg.Profiles[profile.Slug] = profile.Clone()
	return nil
}

func (s *fakeStore) DeleteProfile(_ context.Context, registrySlug RegistrySlug, profileSlug ProfileSlug, _ SaveOptions) error {
	reg, ok := s.registries[registrySlug]
	if !ok || reg == nil {
		return nil
	}
	delete(reg.Profiles, profileSlug)
	return nil
}

func (s *fakeStore) SetDefaultProfile(_ context.Context, registrySlug RegistrySlug, profileSlug ProfileSlug, _ SaveOptions) error {
	reg, ok := s.registries[registrySlug]
	if !ok || reg == nil {
		return nil
	}
	reg.DefaultProfileSlug = profileSlug
	return nil
}

func (s *fakeStore) Close() error {
	s.closeCount++
	return nil
}

func TestOverlayStore_MergesLaterSources(t *testing.T) {
	ctx := context.Background()
	base := &fakeStore{registries: map[RegistrySlug]*ProfileRegistry{
		MustRegistrySlug("default"): {
			Slug:               MustRegistrySlug("default"),
			DisplayName:        "Base",
			DefaultProfileSlug: MustProfileSlug("default"),
			Profiles: map[ProfileSlug]*Profile{
				MustProfileSlug("default"): {Slug: MustProfileSlug("default"), Description: "base-default"},
				MustProfileSlug("agent"):   {Slug: MustProfileSlug("agent"), Description: "base-agent"},
			},
		},
	}}
	ovr := &fakeStore{registries: map[RegistrySlug]*ProfileRegistry{
		MustRegistrySlug("default"): {
			Slug:               MustRegistrySlug("default"),
			DisplayName:        "Overlay",
			DefaultProfileSlug: MustProfileSlug("agent"),
			Profiles: map[ProfileSlug]*Profile{
				MustProfileSlug("agent"): {Slug: MustProfileSlug("agent"), Description: "overlay-agent"},
			},
		},
	}}

	s := NewOverlayStore(base, ovr)
	reg, ok, err := s.GetRegistry(ctx, MustRegistrySlug("default"))
	if err != nil {
		t.Fatalf("GetRegistry failed: %v", err)
	}
	if !ok {
		t.Fatalf("expected merged registry")
	}
	if reg.DisplayName != "Overlay" {
		t.Fatalf("display name mismatch: %q", reg.DisplayName)
	}
	if reg.DefaultProfileSlug != MustProfileSlug("agent") {
		t.Fatalf("default profile mismatch: %q", reg.DefaultProfileSlug)
	}
	if got := reg.Profiles[MustProfileSlug("agent")].Description; got != "overlay-agent" {
		t.Fatalf("overlay profile mismatch: %q", got)
	}
	if got := reg.Profiles[MustProfileSlug("default")].Description; got != "base-default" {
		t.Fatalf("base profile missing: %q", got)
	}
}

func TestOverlayStore_ReadOnlyReturnsError(t *testing.T) {
	s := NewOverlayStoreWithWriter(nil)
	err := s.UpsertRegistry(context.Background(), &ProfileRegistry{Slug: MustRegistrySlug("default")}, SaveOptions{})
	if err != ErrReadOnlyStore {
		t.Fatalf("expected ErrReadOnlyStore, got %v", err)
	}
}

func TestOverlayStore_CloseClosesStores(t *testing.T) {
	base := &fakeStore{}
	overlay := &fakeStore{}
	s := NewOverlayStore(base, overlay)
	if err := s.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}
	if base.closeCount == 0 || overlay.closeCount == 0 {
		t.Fatalf("expected close to be called on both stores")
	}
}

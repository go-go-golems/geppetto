package profiles

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type RegistrySourceKind string

const (
	RegistrySourceKindYAML      RegistrySourceKind = "yaml"
	RegistrySourceKindSQLite    RegistrySourceKind = "sqlite"
	RegistrySourceKindSQLiteDSN RegistrySourceKind = "sqlite-dsn"
)

type RegistrySourceSpec struct {
	Raw  string
	Kind RegistrySourceKind
	Path string
	DSN  string
}

type sourceOwner struct {
	spec          RegistrySourceSpec
	label         string
	writable      bool
	service       *StoreRegistry
	registrySlugs []RegistrySlug
	closer        io.Closer
}

// ChainedRegistry routes reads over all loaded registries and writes to owner sources.
// It resolves profile slugs by stack precedence when no explicit registry is provided.
type ChainedRegistry struct {
	aggregate           *StoreRegistry
	aggregateStore      *InMemoryProfileStore
	registryOwners      map[RegistrySlug]*sourceOwner
	precedenceTopFirst  []RegistrySlug
	defaultRegistrySlug RegistrySlug
	sources             []*sourceOwner
}

var _ Registry = (*ChainedRegistry)(nil)

func ParseProfileRegistrySourceEntries(raw string) ([]string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, nil
	}
	parts := strings.Split(trimmed, ",")
	ret := make([]string, 0, len(parts))
	for i, part := range parts {
		entry := strings.TrimSpace(part)
		if entry == "" {
			return nil, fmt.Errorf("profile registry source entry %d is empty", i)
		}
		ret = append(ret, entry)
	}
	return ret, nil
}

func ParseRegistrySourceSpecs(entries []string) ([]RegistrySourceSpec, error) {
	if len(entries) == 0 {
		return nil, nil
	}
	ret := make([]RegistrySourceSpec, 0, len(entries))
	for i, raw := range entries {
		spec, err := parseRegistrySourceSpec(raw)
		if err != nil {
			return nil, fmt.Errorf("parse profile registry source %d: %w", i, err)
		}
		ret = append(ret, spec)
	}
	return ret, nil
}

func NewChainedRegistryFromSourceSpecs(ctx context.Context, specs []RegistrySourceSpec) (*ChainedRegistry, error) {
	if len(specs) == 0 {
		return nil, &ValidationError{Field: "profile-settings.profile-registries", Reason: "must not be empty"}
	}
	owners := make([]*sourceOwner, 0, len(specs))
	cleanup := func() {
		for i := len(owners) - 1; i >= 0; i-- {
			if owners[i] != nil && owners[i].closer != nil {
				_ = owners[i].closer.Close()
			}
		}
	}

	aggregateStore := NewInMemoryProfileStore()
	aggregateStore.registries = map[RegistrySlug]*ProfileRegistry{}
	ownerByRegistry := map[RegistrySlug]*sourceOwner{}

	for _, spec := range specs {
		owner, registries, err := openRegistrySource(ctx, spec)
		if err != nil {
			cleanup()
			return nil, err
		}
		owners = append(owners, owner)

		for _, reg := range registries {
			if reg == nil {
				continue
			}
			if _, exists := ownerByRegistry[reg.Slug]; exists {
				cleanup()
				return nil, fmt.Errorf("duplicate registry slug %q across profile registry sources", reg.Slug)
			}
			owner.registrySlugs = append(owner.registrySlugs, reg.Slug)
			ownerByRegistry[reg.Slug] = owner
			aggregateStore.registries[reg.Slug] = reg.Clone()
		}
	}

	if len(ownerByRegistry) == 0 {
		cleanup()
		return nil, &ValidationError{Field: "profile-settings.profile-registries", Reason: "no registries loaded from sources"}
	}

	defaultRegistrySlug := RegistrySlug("")
	if len(owners) > 0 {
		for i := len(owners) - 1; i >= 0 && defaultRegistrySlug.IsZero(); i-- {
			if owners[i] == nil || len(owners[i].registrySlugs) == 0 {
				continue
			}
			defaultRegistrySlug = owners[i].registrySlugs[0]
		}
	}
	if defaultRegistrySlug.IsZero() {
		cleanup()
		return nil, fmt.Errorf("could not determine default registry from loaded sources")
	}

	aggregate, err := NewStoreRegistry(aggregateStore, defaultRegistrySlug)
	if err != nil {
		cleanup()
		return nil, err
	}

	precedenceTopFirst := make([]RegistrySlug, 0, len(ownerByRegistry))
	for i := len(owners) - 1; i >= 0; i-- {
		owner := owners[i]
		if owner == nil {
			continue
		}
		precedenceTopFirst = append(precedenceTopFirst, owner.registrySlugs...)
	}

	return &ChainedRegistry{
		aggregate:           aggregate,
		aggregateStore:      aggregateStore,
		registryOwners:      ownerByRegistry,
		precedenceTopFirst:  precedenceTopFirst,
		defaultRegistrySlug: defaultRegistrySlug,
		sources:             owners,
	}, nil
}

func (c *ChainedRegistry) Close() error {
	if c == nil {
		return nil
	}
	var errs []error
	for i := len(c.sources) - 1; i >= 0; i-- {
		owner := c.sources[i]
		if owner == nil || owner.closer == nil {
			continue
		}
		if err := owner.closer.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) == 0 {
		return nil
	}
	return errors.Join(errs...)
}

func (c *ChainedRegistry) ListRegistries(ctx context.Context) ([]RegistrySummary, error) {
	return c.aggregate.ListRegistries(ctx)
}

func (c *ChainedRegistry) GetRegistry(ctx context.Context, registrySlug RegistrySlug) (*ProfileRegistry, error) {
	return c.aggregate.GetRegistry(ctx, c.resolveRegistrySlug(registrySlug))
}

func (c *ChainedRegistry) ListProfiles(ctx context.Context, registrySlug RegistrySlug) ([]*Profile, error) {
	return c.aggregate.ListProfiles(ctx, c.resolveRegistrySlug(registrySlug))
}

func (c *ChainedRegistry) GetProfile(ctx context.Context, registrySlug RegistrySlug, profileSlug ProfileSlug) (*Profile, error) {
	return c.aggregate.GetProfile(ctx, c.resolveRegistrySlug(registrySlug), profileSlug)
}

func (c *ChainedRegistry) ResolveEffectiveProfile(ctx context.Context, in ResolveInput) (*ResolvedProfile, error) {
	if c == nil || c.aggregate == nil {
		return nil, fmt.Errorf("profile registry chain is not initialized")
	}

	next := in
	// If neither registry nor profile is specified, resolve against the top-of-stack
	// default registry and let StoreRegistry apply that registry's default profile slug.
	if next.RegistrySlug.IsZero() && next.ProfileSlug.IsZero() {
		next.RegistrySlug = c.defaultRegistrySlug
	}
	if next.RegistrySlug.IsZero() && !next.ProfileSlug.IsZero() {
		lookupSlug := next.ProfileSlug
		registrySlug, err := c.findRegistrySlugForProfile(ctx, lookupSlug)
		if err != nil {
			return nil, err
		}
		next.RegistrySlug = registrySlug
		next.ProfileSlug = lookupSlug
	}
	return c.aggregate.ResolveEffectiveProfile(ctx, next)
}

func (c *ChainedRegistry) CreateProfile(ctx context.Context, registrySlug RegistrySlug, profile *Profile, opts WriteOptions) (*Profile, error) {
	resolvedRegistrySlug := c.resolveRegistrySlug(registrySlug)
	owner, err := c.ownerForRegistry(resolvedRegistrySlug)
	if err != nil {
		return nil, err
	}
	if !owner.writable {
		return nil, c.readOnlyRegistryWriteError(resolvedRegistrySlug, owner)
	}
	created, err := owner.service.CreateProfile(ctx, resolvedRegistrySlug, profile, opts)
	if err != nil {
		return nil, err
	}
	if err := c.refreshRegistryFromOwner(ctx, resolvedRegistrySlug, owner); err != nil {
		return nil, err
	}
	return created, nil
}

func (c *ChainedRegistry) UpdateProfile(ctx context.Context, registrySlug RegistrySlug, profileSlug ProfileSlug, patch ProfilePatch, opts WriteOptions) (*Profile, error) {
	resolvedRegistrySlug := c.resolveRegistrySlug(registrySlug)
	owner, err := c.ownerForRegistry(resolvedRegistrySlug)
	if err != nil {
		return nil, err
	}
	if !owner.writable {
		return nil, c.readOnlyRegistryWriteError(resolvedRegistrySlug, owner)
	}
	updated, err := owner.service.UpdateProfile(ctx, resolvedRegistrySlug, profileSlug, patch, opts)
	if err != nil {
		return nil, err
	}
	if err := c.refreshRegistryFromOwner(ctx, resolvedRegistrySlug, owner); err != nil {
		return nil, err
	}
	return updated, nil
}

func (c *ChainedRegistry) DeleteProfile(ctx context.Context, registrySlug RegistrySlug, profileSlug ProfileSlug, opts WriteOptions) error {
	resolvedRegistrySlug := c.resolveRegistrySlug(registrySlug)
	owner, err := c.ownerForRegistry(resolvedRegistrySlug)
	if err != nil {
		return err
	}
	if !owner.writable {
		return c.readOnlyRegistryWriteError(resolvedRegistrySlug, owner)
	}
	if err := owner.service.DeleteProfile(ctx, resolvedRegistrySlug, profileSlug, opts); err != nil {
		return err
	}
	return c.refreshRegistryFromOwner(ctx, resolvedRegistrySlug, owner)
}

func (c *ChainedRegistry) SetDefaultProfile(ctx context.Context, registrySlug RegistrySlug, profileSlug ProfileSlug, opts WriteOptions) error {
	resolvedRegistrySlug := c.resolveRegistrySlug(registrySlug)
	owner, err := c.ownerForRegistry(resolvedRegistrySlug)
	if err != nil {
		return err
	}
	if !owner.writable {
		return c.readOnlyRegistryWriteError(resolvedRegistrySlug, owner)
	}
	if err := owner.service.SetDefaultProfile(ctx, resolvedRegistrySlug, profileSlug, opts); err != nil {
		return err
	}
	return c.refreshRegistryFromOwner(ctx, resolvedRegistrySlug, owner)
}

func (c *ChainedRegistry) resolveRegistrySlug(slug RegistrySlug) RegistrySlug {
	if !slug.IsZero() {
		return slug
	}
	return c.defaultRegistrySlug
}

func (c *ChainedRegistry) DefaultRegistrySlug() RegistrySlug {
	if c == nil {
		return ""
	}
	return c.defaultRegistrySlug
}

func (c *ChainedRegistry) ownerForRegistry(slug RegistrySlug) (*sourceOwner, error) {
	owner := c.registryOwners[slug]
	if owner == nil {
		return nil, ErrRegistryNotFound
	}
	return owner, nil
}

func (c *ChainedRegistry) refreshRegistryFromOwner(ctx context.Context, registrySlug RegistrySlug, owner *sourceOwner) error {
	if c == nil || c.aggregateStore == nil || owner == nil || owner.service == nil {
		return nil
	}
	reg, err := owner.service.GetRegistry(ctx, registrySlug)
	if err != nil {
		return err
	}
	c.aggregateStore.registries[registrySlug] = reg.Clone()
	return nil
}

func (c *ChainedRegistry) readOnlyRegistryWriteError(slug RegistrySlug, owner *sourceOwner) error {
	label := "source"
	if owner != nil {
		if owner.label != "" {
			label = owner.label
		} else if owner.spec.Raw != "" {
			label = owner.spec.Raw
		}
	}
	return fmt.Errorf("%w: registry %q is backed by read-only %s", ErrReadOnlyStore, slug, label)
}

func (c *ChainedRegistry) findRegistrySlugForProfile(ctx context.Context, profileSlug ProfileSlug) (RegistrySlug, error) {
	for _, registrySlug := range c.precedenceTopFirst {
		_, err := c.aggregate.GetProfile(ctx, registrySlug, profileSlug)
		if err == nil {
			return registrySlug, nil
		}
		if errors.Is(err, ErrProfileNotFound) {
			continue
		}
		if errors.Is(err, ErrRegistryNotFound) {
			continue
		}
		return "", err
	}
	return "", ErrProfileNotFound
}

func parseRegistrySourceSpec(raw string) (RegistrySourceSpec, error) {
	entry := strings.TrimSpace(raw)
	if entry == "" {
		return RegistrySourceSpec{}, fmt.Errorf("empty profile registry source")
	}

	if rest, ok := strings.CutPrefix(entry, "yaml:"); ok {
		path := strings.TrimSpace(rest)
		if path == "" {
			return RegistrySourceSpec{}, fmt.Errorf("yaml profile registry source path is empty")
		}
		return RegistrySourceSpec{Raw: entry, Kind: RegistrySourceKindYAML, Path: path}, nil
	}
	if rest, ok := strings.CutPrefix(entry, "sqlite:"); ok {
		path := strings.TrimSpace(rest)
		if path == "" {
			return RegistrySourceSpec{}, fmt.Errorf("sqlite profile registry source path is empty")
		}
		return RegistrySourceSpec{Raw: entry, Kind: RegistrySourceKindSQLite, Path: path}, nil
	}
	if rest, ok := strings.CutPrefix(entry, "sqlite-dsn:"); ok {
		dsn := strings.TrimSpace(rest)
		if dsn == "" {
			return RegistrySourceSpec{}, fmt.Errorf("sqlite-dsn profile registry source is empty")
		}
		return RegistrySourceSpec{Raw: entry, Kind: RegistrySourceKindSQLiteDSN, DSN: dsn}, nil
	}

	ext := strings.ToLower(filepath.Ext(entry))
	switch ext {
	case ".db", ".sqlite", ".sqlite3":
		return RegistrySourceSpec{Raw: entry, Kind: RegistrySourceKindSQLite, Path: entry}, nil
	}

	isSQLite, err := fileHasSQLiteHeader(entry)
	if err != nil {
		return RegistrySourceSpec{}, err
	}
	if isSQLite {
		return RegistrySourceSpec{Raw: entry, Kind: RegistrySourceKindSQLite, Path: entry}, nil
	}

	return RegistrySourceSpec{Raw: entry, Kind: RegistrySourceKindYAML, Path: entry}, nil
}

func fileHasSQLiteHeader(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	defer func() {
		_ = f.Close()
	}()
	buf := make([]byte, 16)
	n, err := f.Read(buf)
	if err != nil && !errors.Is(err, io.EOF) {
		return false, err
	}
	if n < 15 {
		return false, nil
	}
	return string(buf[:15]) == "SQLite format 3", nil
}

func openRegistrySource(ctx context.Context, spec RegistrySourceSpec) (*sourceOwner, []*ProfileRegistry, error) {
	switch spec.Kind {
	case RegistrySourceKindYAML:
		registries, err := loadRuntimeYAMLSource(spec.Path)
		if err != nil {
			return nil, nil, err
		}
		if len(registries) == 0 {
			return nil, nil, fmt.Errorf("yaml profile registry source %q did not contain a registry", spec.Path)
		}
		store := NewInMemoryProfileStore()
		for _, reg := range registries {
			if reg == nil {
				continue
			}
			store.registries[reg.Slug] = reg.Clone()
		}
		svc, err := NewStoreRegistry(store, registries[0].Slug)
		if err != nil {
			return nil, nil, err
		}
		ret := &sourceOwner{
			spec:     spec,
			label:    spec.Path,
			writable: false,
			service:  svc,
		}
		return ret, registries, nil
	case RegistrySourceKindSQLite:
		dsn, err := SQLiteProfileDSNForFile(spec.Path)
		if err != nil {
			return nil, nil, err
		}
		return openSQLiteSource(ctx, spec, dsn)
	case RegistrySourceKindSQLiteDSN:
		return openSQLiteSource(ctx, spec, spec.DSN)
	default:
		return nil, nil, fmt.Errorf("unsupported profile registry source kind %q", spec.Kind)
	}
}

func openSQLiteSource(ctx context.Context, spec RegistrySourceSpec, dsn string) (*sourceOwner, []*ProfileRegistry, error) {
	store, err := NewSQLiteProfileStore(dsn, MustRegistrySlug("default"))
	if err != nil {
		return nil, nil, err
	}
	svc, err := NewStoreRegistry(store, MustRegistrySlug("default"))
	if err != nil {
		_ = store.Close()
		return nil, nil, err
	}
	regs, err := store.ListRegistries(ctx)
	if err != nil {
		_ = store.Close()
		return nil, nil, err
	}
	sort.Slice(regs, func(i, j int) bool {
		if regs[i] == nil {
			return false
		}
		if regs[j] == nil {
			return true
		}
		return regs[i].Slug < regs[j].Slug
	})
	ret := &sourceOwner{
		spec:     spec,
		label:    dsn,
		writable: true,
		service:  svc,
		closer:   store,
	}
	return ret, regs, nil
}

func loadRuntimeYAMLSource(path string) ([]*ProfileRegistry, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	reg, err := DecodeRuntimeYAMLSingleRegistry(b)
	if err != nil {
		return nil, err
	}
	if reg == nil {
		return nil, nil
	}
	return []*ProfileRegistry{reg}, nil
}

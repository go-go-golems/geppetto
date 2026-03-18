package engineprofiles

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

const sqliteProfilesSchemaV1 = `
CREATE TABLE IF NOT EXISTS profile_registries (
    slug TEXT PRIMARY KEY,
    payload_json TEXT NOT NULL,
    updated_at_ms INTEGER NOT NULL DEFAULT 0
);
`

// SQLiteEngineProfileStore persists registries in a SQLite database.
//
// Storage format intentionally keeps one JSON payload per registry row so the
// domain schema can evolve without SQL column churn while still using durable
// SQLite persistence and migration/versioning controls.
type SQLiteEngineProfileStore struct {
	mu                  sync.RWMutex
	dsn                 string
	defaultRegistrySlug RegistrySlug
	store               *InMemoryEngineProfileStore
	db                  *sql.DB
	closed              bool
}

func NewSQLiteEngineProfileStore(dsn string, defaultRegistrySlug RegistrySlug) (*SQLiteEngineProfileStore, error) {
	if dsn == "" {
		return nil, fmt.Errorf("sqlite profile store: empty dsn")
	}
	if defaultRegistrySlug.IsZero() {
		defaultRegistrySlug = MustRegistrySlug("default")
	}

	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}

	s := &SQLiteEngineProfileStore{
		dsn:                 dsn,
		defaultRegistrySlug: defaultRegistrySlug,
		store:               NewInMemoryEngineProfileStore(),
		db:                  db,
	}
	if err := s.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}
	if err := s.loadFromDB(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

func (s *SQLiteEngineProfileStore) ListRegistries(ctx context.Context) ([]*EngineProfileRegistry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if err := s.ensureOpen(); err != nil {
		return nil, err
	}
	return s.store.ListRegistries(ctx)
}

func (s *SQLiteEngineProfileStore) GetRegistry(ctx context.Context, registrySlug RegistrySlug) (*EngineProfileRegistry, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if err := s.ensureOpen(); err != nil {
		return nil, false, err
	}
	return s.store.GetRegistry(ctx, registrySlug)
}

func (s *SQLiteEngineProfileStore) ListEngineProfiles(ctx context.Context, registrySlug RegistrySlug) ([]*EngineProfile, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if err := s.ensureOpen(); err != nil {
		return nil, err
	}
	return s.store.ListEngineProfiles(ctx, registrySlug)
}

func (s *SQLiteEngineProfileStore) GetEngineProfile(ctx context.Context, registrySlug RegistrySlug, profileSlug EngineProfileSlug) (*EngineProfile, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if err := s.ensureOpen(); err != nil {
		return nil, false, err
	}
	return s.store.GetEngineProfile(ctx, registrySlug, profileSlug)
}

func (s *SQLiteEngineProfileStore) UpsertRegistry(ctx context.Context, registry *EngineProfileRegistry, opts SaveOptions) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensureOpen(); err != nil {
		return err
	}
	if err := s.store.UpsertRegistry(ctx, registry, opts); err != nil {
		return err
	}
	return s.persistRegistryLocked(ctx, registry.Slug)
}

func (s *SQLiteEngineProfileStore) DeleteRegistry(ctx context.Context, registrySlug RegistrySlug, opts SaveOptions) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensureOpen(); err != nil {
		return err
	}
	if err := s.store.DeleteRegistry(ctx, registrySlug, opts); err != nil {
		return err
	}
	return s.deleteRegistryLocked(ctx, registrySlug)
}

func (s *SQLiteEngineProfileStore) UpsertEngineProfile(ctx context.Context, registrySlug RegistrySlug, profile *EngineProfile, opts SaveOptions) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensureOpen(); err != nil {
		return err
	}
	if err := s.store.UpsertEngineProfile(ctx, registrySlug, profile, opts); err != nil {
		return err
	}
	return s.persistRegistryLocked(ctx, registrySlug)
}

func (s *SQLiteEngineProfileStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil
	}
	s.closed = true
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

func (s *SQLiteEngineProfileStore) migrate() error {
	if s.db == nil {
		return fmt.Errorf("sqlite profile store: db is nil")
	}
	if _, err := s.db.Exec("PRAGMA foreign_keys = ON;"); err != nil {
		return err
	}
	if _, err := s.db.Exec(sqliteProfilesSchemaV1); err != nil {
		return err
	}
	return nil
}

func (s *SQLiteEngineProfileStore) loadFromDB() error {
	if s.db == nil {
		return fmt.Errorf("sqlite profile store: db is nil")
	}
	rows, err := s.db.Query(`SELECT slug, payload_json FROM profile_registries ORDER BY slug ASC`)
	if err != nil {
		return err
	}
	defer func() {
		_ = rows.Close()
	}()

	s.store = NewInMemoryEngineProfileStore()
	s.store.registries = map[RegistrySlug]*EngineProfileRegistry{}

	for rows.Next() {
		var rawSlug string
		var payload string
		if err := rows.Scan(&rawSlug, &payload); err != nil {
			return err
		}

		registry := &EngineProfileRegistry{}
		if err := json.Unmarshal([]byte(payload), registry); err != nil {
			return err
		}

		parsedSlug, err := ParseRegistrySlug(rawSlug)
		if err != nil {
			return err
		}
		if registry.Slug.IsZero() {
			registry.Slug = parsedSlug
		}
		if registry.Slug != parsedSlug {
			return fmt.Errorf("sqlite profile store: slug mismatch payload=%q row=%q", registry.Slug, parsedSlug)
		}
		if registry.Profiles == nil {
			registry.Profiles = map[EngineProfileSlug]*EngineProfile{}
		}
		if err := ValidateRegistry(registry); err != nil {
			return err
		}
		s.store.registries[registry.Slug] = registry.Clone()
	}
	return rows.Err()
}

func (s *SQLiteEngineProfileStore) persistRegistryLocked(ctx context.Context, registrySlug RegistrySlug) error {
	registry, ok, err := s.store.GetRegistry(ctx, registrySlug)
	if err != nil {
		return err
	}
	if !ok || registry == nil {
		return s.deleteRegistryLocked(ctx, registrySlug)
	}
	payload, err := json.Marshal(registry)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(
		ctx,
		`INSERT INTO profile_registries (slug, payload_json, updated_at_ms)
VALUES (?, ?, ?)
ON CONFLICT(slug) DO UPDATE SET payload_json = excluded.payload_json, updated_at_ms = excluded.updated_at_ms`,
		registry.Slug.String(),
		string(payload),
		time.Now().UnixMilli(),
	)
	return err
}

func (s *SQLiteEngineProfileStore) deleteRegistryLocked(ctx context.Context, registrySlug RegistrySlug) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM profile_registries WHERE slug = ?`, registrySlug.String())
	return err
}

func (s *SQLiteEngineProfileStore) ensureOpen() error {
	if s.closed {
		return fmt.Errorf("sqlite profile store closed")
	}
	if s.store == nil {
		return fmt.Errorf("sqlite profile store not initialized")
	}
	if s.db == nil {
		return fmt.Errorf("sqlite profile store db is nil")
	}
	return nil
}

func SQLiteProfileDSNForFile(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("sqlite profile store: empty path")
	}
	return fmt.Sprintf("file:%s?_journal_mode=WAL&_busy_timeout=5000&_foreign_keys=on", path), nil
}

package scopeddb

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

type ToolDescription struct {
	Summary        string
	StarterQueries []string
	Notes          []string
}

type ToolDefinitionSpec struct {
	Name        string
	Description ToolDescription
	Tags        []string
	Version     string
}

type ScopeResolver[Scope any] func(ctx context.Context) (Scope, error)

type DatasetSpec[Scope any, Meta any] struct {
	InMemoryPrefix string
	SchemaLabel    string
	SchemaSQL      string
	AllowedObjects []string
	Tool           ToolDefinitionSpec
	DefaultQuery   QueryOptions
	Materialize    func(ctx context.Context, dst *sql.DB, scope Scope) (Meta, error)
}

type BuildResult[Meta any] struct {
	DB      *sql.DB
	Meta    Meta
	Cleanup func() error
}

func OpenInMemory(ctx context.Context, prefix string, schemaLabel string, schemaSQL string) (*sql.DB, error) {
	dsn, err := uniqueInMemoryDSN(prefix)
	if err != nil {
		return nil, err
	}
	return openSQLite(ctx, dsn, schemaLabel, schemaSQL, false)
}

func OpenFile(ctx context.Context, path string, schemaLabel string, schemaSQL string) (*sql.DB, error) {
	p := strings.TrimSpace(path)
	if p == "" {
		return nil, fmt.Errorf("db path is required")
	}
	return openSQLite(ctx, p, schemaLabel, schemaSQL, true)
}

func EnsureSchema(ctx context.Context, db *sql.DB, schemaLabel string, schemaSQL string) error {
	if db == nil {
		return fmt.Errorf("tool db is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	for _, stmt := range SchemaStatements(schemaSQL) {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("ensure %s: %w", schemaLabel, err)
		}
	}
	return nil
}

func SchemaStatements(schemaSQL string) []string {
	parts := strings.Split(schemaSQL, ";")
	stmts := make([]string, 0, len(parts))
	for _, part := range parts {
		stmt := strings.TrimSpace(part)
		if stmt == "" {
			continue
		}
		stmts = append(stmts, stmt+";")
	}
	return stmts
}

func BuildInMemory[Scope any, Meta any](ctx context.Context, spec DatasetSpec[Scope, Meta], scope Scope) (*BuildResult[Meta], error) {
	prefix := strings.TrimSpace(spec.InMemoryPrefix)
	if prefix == "" {
		prefix = strings.TrimSpace(spec.Tool.Name)
	}
	if prefix == "" {
		prefix = "scopeddb"
	}
	db, err := OpenInMemory(ctx, prefix, spec.SchemaLabel, spec.SchemaSQL)
	if err != nil {
		return nil, err
	}
	meta, err := materializeScope(ctx, spec, db, scope)
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	return &BuildResult[Meta]{
		DB:   db,
		Meta: meta,
		Cleanup: func() error {
			return db.Close()
		},
	}, nil
}

func BuildFile[Scope any, Meta any](ctx context.Context, path string, spec DatasetSpec[Scope, Meta], scope Scope) (*BuildResult[Meta], error) {
	db, err := OpenFile(ctx, path, spec.SchemaLabel, spec.SchemaSQL)
	if err != nil {
		return nil, err
	}
	meta, err := materializeScope(ctx, spec, db, scope)
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	return &BuildResult[Meta]{
		DB:   db,
		Meta: meta,
		Cleanup: func() error {
			return db.Close()
		},
	}, nil
}

func materializeScope[Scope any, Meta any](ctx context.Context, spec DatasetSpec[Scope, Meta], db *sql.DB, scope Scope) (Meta, error) {
	var zero Meta
	if spec.Materialize == nil {
		return zero, fmt.Errorf("materialize callback is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return spec.Materialize(ctx, db, scope)
}

func openSQLite(ctx context.Context, dsn string, schemaLabel string, schemaSQL string, ensureDir bool) (*sql.DB, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if ensureDir {
		if err := os.MkdirAll(filepath.Dir(dsn), 0o755); err != nil {
			return nil, fmt.Errorf("create output directory: %w", err)
		}
	}
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	if _, err := db.ExecContext(ctx, `PRAGMA foreign_keys=ON;`); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}
	if err := EnsureSchema(ctx, db, schemaLabel, schemaSQL); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

func uniqueInMemoryDSN(prefix string) (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("randomize in-memory sqlite name: %w", err)
	}
	name := prefix + "_" + hex.EncodeToString(b)
	return "file:" + name + "?mode=memory&cache=shared", nil
}

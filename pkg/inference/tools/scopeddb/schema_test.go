package scopeddb

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
)

func TestOpenInMemoryAppliesSchema(t *testing.T) {
	ctx := context.Background()
	db, err := OpenInMemory(ctx, "scopeddb-test", "test schema", `
CREATE TABLE scope(run_id TEXT PRIMARY KEY);
CREATE TABLE items(id TEXT PRIMARY KEY, value TEXT);
`)
	if err != nil {
		t.Fatalf("OpenInMemory failed: %v", err)
	}
	defer func() { _ = db.Close() }()

	var count int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(1) FROM sqlite_master WHERE type='table' AND name='items'`).Scan(&count); err != nil {
		t.Fatalf("query sqlite_master failed: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected items table to exist, got %d", count)
	}
}

func TestBuildFileMaterializesAndKeepsMeta(t *testing.T) {
	type scope struct {
		Value string
	}
	type meta struct {
		Count int
	}
	spec := DatasetSpec[scope, meta]{
		SchemaLabel: "test schema",
		SchemaSQL: `
CREATE TABLE scope(run_id TEXT PRIMARY KEY);
CREATE TABLE items(id TEXT PRIMARY KEY, value TEXT);
`,
		Materialize: func(ctx context.Context, dst *sql.DB, in scope) (meta, error) {
			if _, err := dst.ExecContext(ctx, `INSERT INTO items(id, value) VALUES(?, ?)`, "item-1", in.Value); err != nil {
				return meta{}, err
			}
			return meta{Count: 1}, nil
		},
	}
	handle, err := BuildFile(context.Background(), filepath.Join(t.TempDir(), "scope.sqlite"), spec, scope{Value: "hello"})
	if err != nil {
		t.Fatalf("BuildFile failed: %v", err)
	}
	defer func() { _ = handle.Cleanup() }()

	if handle.Meta.Count != 1 {
		t.Fatalf("expected meta count 1, got %d", handle.Meta.Count)
	}
	var got string
	if err := handle.DB.QueryRow(`SELECT value FROM items WHERE id='item-1'`).Scan(&got); err != nil {
		t.Fatalf("query materialized item failed: %v", err)
	}
	if got != "hello" {
		t.Fatalf("expected value hello, got %q", got)
	}
}

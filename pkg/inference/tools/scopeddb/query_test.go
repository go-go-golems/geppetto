package scopeddb

import (
	"context"
	"testing"
	"time"
)

func seedQueryRunnerDB(t *testing.T) *QueryRunner {
	t.Helper()
	ctx := context.Background()
	db, err := OpenInMemory(ctx, "query-test", "query schema", `
CREATE TABLE scope(run_id TEXT PRIMARY KEY);
CREATE TABLE items(id TEXT PRIMARY KEY, value TEXT);
`)
	if err != nil {
		t.Fatalf("OpenInMemory failed: %v", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO items(id, value) VALUES('item-1', 'hello'), ('item-2', 'world')`); err != nil {
		t.Fatalf("seed items failed: %v", err)
	}
	runner, err := NewQueryRunner(db, AllowedObjectMap([]string{"scope", "items"}), QueryOptions{MaxRows: 10, Timeout: 2 * time.Second})
	if err != nil {
		t.Fatalf("NewQueryRunner failed: %v", err)
	}
	return runner
}

func TestQueryRunnerAllowsScopedSelect(t *testing.T) {
	runner := seedQueryRunnerDB(t)
	out, err := runner.Run(context.Background(), QueryInput{SQL: `SELECT id, value FROM items ORDER BY id ASC`})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if out.Error != "" {
		t.Fatalf("expected empty error, got %q", out.Error)
	}
	if out.Count != 2 {
		t.Fatalf("expected 2 rows, got %d", out.Count)
	}
}

func TestQueryRunnerRejectsUnsafeQueries(t *testing.T) {
	runner := seedQueryRunnerDB(t)
	for _, tc := range []struct {
		name string
		sql  string
	}{
		{name: "delete", sql: `DELETE FROM items`},
		{name: "multiple", sql: `SELECT * FROM items; SELECT * FROM scope`},
		{name: "disallowed", sql: `SELECT name FROM sqlite_master ORDER BY name ASC`},
	} {
		out, err := runner.Run(context.Background(), QueryInput{SQL: tc.sql})
		if err != nil {
			t.Fatalf("%s: Run failed: %v", tc.name, err)
		}
		if out.Error == "" {
			t.Fatalf("%s: expected error output for %q", tc.name, tc.sql)
		}
	}
}

func TestQueryRunnerRequireOrderBy(t *testing.T) {
	runner := seedQueryRunnerDB(t)
	runner.opts.RequireOrderBy = true
	out, err := runner.Run(context.Background(), QueryInput{SQL: `SELECT id, value FROM items`})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if out.Error == "" {
		t.Fatalf("expected ORDER BY validation error")
	}
}

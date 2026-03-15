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

func seedViewQueryRunnerDB(t *testing.T, allowed []string) *QueryRunner {
	t.Helper()
	ctx := context.Background()
	db, err := OpenInMemory(ctx, "query-view-test", "query view schema", `
CREATE TABLE items(id TEXT PRIMARY KEY, value TEXT, active INTEGER NOT NULL);
CREATE VIEW active_items AS
SELECT id, value
FROM items
WHERE active = 1;
`)
	if err != nil {
		t.Fatalf("OpenInMemory failed: %v", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO items(id, value, active) VALUES('item-1', 'hello', 1), ('item-2', 'world', 0)`); err != nil {
		t.Fatalf("seed items failed: %v", err)
	}
	runner, err := NewQueryRunner(db, AllowedObjectMap(allowed), QueryOptions{MaxRows: 10, Timeout: 2 * time.Second})
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

func TestQueryRunnerAllowsCTEAliasesWhenBaseObjectsAreAllowed(t *testing.T) {
	runner := seedQueryRunnerDB(t)
	out, err := runner.Run(context.Background(), QueryInput{
		SQL: `WITH recent AS (
			SELECT id, value
			FROM items
		)
		SELECT id, value
		FROM recent
		ORDER BY id ASC`,
	})
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

func TestQueryRunnerSupportsStringBindParams(t *testing.T) {
	runner := seedQueryRunnerDB(t)
	out, err := runner.Run(context.Background(), QueryInput{
		SQL:    `SELECT id, value FROM items WHERE id = ? ORDER BY id ASC`,
		Params: []string{"item-2"},
	})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if out.Error != "" {
		t.Fatalf("expected empty error, got %q", out.Error)
	}
	if out.Count != 1 {
		t.Fatalf("expected 1 row, got %d", out.Count)
	}
	if got := out.Rows[0]["id"]; got != "item-2" {
		t.Fatalf("expected item-2, got %v", got)
	}
}

func TestQueryRunnerAllowsReadsThroughPermittedView(t *testing.T) {
	runner := seedViewQueryRunnerDB(t, []string{"active_items"})
	out, err := runner.Run(context.Background(), QueryInput{
		SQL: `SELECT id, value FROM active_items ORDER BY id ASC`,
	})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if out.Error != "" {
		t.Fatalf("expected empty error, got %q", out.Error)
	}
	if out.Count != 1 {
		t.Fatalf("expected 1 row, got %d", out.Count)
	}
	if got := out.Rows[0]["id"]; got != "item-1" {
		t.Fatalf("expected item-1, got %v", got)
	}
}

func TestQueryRunnerStillRejectsDirectBaseTableReadWhenOnlyViewIsAllowed(t *testing.T) {
	runner := seedViewQueryRunnerDB(t, []string{"active_items"})
	out, err := runner.Run(context.Background(), QueryInput{
		SQL: `SELECT id, value FROM items ORDER BY id ASC`,
	})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if out.Error == "" {
		t.Fatalf("expected disallowed table error")
	}
}

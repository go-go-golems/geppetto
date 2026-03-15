package scopeddb

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	"github.com/go-go-golems/geppetto/pkg/inference/tools"
)

func TestBuildDescriptionIncludesAllowedObjectsAndStarterQueries(t *testing.T) {
	desc := BuildDescription(ToolDescription{
		Summary: "Query scoped extraction history from a run-local in-memory SQLite database",
		Notes: []string{
			"Prefer bind params with ? placeholders instead of inline literal values",
		},
		StarterQueries: []string{
			"SELECT * FROM latest_session_arc",
			"SELECT commitment_id FROM commitments ORDER BY commitment_id",
		},
	}, []string{"scope", "commitments", "latest_session_arc"}, QueryOptions{})

	for _, fragment := range []string{
		"Allowed tables/views:",
		"Prefer bind params",
		"latest_session_arc",
		"SELECT commitment_id FROM commitments ORDER BY commitment_id",
	} {
		if !strings.Contains(desc, fragment) {
			t.Fatalf("expected description to contain %q, got %q", fragment, desc)
		}
	}
}

func TestRegisterPrebuiltAndNewLazyRegistrar(t *testing.T) {
	type scope struct {
		Value string
	}
	type meta struct {
		Value string
	}
	spec := DatasetSpec[scope, meta]{
		InMemoryPrefix: "lazy-test",
		SchemaLabel:    "lazy schema",
		SchemaSQL: `
CREATE TABLE scope(run_id TEXT PRIMARY KEY);
CREATE TABLE items(id TEXT PRIMARY KEY, value TEXT);
`,
		AllowedObjects: []string{"scope", "items"},
		Tool: ToolDefinitionSpec{
			Name: "query_items",
			Description: ToolDescription{
				Summary:        "Query items from a scoped database",
				StarterQueries: []string{"SELECT id, value FROM items ORDER BY id"},
			},
			Tags:    []string{"sqlite"},
			Version: "v1",
		},
		DefaultQuery: DefaultQueryOptions(),
		Materialize: func(ctx context.Context, dst *sql.DB, s scope) (meta, error) {
			if _, err := dst.ExecContext(ctx, `INSERT INTO items(id, value) VALUES('item-1', ?)`, s.Value); err != nil {
				return meta{}, err
			}
			return meta(s), nil
		},
	}

	prebuilt, err := BuildInMemory(context.Background(), spec, scope{Value: "from-prebuilt"})
	if err != nil {
		t.Fatalf("BuildInMemory failed: %v", err)
	}
	defer func() { _ = prebuilt.Cleanup() }()

	reg := tools.NewInMemoryToolRegistry()
	if err := RegisterPrebuilt(reg, spec, prebuilt.DB, spec.DefaultQuery); err != nil {
		t.Fatalf("RegisterPrebuilt failed: %v", err)
	}
	if !reg.HasTool("query_items") {
		t.Fatalf("expected prebuilt registry to contain query_items")
	}

	type scopeKey struct{}
	lazyReg := tools.NewInMemoryToolRegistry()
	registrar := NewLazyRegistrar(spec, func(ctx context.Context) (scope, error) {
		v, _ := ctx.Value(scopeKey{}).(scope)
		return v, nil
	}, spec.DefaultQuery)
	if err := registrar(lazyReg); err != nil {
		t.Fatalf("NewLazyRegistrar failed: %v", err)
	}
	def, err := lazyReg.GetTool("query_items")
	if err != nil {
		t.Fatalf("GetTool failed: %v", err)
	}
	result, err := def.Function.ExecuteWithContext(context.WithValue(context.Background(), scopeKey{}, scope{Value: "from-lazy"}), []byte(`{"sql":"SELECT value FROM items ORDER BY id"}`))
	if err != nil {
		t.Fatalf("ExecuteWithContext failed: %v", err)
	}
	out, ok := result.(QueryOutput)
	if !ok {
		t.Fatalf("expected QueryOutput, got %T", result)
	}
	if out.Error != "" || out.Count != 1 {
		t.Fatalf("unexpected lazy query output: %#v", out)
	}
}

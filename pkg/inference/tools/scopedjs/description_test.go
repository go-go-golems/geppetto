package scopedjs

import (
	"strings"
	"testing"
)

func TestBuildDescriptionIncludesManifestAndStateMode(t *testing.T) {
	desc := BuildDescription(ToolDescription{
		Summary: "Execute application automation code against the scoped dbserver runtime",
		Notes: []string{
			"Prefer structured return values over printing JSON to console",
		},
		StarterSnippets: []string{
			"const rows = await db.query('SELECT * FROM users ORDER BY id'); return rows;",
		},
	}, EnvironmentManifest{
		Modules: []ModuleDoc{
			{Name: "fs", Exports: []string{"readFileSync", "writeFileSync"}},
			{Name: "webserver"},
		},
		Globals: []GlobalDoc{
			{Name: "db", Type: "DatabaseClient"},
		},
		Helpers: []HelperDoc{
			{Name: "serve", Signature: "serve(port)"},
		},
		BootstrapFiles: []string{"bootstrap/router.js"},
	}, EvalOptions{
		StateMode: StatePerSession,
	})

	for _, fragment := range []string{
		"Available modules:",
		"fs (readFileSync, writeFileSync)",
		"Available globals: db (DatabaseClient).",
		"Helpers: serve(port).",
		"Preloaded bootstrap files: bootstrap/router.js.",
		"Runtime state persists within the current session.",
		"Prefer structured return values",
		"Starter snippets:",
	} {
		if !strings.Contains(desc, fragment) {
			t.Fatalf("expected description to contain %q, got %q", fragment, desc)
		}
	}
}

func TestBuildDescriptionDefaultsAndFreshStateNote(t *testing.T) {
	desc := BuildDescription(ToolDescription{}, EnvironmentManifest{}, DefaultEvalOptions())
	for _, fragment := range []string{
		"Execute JavaScript inside a prepared scoped runtime.",
		"Use return to provide the final result.",
		"Each call uses a fresh runtime.",
	} {
		if !strings.Contains(desc, fragment) {
			t.Fatalf("expected description to contain %q, got %q", fragment, desc)
		}
	}
}

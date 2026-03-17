package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/dop251/goja"
	gojengine "github.com/go-go-golems/go-go-goja/engine"
	ggjmodules "github.com/go-go-golems/go-go-goja/modules"
	_ "github.com/go-go-golems/go-go-goja/modules/fs"

	"github.com/go-go-golems/geppetto/pkg/inference/tools"
	"github.com/go-go-golems/geppetto/pkg/inference/tools/scopedjs"
)

type webserverModule struct{}

func (m *webserverModule) Name() string { return "webserver" }

func (m *webserverModule) Doc() string {
	return "webserver exposes get(path, payload) and routes() for registering demo HTTP routes."
}

func (m *webserverModule) Loader(_ *goja.Runtime, moduleObj *goja.Object) {
	exports := moduleObj.Get("exports").(*goja.Object)
	routes := []map[string]any{}
	_ = exports.Set("get", func(path string, payload any) map[string]any {
		route := map[string]any{
			"method":  "GET",
			"path":    path,
			"payload": payload,
		}
		routes = append(routes, route)
		return route
	})
	_ = exports.Set("routes", func() []map[string]any {
		return routes
	})
}

type obsidianModule struct{}

func (m *obsidianModule) Name() string { return "obsidian" }

func (m *obsidianModule) Doc() string {
	return "obsidian exposes createNote(title, body) and returns metadata for the created note."
}

func (m *obsidianModule) Loader(_ *goja.Runtime, moduleObj *goja.Object) {
	exports := moduleObj.Get("exports").(*goja.Object)
	_ = exports.Set("createNote", func(title string, body string) map[string]any {
		slug := title
		return map[string]any{
			"title":       title,
			"bodyPreview": body,
			"path":        filepath.ToSlash(filepath.Join("vault", slug+".md")),
		}
	})
}

func main() {
	ctx := context.Background()

	fsModule := ggjmodules.GetModule("fs")
	if fsModule == nil {
		log.Fatal("fs module is not registered")
	}

	scopeDir, err := os.MkdirTemp("", "scopedjs-dbserver-*")
	if err != nil {
		log.Fatalf("create temp dir: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(scopeDir)
	}()

	spec := scopedjs.EnvironmentSpec[string, struct{}]{
		RuntimeLabel: "dbserver-demo",
		Tool: scopedjs.ToolDefinitionSpec{
			Name: "eval_dbserver_demo",
			Description: scopedjs.ToolDescription{
				Summary: "Execute JavaScript in a composed runtime with fs, webserver, obsidian, and db capabilities.",
				Notes: []string{
					"The db global is scoped to this demo and returns a fixed result set.",
					"Use require(\"webserver\") to register demo HTTP routes and require(\"obsidian\") for note metadata helpers.",
				},
				StarterSnippets: []string{
					`const rows = db.query("SELECT * FROM notes"); return rows;`,
					`const webserver = require("webserver"); webserver.get("/notes", { ok: true }); return webserver.routes();`,
				},
			},
			Tags: []string{"javascript", "db", "webserver", "obsidian", "example"},
		},
		DefaultEval: scopedjs.DefaultEvalOptions(),
		Describe: func() (scopedjs.EnvironmentManifest, error) {
			return scopedjs.EnvironmentManifest{
				Modules: []scopedjs.ModuleDoc{
					{Name: "fs"},
					{Name: "webserver", Exports: []string{"get", "routes"}},
					{Name: "obsidian", Exports: []string{"createNote"}},
				},
				Globals: []scopedjs.GlobalDoc{
					{Name: "workspaceRoot", Type: "string"},
					{Name: "db", Type: "object"},
				},
				Helpers: []scopedjs.HelperDoc{
					{Name: "joinPath", Signature: "joinPath(a, b)"},
				},
				BootstrapFiles: []string{"helpers.js"},
			}, nil
		},
		Configure: func(ctx context.Context, b *scopedjs.Builder, root string) (struct{}, error) {
			if err := b.AddNativeModule(fsModule); err != nil {
				return struct{}{}, err
			}
			if err := b.AddNativeModule(&webserverModule{}); err != nil {
				return struct{}{}, err
			}
			if err := b.AddNativeModule(&obsidianModule{}); err != nil {
				return struct{}{}, err
			}
			if err := b.AddGlobal("workspaceRoot", func(ctx *gojengine.RuntimeContext) error {
				return ctx.VM.Set("workspaceRoot", root)
			}, scopedjs.GlobalDoc{
				Type:        "string",
				Description: "Writable temp directory used by the demo runtime.",
			}); err != nil {
				return struct{}{}, err
			}
			if err := b.AddGlobal("db", func(ctx *gojengine.RuntimeContext) error {
				return ctx.VM.Set("db", map[string]any{
					"query": func(sql string) []map[string]any {
						return []map[string]any{
							{"id": 1, "title": "Inbox", "sql": sql},
							{"id": 2, "title": "Projects", "sql": sql},
						}
					},
				})
			}, scopedjs.GlobalDoc{
				Type:        "object",
				Description: "Demo database facade exposing db.query(sql).",
			}); err != nil {
				return struct{}{}, err
			}
			if err := b.AddBootstrapSource("helpers.js", `
function joinPath(a, b) {
  return a.replace(/\/$/, "") + "/" + b.replace(/^\//, "");
}
`); err != nil {
				return struct{}{}, err
			}
			if err := b.AddHelper("joinPath", "joinPath(a, b)", "Join workspace-relative paths in bootstrap code."); err != nil {
				return struct{}{}, err
			}
			return struct{}{}, nil
		},
	}

	handle, err := scopedjs.BuildRuntime(ctx, spec, scopeDir)
	if err != nil {
		log.Fatalf("build runtime: %v", err)
	}
	defer func() {
		_ = handle.Cleanup()
	}()

	registry := tools.NewInMemoryToolRegistry()
	if err := scopedjs.RegisterPrebuilt(registry, spec, handle, scopedjs.EvalOptionOverrides{}); err != nil {
		log.Fatalf("register tool: %v", err)
	}

	def, err := registry.GetTool("eval_dbserver_demo")
	if err != nil {
		log.Fatalf("get tool: %v", err)
	}

	args, err := json.Marshal(scopedjs.EvalInput{
		Code: `
const fs = require("fs");
const webserver = require("webserver");
const obsidian = require("obsidian");

const rows = db.query("SELECT id, title FROM notes ORDER BY id");
const previewPath = joinPath(workspaceRoot, "preview.json");
fs.writeFileSync(previewPath, JSON.stringify(rows));

const note = obsidian.createNote("notes-preview", fs.readFileSync(previewPath));
webserver.get("/notes", {
  count: rows.length,
  notePath: note.path,
});

return {
  previewPath,
  note,
  routes: webserver.routes(),
  rows,
};
`,
	})
	if err != nil {
		log.Fatalf("marshal args: %v", err)
	}

	result, err := def.Function.ExecuteWithContext(ctx, args)
	if err != nil {
		log.Fatalf("execute tool: %v", err)
	}

	out, ok := result.(scopedjs.EvalOutput)
	if !ok {
		log.Fatalf("unexpected result type %T", result)
	}
	fmt.Printf("tool result: %#v\n", out)
}

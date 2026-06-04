package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/dop251/goja"
	ggjmodules "github.com/go-go-golems/go-go-goja/modules"
	_ "github.com/go-go-golems/go-go-goja/modules/fs"
	gojengine "github.com/go-go-golems/go-go-goja/pkg/engine"

	"github.com/go-go-golems/geppetto/cmd/examples/internal/examplecmd"
	"github.com/go-go-golems/geppetto/pkg/inference/tools"
	"github.com/go-go-golems/geppetto/pkg/inference/tools/scopedjs"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	"github.com/go-go-golems/glazed/pkg/types"
	"github.com/spf13/cobra"
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

type runCommand struct {
	*cmds.CommandDescription
}

var _ cmds.GlazeCommand = (*runCommand)(nil)

func newRunCommand() (*runCommand, error) {
	description := cmds.NewCommandDescription(
		"run",
		cmds.WithShort("Execute the composed scoped JavaScript db/server example"),
	)
	return &runCommand{CommandDescription: description}, nil
}

func (c *runCommand) RunIntoGlazeProcessor(ctx context.Context, _ *values.Values, gp middlewares.Processor) error {
	fsModule := ggjmodules.GetModule("fs")
	if fsModule == nil {
		return fmt.Errorf("fs module is not registered")
	}

	scopeDir, err := os.MkdirTemp("", "scopedjs-dbserver-*")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(scopeDir) }()

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
				Helpers:        []scopedjs.HelperDoc{{Name: "joinPath", Signature: "joinPath(a, b)"}},
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
			if err := b.AddGlobal("workspaceRoot", func(ctx *gojengine.RuntimeInitializationContext) error {
				return ctx.VM.Set("workspaceRoot", root)
			}, scopedjs.GlobalDoc{Type: "string", Description: "Writable temp directory used by the demo runtime."}); err != nil {
				return struct{}{}, err
			}
			if err := b.AddGlobal("db", func(ctx *gojengine.RuntimeInitializationContext) error {
				return ctx.VM.Set("db", map[string]any{
					"query": func(sql string) []map[string]any {
						return []map[string]any{
							{"id": 1, "title": "Inbox", "sql": sql},
							{"id": 2, "title": "Projects", "sql": sql},
						}
					},
				})
			}, scopedjs.GlobalDoc{Type: "object", Description: "Demo database facade exposing db.query(sql)."}); err != nil {
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
		return err
	}
	defer func() { _ = handle.Cleanup() }()

	registry := tools.NewInMemoryToolRegistry()
	if err := scopedjs.RegisterPrebuilt(registry, spec, handle, scopedjs.EvalOptionOverrides{}); err != nil {
		return err
	}

	def, err := registry.GetTool("eval_dbserver_demo")
	if err != nil {
		return err
	}

	args, err := json.Marshal(scopedjs.EvalInput{Code: `
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
`})
	if err != nil {
		return err
	}

	result, err := def.Function.ExecuteWithContext(ctx, args)
	if err != nil {
		return err
	}
	out, ok := result.(scopedjs.EvalOutput)
	if !ok {
		return fmt.Errorf("unexpected result type %T", result)
	}
	return gp.AddRow(ctx, types.NewRow(
		types.MRP("tool", "eval_dbserver_demo"),
		types.MRP("workspace", scopeDir),
		types.MRP("result", out.Result),
		types.MRP("console", out.Console),
		types.MRP("duration_ms", out.DurationMs),
		types.MRP("error", out.Error),
	))
}

func main() {
	root := examplecmd.NewRoot("scopedjs-dbserver", "Composed scoped JavaScript db/server example")
	cmd, err := newRunCommand()
	cobra.CheckErr(err)
	cobra.CheckErr(examplecmd.ExecuteSingleCommand(root, "geppetto", cmd))
}

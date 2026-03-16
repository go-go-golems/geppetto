package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	gojengine "github.com/go-go-golems/go-go-goja/engine"
	ggjmodules "github.com/go-go-golems/go-go-goja/modules"
	_ "github.com/go-go-golems/go-go-goja/modules/fs"

	"github.com/go-go-golems/geppetto/pkg/inference/tools"
	"github.com/go-go-golems/geppetto/pkg/inference/tools/scopedjs"
)

func main() {
	ctx := context.Background()

	fsModule := ggjmodules.GetModule("fs")
	if fsModule == nil {
		log.Fatal("fs module is not registered")
	}

	workspaceDir, err := os.MkdirTemp("", "scopedjs-example-*")
	if err != nil {
		log.Fatalf("create temp dir: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(workspaceDir)
	}()

	notePath := filepath.Join(workspaceDir, "note.txt")
	if err := os.WriteFile(notePath, []byte("hello from scopedjs"), 0o644); err != nil {
		log.Fatalf("write note: %v", err)
	}

	spec := scopedjs.EnvironmentSpec[string, struct{}]{
		RuntimeLabel: "fs-demo",
		Tool: scopedjs.ToolDefinitionSpec{
			Name: "eval_fs_demo",
			Description: scopedjs.ToolDescription{
				Summary: "Execute JavaScript against a scoped workspace with fs access.",
				Notes: []string{
					"The workspaceRoot global points at the allowed demo directory.",
				},
				StarterSnippets: []string{
					"const fs = require(\"fs\"); return fs.readFileSync(input.path);",
				},
			},
			Tags: []string{"javascript", "fs", "example"},
		},
		DefaultEval: scopedjs.DefaultEvalOptions(),
		Configure: func(ctx context.Context, b *scopedjs.Builder, root string) (struct{}, error) {
			if err := b.AddNativeModule(fsModule); err != nil {
				return struct{}{}, err
			}
			if err := b.AddGlobal("workspaceRoot", func(ctx *gojengine.RuntimeContext) error {
				return ctx.VM.Set("workspaceRoot", root)
			}, scopedjs.GlobalDoc{
				Type:        "string",
				Description: "Scoped root directory for the demo workspace.",
			}); err != nil {
				return struct{}{}, err
			}
			if err := b.AddBootstrapSource("helpers.js", `
function joinPath(a, b) {
  return a + "/" + b;
}
`); err != nil {
				return struct{}{}, err
			}
			return struct{}{}, nil
		},
	}

	handle, err := scopedjs.BuildRuntime(ctx, spec, workspaceDir)
	if err != nil {
		log.Fatalf("build runtime: %v", err)
	}
	defer func() {
		_ = handle.Cleanup()
	}()

	registry := tools.NewInMemoryToolRegistry()
	if err := scopedjs.RegisterPrebuilt(registry, spec, handle, scopedjs.EvalOptions{}); err != nil {
		log.Fatalf("register tool: %v", err)
	}

	def, err := registry.GetTool("eval_fs_demo")
	if err != nil {
		log.Fatalf("get tool: %v", err)
	}

	args, err := json.Marshal(scopedjs.EvalInput{
		Code: `
const fs = require("fs");
console.log("workspace", workspaceRoot);
return {
  helperPath: joinPath("notes", "daily.txt"),
  content: fs.readFileSync(input.path),
  workspaceRoot,
};
`,
		Input: map[string]any{
			"path": notePath,
		},
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

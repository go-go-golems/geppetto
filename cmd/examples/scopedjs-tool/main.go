package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	gojengine "github.com/go-go-golems/go-go-goja/engine"
	ggjmodules "github.com/go-go-golems/go-go-goja/modules"
	_ "github.com/go-go-golems/go-go-goja/modules/fs"

	"github.com/go-go-golems/geppetto/cmd/examples/internal/examplecmd"
	"github.com/go-go-golems/geppetto/pkg/inference/tools"
	"github.com/go-go-golems/geppetto/pkg/inference/tools/scopedjs"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	"github.com/go-go-golems/glazed/pkg/types"
	"github.com/spf13/cobra"
)

type runCommand struct {
	*cmds.CommandDescription
}

var _ cmds.GlazeCommand = (*runCommand)(nil)

func newRunCommand() (*runCommand, error) {
	description := cmds.NewCommandDescription(
		"run",
		cmds.WithShort("Execute the scoped JavaScript fs tool example"),
	)
	return &runCommand{CommandDescription: description}, nil
}

func (c *runCommand) RunIntoGlazeProcessor(ctx context.Context, _ *values.Values, gp middlewares.Processor) error {
	fsModule := ggjmodules.GetModule("fs")
	if fsModule == nil {
		return os.ErrNotExist
	}

	workspaceDir, err := os.MkdirTemp("", "scopedjs-example-*")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(workspaceDir) }()

	notePath := filepath.Join(workspaceDir, "note.txt")
	if err := os.WriteFile(notePath, []byte("hello from scopedjs"), 0o644); err != nil {
		return err
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
		Describe: func() (scopedjs.EnvironmentManifest, error) {
			return scopedjs.EnvironmentManifest{
				Modules:        []scopedjs.ModuleDoc{{Name: "fs"}},
				Globals:        []scopedjs.GlobalDoc{{Name: "workspaceRoot", Type: "string"}},
				BootstrapFiles: []string{"helpers.js"},
			}, nil
		},
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
		return err
	}
	defer func() { _ = handle.Cleanup() }()

	registry := tools.NewInMemoryToolRegistry()
	if err := scopedjs.RegisterPrebuilt(registry, spec, handle, scopedjs.EvalOptionOverrides{}); err != nil {
		return err
	}

	def, err := registry.GetTool("eval_fs_demo")
	if err != nil {
		return err
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
		types.MRP("tool", "eval_fs_demo"),
		types.MRP("workspace", workspaceDir),
		types.MRP("result", out.Result),
		types.MRP("console", out.Console),
		types.MRP("duration_ms", out.DurationMs),
		types.MRP("error", out.Error),
	))
}

func main() {
	root := examplecmd.NewRoot("scopedjs-tool", "Scoped JavaScript tool example")
	cmd, err := newRunCommand()
	cobra.CheckErr(err)
	cobra.CheckErr(examplecmd.ExecuteSingleCommand(root, "geppetto", cmd))
}

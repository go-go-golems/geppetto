//go:build ignore

// Experiment 06: Glazed CLI Integration
//
// Shows how the runner could integrate with glazed for structured output.
// This combines the runner's simplicity with glazed's output formatting
// to build CLI tools that can output JSON, YAML, CSV, table, etc.
//
// This file is a design sketch and is excluded from normal builds.

package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/go-go-golems/geppetto/pkg/runner"
	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/spf13/cobra"
)

// --- The dream: a powerful CLI tool in ~50 lines ---

type AnalyzeCommand struct {
	description *cmds.CommandDescription
}

func NewAnalyzeCommand() *AnalyzeCommand {
	return &AnalyzeCommand{
		description: cmds.NewCommandDescription("analyze",
			cmds.WithShort("Analyze code using LLM with tools"),
			cmds.WithFlags(
				fields.New("prompt", fields.TypeString,
					fields.WithHelp("What to analyze"),
					fields.WithRequired(true),
				),
				fields.New("dir", fields.TypeString,
					fields.WithHelp("Directory to analyze"),
					fields.WithDefault("."),
				),
			),
		),
	}
}

func (c *AnalyzeCommand) RunIntoWriter(ctx context.Context, parsed *values.Values, w io.Writer) error {
	var settings struct {
		Prompt string `glazed:"prompt"`
		Dir    string `glazed:"dir"`
	}
	if err := parsed.DecodeSectionInto(schema.DefaultSlug, &settings); err != nil {
		return err
	}

	// The runner handles all LLM infrastructure.
	// We just declare tools and prompts.
	result, err := runner.Run(ctx, settings.Prompt,
		runner.System(fmt.Sprintf("You are a code analyst. Analyze code in %s.", settings.Dir)),
		runner.Tool("read_file", "Read a source file", func(input struct {
			Path string `json:"path"`
		}) (string, error) {
			data, err := os.ReadFile(input.Path)
			return string(data), err
		}),
		runner.Tool("list_files", "List files matching a pattern", func(input struct {
			Pattern string `json:"pattern"`
		}) ([]string, error) {
			return filepath.Glob(input.Pattern)
		}),
		runner.Tool("search", "Search for patterns in code", func(input struct {
			Pattern string `json:"pattern"`
			Dir     string `json:"dir"`
		}) (string, error) {
			out, _ := exec.Command("rg", "--json", "-m5", input.Pattern, input.Dir).Output()
			return string(out), nil
		}),
		runner.MaxTools(20),
		runner.Stream(func(delta string) { fmt.Fprint(w, delta) }),
	)
	if err != nil {
		return err
	}

	// Print tool call summary
	fmt.Fprintf(w, "\n\n---\nTool calls: %d | Input tokens: %d | Output tokens: %d\n",
		len(result.ToolCalls), result.Usage.InputTokens, result.Usage.OutputTokens)

	return nil
}

func main() {
	cmd := NewAnalyzeCommand()
	cobraCmd, _ := cli.BuildCobraCommand(cmd, cli.WithParserConfig(cli.CobraParserConfig{
		ShortHelpSections: []string{schema.DefaultSlug},
		MiddlewaresFunc:   cli.CobraCommandDefaultMiddlewares,
	}))
	cobraCmd.Execute()
}

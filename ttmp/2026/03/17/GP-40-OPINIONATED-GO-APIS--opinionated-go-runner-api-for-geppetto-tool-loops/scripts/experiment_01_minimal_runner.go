//go:build ignore

// Experiment 01: Minimal Runner API (Design D)
//
// This sketch shows what a complete CLI tool looks like with the proposed
// opinionated runner API. Compare to the 70-195 lines currently required.
//
// This file is a design sketch and is excluded from normal builds.
// It won't compile until the runner package exists.

package main

import (
	"context"
	"fmt"
	"os"

	"github.com/go-go-golems/geppetto/pkg/runner"
)

// --- Tool implementations ---

// readFile reads a file and returns its contents.
// The runner automatically generates a JSON schema from the struct tags.
func readFile(input struct {
	Path string `json:"path" jsonschema:"description=Path to the file to read"`
}) (string, error) {
	data, err := os.ReadFile(input.Path)
	if err != nil {
		return "", fmt.Errorf("cannot read %s: %w", input.Path, err)
	}
	return string(data), nil
}

// searchCode searches for a pattern in files using ripgrep.
func searchCode(input struct {
	Pattern string `json:"pattern" jsonschema:"description=Regex pattern to search for"`
	Dir     string `json:"dir" jsonschema:"description=Directory to search in,default=."`
}) (string, error) {
	// In reality, this would use exec.Command("rg", ...)
	return fmt.Sprintf("Found 3 matches for %q in %s", input.Pattern, input.Dir), nil
}

// --- Main ---

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: %s <prompt>\n", os.Args[0])
		os.Exit(1)
	}

	ctx := context.Background()
	prompt := os.Args[1]

	// This is the ENTIRE setup. No StepSettings, no Session, no EngineBuilder.
	// The runner auto-detects Claude from ANTHROPIC_API_KEY env var.
	result, err := runner.Run(ctx, prompt,
		runner.System("You are a code review assistant. Use the available tools to examine code and provide detailed reviews."),
		runner.Tool("read_file", "Read the contents of a file", readFile),
		runner.Tool("search_code", "Search for patterns in source code", searchCode),
		runner.MaxTools(15),
		runner.Stream(func(delta string) {
			fmt.Print(delta) // Stream to stdout as it arrives
		}),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	// result.ToolCalls shows what tools were invoked
	fmt.Fprintf(os.Stderr, "\n--- %d tool calls, %d input tokens, %d output tokens ---\n",
		len(result.ToolCalls), result.Usage.InputTokens, result.Usage.OutputTokens)
}

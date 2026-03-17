//go:build ignore

// Experiment 04: Temporal Relationships Extraction Rewrite
//
// Shows how the batch extraction loop (currently ~150 lines across
// loop.go, config.go, tools_persistence.go) would look with the runner.
//
// Key insight: The temporal-relationships project has an OUTER loop
// (multiple extraction iterations with different prompts) wrapping
// geppetto's INNER tool loop. The runner handles the inner loop;
// the outer loop stays in application code.
//
// This file is a design sketch and is excluded from normal builds.

package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-go-golems/geppetto/pkg/runner"
)

// --- Current implementation: ~150 lines across 3 files ---
//
// config.go:   resolveEngine() → 130 lines of profile/settings cascade
// tools.go:    buildToolRegistry() → 80 lines of scoped DB + registration
// loop.go:     runExtractionLoop() → 70 lines of session/builder/inference
//
// Total: ~280 lines of infrastructure

// --- Proposed implementation ---

// ExtractionConfig holds domain-specific settings.
// Infrastructure settings (model, API keys) come from profiles.
type ExtractionConfig struct {
	ProfileRegistry string
	ProfileName     string
	MaxIterations   int
	StopSequences   []string
}

// RunExtraction performs the multi-iteration extraction pipeline.
// The OUTER loop is application logic. The INNER tool loop is handled by runner.
func RunExtraction(ctx context.Context, cfg ExtractionConfig, inputText string,
	queryHistory func(input struct{ Query string }) (string, error),
	queryTranscripts func(input struct{ Query string }) (string, error),
) ([]string, error) {

	var allArtifacts []string

	// Create a runner with tools. The runner handles:
	// - Profile resolution (finding the right model/settings)
	// - Engine creation (provider-specific API client)
	// - Tool registry (registering Go functions as LLM tools)
	// - Tool loop (inference → tool calls → execution → re-inference)
	// - Event publishing
	r := runner.New(
		runner.Profile(cfg.ProfileName, cfg.ProfileRegistry),
		runner.System(extractionSystemPrompt),
		runner.Tool("query_entity_history", "Query entity change history database", queryHistory),
		runner.Tool("query_transcript_history", "Query session transcript database", queryTranscripts),
		runner.MaxTools(6),
		runner.Timeout(2*time.Minute),
	)

	// Outer loop: multiple extraction passes
	for i := 0; i < cfg.MaxIterations; i++ {
		prompt := buildIterationPrompt(inputText, i, allArtifacts)

		// Run() handles the full inner tool loop:
		// LLM call → detect tool calls → execute tools → feed results back → repeat
		result, err := r.Run(ctx, prompt)
		if err != nil {
			return allArtifacts, fmt.Errorf("iteration %d failed: %w", i, err)
		}

		// Check stop conditions
		artifacts := parseExtractionArtifacts(result.Text)
		allArtifacts = append(allArtifacts, artifacts...)

		if containsStopSequence(result.Text, cfg.StopSequences) {
			break
		}
		if result.StopReason == "end_turn" && len(artifacts) == 0 {
			break // LLM has nothing more to extract
		}
	}

	return allArtifacts, nil
}

// --- Helpers (stubs) ---

const extractionSystemPrompt = `You are an entity extraction assistant.
Use the available tools to query historical data and identify temporal relationships.
Output structured artifacts in YAML format.`

func buildIterationPrompt(input string, iteration int, prior []string) string {
	if iteration == 0 {
		return fmt.Sprintf("Extract temporal relationships from:\n\n%s", input)
	}
	return fmt.Sprintf("Continue extraction. Previously found %d artifacts:\n%s\n\nFind additional relationships.",
		len(prior), strings.Join(prior, "\n"))
}

func parseExtractionArtifacts(text string) []string {
	// Parse YAML blocks from LLM output
	return []string{"artifact1", "artifact2"}
}

func containsStopSequence(text string, sequences []string) bool {
	for _, seq := range sequences {
		if strings.Contains(text, seq) {
			return true
		}
	}
	return false
}

func main() {
	ctx := context.Background()
	cfg := ExtractionConfig{
		ProfileRegistry: "./profile-registry.yaml",
		ProfileName:     "extraction",
		MaxIterations:   4,
		StopSequences:   []string{"NO_MORE_ENTITIES"},
	}

	artifacts, err := RunExtraction(ctx, cfg, "Sample input text...",
		func(input struct{ Query string }) (string, error) { return "history results", nil },
		func(input struct{ Query string }) (string, error) { return "transcript results", nil },
	)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	fmt.Printf("Extracted %d artifacts\n", len(artifacts))
}

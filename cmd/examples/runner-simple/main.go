package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/go-go-golems/geppetto/cmd/examples/internal/runnerexample"
	"github.com/go-go-golems/geppetto/pkg/inference/runner"
	"github.com/go-go-golems/geppetto/pkg/turns"
)

func main() {
	var (
		profile           = flag.String("profile", "gpt-5-nano-low", "engine profile slug")
		profileRegistries = flag.String("profile-registries", runnerexample.PinocchioProfileRegistryPath(), "comma-separated engine profile registry paths")
		prompt            = flag.String("prompt", "Give me a one-sentence explanation of why event-driven inference loops are useful.", "prompt to run")
	)
	flag.Parse()

	stepSettings, closeProfiles, err := runnerexample.OpenAIInferenceSettingsFromProfiles(context.Background(), *profileRegistries, *profile, true)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer func() { _ = closeProfiles() }()

	r := runner.New()
	_, out, err := r.Run(context.Background(), runner.StartRequest{
		Prompt: *prompt,
		Runtime: runner.Runtime{
			InferenceSettings: stepSettings,
			SystemPrompt:      "You are a concise assistant.",
		},
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	turns.FprintTurn(os.Stdout, out)
}

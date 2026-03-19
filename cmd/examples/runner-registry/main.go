package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/go-go-golems/geppetto/cmd/examples/internal/runnerexample"
	profiles "github.com/go-go-golems/geppetto/pkg/engineprofiles"
	"github.com/go-go-golems/geppetto/pkg/inference/runner"
	"github.com/go-go-golems/geppetto/pkg/turns"
)

func main() {
	var (
		profileRegistries = flag.String("profile-registries", runnerexample.ExampleEngineProfileRegistryPath(), "comma-separated profile registry sources")
		profile           = flag.String("profile", "openai-fast", "engine profile slug to resolve")
		prompt            = flag.String("prompt", "Explain how profile registries and runner APIs fit together.", "prompt to run")
	)
	flag.Parse()

	registryEntries, err := profiles.ParseEngineProfileRegistrySourceEntries(*profileRegistries)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	stepSettings, closeRegistry, err := runnerexample.ResolveInferenceSettingsFromRegistry(context.Background(), registryEntries, *profile)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer func() {
		if closeRegistry != nil {
			_ = closeRegistry()
		}
	}()

	r := runner.New()
	_, out, err := r.Run(context.Background(), runner.StartRequest{
		Prompt: *prompt,
		Runtime: runner.Runtime{
			InferenceSettings: stepSettings,
		},
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	turns.FprintTurn(os.Stdout, out)
}

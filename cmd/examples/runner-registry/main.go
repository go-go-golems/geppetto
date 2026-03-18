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
		model             = flag.String("model", "gpt-4o-mini", "model name")
		profileRegistries = flag.String("profile-registries", runnerexample.ExampleProfileRegistryPath(), "comma-separated profile registry sources")
		profile           = flag.String("profile", "concise", "profile slug to resolve")
		prompt            = flag.String("prompt", "Explain how profile registries and runner APIs fit together.", "prompt to run")
	)
	flag.Parse()

	stepSettings, err := runnerexample.OpenAIInferenceSettingsFromEnv(*model, true)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	rt, closeRegistry, err := runnerexample.ResolveRuntimeFromRegistry(context.Background(), stepSettings, *profileRegistries, *profile)
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
		Prompt:  *prompt,
		Runtime: rt,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	turns.FprintTurn(os.Stdout, out)
}

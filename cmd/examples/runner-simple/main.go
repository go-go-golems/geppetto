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
		model  = flag.String("model", "gpt-4o-mini", "model name")
		prompt = flag.String("prompt", "Give me a one-sentence explanation of why event-driven inference loops are useful.", "prompt to run")
	)
	flag.Parse()

	stepSettings, err := runnerexample.OpenAIInferenceSettingsFromEnv(*model, true)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

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

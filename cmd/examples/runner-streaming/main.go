package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/go-go-golems/geppetto/cmd/examples/internal/runnerexample"
	"github.com/go-go-golems/geppetto/pkg/events"
	"github.com/go-go-golems/geppetto/pkg/inference/runner"
	"github.com/go-go-golems/geppetto/pkg/turns"
)

type stdoutSink struct{}

func (s *stdoutSink) PublishEvent(event events.Event) error {
	fmt.Fprintf(os.Stdout, "event: %s\n", event.Type())
	return nil
}

func main() {
	var (
		profile           = flag.String("profile", "gpt-5-nano-low", "engine profile slug")
		profileRegistries = flag.String("profile-registries", runnerexample.PinocchioProfileRegistryPath(), "comma-separated engine profile registry paths")
		prompt            = flag.String("prompt", "Explain, in a few sentences, how event sinks help streaming applications.", "prompt to run")
	)
	flag.Parse()

	stepSettings, closeProfiles, err := runnerexample.OpenAIInferenceSettingsFromProfiles(context.Background(), *profileRegistries, *profile, true)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer func() { _ = closeProfiles() }()

	r := runner.New()
	prepared, handle, err := r.Start(context.Background(), runner.StartRequest{
		Prompt: *prompt,
		Runtime: runner.Runtime{
			InferenceSettings: stepSettings,
			SystemPrompt:      "You are a concise assistant.",
		},
		EventSinks: []events.EventSink{&stdoutSink{}},
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stdout, "session: %s\n", prepared.Session.SessionID)
	out, err := handle.Wait()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fmt.Fprintln(os.Stdout, "\nfinal turn:")
	turns.FprintTurn(os.Stdout, out)
}

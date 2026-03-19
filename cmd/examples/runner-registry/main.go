package main

import (
	"context"
	"fmt"
	"os"

	"github.com/go-go-golems/geppetto/cmd/examples/internal/runnerexample"
	"github.com/go-go-golems/geppetto/pkg/inference/runner"
	geppettosections "github.com/go-go-golems/geppetto/pkg/sections"
	"github.com/go-go-golems/geppetto/pkg/turns"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/spf13/cobra"
)

func newRootCmd() (*cobra.Command, error) {
	cmd := &cobra.Command{
		Use:   "runner-registry",
		Short: "Runner example that mounts the shared profile settings section on Cobra",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			prompt, err := cmd.Flags().GetString("prompt")
			if err != nil {
				return err
			}
			profile, err := cmd.Flags().GetString("profile")
			if err != nil {
				return err
			}
			profileRegistries, err := cmd.Flags().GetStringSlice("profile-registries")
			if err != nil {
				return err
			}

			return run(cmd.Context(), cmd, prompt, profile, profileRegistries)
		},
	}

	profileSettingsSection, err := geppettosections.NewProfileSettingsSection(
		geppettosections.WithProfileDefault("openai-fast"),
		geppettosections.WithProfileRegistriesDefault(runnerexample.ExampleEngineProfileRegistryPath()),
	)
	if err != nil {
		return nil, err
	}
	if err := profileSettingsSection.(schema.CobraSection).AddSectionToCobraCommand(cmd); err != nil {
		return nil, err
	}

	cmd.Flags().String("prompt", "Explain how profile registries and runner APIs fit together.", "prompt to run")
	return cmd, nil
}

func run(ctx context.Context, cmd *cobra.Command, prompt string, profile string, profileRegistries []string) error {
	stepSettings, closeRegistry, err := runnerexample.ResolveInferenceSettingsFromRegistry(ctx, profileRegistries, profile)
	if err != nil {
		return err
	}
	defer func() {
		if closeRegistry != nil {
			_ = closeRegistry()
		}
	}()

	r := runner.New()
	_, out, err := r.Run(ctx, runner.StartRequest{
		Prompt: prompt,
		Runtime: runner.Runtime{
			InferenceSettings: stepSettings,
		},
	})
	if err != nil {
		return err
	}

	turns.FprintTurn(cmd.OutOrStdout(), out)
	return nil
}

func main() {
	cmd, err := newRootCmd()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if err := cmd.ExecuteContext(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

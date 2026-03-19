package main

import (
	"context"
	"fmt"
	"io"
	"os"

	clay "github.com/go-go-golems/clay/pkg"
	"github.com/go-go-golems/geppetto/cmd/examples/internal/runnerexample"
	"github.com/go-go-golems/geppetto/pkg/inference/runner"
	geppettosections "github.com/go-go-golems/geppetto/pkg/sections"
	"github.com/go-go-golems/geppetto/pkg/turns"
	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/logging"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/glazed/pkg/help"
	help_cmd "github.com/go-go-golems/glazed/pkg/help/cmd"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "runner-glazed-registry-flags",
	Short: "Glazed-driven runner example with only registry-selection flags",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return logging.InitLoggerFromCobra(cmd)
	},
}

type registryFlagsCommand struct {
	*cmds.CommandDescription
}

var _ cmds.WriterCommand = (*registryFlagsCommand)(nil)

type registryFlagsSettings struct {
	Prompt string `glazed:"prompt"`
}

func newRegistryFlagsCommand() (*registryFlagsCommand, error) {
	profileSettingsSection, err := geppettosections.NewProfileSettingsSection(
		geppettosections.WithProfileDefault("openai-fast"),
		geppettosections.WithProfileRegistriesDefault(runnerexample.ExampleEngineProfileRegistryPath()),
	)
	if err != nil {
		return nil, errors.Wrap(err, "create profile settings section")
	}

	description := cmds.NewCommandDescription(
		"run",
		cmds.WithShort("Run inference via pkg/inference/runner with only registry selection exposed through Glazed"),
		cmds.WithArguments(
			fields.New("prompt", fields.TypeString, fields.WithHelp("Prompt to run"), fields.WithRequired(true)),
		),
		cmds.WithSections(profileSettingsSection),
	)

	return &registryFlagsCommand{CommandDescription: description}, nil
}

func (c *registryFlagsCommand) RunIntoWriter(ctx context.Context, parsedValues *values.Values, w io.Writer) error {
	s := &registryFlagsSettings{}
	if err := parsedValues.DecodeSectionInto(values.DefaultSlug, s); err != nil {
		return err
	}
	profileSettings := &geppettosections.ProfileSettings{}
	if err := parsedValues.DecodeSectionInto(geppettosections.ProfileSettingsSectionSlug, profileSettings); err != nil {
		return err
	}

	stepSettings, closeRegistry, err := runnerexample.ResolveInferenceSettingsFromRegistry(ctx, profileSettings.ProfileRegistries, profileSettings.Profile)
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
		Prompt: s.Prompt,
		Runtime: runner.Runtime{
			InferenceSettings: stepSettings,
		},
	})
	if err != nil {
		return err
	}

	turns.FprintTurn(w, out)
	return nil
}

func main() {
	err := clay.InitGlazed("geppetto", rootCmd)
	cobra.CheckErr(err)

	helpSystem := help.NewHelpSystem()
	help_cmd.SetupCobraRootCommand(helpSystem, rootCmd)

	command, err := newRegistryFlagsCommand()
	cobra.CheckErr(err)

	cobraCommand, err := cli.BuildCobraCommand(command)
	cobra.CheckErr(err)

	rootCmd.AddCommand(cobraCommand)
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

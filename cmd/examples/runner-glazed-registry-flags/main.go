package main

import (
	"context"
	"fmt"
	"io"
	"os"

	clay "github.com/go-go-golems/clay/pkg"
	"github.com/go-go-golems/geppetto/cmd/examples/internal/runnerexample"
	"github.com/go-go-golems/geppetto/pkg/inference/runner"
	"github.com/go-go-golems/geppetto/pkg/turns"
	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/logging"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
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
	Prompt            string `glazed:"prompt"`
	Profile           string `glazed:"profile"`
	ProfileRegistries string `glazed:"profile-registries"`
}

func profileRegistrySettingsSection() (schema.Section, error) {
	return schema.NewSection(
		"profile-settings",
		"Profile settings",
		schema.WithFields(
			fields.New("profile", fields.TypeString, fields.WithHelp("Profile slug to resolve"), fields.WithDefault("concise")),
			fields.New(
				"profile-registries",
				fields.TypeString,
				fields.WithHelp("Comma-separated profile registry sources (yaml/sqlite/sqlite-dsn)"),
				fields.WithDefault(runnerexample.ExampleEngineProfileRegistryPath()),
			),
		),
	)
}

func newRegistryFlagsCommand() (*registryFlagsCommand, error) {
	profileSection, err := profileRegistrySettingsSection()
	if err != nil {
		return nil, errors.Wrap(err, "create profile settings section")
	}

	description := cmds.NewCommandDescription(
		"runner-glazed-registry-flags",
		cmds.WithShort("Run inference via pkg/inference/runner with only registry selection exposed through Glazed"),
		cmds.WithArguments(
			fields.New("prompt", fields.TypeString, fields.WithHelp("Prompt to run")),
		),
		cmds.WithSections(profileSection),
	)

	return &registryFlagsCommand{CommandDescription: description}, nil
}

func (c *registryFlagsCommand) RunIntoWriter(ctx context.Context, parsedValues *values.Values, w io.Writer) error {
	s := &registryFlagsSettings{}
	if err := parsedValues.DecodeSectionInto(values.DefaultSlug, s); err != nil {
		return err
	}
	if err := parsedValues.DecodeSectionInto("profile-settings", s); err != nil {
		return err
	}

	// This is the small-CLI pattern: keep engine bootstrap app-owned and hidden,
	// while exposing only profile registry selection through Glazed.
	stepSettings, err := runnerexample.BaseInferenceSettingsFromDefaults()
	if err != nil {
		return err
	}

	rt, closeRegistry, err := runnerexample.ResolveRuntimeFromRegistry(ctx, stepSettings, s.ProfileRegistries, s.Profile)
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
		Prompt:  s.Prompt,
		Runtime: rt,
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

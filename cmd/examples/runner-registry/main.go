package main

import (
	"context"
	"io"

	"github.com/go-go-golems/geppetto/cmd/examples/internal/examplecmd"
	"github.com/go-go-golems/geppetto/cmd/examples/internal/runnerexample"
	"github.com/go-go-golems/geppetto/pkg/inference/runner"
	geppettosections "github.com/go-go-golems/geppetto/pkg/sections"
	"github.com/go-go-golems/geppetto/pkg/turns"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type runCommand struct {
	*cmds.CommandDescription
}

var _ cmds.WriterCommand = (*runCommand)(nil)

type runSettings struct {
	Prompt string `glazed:"prompt"`
}

func newRunCommand() (*runCommand, error) {
	profileSettingsSection, err := geppettosections.NewProfileSettingsSection(
		geppettosections.WithProfileDefault("openai-fast"),
		geppettosections.WithProfileRegistriesDefault(runnerexample.ExampleEngineProfileRegistryPath()),
	)
	if err != nil {
		return nil, err
	}

	description := cmds.NewCommandDescription(
		"run",
		cmds.WithShort("Run inference using the shared profile settings Glazed section"),
		cmds.WithArguments(
			fields.New(
				"prompt",
				fields.TypeString,
				fields.WithHelp("Prompt to run"),
				fields.WithDefault("Explain how profile registries and runner APIs fit together."),
			),
		),
		cmds.WithSections(profileSettingsSection),
	)

	return &runCommand{CommandDescription: description}, nil
}

func (c *runCommand) RunIntoWriter(ctx context.Context, parsedValues *values.Values, w io.Writer) error {
	s := &runSettings{}
	if err := parsedValues.DecodeSectionInto(values.DefaultSlug, s); err != nil {
		return errors.Wrap(err, "decode run settings")
	}
	profileSettings := &geppettosections.ProfileSettings{}
	if err := parsedValues.DecodeSectionInto(geppettosections.ProfileSettingsSectionSlug, profileSettings); err != nil {
		return errors.Wrap(err, "decode profile settings")
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
	root := examplecmd.NewRoot("runner-registry", "Runner example that uses the shared profile settings Glazed section")
	cmd, err := newRunCommand()
	cobra.CheckErr(err)
	cobra.CheckErr(examplecmd.ExecuteSingleCommand(root, "geppetto", cmd))
}

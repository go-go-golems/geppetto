package main

import (
	"context"
	"fmt"
	"io"
	"os"

	clay "github.com/go-go-golems/clay/pkg"
	"github.com/go-go-golems/geppetto/pkg/inference/runner"
	geppettosections "github.com/go-go-golems/geppetto/pkg/sections"
	aistepsettings "github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
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
	Use:   "runner-glazed-full-flags",
	Short: "Glazed-driven runner example with full Geppetto runtime flags",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return logging.InitLoggerFromCobra(cmd)
	},
}

type runnerGlazedCommand struct {
	*cmds.CommandDescription
}

var _ cmds.WriterCommand = (*runnerGlazedCommand)(nil)

type runnerGlazedSettings struct {
	Prompt       string `glazed:"prompt"`
	SystemPrompt string `glazed:"system-prompt"`
}

func newRunnerGlazedCommand() (*runnerGlazedCommand, error) {
	geppettoSections, err := geppettosections.CreateGeppettoSections()
	if err != nil {
		return nil, errors.Wrap(err, "create geppetto sections")
	}

	description := cmds.NewCommandDescription(
		"runner-glazed-full-flags",
		cmds.WithShort("Run inference via pkg/inference/runner using full Geppetto sections"),
		cmds.WithArguments(
			fields.New("prompt", fields.TypeString, fields.WithHelp("Prompt to run")),
		),
		cmds.WithFlags(
			fields.New(
				"system-prompt",
				fields.TypeString,
				fields.WithHelp("Optional system prompt applied through runner middleware"),
				fields.WithDefault("You are a concise assistant."),
			),
		),
		cmds.WithSections(geppettoSections...),
	)

	return &runnerGlazedCommand{CommandDescription: description}, nil
}

func (c *runnerGlazedCommand) RunIntoWriter(ctx context.Context, parsedValues *values.Values, w io.Writer) error {
	s := &runnerGlazedSettings{}
	if err := parsedValues.DecodeSectionInto(values.DefaultSlug, s); err != nil {
		return err
	}

	stepSettings, err := aistepsettings.NewStepSettingsFromParsedValues(parsedValues)
	if err != nil {
		return err
	}

	r := runner.New()
	_, out, err := r.Run(ctx, runner.StartRequest{
		Prompt: s.Prompt,
		Runtime: runner.Runtime{
			StepSettings: stepSettings,
			SystemPrompt: s.SystemPrompt,
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

	command, err := newRunnerGlazedCommand()
	cobra.CheckErr(err)

	cobraCommand, err := cli.BuildCobraCommand(
		command,
		cli.WithCobraMiddlewaresFunc(geppettosections.GetCobraCommandGeppettoMiddlewares),
	)
	cobra.CheckErr(err)

	rootCmd.AddCommand(cobraCommand)
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

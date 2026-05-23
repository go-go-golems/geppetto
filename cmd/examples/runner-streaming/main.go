package main

import (
	"context"
	"fmt"
	"io"

	"github.com/go-go-golems/geppetto/cmd/examples/internal/examplecmd"
	"github.com/go-go-golems/geppetto/cmd/examples/internal/runnerexample"
	"github.com/go-go-golems/geppetto/pkg/events"
	"github.com/go-go-golems/geppetto/pkg/inference/runner"
	geppettosections "github.com/go-go-golems/geppetto/pkg/sections"
	"github.com/go-go-golems/geppetto/pkg/turns"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type writerSink struct {
	w io.Writer
}

func (s *writerSink) PublishEvent(event events.Event) error {
	_, err := fmt.Fprintf(s.w, "event: %s\n", event.Type())
	return err
}

type runCommand struct {
	*cmds.CommandDescription
}

var _ cmds.WriterCommand = (*runCommand)(nil)

type runSettings struct {
	Prompt string `glazed:"prompt"`
}

func newRunCommand() (*runCommand, error) {
	profileSettingsSection, err := geppettosections.NewProfileSettingsSection(
		geppettosections.WithProfileDefault("gpt-5-nano-low"),
		geppettosections.WithProfileRegistriesDefault(runnerexample.PinocchioProfileRegistryPath()),
	)
	if err != nil {
		return nil, err
	}

	description := cmds.NewCommandDescription(
		"run",
		cmds.WithShort("Run a profile-backed streaming inference request"),
		cmds.WithArguments(
			fields.New(
				"prompt",
				fields.TypeString,
				fields.WithHelp("Prompt to run"),
				fields.WithDefault("Explain, in a few sentences, how event sinks help streaming applications."),
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

	stepSettings, closeProfiles, err := runnerexample.ResolveInferenceSettingsFromRegistry(ctx, profileSettings.ProfileRegistries, profileSettings.Profile)
	if err != nil {
		return err
	}
	defer func() {
		if closeProfiles != nil {
			_ = closeProfiles()
		}
	}()

	r := runner.New()
	prepared, handle, err := r.Start(ctx, runner.StartRequest{
		Prompt: s.Prompt,
		Runtime: runner.Runtime{
			InferenceSettings: stepSettings,
			SystemPrompt:      "You are a concise assistant.",
		},
		EventSinks: []events.EventSink{&writerSink{w: w}},
	})
	if err != nil {
		return err
	}

	fmt.Fprintf(w, "session: %s\n", prepared.Session.SessionID)
	out, err := handle.Wait()
	if err != nil {
		return err
	}

	fmt.Fprintln(w, "\nfinal turn:")
	turns.FprintTurn(w, out)
	return nil
}

func main() {
	root := examplecmd.NewRoot("runner-streaming", "Profile-backed streaming runner example")
	cmd, err := newRunCommand()
	cobra.CheckErr(err)
	cobra.CheckErr(examplecmd.ExecuteSingleCommand(root, "geppetto", cmd))
}

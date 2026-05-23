package main

import (
	"context"
	"fmt"
	"io"
	"strings"

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

type CalculatorRequest struct {
	A  float64 `json:"a"`
	B  float64 `json:"b"`
	Op string  `json:"op"`
}

type CalculatorResponse struct {
	Result float64 `json:"result"`
}

func calculatorTool(req CalculatorRequest) (CalculatorResponse, error) {
	switch strings.TrimSpace(req.Op) {
	case "", "add":
		return CalculatorResponse{Result: req.A + req.B}, nil
	case "sub":
		return CalculatorResponse{Result: req.A - req.B}, nil
	case "mul":
		return CalculatorResponse{Result: req.A * req.B}, nil
	case "div":
		if req.B == 0 {
			return CalculatorResponse{}, fmt.Errorf("division by zero")
		}
		return CalculatorResponse{Result: req.A / req.B}, nil
	default:
		return CalculatorResponse{}, fmt.Errorf("unsupported op %q", req.Op)
	}
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
		cmds.WithShort("Run a profile-backed inference request with a calculator tool"),
		cmds.WithArguments(
			fields.New(
				"prompt",
				fields.TypeString,
				fields.WithHelp("Prompt to run"),
				fields.WithDefault("Use the calculator tool to multiply 17 by 23, then explain the answer briefly."),
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

	r := runner.New(
		runner.WithFuncTool("calculator", "Basic arithmetic calculator", calculatorTool),
	)
	_, out, err := r.Run(ctx, runner.StartRequest{
		Prompt: s.Prompt,
		Runtime: runner.Runtime{
			InferenceSettings: stepSettings,
			SystemPrompt:      "You are a concise assistant that uses tools when needed.",
			ToolNames:         []string{"calculator"},
		},
	})
	if err != nil {
		return err
	}

	turns.FprintTurn(w, out)
	return nil
}

func main() {
	root := examplecmd.NewRoot("runner-tools", "Profile-backed runner example with a tool")
	cmd, err := newRunCommand()
	cobra.CheckErr(err)
	cobra.CheckErr(examplecmd.ExecuteSingleCommand(root, "geppetto", cmd))
}

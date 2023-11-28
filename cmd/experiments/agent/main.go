package main

import (
	"context"
	"fmt"
	clay "github.com/go-go-golems/clay/pkg"
	geppetto_context "github.com/go-go-golems/geppetto/pkg/context"
	"github.com/go-go-golems/geppetto/pkg/helpers"
	"github.com/go-go-golems/geppetto/pkg/steps"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/openai"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	openai2 "github.com/go-go-golems/geppetto/pkg/steps/ai/settings/openai"
	"github.com/go-go-golems/geppetto/pkg/steps/utils"
	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/help"
	"github.com/spf13/cobra"
	"strings"
)

var rootCmd = &cobra.Command{
	Use:   "agent",
	Short: "agent test",
	Run: func(cmd *cobra.Command, args []string) {
		// reinitialize the logger because we can now parse --log-level and co
		// from the command line flag
		err := clay.InitLogger()
		cobra.CheckErr(err)

		fmt.Println("foobar")

		runAgent()
	}}

var cmd *cobra.Command

func parseLayersFromCobraCommand(cmd *cobra.Command, layers_ []cli.CobraParameterLayer) (
	map[string]*layers.ParsedParameterLayer,
	error,
) {
	ret := map[string]*layers.ParsedParameterLayer{}

	for _, layer := range layers_ {
		ps, err := layer.ParseFlagsFromCobraCommand(cmd)
		if err != nil {
			return nil, err
		}
		ret[layer.GetSlug()] = &layers.ParsedParameterLayer{
			Layer:      layer,
			Parameters: ps,
		}
	}

	return ret, nil
}

func runAgent() {
	layer, err := openai2.NewParameterLayer()
	cobra.CheckErr(err)
	aiLayer, err := settings.NewChatParameterLayer()
	cobra.CheckErr(err)

	// TODO(manuel, 2023-11-28) Turn this into a "add all flags to command"
	// function to create commands, like glazedParameterLayer
	parsedLayers, err := parseLayersFromCobraCommand(cmd, []cli.CobraParameterLayer{layer, aiLayer})
	cobra.CheckErr(err)

	stepSettings := settings.NewStepSettings()
	err = stepSettings.UpdateFromParsedLayers(parsedLayers)
	cobra.CheckErr(err)

	ctx, cancel := context.WithCancel(cmd.Context())
	defer cancel()
	messages := []*geppetto_context.Message{
		{
			Text: "Hello, my friend?",
			Role: geppetto_context.RoleUser,
		},
	}

	// LLM completion step
	step := &openai.Step{
		Settings: stepSettings,
	}
	step.SetStreaming(true)

	// uppercase lambda step
	uppercaseStep := &utils.LambdaStep[string, string]{
		Function: func(s string) helpers.Result[string] {
			return helpers.NewValueResult(strings.ToUpper(s))
		},
	}

	// start the LLM completion
	res, err := step.Start(ctx, messages)
	cobra.CheckErr(err)

	// chain the result through the uppercaseStep
	res_ := steps.Bind[string, string](ctx, res, uppercaseStep)

	c := res_.GetChannel()
	for i := range c {
		s, err := i.Value()
		cobra.CheckErr(err)
		fmt.Printf("%s", s)
	}
}

func main() {
	helpSystem := help.NewHelpSystem()

	helpSystem.SetupCobraRootCommand(rootCmd)

	err := clay.InitViper("pinocchio", rootCmd)
	cobra.CheckErr(err)
	err = clay.InitLogger()
	cobra.CheckErr(err)

	layer, err := openai2.NewParameterLayer()
	cobra.CheckErr(err)

	err = layer.AddFlagsToCobraCommand(rootCmd)
	cobra.CheckErr(err)

	aiLayer, err := settings.NewChatParameterLayer()
	cobra.CheckErr(err)

	err = aiLayer.AddFlagsToCobraCommand(rootCmd)
	cobra.CheckErr(err)

	cmd = rootCmd

	err = rootCmd.Execute()
	cobra.CheckErr(err)

}

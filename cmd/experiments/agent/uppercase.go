package main

import (
	"context"
	"fmt"
	"github.com/go-go-golems/geppetto/pkg/cmds"
	geppetto_context "github.com/go-go-golems/geppetto/pkg/context"
	"github.com/go-go-golems/geppetto/pkg/helpers"
	"github.com/go-go-golems/geppetto/pkg/steps"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/openai"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/go-go-golems/geppetto/pkg/steps/utils"
	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/spf13/cobra"
	"strings"
)

var upperCaseCmd = &cobra.Command{
	Use:   "uppercase",
	Short: "uppercase test",
	Run: func(cmd *cobra.Command, args []string) {
		stepSettings := settings.NewStepSettings()
		geppettoLayers, err := cmds.CreateGeppettoLayers(stepSettings)
		cobra.CheckErr(err)
		layers_ := layers.NewParameterLayers(layers.WithLayers(geppettoLayers...))

		cobraParser, err := cli.NewCobraParserFromLayers(
			layers_,
			cli.WithCobraMiddlewaresFunc(
				cmds.GetCobraCommandGeppettoMiddlewares,
			))
		cobra.CheckErr(err)

		parsedLayers, err := cobraParser.Parse(cmd, args)
		cobra.CheckErr(err)

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

		stepSettings.Chat.Stream = true
		// LLM completion step
		step := openai.NewStep(stepSettings)
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
	},
}

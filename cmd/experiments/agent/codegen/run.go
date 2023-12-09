package codegen

import (
	"context"
	"github.com/go-go-golems/geppetto/cmd/experiments/agent/helpers"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/chat"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	openai2 "github.com/go-go-golems/geppetto/pkg/steps/ai/settings/openai"
	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/spf13/cobra"
)

var CodegenTestCmd = &cobra.Command{
	Use:   "codegen-test",
	Short: "Test codegen prompt",
	Run: func(cmd *cobra.Command, args []string) {
		layer, err := openai2.NewParameterLayer()
		cobra.CheckErr(err)
		aiLayer, err := settings.NewChatParameterLayer()
		cobra.CheckErr(err)

		// TODO(manuel, 2023-11-28) Turn this into a "add all flags to command"
		// function to create commands, like glazedParameterLayer
		parsedLayers, err := helpers.ParseLayersFromCobraCommand(cmd, []cli.CobraParameterLayer{layer, aiLayer})
		cobra.CheckErr(err)

		stepSettings := settings.NewStepSettings()
		err = stepSettings.UpdateFromParsedLayers(parsedLayers)
		cobra.CheckErr(err)

		stepSettings.Chat.Stream = true

		stepFactory := &chat.StandardStepFactory{
			Settings: stepSettings,
		}

		c, err := NewTestCodegenCommand()
		cobra.CheckErr(err)

		c.StepFactory = stepFactory

		params := &TestCodegenCommandParameters{
			Pretend: "Scientist",
			What:    "Size of the moon",
			Of:      "My heart",
			Query:   []string{"What is the size of the moon?"},
		}

		ctx := context.Background()
		err = c.RunIntoWriter(ctx, params, cmd.OutOrStdout())
		cobra.CheckErr(err)
	},
}

var MultiStepCodgenTestCmd = &cobra.Command{
	Use:   "multi-step-codegen-test",
	Short: "Test codegen prompt",
	Run: func(cmd *cobra.Command, args []string) {
	},
}

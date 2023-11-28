package main

import (
	clay "github.com/go-go-golems/clay/pkg"
	"github.com/go-go-golems/geppetto/cmd/experiments/agent/tool"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	openai2 "github.com/go-go-golems/geppetto/pkg/steps/ai/settings/openai"
	"github.com/go-go-golems/glazed/pkg/help"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "agent",
	Short: "agent test",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// reinitialize the logger because we can now parse --log-level and co
		// from the command line flag
		err := clay.InitLogger()
		cobra.CheckErr(err)
	}}

func main() {
	helpSystem := help.NewHelpSystem()

	helpSystem.SetupCobraRootCommand(rootCmd)

	err := clay.InitViper("pinocchio", rootCmd)
	cobra.CheckErr(err)
	err = clay.InitLogger()
	cobra.CheckErr(err)

	layer, err := openai2.NewParameterLayer()
	cobra.CheckErr(err)
	aiLayer, err := settings.NewChatParameterLayer()
	cobra.CheckErr(err)

	err = layer.AddFlagsToCobraCommand(upperCaseCmd)
	cobra.CheckErr(err)
	err = aiLayer.AddFlagsToCobraCommand(upperCaseCmd)
	cobra.CheckErr(err)
	rootCmd.AddCommand(upperCaseCmd)

	err = layer.AddFlagsToCobraCommand(tool.ToolCallCmd)
	cobra.CheckErr(err)
	err = aiLayer.AddFlagsToCobraCommand(tool.ToolCallCmd)
	cobra.CheckErr(err)
	rootCmd.AddCommand(tool.ToolCallCmd)

	err = rootCmd.Execute()
	cobra.CheckErr(err)

}

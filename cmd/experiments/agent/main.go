package main

import (
	clay "github.com/go-go-golems/clay/pkg"
	"github.com/go-go-golems/geppetto/cmd/experiments/agent/codegen"
	"github.com/go-go-golems/geppetto/cmd/experiments/agent/tool"
	"github.com/go-go-golems/geppetto/pkg/cmds"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
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

	stepSettings := settings.NewStepSettings()
	geppettoLayers, err := cmds.CreateGeppettoLayers(stepSettings)
	cobra.CheckErr(err)

	pLayers := layers.NewParameterLayers(layers.WithLayers(geppettoLayers...))

	err = clay.InitViper("pinocchio", rootCmd)
	cobra.CheckErr(err)
	err = clay.InitLogger()
	cobra.CheckErr(err)

	err = pLayers.AddToCobraCommand(upperCaseCmd)
	cobra.CheckErr(err)
	rootCmd.AddCommand(upperCaseCmd)

	err = pLayers.AddToCobraCommand(tool.ToolCallCmd)
	cobra.CheckErr(err)
	rootCmd.AddCommand(tool.ToolCallCmd)

	err = pLayers.AddToCobraCommand(codegen.CodegenTestCmd)
	cobra.CheckErr(err)
	rootCmd.AddCommand(codegen.CodegenTestCmd)

	err = pLayers.AddToCobraCommand(codegen.MultiStepCodgenTestCmd)
	cobra.CheckErr(err)
	rootCmd.AddCommand(codegen.MultiStepCodgenTestCmd)

	err = rootCmd.Execute()
	cobra.CheckErr(err)
}

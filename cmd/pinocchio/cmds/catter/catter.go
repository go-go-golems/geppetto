package catter

import (
	"github.com/go-go-golems/geppetto/cmd/pinocchio/cmds/catter/cmds"
	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds/middlewares"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
	"github.com/spf13/cobra"
)

var catterCmd = &cobra.Command{
	Use:   "catter",
	Short: "Catter - File content and statistics tool",
	Long:  "A CLI tool to print file contents, recursively process directories, and count tokens for LLM context preparation.",
}

func AddToRootCommand(rootCmd *cobra.Command) {
	catterPrintCommand, err := cmds.NewCatterPrintCommand()
	cobra.CheckErr(err)

	catterStatsCmd, err := cmds.NewCatterStatsCommand()
	cobra.CheckErr(err)

	catterCobraCmd, err := cli.BuildCobraCommandFromGlazeCommand(catterPrintCommand,
		cli.WithCobraMiddlewaresFunc(getMiddlewares),
	)
	cobra.CheckErr(err)

	catterStatsCobraCmd, err := cli.BuildCobraCommandFromGlazeCommand(catterStatsCmd,
		cli.WithCobraMiddlewaresFunc(getMiddlewares),
	)
	cobra.CheckErr(err)

	catterCmd.AddCommand(catterCobraCmd)
	catterCmd.AddCommand(catterStatsCobraCmd)
	rootCmd.AddCommand(catterCmd)
}

func getMiddlewares(
	_ *cli.GlazedCommandSettings,
	cmd *cobra.Command,
	args []string,
) ([]middlewares.Middleware, error) {
	return []middlewares.Middleware{
		middlewares.ParseFromCobraCommand(cmd),
		middlewares.GatherArguments(args),
		middlewares.GatherSpecificFlagsFromViper(
			[]string{"filter-profile"},
			parameters.WithParseStepSource("viper"),
		),
		middlewares.SetFromDefaults(),
	}, nil
}

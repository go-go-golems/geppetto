package kagi

import (
	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/spf13/cobra"
)

func RegisterKagiCommands() *cobra.Command {

	kagiCmd := &cobra.Command{
		Use:   "kagi",
		Short: "Commands for kagi.com LLM APIs",
	}
	enrichCmd, err := NewEnrichWebCommand()
	cobra.CheckErr(err)
	command, err := cli.BuildCobraCommandFromGlazeCommand(enrichCmd)
	cobra.CheckErr(err)
	kagiCmd.AddCommand(command)

	summarizeCmd, err := NewSummarizeCommand()
	cobra.CheckErr(err)
	command, err = cli.BuildCobraCommandFromWriterCommand(summarizeCmd)
	cobra.CheckErr(err)
	kagiCmd.AddCommand(command)

	fastGPTCmd, err := NewFastGPTCommand()
	cobra.CheckErr(err)
	command, err = cli.BuildCobraCommandFromWriterCommand(fastGPTCmd)
	cobra.CheckErr(err)
	kagiCmd.AddCommand(command)

	return kagiCmd
}

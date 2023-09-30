package tokens

import (
	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/spf13/cobra"
	"github.com/tiktoken-go/tokenizer"
	"log"
)

func getCodec(model, encoding string) tokenizer.Codec {
	if model != "" {
		c, err := tokenizer.ForModel(tokenizer.Model(model))
		if err != nil {
			log.Fatalf("error creating tokenizer: %v", err)
		}
		return c
	} else {
		c, err := tokenizer.Get(tokenizer.Encoding(encoding))
		if err != nil {
			log.Fatalf("error creating tokenizer: %v", err)
		}
		return c
	}
}

func RegisterTokenCommands(tokensCmd *cobra.Command) {
	countCmdInstance, err := NewCountCommand()
	cobra.CheckErr(err)
	countCommand, err := cli.BuildCobraCommandFromWriterCommand(countCmdInstance)
	cobra.CheckErr(err)
	tokensCmd.AddCommand(countCommand)

	decodeCmdInstance, err := NewDecodeCommand()
	cobra.CheckErr(err)
	decodeCommand, err := cli.BuildCobraCommandFromWriterCommand(decodeCmdInstance)
	cobra.CheckErr(err)
	tokensCmd.AddCommand(decodeCommand)

	encodeCmdInstance, err := NewEncodeCommand()
	cobra.CheckErr(err)
	encodeCommand, err := cli.BuildCobraCommandFromWriterCommand(encodeCmdInstance)
	cobra.CheckErr(err)
	tokensCmd.AddCommand(encodeCommand)

	listModelsCmdInstance, err := NewListModelsCommand()
	cobra.CheckErr(err)
	listModelsCommand, err := cli.BuildCobraCommandFromGlazeCommand(listModelsCmdInstance)
	cobra.CheckErr(err)
	tokensCmd.AddCommand(listModelsCommand)

	listCodecsCmdInstance, err := NewListCodecsCommand()
	cobra.CheckErr(err)
	listCodecsCommand, err := cli.BuildCobraCommandFromGlazeCommand(listCodecsCmdInstance)
	cobra.CheckErr(err)
	tokensCmd.AddCommand(listCodecsCommand)
}

func RegisterCommands(rootCmd *cobra.Command) {
	tokensCmd := &cobra.Command{
		Use:   "tokens",
		Short: "Commands related to tokens",
	}
	RegisterTokenCommands(tokensCmd)
	rootCmd.AddCommand(tokensCmd)
}

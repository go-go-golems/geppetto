package cmds

import (
	"context"
	"github.com/spf13/cobra"
	"github.com/wesen/geppetto/pkg/steps/openai"
	"math"
	"os"
)

// TODO(manuel, 2023-01-27) This actually just sends a file to the prompt API
// this should be part of the openai verbs
// https://github.com/wesen/geppetto/issues/12

var OpenaiCmd = &cobra.Command{
	Use:   "openai",
	Short: "OpenAI commands",
}

var CompletionCmd = &cobra.Command{
	Use:   "completion",
	Short: "send a prompt to the completion API",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		file := args[0]

		if file == "-" {
			file = "/dev/stdin"
		}

		f, err := os.ReadFile(file)
		cobra.CheckErr(err)

		// TODO(manuel, 2023-01-28) we have GenericStepFactory now
		factorySettings, err := openai.NewCompletionStepSettingsFromCobra(cmd)
		cobra.CheckErr(err)
		factory := openai.NewCompletionStepFactory(factorySettings, openai.NewClientSettings())

		step := factory.CreateCompletionStep()

		ctx := context.Background()
		err = step.Start(ctx, string(f))
		cobra.CheckErr(err)

		result := <-step.GetOutput()
		v, err := result.Value()
		cobra.CheckErr(err)

		_, err = os.Stdout.Write([]byte(v))
		cobra.CheckErr(err)
	},
}

func init() {
	OpenaiCmd.PersistentFlags().Int("timeout", 60, "timeout in seconds")
	OpenaiCmd.PersistentFlags().String("organization", "", "organization to use")
	OpenaiCmd.PersistentFlags().String("user-agent", "Geppetto", "user agent to use")
	OpenaiCmd.PersistentFlags().String("base-url", "https://api.openai.com/v1", "base url to use")
	OpenaiCmd.PersistentFlags().String("default-engine", "", "default engine to use")

	CompletionCmd.Flags().String("engine", "", "engine to use")
	CompletionCmd.Flags().Int("max-response-tokens", -1, "max response tokens to use")
	CompletionCmd.Flags().Float64("temperature", math.NaN(), "temperature to use")
	CompletionCmd.Flags().Float64("top-p", math.NaN(), "top p to use")
	CompletionCmd.Flags().StringSlice("stop", []string{}, "stopwords to use")
	CompletionCmd.Flags().Int("log-probabilities", 0, "log probabilities of n tokens")
	CompletionCmd.Flags().Int("n", 1, "n to use")
	CompletionCmd.Flags().Bool("stream", true, "stream to use")

	OpenaiCmd.AddCommand(CompletionCmd)
}

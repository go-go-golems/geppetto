package cmds

import (
	"context"
	"fmt"
	"github.com/mb0/glob"
	"github.com/spf13/cobra"
	"github.com/wesen/geppetto/pkg/steps/openai"
	"github.com/wesen/glazed/pkg/cli"
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

var ListEnginesCmd = &cobra.Command{
	Use:   "list-engines",
	Short: "list engines",
	Run: func(cmd *cobra.Command, args []string) {
		clientSettings, err := openai.NewClientSettingsFromCobra(cmd)
		cobra.CheckErr(err)

		client, err := clientSettings.CreateClient()
		cobra.CheckErr(err)

		ctx := context.Background()
		resp, err := client.Engines(ctx)
		cobra.CheckErr(err)

		gp, of, err := cli.SetupProcessor(cmd)
		cobra.CheckErr(err)

		idGlob, _ := cmd.Flags().GetString("id")
		ownerGlob, _ := cmd.Flags().GetString("owner")
		ready, _ := cmd.Flags().GetBool("ready")

		for _, engine := range resp.Data {
			if idGlob != "" {
				// check if idGlob  matches id
				matching, err := glob.Match(idGlob, engine.ID)
				cobra.CheckErr(err)

				if !matching {
					continue
				}
			}

			if ownerGlob != "" {
				// check if ownerGlob matches owner
				matching, err := glob.Match(ownerGlob, engine.Owner)
				cobra.CheckErr(err)

				if !matching {
					continue
				}
			}

			if cmd.Flags().Changed("ready") {
				// check if ready matches ready
				if ready != engine.Ready {
					continue
				}
			}

			row := map[string]interface{}{
				"id":     engine.ID,
				"owner":  engine.Owner,
				"ready":  engine.Ready,
				"object": engine.Object,
			}
			err = gp.ProcessInputObject(row)
			cobra.CheckErr(err)
		}

		s, err := of.Output()
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Error rendering output: %s\n", err)
			os.Exit(1)
		}
		fmt.Print(s)
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

	ListEnginesCmd.Flags().String("id", "", "glob pattern to match engine id")
	ListEnginesCmd.Flags().String("owner", "", "glob pattern to match engine owner")
	ListEnginesCmd.Flags().Bool("ready", false, "glob pattern to match engine ready")
	cli.AddFlags(ListEnginesCmd, cli.NewFlagsDefaults())
	OpenaiCmd.AddCommand(ListEnginesCmd)
}

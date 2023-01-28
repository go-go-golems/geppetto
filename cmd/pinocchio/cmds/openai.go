package cmds

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/PullRequestInc/go-gpt3"
	"github.com/mb0/glob"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/wesen/geppetto/pkg/steps/openai"
	"github.com/wesen/glazed/pkg/cli"
	"os"
	"strings"
)

var OpenaiCmd = &cobra.Command{
	Use:   "openai",
	Short: "OpenAI commands",
}

var completionStepFactory *openai.CompletionStepFactory

var CompletionCmd = &cobra.Command{
	Use:   "completion",
	Short: "send a prompt to the completion API",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		prompts := []string{}

		for _, file := range args {

			if file == "-" {
				file = "/dev/stdin"
			}

			f, err := os.ReadFile(file)
			cobra.CheckErr(err)

			prompts = append(prompts, string(f))
		}

		// TODO(manuel, 2023-01-28) actually I don't think it's a good idea to go through the stepfactory here
		// we just want to have the RAW api access with all its outputs

		clientSettings, err := openai.NewClientSettingsFromCobra(cmd)
		cobra.CheckErr(err)

		err = completionStepFactory.UpdateFromCobra(cmd)
		cobra.CheckErr(err)

		client, err := clientSettings.CreateClient()
		cobra.CheckErr(err)

		ctx := context.Background()
		settings := completionStepFactory.StepSettings
		if settings.Engine == nil {
			cobra.CheckErr(fmt.Errorf("engine is required"))
		}
		resp, err := client.CompletionWithEngine(ctx, *settings.Engine,
			gpt3.CompletionRequest{
				Prompt:           prompts,
				MaxTokens:        settings.MaxResponseTokens,
				Temperature:      settings.Temperature,
				TopP:             settings.TopP,
				N:                settings.N,
				LogProbs:         settings.LogProbs,
				Echo:             false,
				Stop:             settings.Stop,
				PresencePenalty:  0,
				FrequencyPenalty: 0,
				Stream:           false,
			},
		)
		cobra.CheckErr(err)

		printUsage, _ := cmd.Flags().GetBool("print-usage")
		usage := resp.Usage
		evt := log.Debug()
		if printUsage {
			evt = log.Info()
		}
		evt.
			Int("prompt-tokens", usage.PromptTokens).
			Int("completion-tokens", usage.CompletionTokens).
			Int("total-tokens", usage.TotalTokens).
			Msg("Usage")

		gp, of, err := cli.SetupProcessor(cmd)
		cobra.CheckErr(err)

		printRawResponse, _ := cmd.Flags().GetBool("print-raw-response")

		if printRawResponse {
			// serialize resp to json
			rawResponse, err := json.MarshalIndent(resp, "", "  ")
			cobra.CheckErr(err)

			// deserialize to map[string]interface{}
			var rawResponseMap map[string]interface{}
			err = json.Unmarshal(rawResponse, &rawResponseMap)
			cobra.CheckErr(err)

			err = gp.ProcessInputObject(rawResponseMap)
			cobra.CheckErr(err)

		} else {
			for i, choice := range resp.Choices {
				logProbs := map[string]interface{}{}

				logProbJson, err := json.MarshalIndent(choice.LogProbs, "", "  ")
				cobra.CheckErr(err)

				// deserialize to map[string]interface{}
				var logProbMap map[string]interface{}
				err = json.Unmarshal(logProbJson, &logProbMap)
				cobra.CheckErr(err)

				idx := i / len(resp.Choices)
				prompt := ""
				if len(prompts) > idx {
					prompt = prompts[idx]
				}

				// escape newline in response
				text := strings.Trim(choice.Text, " \t\n")
				text = strings.ReplaceAll(text, "\n", "\\n")
				prompt = strings.Trim(prompt, " \t\n")
				prompt = strings.ReplaceAll(prompt, "\n", "\\n")
				// trim whitespace

				row := map[string]interface{}{
					"index":         choice.Index,
					"text":          text,
					"logprobs":      logProbs,
					"prompt":        prompt,
					"finish_reason": choice.FinishReason,
					"engine":        *settings.Engine,
				}
				err = gp.ProcessInputObject(row)
				cobra.CheckErr(err)
			}
		}

		s, err := of.Output()
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Error rendering output: %s\n", err)
			os.Exit(1)
		}
		fmt.Print(s)
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

var EngineInfoCmd = &cobra.Command{
	Use:   "engine-info",
	Short: "get engine info",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		engine := args[0]

		clientSettings, err := openai.NewClientSettingsFromCobra(cmd)
		cobra.CheckErr(err)

		client, err := clientSettings.CreateClient()
		cobra.CheckErr(err)

		ctx := context.Background()
		resp, err := client.Engine(ctx, engine)
		cobra.CheckErr(err)

		gp, of, err := cli.SetupProcessor(cmd)
		cobra.CheckErr(err)

		row := map[string]interface{}{
			"id":     resp.ID,
			"owner":  resp.Owner,
			"ready":  resp.Ready,
			"object": resp.Object,
		}
		err = gp.ProcessInputObject(row)
		cobra.CheckErr(err)

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

	ListEnginesCmd.Flags().String("id", "", "glob pattern to match engine id")
	ListEnginesCmd.Flags().String("owner", "", "glob pattern to match engine owner")
	ListEnginesCmd.Flags().Bool("ready", false, "glob pattern to match engine ready")
	cli.AddFlags(ListEnginesCmd, cli.NewFlagsDefaults())
	OpenaiCmd.AddCommand(ListEnginesCmd)

	completionStepFactory = openai.NewCompletionStepFactory(
		openai.NewCompletionStepSettings(),
		openai.NewClientSettings(),
	)
	defaultEngine := "text-davinci-002"
	maxResponseTokens := 256
	err := completionStepFactory.AddFlags(CompletionCmd, "", &openai.CompletionStepFactoryFlagsDefaults{
		Engine:            &defaultEngine,
		MaxResponseTokens: &maxResponseTokens,
	})
	cobra.CheckErr(err)

	CompletionCmd.Flags().Bool("print-usage", false, "print usage")
	CompletionCmd.Flags().Bool("print-raw-response", false, "print raw response as object")

	cli.AddFlags(CompletionCmd, cli.NewFlagsDefaults())
	OpenaiCmd.AddCommand(CompletionCmd)

	cli.AddFlags(EngineInfoCmd, cli.NewFlagsDefaults())
	OpenaiCmd.AddCommand(EngineInfoCmd)
}

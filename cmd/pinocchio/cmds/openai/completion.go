package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/PullRequestInc/go-gpt3"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/wesen/geppetto/pkg/steps/openai"
	"github.com/wesen/glazed/pkg/cli"
	"os"
	"strings"
)

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

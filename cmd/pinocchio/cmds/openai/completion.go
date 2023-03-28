package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/PullRequestInc/go-gpt3"
	"github.com/go-go-golems/geppetto/pkg/steps/openai"
	"github.com/go-go-golems/geppetto/pkg/steps/openai/completion"
	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"os"
	"strings"
)

type CompletionCommand struct {
	description *cmds.CommandDescription
}

func NewCompletionCommand() (*CompletionCommand, error) {
	glazedParameterLayer, err := cli.NewGlazedParameterLayers()
	if err != nil {
		return nil, err
	}
	openaiParameterLayer, err := openai.NewClientParameterLayer()
	if err != nil {
		return nil, err
	}
	completionParameterLayer, err := completion.NewParameterLayer()
	if err != nil {
		return nil, err
	}

	return &CompletionCommand{
		description: cmds.NewCommandDescription(
			"completion",
			cmds.WithShort("send a prompt to the completion API"),
			cmds.WithFlags(
				parameters.NewParameterDefinition(
					"print-usage",
					parameters.ParameterTypeBool,
					parameters.WithHelp("print usage"),
					parameters.WithDefault(false),
				),
				parameters.NewParameterDefinition(
					"print-raw-response",
					parameters.ParameterTypeBool,
					parameters.WithHelp("print raw response as object"),
					parameters.WithDefault(false),
				),
			),
			cmds.WithArguments(
				parameters.NewParameterDefinition(
					"input-files",
					parameters.ParameterTypeStringList,
					parameters.WithRequired(true),
				),
			),
			cmds.WithLayers(
				glazedParameterLayer,
				completionParameterLayer,
				openaiParameterLayer,
			),
		),
	}, nil
}

func (j *CompletionCommand) Run(
	ctx context.Context,
	parsedLayers map[string]*layers.ParsedParameterLayer,
	ps map[string]interface{},
	gp cmds.Processor,
) error {
	prompts := []string{}

	inputFiles, ok := ps["input-files"].([]string)
	if !ok {
		return fmt.Errorf("input-files is not a string list")
	}

	for _, file := range inputFiles {

		if file == "-" {
			file = "/dev/stdin"
		}

		f, err := os.ReadFile(file)
		cobra.CheckErr(err)

		prompts = append(prompts, string(f))
	}

	// TODO(manuel, 2023-01-28) actually I don't think it's a good idea to go through the stepfactory here
	// we just want to have the RAW api access with all its outputs

	clientSettings, err := openai.NewClientSettingsFromParameters(ps)
	cobra.CheckErr(err)

	completionSettings, err := completion.NewStepSettingsFromParameters(ps)
	cobra.CheckErr(err)

	completionStepFactory := completion.NewStepFactory(completionSettings, clientSettings)

	client, err := clientSettings.CreateClient()
	cobra.CheckErr(err)

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

	printUsage, _ := ps["print-usage"].(bool)
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

	printRawResponse, _ := ps["print-raw-response"].(bool)

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

	return nil
}

func (j *CompletionCommand) Description() *cmds.CommandDescription {
	return j.description
}

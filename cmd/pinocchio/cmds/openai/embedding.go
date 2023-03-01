package openai

import (
	"context"
	"encoding/json"
	"github.com/PullRequestInc/go-gpt3"
	"github.com/go-go-golems/geppetto/pkg/steps/openai"
	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"os"
)

type EmbeddingsCommand struct {
	description *cmds.CommandDescription
}

func NewEmbeddingsCommand() (*EmbeddingsCommand, error) {
	glazedParameterLayer, err := cli.NewGlazedParameterLayers(
		cli.WithOutputParameterLayerOptions(
			layers.WithDefaults(map[string]interface{}{
				"output": "json",
			},
			),
		),
	)
	if err != nil {
		return nil, err
	}
	openaiParameterLayer, err := openai.NewClientParameterLayer()
	if err != nil {
		return nil, err
	}
	completionParameterLayer, err := openai.NewCompletionParameterLayer()
	if err != nil {
		return nil, err
	}

	return &EmbeddingsCommand{
		description: cmds.NewCommandDescription(
			"embeddings",
			cmds.WithShort("send a prompt to the embeddings API"),
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
				parameters.NewParameterDefinition(
					"model",
					parameters.ParameterTypeString,
					parameters.WithHelp("model to use"),
					parameters.WithDefault("text-embedding-ada-002"),
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

func (c *EmbeddingsCommand) Description() *cmds.CommandDescription {
	return c.description
}

func (c *EmbeddingsCommand) Run(
	ctx context.Context,
	parsedLayers map[string]*layers.ParsedParameterLayer,
	ps map[string]interface{},
	gp *cmds.GlazeProcessor,
) error {
	user, _ := ps["user"].(string)

	inputFiles, _ := ps["input-files"].([]string)
	prompts := []string{}

	for _, file := range inputFiles {

		if file == "-" {
			file = "/dev/stdin"
		}

		f, err := os.ReadFile(file)
		cobra.CheckErr(err)

		prompts = append(prompts, string(f))
	}

	clientSettings, err := openai.NewClientSettingsFromParameters(ps)
	cobra.CheckErr(err)

	client, err := clientSettings.CreateClient()
	cobra.CheckErr(err)

	engine, _ := ps["model"].(string)

	resp, err := client.Embeddings(ctx, gpt3.EmbeddingsRequest{
		Input: prompts,
		Model: engine,
		User:  user,
	})
	cobra.CheckErr(err)

	printUsage, _ := ps["print-usage"].(bool)
	printRawResponse, _ := ps["print-raw-response"].(bool)

	usage := resp.Usage
	evt := log.Debug()
	if printUsage {
		evt = log.Info()
	}
	evt.
		Int("prompt-tokens", usage.PromptTokens).
		Int("total-tokens", usage.TotalTokens).
		Msg("Usage")

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
		for _, embedding := range resp.Data {
			row := map[string]interface{}{
				"object":    embedding.Object,
				"embedding": embedding.Embedding,
				"index":     embedding.Index,
			}
			err = gp.ProcessInputObject(row)
			cobra.CheckErr(err)
		}
	}

	return nil
}

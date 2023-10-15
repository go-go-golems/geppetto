package openai

import (
	"context"
	_ "embed"
	get_conversation "github.com/go-go-golems/geppetto/cmd/pinocchio/cmds/openai/get-conversation"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings/openai"
	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	"github.com/go-go-golems/glazed/pkg/settings"
	"github.com/go-go-golems/glazed/pkg/types"
	"github.com/mb0/glob"
	openai2 "github.com/sashabaranov/go-openai"
	"github.com/spf13/cobra"
)

var OpenaiCmd = &cobra.Command{
	Use:   "openai",
	Short: "OpenAI commands",
}

type ListEnginesCommand struct {
	*cmds.CommandDescription
}

func NewListEngineCommand() (*ListEnginesCommand, error) {
	glazedParameterLayer, err := settings.NewGlazedParameterLayers()
	if err != nil {
		return nil, err
	}
	openaiParameterLayer, err := openai.NewParameterLayer()
	if err != nil {
		return nil, err
	}

	return &ListEnginesCommand{
		CommandDescription: cmds.NewCommandDescription(
			"list-engines",
			cmds.WithShort("list engines"),
			cmds.WithFlags(
				parameters.NewParameterDefinition(
					"id",
					parameters.ParameterTypeString,
					parameters.WithHelp("glob to match engine id"),
				),
				parameters.NewParameterDefinition(
					"owner",
					parameters.ParameterTypeString,
					parameters.WithHelp("glob to match engine owner"),
				),

				parameters.NewParameterDefinition(
					"only-ready",
					parameters.ParameterTypeBool,
					parameters.WithHelp("glob to match engine ready"),
					parameters.WithDefault(false),
				),
			),
			cmds.WithLayers(
				glazedParameterLayer,
				openaiParameterLayer,
			),
		),
	}, nil
}

func (c *ListEnginesCommand) Run(
	ctx context.Context,
	parsedLayers map[string]*layers.ParsedParameterLayer,
	ps map[string]interface{},
	gp middlewares.Processor,
) error {
	openaiSettings, err := openai.NewSettingsFromParsedLayer(
		parsedLayers["openai-chat"],
	)
	cobra.CheckErr(err)

	client := openai2.NewClient(*openaiSettings.APIKey)

	engines, err := client.ListEngines(ctx)
	cobra.CheckErr(err)

	idGlob, _ := ps["id"].(string)
	ownerGlob, _ := ps["owner"].(string)
	onlyReady, _ := ps["onlyready"].(bool)

	for _, engine := range engines.Engines {
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

		if onlyReady {
			if !engine.Ready {
				continue
			}
		}

		row := types.NewRow(
			types.MRP("id", engine.ID),
			types.MRP("owner", engine.Owner),
			types.MRP("ready", engine.Ready),
			types.MRP("object", engine.Object),
		)
		err = gp.AddRow(ctx, row)
		cobra.CheckErr(err)
	}

	return nil
}

type EngineInfoCommand struct {
	*cmds.CommandDescription
}

func NewEngineInfoCommand() (*EngineInfoCommand, error) {
	glazedParameterLayer, err := settings.NewGlazedParameterLayers()
	if err != nil {
		return nil, err
	}
	openaiParameterLayer, err := openai.NewParameterLayer()
	if err != nil {
		return nil, err
	}

	return &EngineInfoCommand{
		CommandDescription: cmds.NewCommandDescription(
			"engine-info",
			cmds.WithShort("get engine info"),
			cmds.WithArguments(
				parameters.NewParameterDefinition(
					"engine",
					parameters.ParameterTypeString,
					parameters.WithHelp("engine id"),
				),
			),
			cmds.WithLayers(
				glazedParameterLayer,
				openaiParameterLayer,
			),
		),
	}, nil
}

func (c *EngineInfoCommand) Run(
	ctx context.Context,
	parsedLayers map[string]*layers.ParsedParameterLayer,
	ps map[string]interface{},
	gp middlewares.Processor,
) error {
	openaiSettings, err := openai.NewSettingsFromParsedLayer(
		parsedLayers["openai-chat"],
	)
	cobra.CheckErr(err)

	client := openai2.NewClient(*openaiSettings.APIKey)

	engine, _ := ps["engine"].(string)

	resp, err := client.GetEngine(ctx, engine)
	cobra.CheckErr(err)

	row := types.NewRow(
		types.MRP("id", resp.ID),
		types.MRP("owner", resp.Owner),
		types.MRP("ready", resp.Ready),
		types.MRP("object", resp.Object),
	)
	err = gp.AddRow(ctx, row)
	cobra.CheckErr(err)

	return nil
}

func init() {
	listEnginesCommand, err := NewListEngineCommand()
	cobra.CheckErr(err)
	listEnginesCobraCommand, err := cli.BuildCobraCommandFromGlazeCommand(listEnginesCommand)
	cobra.CheckErr(err)
	OpenaiCmd.AddCommand(listEnginesCobraCommand)

	engineInfoCommand, err := NewEngineInfoCommand()
	cobra.CheckErr(err)
	cobraEngineInfoCommand, err := cli.BuildCobraCommandFromGlazeCommand(engineInfoCommand)
	cobra.CheckErr(err)
	OpenaiCmd.AddCommand(cobraEngineInfoCommand)

	getConversationCommand, err := get_conversation.NewGetConversationCommand()
	cobra.CheckErr(err)
	cobraGetConversationCommand, err := cli.BuildCobraCommandFromWriterCommand(getConversationCommand)
	cobra.CheckErr(err)
	OpenaiCmd.AddCommand(cobraGetConversationCommand)

	transcribeCommand, err := NewTranscribeCommand()
	cobra.CheckErr(err)
	cobraTranscribeCommand, err := cli.BuildCobraCommandFromGlazeCommand(transcribeCommand)
	cobra.CheckErr(err)
	OpenaiCmd.AddCommand(cobraTranscribeCommand)
}

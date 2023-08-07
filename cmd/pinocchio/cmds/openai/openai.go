package openai

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"github.com/go-go-golems/geppetto/pkg/steps/openai"
	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
	"github.com/go-go-golems/glazed/pkg/help"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	"github.com/go-go-golems/glazed/pkg/settings"
	"github.com/go-go-golems/glazed/pkg/types"
	"github.com/mb0/glob"
	"github.com/pkg/errors"
	openai2 "github.com/sashabaranov/go-openai"
	"github.com/spf13/cobra"
	"strings"
	"text/template"
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
	openaiParameterLayer, err := openai.NewClientParameterLayer()
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
	clientSettings, err := openai.NewClientSettingsFromParameters(ps)
	cobra.CheckErr(err)

	client := openai2.NewClient(*clientSettings.APIKey)

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
	openaiParameterLayer, err := openai.NewClientParameterLayer()
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
	clientSettings, err := openai.NewClientSettingsFromParameters(ps)
	cobra.CheckErr(err)

	client := openai2.NewClient(*clientSettings.APIKey)

	cobra.CheckErr(err)

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

//go:embed help/models.json
var modelsJson string

//go:embed help/help-family-template.md
var helpFamilyTemplate string

//go:embed help/help-completion-template.md
var helpCompletionTemplate string

type Completion struct {
	Name                   string `json:"name"`
	Family                 string `json:"family"`
	Description            string `json:"description"`
	MaxTokens              int    `json:"max_tokens"`
	TrainingDataCutoffDate string `json:"training_data_cutoff_date"`
}

type Family struct {
	Name             string   `json:"name"`
	Description      string   `json:"description"`
	PricePer1kTokens float64  `json:"price_per_1k_tokens"`
	GoodAt           []string `json:"good_at"`
	KeyPoints        []string `json:"key_points"`
	Subtitle         string   `json:"subtitle"`
	Short            string   `json:"short"`
}

type ModelsJSON struct {
	Completion []Completion `json:"completion"`
	Families   []Family     `json:"families"`
}

type SimpleModelsJSON struct {
	Completion []types.Row `json:"completion"`
	Families   []types.Row `json:"families"`
}

var FamiliesCmd = &cobra.Command{
	Use:   "ls-families",
	Short: "list families",
	Run: func(cmd *cobra.Command, args []string) {
		models := SimpleModelsJSON{}
		err := json.Unmarshal([]byte(modelsJson), &models)
		cobra.CheckErr(err)

		ctx := cmd.Context()

		gp, _, err := cli.CreateGlazedProcessorFromCobra(cmd)
		cobra.CheckErr(err)

		for _, family := range models.Families {
			err = gp.AddRow(ctx, family)
			cobra.CheckErr(err)
		}

		err = gp.Close(ctx)
		cobra.CheckErr(err)
	},
}

var ModelsCmd = &cobra.Command{
	Use:   "ls-models",
	Short: "list models",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()

		models := SimpleModelsJSON{}
		err := json.Unmarshal([]byte(modelsJson), &models)
		cobra.CheckErr(err)

		gp, _, err := cli.CreateGlazedProcessorFromCobra(cmd)
		cobra.CheckErr(err)

		for _, completion := range models.Completion {
			err = gp.AddRow(ctx, completion)
			cobra.CheckErr(err)
		}

		err = gp.Close(ctx)
		cobra.CheckErr(err)
	},
}

func LoadModelsHelpFiles() ([]*help.Section, error) {
	models := ModelsJSON{}
	err := json.Unmarshal([]byte(modelsJson), &models)
	if err != nil {
		return nil, err
	}

	ret := []*help.Section{}

	familyTemplate, err := template.New("family").Parse(helpFamilyTemplate)
	if err != nil {
		return nil, err
	}

	completionTemplate, err := template.New("completion").Parse(helpCompletionTemplate)
	if err != nil {
		return nil, err
	}

	families := map[string]*Family{}

	for _, family := range models.Families {
		families[family.Name] = &family

		buf := &bytes.Buffer{}
		err = familyTemplate.Execute(buf, family)
		if err != nil {
			return nil, errors.Wrapf(err, "error executing family template for family %s", family.Name)
		}

		familySection := &help.Section{
			Slug:        strings.ToLower(family.Name),
			SectionType: help.SectionGeneralTopic,
			Title:       fmt.Sprintf("OpenAI Family %s", family.Name),
			SubTitle:    family.Subtitle,
			Short:       family.Short,
			Content:     buf.String(),
			Topics: []string{
				"models",
				"openai",
				"families",
			},
			Flags: []string{
				"engine",
			},
			Commands:       nil,
			IsTopLevel:     true,
			IsTemplate:     false,
			ShowPerDefault: false,
			Order:          0,
		}

		ret = append(ret, familySection)
	}

	for _, completion := range models.Completion {
		family, ok := families[completion.Family]
		if !ok {
			return nil, fmt.Errorf("family %s not found", completion.Family)
		}

		data := map[string]interface{}{
			"Completion": completion,
			"Family":     family,
		}

		buf := &bytes.Buffer{}
		err = completionTemplate.Execute(buf, data)
		if err != nil {
			return nil, errors.Wrapf(err, "error executing template for completion %s", completion.Name)
		}

		completionSection := &help.Section{
			Slug:        strings.ToLower(completion.Name),
			SectionType: help.SectionGeneralTopic,
			Title:       completion.Name,
			SubTitle:    family.Subtitle,
			Short:       completion.Description,
			Content:     buf.String(),
			Topics: []string{
				"completion",
				"openai",
				"models",
			},
			Flags: []string{
				"engine",
			},
			Commands:       []string{"completion"},
			IsTopLevel:     true,
			IsTemplate:     false,
			ShowPerDefault: false,
			Order:          1,
		}

		ret = append(ret, completionSection)
	}

	return ret, nil
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

	err = cli.AddGlazedProcessorFlagsToCobraCommand(FamiliesCmd)
	if err != nil {
		panic(err)
	}
	OpenaiCmd.AddCommand(FamiliesCmd)

	err = cli.AddGlazedProcessorFlagsToCobraCommand(ModelsCmd)
	if err != nil {
		panic(err)
	}
	OpenaiCmd.AddCommand(ModelsCmd)
}

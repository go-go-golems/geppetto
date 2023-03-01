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
	"github.com/mb0/glob"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"os"
	"strings"
	"text/template"
)

var OpenaiCmd = &cobra.Command{
	Use:   "openai",
	Short: "OpenAI commands",
}

type ListEnginesCommand struct {
	description *cmds.CommandDescription
}

func NewListEngineCommand() (*ListEnginesCommand, error) {
	glazedParameterLayer, err := cli.NewGlazedParameterLayers()
	if err != nil {
		return nil, err
	}
	openaiParameterLayer, err := openai.NewClientParameterLayer()
	if err != nil {
		return nil, err
	}

	return &ListEnginesCommand{
		description: cmds.NewCommandDescription(
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

func (c *ListEnginesCommand) Description() *cmds.CommandDescription {
	return c.description
}

func (c *ListEnginesCommand) Run(
	ctx context.Context,
	parsedLayers map[string]*layers.ParsedParameterLayer,
	ps map[string]interface{},
	gp cmds.Processor,
) error {
	clientSettings, err := openai.NewClientSettingsFromParameters(ps)
	cobra.CheckErr(err)

	client, err := clientSettings.CreateClient()
	cobra.CheckErr(err)

	resp, err := client.Engines(ctx)
	cobra.CheckErr(err)

	idGlob, _ := ps["id"].(string)
	ownerGlob, _ := ps["owner"].(string)
	onlyReady, _ := ps["onlyready"].(bool)

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

		if onlyReady {
			if !engine.Ready {
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

	return nil
}

type EngineInfoCommand struct {
	description *cmds.CommandDescription
}

func NewEngineInfoCommand() (*EngineInfoCommand, error) {
	glazedParameterLayer, err := cli.NewGlazedParameterLayers()
	if err != nil {
		return nil, err
	}
	openaiParameterLayer, err := openai.NewClientParameterLayer()
	if err != nil {
		return nil, err
	}

	return &EngineInfoCommand{
		description: cmds.NewCommandDescription(
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

func (c *EngineInfoCommand) Description() *cmds.CommandDescription {
	return c.description
}

func (c *EngineInfoCommand) Run(
	ctx context.Context,
	parsedLayers map[string]*layers.ParsedParameterLayer,
	ps map[string]interface{},
	gp cmds.Processor,
) error {
	clientSettings, err := openai.NewClientSettingsFromParameters(ps)
	cobra.CheckErr(err)

	client, err := clientSettings.CreateClient()
	cobra.CheckErr(err)

	engine, _ := ps["engine"].(string)

	resp, err := client.Engine(ctx, engine)
	cobra.CheckErr(err)

	row := map[string]interface{}{
		"id":     resp.ID,
		"owner":  resp.Owner,
		"ready":  resp.Ready,
		"object": resp.Object,
	}
	err = gp.ProcessInputObject(row)
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
	Completion []map[string]interface{} `json:"completion"`
	Families   []map[string]interface{} `json:"families"`
}

var FamiliesCmd = &cobra.Command{
	Use:   "ls-families",
	Short: "list families",
	Run: func(cmd *cobra.Command, args []string) {
		models := SimpleModelsJSON{}
		err := json.Unmarshal([]byte(modelsJson), &models)
		cobra.CheckErr(err)

		gp, of, err := cli.CreateGlazedProcessorFromCobra(cmd)
		cobra.CheckErr(err)

		for _, family := range models.Families {
			err = gp.ProcessInputObject(family)
			cobra.CheckErr(err)
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

var ModelsCmd = &cobra.Command{
	Use:   "ls-models",
	Short: "list models",
	Run: func(cmd *cobra.Command, args []string) {

		models := SimpleModelsJSON{}
		err := json.Unmarshal([]byte(modelsJson), &models)
		cobra.CheckErr(err)

		gp, of, err := cli.CreateGlazedProcessorFromCobra(cmd)
		cobra.CheckErr(err)

		for _, completion := range models.Completion {
			err = gp.ProcessInputObject(completion)
			cobra.CheckErr(err)
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
	listEnginesCobraCommand, err := cli.BuildCobraCommand(listEnginesCommand)
	cobra.CheckErr(err)
	OpenaiCmd.AddCommand(listEnginesCobraCommand)

	completionCommand, err := NewCompletionCommand()
	cobra.CheckErr(err)
	cobraCompletionCommand, err := cli.BuildCobraCommand(completionCommand)
	cobra.CheckErr(err)
	OpenaiCmd.AddCommand(cobraCompletionCommand)

	engineInfoCommand, err := NewEngineInfoCommand()
	cobra.CheckErr(err)
	cobraEngineInfoCommand, err := cli.BuildCobraCommand(engineInfoCommand)
	cobra.CheckErr(err)
	OpenaiCmd.AddCommand(cobraEngineInfoCommand)

	embeddingsCommand, err := NewEmbeddingsCommand()
	cobra.CheckErr(err)
	cobraEmbeddingsCommand, err := cli.BuildCobraCommand(embeddingsCommand)
	cobra.CheckErr(err)
	OpenaiCmd.AddCommand(cobraEmbeddingsCommand)

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

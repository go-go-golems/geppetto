package openai

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"github.com/PullRequestInc/go-gpt3"
	"github.com/mb0/glob"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/wesen/geppetto/pkg/steps/openai"
	"github.com/wesen/glazed/pkg/cli"
	"github.com/wesen/glazed/pkg/help"
	"os"
	"strings"
	"text/template"
)

var OpenaiCmd = &cobra.Command{
	Use:   "openai",
	Short: "OpenAI commands",
}

var EditsCmd = &cobra.Command{
	Use:   "edits",
	Short: "Compute edits for a file",
	Run: func(cmd *cobra.Command, args []string) {

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

// embed the models.json file

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

		gp, of, err := cli.SetupProcessor(cmd)
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

		gp, of, err := cli.SetupProcessor(cmd)
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
	OpenaiCmd.PersistentFlags().Int("timeout", 60, "timeout in seconds")
	OpenaiCmd.PersistentFlags().String("organization", "", "organization to use")
	OpenaiCmd.PersistentFlags().String("user-agent", "Geppetto", "user agent to use")
	OpenaiCmd.PersistentFlags().String("base-url", "https://api.openai.com/v1", "base url to use")
	OpenaiCmd.PersistentFlags().String("default-engine", "", "default engine to use")
	OpenaiCmd.PersistentFlags().String("user", "", "user (hash) to use")

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

	EmbeddingsCmd.Flags().Bool("print-usage", false, "print usage")
	EmbeddingsCmd.Flags().Bool("print-raw-response", false, "print raw response as object")
	cli.AddFlags(EmbeddingsCmd, cli.NewFlagsDefaults())
	EmbeddingsCmd.Flags().String("engine", gpt3.TextDavinci002Engine, "engine to use")
	OpenaiCmd.AddCommand(EmbeddingsCmd)

	OpenaiCmd.AddCommand(EditsCmd)

	cli.AddFlags(FamiliesCmd, cli.NewFlagsDefaults())
	OpenaiCmd.AddCommand(FamiliesCmd)

	cli.AddFlags(ModelsCmd, cli.NewFlagsDefaults())
	OpenaiCmd.AddCommand(ModelsCmd)
}

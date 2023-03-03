package main

import (
	"embed"
	"fmt"
	clay "github.com/go-go-golems/clay/pkg"
	"github.com/go-go-golems/geppetto/cmd/pinocchio/cmds/openai"
	"github.com/go-go-golems/geppetto/cmd/pinocchio/cmds/openai/ui"
	"github.com/go-go-golems/geppetto/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cli"
	glazed_cmds "github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/help"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
)

//go:embed doc/*
var docFS embed.FS

//go:embed prompts/*
var promptsFS embed.FS

var rootCmd = &cobra.Command{
	Use:   "pinocchio",
	Short: "pinocchio is a tool to run LLM applications",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// reinitialize the logger because we can now parse --log-level and co
		// from the command line flag
		err := clay.InitLogger()
		cobra.CheckErr(err)
	},
}

func main() {
	_ = rootCmd.Execute()
}

func init() {
	helpSystem := help.NewHelpSystem()
	err := helpSystem.LoadSectionsFromFS(docFS, ".")
	if err != nil {
		panic(err)
	}

	helpFunc, usageFunc := help.GetCobraHelpUsageFuncs(helpSystem)
	helpTemplate, usageTemplate := help.GetCobraHelpUsageTemplates(helpSystem)

	_ = usageFunc
	_ = usageTemplate

	//sections, err := openai.LoadModelsHelpFiles()
	//if err != nil {
	//	log.Error().Err(err).Msg("Error loading models help files")
	//}
	//for _, section := range sections {
	//	helpSystem.AddSection(section)
	//}
	//
	rootCmd.SetHelpFunc(helpFunc)
	rootCmd.SetUsageFunc(usageFunc)
	rootCmd.SetHelpTemplate(helpTemplate)
	rootCmd.SetUsageTemplate(usageTemplate)

	helpCmd := help.NewCobraHelpCommand(helpSystem)
	rootCmd.SetHelpCommand(helpCmd)

	err = clay.InitViper("pinocchio", rootCmd)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error initializing config: %s\n", err)
		os.Exit(1)
	}
	err = clay.InitLogger()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error initializing logger: %s\n", err)
		os.Exit(1)
	}

	repositories := viper.GetStringSlice("repositories")

	defaultDirectory := "$HOME/.pinocchio/prompts"
	repositories = append(repositories, defaultDirectory)

	locations := clay.CommandLocations{
		Embedded: []clay.EmbeddedCommandLocation{
			{
				FS:      promptsFS,
				Name:    "embed",
				Root:    ".",
				DocRoot: "prompts/doc",
			},
		},
		Repositories: repositories,
	}

	glazedParameterLayer, err := cli.NewGlazedParameterLayers(
		cli.WithSelectParameterLayerOptions(
			layers.WithDefaults(
				&cli.SelectSettings{
					SelectField: "response",
				},
			),
		),
	)
	if err != nil {
		panic(err)
	}

	yamlLoader := glazed_cmds.NewYAMLFSCommandLoader(
		&cmds.GeppettoCommandLoader{}, "", "")
	commands, aliases, err := locations.LoadCommands(
		yamlLoader, helpSystem, rootCmd, glazed_cmds.WithReplaceLayers(glazedParameterLayer))

	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error initializing commands: %s\n", err)
		os.Exit(1)
	}
	_ = commands
	_ = aliases

	rootCmd.AddCommand(openai.OpenaiCmd)

	rootCmd.AddCommand(ui.UiCmd)
}

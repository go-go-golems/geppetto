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
	cobra.CheckErr(err)

	helpSystem.SetupCobraRootCommand(rootCmd)

	//sections, err := openai.LoadModelsHelpFiles()
	//if err != nil {
	//	log.Error().Err(err).Msg("Error loading models help files")
	//}
	//for _, section := range sections {
	//	helpSystem.AddSection(section)
	//}
	//

	err = clay.InitViper("pinocchio", rootCmd)
	cobra.CheckErr(err)
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
	commandLoader := clay.NewCommandLoader[*cmds.GeppettoCommand](&locations)
	commands, aliases, err := commandLoader.LoadCommands(
		yamlLoader, helpSystem, rootCmd, glazed_cmds.WithReplaceLayers(glazedParameterLayer))

	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error initializing commands: %s\n", err)
		os.Exit(1)
	}

	glazeCommands, ok := clay.CastList[glazed_cmds.GlazeCommand](commands)
	if !ok {
		_, _ = fmt.Fprintf(os.Stderr, "Error initializing commands: %s\n", err)
		os.Exit(1)
	}
	err = cli.AddCommandsToRootCommand(rootCmd, glazeCommands, aliases)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error initializing commands: %s\n", err)
		os.Exit(1)
	}

	rootCmd.AddCommand(openai.OpenaiCmd)

	rootCmd.AddCommand(ui.UiCmd)
}

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
	"github.com/go-go-golems/glazed/pkg/helpers/cast"
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
	// first, check if the args are "run-command file.yaml",
	// because we need to load the file and then run the command itself.
	// we need to do this before cobra, because we don't know which flags to load yet
	if len(os.Args) >= 3 && os.Args[1] == "run-command" && os.Args[2] != "--help" {
		// load the command
		loader := &cmds.GeppettoCommandLoader{}
		f, err := os.Open(os.Args[2])
		if err != nil {
			fmt.Printf("Could not open file: %v\n", err)
			os.Exit(1)
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
		cobra.CheckErr(err)

		cmds_, err := loader.LoadCommandFromYAML(f, glazed_cmds.WithReplaceLayers(glazedParameterLayer))
		if err != nil {
			fmt.Printf("Could not load command: %v\n", err)
			os.Exit(1)
		}
		if len(cmds_) != 1 {
			fmt.Printf("Expected exactly one command, got %d", len(cmds_))
		}

		glazeCommand, ok := cmds_[0].(glazed_cmds.GlazeCommand)
		if !ok {
			fmt.Printf("Expected GlazeCommand, got %T", cmds_[0])
			os.Exit(1)
		}

		cobraCommand, err := cli.BuildCobraCommandFromGlazeCommand(glazeCommand)
		if err != nil {
			fmt.Printf("Could not build cobra command: %v\n", err)
			os.Exit(1)
		}

		_, err = initRootCmd()
		cobra.CheckErr(err)

		rootCmd.AddCommand(cobraCommand)
		restArgs := os.Args[3:]
		os.Args = append([]string{os.Args[0], cobraCommand.Use}, restArgs...)
	} else {
		helpSystem, err := initRootCmd()
		cobra.CheckErr(err)

		err = initAllCommands(helpSystem)
		cobra.CheckErr(err)
	}

	err := rootCmd.Execute()
	cobra.CheckErr(err)
}

var runCommandCmd = &cobra.Command{
	Use:   "run-command",
	Short: "Run a command from a file",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		panic(fmt.Errorf("not implemented"))
	},
}

func initRootCmd() (*help.HelpSystem, error) {
	helpSystem := help.NewHelpSystem()
	err := helpSystem.LoadSectionsFromFS(docFS, ".")
	cobra.CheckErr(err)

	helpSystem.SetupCobraRootCommand(rootCmd)

	err = clay.InitViper("pinocchio", rootCmd)
	cobra.CheckErr(err)
	err = clay.InitLogger()
	cobra.CheckErr(err)

	rootCmd.AddCommand(runCommandCmd)
	return helpSystem, nil
}

func initAllCommands(helpSystem *help.HelpSystem) error {
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
		return err
	}

	yamlLoader := glazed_cmds.NewYAMLFSCommandLoader(
		&cmds.GeppettoCommandLoader{}, "", "")
	commandLoader := clay.NewCommandLoader[*cmds.GeppettoCommand](&locations)
	commands, aliases, err := commandLoader.LoadCommands(
		yamlLoader, helpSystem, rootCmd, glazed_cmds.WithReplaceLayers(glazedParameterLayer))

	if err != nil {
		return err
	}

	glazeCommands, ok := cast.CastList[glazed_cmds.GlazeCommand](commands)
	if !ok {
		return fmt.Errorf("could not cast commands to GlazeCommand")
	}
	err = cli.AddCommandsToRootCommand(rootCmd, glazeCommands, aliases)
	if err != nil {
		return err
	}

	rootCmd.AddCommand(openai.OpenaiCmd)

	rootCmd.AddCommand(ui.UiCmd)

	return nil
}

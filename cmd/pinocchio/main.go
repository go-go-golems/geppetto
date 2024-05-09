package main

import (
	"embed"
	"fmt"
	clay "github.com/go-go-golems/clay/pkg"
	edit_command "github.com/go-go-golems/clay/pkg/cmds/edit-command"
	ls_commands "github.com/go-go-golems/clay/pkg/cmds/ls-commands"
	"github.com/go-go-golems/clay/pkg/repositories"
	"github.com/go-go-golems/clay/pkg/sql"
	pinocchio_cmds "github.com/go-go-golems/geppetto/cmd/pinocchio/cmds"
	"github.com/go-go-golems/geppetto/cmd/pinocchio/cmds/kagi"
	"github.com/go-go-golems/geppetto/cmd/pinocchio/cmds/openai"
	"github.com/go-go-golems/geppetto/cmd/pinocchio/cmds/tokens"
	"github.com/go-go-golems/geppetto/pkg/cmds"
	"github.com/go-go-golems/geppetto/pkg/doc"
	"github.com/go-go-golems/glazed/pkg/cli"
	glazed_cmds "github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/alias"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/cmds/loaders"
	"github.com/go-go-golems/glazed/pkg/help"
	"github.com/go-go-golems/glazed/pkg/types"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
)

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

		fs_, filePath, err := loaders.FileNameToFsFilePath(os.Args[2])
		if err != nil {
			fmt.Printf("Could not get absolute path: %v\n", err)
			os.Exit(1)
		}
		cmds_, err := loaders.LoadCommandsFromFS(fs_, filePath, os.Args[2], loader, []glazed_cmds.CommandDescriptionOption{}, []alias.Option{})
		if err != nil {
			fmt.Printf("Could not load command: %v\n", err)
			os.Exit(1)
		}
		if len(cmds_) != 1 {
			fmt.Printf("Expected exactly one command, got %d", len(cmds_))
		}

		cobraCommand, err := cmds.BuildCobraCommandWithGeppettoMiddlewares(cmds_[0])
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
	err := doc.AddDocToHelpSystem(helpSystem)
	cobra.CheckErr(err)

	helpSystem.SetupCobraRootCommand(rootCmd)

	err = clay.InitViper("pinocchio", rootCmd)
	cobra.CheckErr(err)
	err = clay.InitLogger()
	cobra.CheckErr(err)

	rootCmd.AddCommand(runCommandCmd)
	rootCmd.AddCommand(pinocchio_cmds.NewCodegenCommand())
	return helpSystem, nil
}

func initAllCommands(helpSystem *help.HelpSystem) error {
	repositoryPaths := viper.GetStringSlice("repositories")

	defaultDirectory := "$HOME/.pinocchio/prompts"
	repositoryPaths = append(repositoryPaths, defaultDirectory)

	loader := &cmds.GeppettoCommandLoader{}

	directories := []repositories.Directory{
		{
			FS:               promptsFS,
			RootDirectory:    "prompts",
			RootDocDirectory: "prompts/doc",
			Name:             "pinocchio",
			SourcePrefix:     "embed",
		}}

	for _, repositoryPath := range repositoryPaths {
		dir := os.ExpandEnv(repositoryPath)
		// check if dir exists
		if fi, err := os.Stat(dir); os.IsNotExist(err) || !fi.IsDir() {
			continue
		}
		directories = append(directories, repositories.Directory{
			FS:               os.DirFS(dir),
			RootDirectory:    ".",
			RootDocDirectory: "doc",
			WatchDirectory:   dir,
			Name:             dir,
			SourcePrefix:     "file",
		})
	}

	repositories_ := []*repositories.Repository{
		repositories.NewRepository(
			repositories.WithDirectories(directories...),
			repositories.WithCommandLoader(loader),
		),
	}

	allCommands, err := repositories.LoadRepositories(
		helpSystem,
		rootCmd,
		repositories_,
		cli.WithCobraMiddlewaresFunc(cmds.GetCobraCommandGeppettoMiddlewares),
		cli.WithCobraShortHelpLayers(layers.DefaultSlug, cmds.GeppettoHelpersSlug),
	)
	if err != nil {
		return err
	}

	lsCommandsCommand, err := ls_commands.NewListCommandsCommand(allCommands,
		ls_commands.WithCommandDescriptionOptions(
			glazed_cmds.WithShort("Commands related to sqleton queries"),
		),
		ls_commands.WithAddCommandToRowFunc(func(
			command glazed_cmds.Command,
			row types.Row,
			parsedLayers *layers.ParsedLayers,
		) ([]types.Row, error) {
			ret := []types.Row{row}
			switch c := command.(type) {
			case *cmds.GeppettoCommand:
				row.Set("prompt", c.Prompt)
				row.Set("type", "geppetto")
			default:
			}

			return ret, nil
		}),
	)
	if err != nil {
		return err
	}
	cobraQueriesCommand, err := sql.BuildCobraCommandWithSqletonMiddlewares(lsCommandsCommand)
	if err != nil {
		return err
	}
	rootCmd.AddCommand(cobraQueriesCommand)

	rootCmd.AddCommand(openai.OpenaiCmd)

	tokens.RegisterCommands(rootCmd)

	kagiCmd := kagi.RegisterKagiCommands()
	rootCmd.AddCommand(kagiCmd)

	listCommandsCommand, err := ls_commands.NewListCommandsCommand(allCommands,
		ls_commands.WithCommandDescriptionOptions(
			glazed_cmds.WithShort("Commands related to sqleton queries"),
		),
	)

	if err != nil {
		return err
	}
	cobraListCommandsCommand, err := sql.BuildCobraCommandWithSqletonMiddlewares(listCommandsCommand)
	if err != nil {
		return err
	}
	rootCmd.AddCommand(cobraListCommandsCommand)

	editCommandCommand, err := edit_command.NewEditCommand(allCommands)
	if err != nil {
		return err
	}
	cobraEditCommandCommand, err := sql.BuildCobraCommandWithSqletonMiddlewares(editCommandCommand)
	if err != nil {
		return err
	}
	rootCmd.AddCommand(cobraEditCommandCommand)

	return nil
}

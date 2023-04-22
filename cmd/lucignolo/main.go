package main

import (
	"context"
	"embed"
	"fmt"
	clay "github.com/go-go-golems/clay/pkg"
	clay_cmds "github.com/go-go-golems/clay/pkg/cmds"
	"github.com/go-go-golems/geppetto/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds/loaders"
	"github.com/go-go-golems/glazed/pkg/help"
	"github.com/go-go-golems/glazed/pkg/helpers"
	parka "github.com/go-go-golems/parka/pkg"
	"github.com/go-go-golems/parka/pkg/glazed"
	"github.com/go-go-golems/parka/pkg/render"
	"github.com/spf13/cobra"
	"os"
)

//go:embed doc/*
var docFS embed.FS

//go:embed prompts/*
var promptsFS embed.FS

var rootCmd = &cobra.Command{
	Use:   "lucignolo",
	Short: "lucignolo is a little proof-of-concept web service for LLM applications",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// reinitialize the logger because we can now parse --log-level and co
		// from the command line flag
		err := clay.InitLogger()
		cobra.CheckErr(err)
	},
}

func main() {
	helpSystem := help.NewHelpSystem()
	err := helpSystem.LoadSectionsFromFS(docFS, ".")
	if err != nil {
		panic(err)
	}

	helpSystem.SetupCobraRootCommand(rootCmd)

	err = clay.InitViper("lucignolo", rootCmd)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error initializing config: %s\n", err)
		os.Exit(1)
	}
	err = clay.InitLogger()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error initializing logger: %s\n", err)
		os.Exit(1)
	}

	locations := clay_cmds.CommandLocations{
		Embedded: []clay_cmds.EmbeddedCommandLocation{
			{
				FS:      promptsFS,
				Name:    "embed",
				Root:    ".",
				DocRoot: "prompts/doc",
			},
		},
	}

	yamlLoader := loaders.NewYAMLFSCommandLoader(&cmds.GeppettoCommandLoader{})
	commandLoader := clay_cmds.NewCommandLoader[*cmds.GeppettoCommand](&locations)
	commands, aliases, err := commandLoader.LoadCommands(
		yamlLoader, helpSystem)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error initializing commands: %s\n", err)
		os.Exit(1)
	}
	_ = commands
	_ = aliases

	var ServeCmd = &cobra.Command{
		Use:   "serve",
		Short: "Starts lucignolo",
		Run: func(cmd *cobra.Command, args []string) {
			_, err := cmd.Flags().GetUint16("port")
			cobra.CheckErr(err)

			serverOptions := []parka.ServerOption{}

			templateDir, err := cmd.Flags().GetString("template-dir")
			cobra.CheckErr(err)
			if templateDir != "" {
				serverOptions = append(
					serverOptions,
					parka.WithPrependTemplateLookups(render.LookupTemplateFromDirectory(templateDir)),
				)
			}

			s, _ := parka.NewServer(serverOptions...)

			glazedParameterLayers, err := cli.NewGlazedParameterLayers()
			cobra.CheckErr(err)

			for _, command := range commands {
				d := command.Description()
				s.Router.GET("/api/"+d.Name, s.HandleSimpleQueryCommand(command,
					glazed.WithParserOptions(
						glazed.WithGlazeOutputParserOption(glazedParameterLayers, "table", "html"),
					),
				))
				s.Router.POST("api/"+d.Name, s.HandleSimpleFormCommand(command))
			}
			for _, alias := range aliases {
				d := alias.Description()
				s.Router.GET("/api/"+d.Name, s.HandleSimpleQueryCommand(alias,
					glazed.WithParserOptions(
						glazed.WithGlazeOutputParserOption(glazedParameterLayers, "table", "html"),
					),
				))
				s.Router.POST("api/"+d.Name, s.HandleSimpleFormCommand(alias))
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			go func() {
				err := helpers.CancelOnSignal(ctx, os.Interrupt, cancel)
				if err != nil && err != context.Canceled {
					fmt.Println(err)
				}
			}()
			err = s.Run(ctx)

			cobra.CheckErr(err)
		},
	}

	ServeCmd.Flags().Uint16P("port", "p", 8080, "Port to listen on")
	ServeCmd.Flags().StringP("template-dir", "t", "", "Directory containing the templates")

	rootCmd.AddCommand(ServeCmd)
	_ = rootCmd.Execute()
}

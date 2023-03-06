package main

import (
	"embed"
	"fmt"
	clay "github.com/go-go-golems/clay/pkg"
	"github.com/go-go-golems/geppetto/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cli"
	glazed_cmds "github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/help"
	parka "github.com/go-go-golems/parka/pkg"
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

	locations := clay.CommandLocations{
		Embedded: []clay.EmbeddedCommandLocation{
			{
				FS:      promptsFS,
				Name:    "embed",
				Root:    ".",
				DocRoot: "prompts/doc",
			},
		},
	}

	yamlLoader := glazed_cmds.NewYAMLFSCommandLoader(
		&cmds.GeppettoCommandLoader{}, "", "")
	commands, aliases, err := locations.LoadCommands(
		yamlLoader, helpSystem, rootCmd)
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
					parka.WithPrependTemplateLookups(parka.LookupTemplateFromDirectory(templateDir)),
				)
			}

			s, _ := parka.NewServer(serverOptions...)

			glazedParameterLayers, err := cli.NewGlazedParameterLayers()
			cobra.CheckErr(err)

			for _, command := range commands {
				d := command.Description()
				s.Router.GET("/api/"+d.Name, s.HandleSimpleQueryCommand(command,
					[]parka.ParserHandlerOption{
						parka.WithGlazeOutputParserOption(glazedParameterLayers, "table", "html"),
					},
				))
				s.Router.POST("api/"+d.Name, s.HandleSimpleFormCommand(command))
			}
			for _, alias := range aliases {
				d := alias.Description()
				s.Router.GET("/api/"+d.Name, s.HandleSimpleQueryCommand(alias,
					[]parka.ParserHandlerOption{
						parka.WithGlazeOutputParserOption(glazedParameterLayers, "table", "html"),
					},
				))
				s.Router.POST("api/"+d.Name, s.HandleSimpleFormCommand(alias))
			}

			err = s.Run()
			cobra.CheckErr(err)
		},
	}

	ServeCmd.Flags().Uint16P("port", "p", 8080, "Port to listen on")
	ServeCmd.Flags().StringP("template-dir", "t", "", "Directory containing the templates")

	rootCmd.AddCommand(ServeCmd)
	_ = rootCmd.Execute()
}

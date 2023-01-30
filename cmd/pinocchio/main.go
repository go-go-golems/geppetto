package main

import (
	"embed"
	"fmt"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/wesen/geppetto/cmd/pinocchio/cmds/openai"
	geppetto_cmds "github.com/wesen/geppetto/pkg/cmds"
	glazed_cmds "github.com/wesen/glazed/pkg/cmds"
	"github.com/wesen/glazed/pkg/help"
	"gopkg.in/natefinch/lumberjack.v2"
	"io"
	"os"
	"strings"
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
		initLogger()
	},
}

func initLogger() {
	logLevel := viper.GetString("log-level")
	verbose := viper.GetBool("verbose")
	if verbose && logLevel != "trace" {
		logLevel = "debug"
	}

	err := InitLogger(&logConfig{
		Level:      logLevel,
		LogFile:    viper.GetString("log-file"),
		LogFormat:  viper.GetString("log-format"),
		WithCaller: viper.GetBool("with-caller"),
	})
	cobra.CheckErr(err)
}

type logConfig struct {
	WithCaller bool
	Level      string
	LogFormat  string
	LogFile    string
}

func loadRepositoryCommands(helpSystem *help.HelpSystem) ([]*geppetto_cmds.GeppettoCommand, []*glazed_cmds.CommandAlias, error) {
	repositories := viper.GetStringSlice("repositories")

	loader := &geppetto_cmds.GeppettoCommandLoader{}

	xdgDirectory, err := os.UserConfigDir()
	if err != nil {
		return nil, nil, err
	}
	defaultDirectory := fmt.Sprintf("%s/pinocchio/queries", xdgDirectory)
	repositories = append(repositories, defaultDirectory)

	commands := make([]*geppetto_cmds.GeppettoCommand, 0)
	aliases := make([]*glazed_cmds.CommandAlias, 0)

	for _, repository := range repositories {
		repository = os.ExpandEnv(repository)

		// check that repository exists and is a directory
		s, err := os.Stat(repository)

		if os.IsNotExist(err) {
			log.Debug().Msgf("Repository %s does not exist", repository)
			continue
		} else if err != nil {
			log.Warn().Msgf("Error while checking directory %s: %s", repository, err)
			continue
		}

		if s == nil || !s.IsDir() {
			log.Warn().Msgf("Repository %s is not a directory", repository)
		} else {
			docDir := fmt.Sprintf("%s/doc", repository)
			commands_, aliases_, err := glazed_cmds.LoadCommandsFromDirectory(loader, repository, repository)
			if err != nil {
				return nil, nil, err
			}
			for _, command := range commands_ {
				commands = append(commands, command.(*geppetto_cmds.GeppettoCommand))
			}
			aliases = append(aliases, aliases_...)

			_, err = os.Stat(docDir)
			if os.IsNotExist(err) {
				continue
			} else if err != nil {
				log.Debug().Err(err).Msgf("Error while checking directory %s", docDir)
				continue
			}
			err = helpSystem.LoadSectionsFromDirectory(docDir)
			if err != nil {
				log.Warn().Err(err).Msgf("Error while loading help sections from directory %s", repository)
				continue
			}
		}
	}
	return commands, aliases, nil
}

func initCommands(
	rootCmd *cobra.Command, configPath string, helpSystem *help.HelpSystem,
) ([]*geppetto_cmds.GeppettoCommand, []*glazed_cmds.CommandAlias, error) {
	// Load the variables from the environment
	viper.SetEnvPrefix("pinocchio")

	if configPath != "" {
		viper.SetConfigFile(configPath)
	} else {
		viper.AddConfigPath(".")
		viper.AddConfigPath("/etc/pinocchio")

		// get XDG config path for pinocchio
		xdgConfigPath, err := os.UserConfigDir()
		if err == nil {
			viper.AddConfigPath(xdgConfigPath + "/pinocchio")
		}
	}

	// Read the configuration file into Viper
	err := viper.ReadInConfig()
	// if the file does not exist, continue normally
	if _, ok := err.(viper.ConfigFileNotFoundError); ok {
		// Config file not found; ignore error
	} else if err != nil {
		// Config file was found but another error was produced
		return nil, nil, err
	}
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()

	// Bind the variables to the command-line flags
	err = viper.BindPFlags(rootCmd.PersistentFlags())
	if err != nil {
		return nil, nil, err
	}

	// this still won't pick up on --verbose to show debug logging when the commands
	// are parsed, but at least it will configure it based on the config file
	initLogger()

	log.Debug().
		Str("config", viper.ConfigFileUsed()).
		Msg("Loaded configuration")

	loader := &geppetto_cmds.GeppettoCommandLoader{}
	var commands []*geppetto_cmds.GeppettoCommand
	commands_, aliases, err := glazed_cmds.LoadCommandsFromEmbedFS(loader, promptsFS, ".", "prompts/")
	if err != nil {
		return nil, nil, err
	}
	for _, command := range commands_ {
		commands = append(commands, command.(*geppetto_cmds.GeppettoCommand))
	}

	err = helpSystem.LoadSectionsFromEmbedFS(promptsFS, "prompts/doc")
	if err != nil {
		return nil, nil, err
	}

	repositoryCommands, repositoryAliases, err := loadRepositoryCommands(helpSystem)
	if err != nil {
		return nil, nil, err
	}

	commands = append(commands, repositoryCommands...)
	aliases = append(aliases, repositoryAliases...)

	var cobraCommands []glazed_cmds.CobraCommand
	for _, command := range commands {
		cobraCommands = append(cobraCommands, command)
	}

	err = glazed_cmds.AddCommandsToRootCommand(rootCmd, cobraCommands, aliases)
	if err != nil {
		return nil, nil, err
	}

	return commands, aliases, nil
}

func InitLogger(config *logConfig) error {
	if config.WithCaller {
		log.Logger = log.With().Caller().Logger()
	}
	// default is json
	var logWriter io.Writer
	if config.LogFormat == "text" {
		logWriter = zerolog.ConsoleWriter{Out: os.Stderr}
	} else {
		logWriter = os.Stderr
	}

	if config.LogFile != "" {
		logWriter = io.MultiWriter(
			logWriter,
			zerolog.ConsoleWriter{
				NoColor: true,
				Out: &lumberjack.Logger{
					Filename:   config.LogFile,
					MaxSize:    10, // megabytes
					MaxBackups: 3,
					MaxAge:     28,    //days
					Compress:   false, // disabled by default
				},
			})
	}

	log.Logger = log.Output(logWriter)

	switch config.Level {
	case "debug":
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case "info":
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	case "warn":
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	case "error":
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	case "fatal":
		zerolog.SetGlobalLevel(zerolog.FatalLevel)
	}

	return nil
}

func main() {
	_ = rootCmd.Execute()
}

func init() {
	helpSystem := help.NewHelpSystem()
	err := helpSystem.LoadSectionsFromEmbedFS(docFS, ".")
	if err != nil {
		panic(err)
	}

	helpFunc, usageFunc := help.GetCobraHelpUsageFuncs(helpSystem)
	helpTemplate, usageTemplate := help.GetCobraHelpUsageTemplates(helpSystem)

	_ = usageFunc
	_ = usageTemplate

	sections, err := openai.LoadModelsHelpFiles()
	if err != nil {
		log.Error().Err(err).Msg("Error loading models help files")
	}
	for _, section := range sections {
		helpSystem.AddSection(section)
	}

	rootCmd.SetHelpFunc(helpFunc)
	rootCmd.SetUsageFunc(usageFunc)
	rootCmd.SetHelpTemplate(helpTemplate)
	rootCmd.SetUsageTemplate(usageTemplate)

	helpCmd := help.NewCobraHelpCommand(helpSystem)
	rootCmd.SetHelpCommand(helpCmd)

	// db connection persistent base flags
	// logging flags
	rootCmd.PersistentFlags().Bool("with-caller", false, "Log caller")
	rootCmd.PersistentFlags().String("log-level", "info", "Log level (debug, info, warn, error, fatal)")
	rootCmd.PersistentFlags().String("log-format", "text", "Log format (json, text)")
	rootCmd.PersistentFlags().String("log-file", "", "Log file (default: stderr)")

	rootCmd.PersistentFlags().String("config", "", "Path to config file (default ~/.pinocchio/config.yml)")
	rootCmd.PersistentFlags().Bool("verbose", false, "Verbose output")

	rootCmd.PersistentFlags().String("openai-api-key", "", "OpenAI API key")

	// parse the flags one time just to catch --config
	configFile := ""
	for idx, arg := range os.Args {
		if arg == "--config" {
			if len(os.Args) > idx+1 {
				configFile = os.Args[idx+1]
			}
		}
	}
	_ = configFile

	commands, aliases, err := initCommands(rootCmd, configFile, helpSystem)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error initializing commands: %s\n", err)
		os.Exit(1)
	}
	_ = commands
	_ = aliases

	rootCmd.AddCommand(openai.OpenaiCmd)
}

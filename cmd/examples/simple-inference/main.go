package main

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/go-go-golems/geppetto/pkg/conversation"
	"github.com/go-go-golems/geppetto/pkg/conversation/builder"
	"github.com/go-go-golems/geppetto/pkg/inference"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"

	clay "github.com/go-go-golems/clay/pkg"
	geppettolayers "github.com/go-go-golems/geppetto/pkg/layers"
	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
	"github.com/go-go-golems/glazed/pkg/help"
	help_cmd "github.com/go-go-golems/glazed/pkg/help/cmd"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var rootCmd = &cobra.Command{
	Use:   "simple-inference",
	Short: "Simple inference example with Engine-first architecture",
}

type SimpleInferenceCommand struct {
	*cmds.CommandDescription
}

var _ cmds.WriterCommand = (*SimpleInferenceCommand)(nil)

type SimpleInferenceSettings struct {
	PinocchioProfile string `glazed.parameter:"pinocchio-profile"`
	Debug            bool   `glazed.parameter:"debug"`
	WithLogging      bool   `glazed.parameter:"with-logging"`
	Prompt           string `glazed.parameter:"prompt"`
}

func NewSimpleInferenceCommand() (*SimpleInferenceCommand, error) {
	geppettoLayers, err := geppettolayers.CreateGeppettoLayers()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create geppetto parameter layer")
	}
	description := cmds.NewCommandDescription(
		"simple-inference",
		cmds.WithShort("Simple inference with Engine-first architecture"),
		cmds.WithArguments(
			parameters.NewParameterDefinition(
				"prompt",
				parameters.ParameterTypeString,
				parameters.WithHelp("Prompt to run"),
			),
		),
		cmds.WithFlags(
			parameters.NewParameterDefinition("pinocchio-profile",
				parameters.ParameterTypeString,
				parameters.WithHelp("Pinocchio profile"),
				parameters.WithDefault("4o-mini"),
			),
			parameters.NewParameterDefinition("debug",
				parameters.ParameterTypeBool,
				parameters.WithHelp("Debug mode - show parsed layers"),
				parameters.WithDefault(false),
			),
			parameters.NewParameterDefinition("with-logging",
				parameters.ParameterTypeBool,
				parameters.WithHelp("Enable logging middleware"),
				parameters.WithDefault(false),
			),
		),
		cmds.WithLayersList(
			geppettoLayers...,
		),
	)

	return &SimpleInferenceCommand{
		CommandDescription: description,
	}, nil
}

func (c *SimpleInferenceCommand) RunIntoWriter(ctx context.Context, parsedLayers *layers.ParsedLayers, w io.Writer) error {
	log.Info().Msg("Starting simple inference command")

	s := &SimpleInferenceSettings{}
	err := parsedLayers.InitializeStruct(layers.DefaultSlug, s)
	if err != nil {
		return errors.Wrap(err, "failed to initialize settings")
	}

	if s.Debug {
		b, err := yaml.Marshal(parsedLayers)
		if err != nil {
			return err
		}
		fmt.Fprintln(w, "=== Parsed Layers Debug ===")
		fmt.Fprintln(w, string(b))
		fmt.Fprintln(w, "=========================")
		return nil
	}

	stepSettings, err := settings.NewStepSettings()
	if err != nil {
		return err
	}
	err = stepSettings.UpdateFromParsedLayers(parsedLayers)
	if err != nil {
		return err
	}

	engine, err := inference.NewEngineFromParsedLayers(parsedLayers)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create engine")
		return errors.Wrap(err, "failed to create engine")
	}

	if s.WithLogging {
		loggingMiddleware := func(next inference.HandlerFunc) inference.HandlerFunc {
			return func(ctx context.Context, messages conversation.Conversation) (conversation.Conversation, error) {
				logger := log.With().Int("message_count", len(messages)).Logger()
				logger.Info().Msg("Starting inference")

				result, err := next(ctx, messages)
				if err != nil {
					logger.Error().Err(err).Msg("Inference failed")
				} else {
					logger.Info().Int("result_message_count", len(result)).Msg("Inference completed")
				}
				return result, err
			}
		}
		engine = inference.NewEngineWithMiddleware(engine, loggingMiddleware)
	}

	b := builder.NewManagerBuilder().
		WithSystemPrompt("You are a helpful assistant. Answer the question in a short and concise manner. ").
		WithPrompt(s.Prompt)

	manager, err := b.Build()
	if err != nil {
		log.Error().Err(err).Msg("Failed to build conversation manager")
		return err
	}

	conversation_ := manager.GetConversation()

	msg, err := engine.RunInference(ctx, conversation_)
	if err != nil {
		log.Error().Err(err).Msg("Inference failed")
		return fmt.Errorf("inference failed: %w", err)
	}

	if err := manager.AppendMessages(msg); err != nil {
		log.Error().Err(err).Msg("Failed to append message to conversation")
		return fmt.Errorf("failed to append message: %w", err)
	}

	messages := manager.GetConversation()

	fmt.Fprintln(w, "\n=== Final Conversation ===")
	for _, msg := range messages {
		if chatMsg, ok := msg.Content.(*conversation.ChatMessageContent); ok {
			fmt.Fprintf(w, "%s: %s\n", chatMsg.Role, chatMsg.Text)
		} else {
			fmt.Fprintf(w, "%s: %s\n", msg.Content.ContentType(), msg.Content.String())
		}
	}

	log.Info().Int("total_messages", len(messages)).Msg("Simple inference command completed successfully")
	return nil
}

func main() {
	// Initialize zerolog with pretty console output
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	err := clay.InitViper("pinocchio", rootCmd)
	cobra.CheckErr(err)

	helpSystem := help.NewHelpSystem()
	help_cmd.SetupCobraRootCommand(helpSystem, rootCmd)
	cobra.CheckErr(err)

	simpleCmd, err := NewSimpleInferenceCommand()
	cobra.CheckErr(err)

	command, err := cli.BuildCobraCommand(simpleCmd,
		cli.WithCobraMiddlewaresFunc(geppettolayers.GetCobraCommandGeppettoMiddlewares),
	)
	cobra.CheckErr(err)
	rootCmd.AddCommand(command)

	cobra.CheckErr(rootCmd.Execute())
}

package main

import (
	"context"
	"fmt"
	"io"

	clay "github.com/go-go-golems/clay/pkg"
	"github.com/go-go-golems/geppetto/pkg/conversation"
	"github.com/go-go-golems/geppetto/pkg/conversation/builder"
	"github.com/go-go-golems/geppetto/pkg/inference/engine/factory"
	"github.com/go-go-golems/geppetto/pkg/inference/middleware"
	geppettolayers "github.com/go-go-golems/geppetto/pkg/layers"
	"github.com/go-go-golems/geppetto/pkg/turns"
	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/cmds/logging"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
	"github.com/go-go-golems/glazed/pkg/help"
	help_cmd "github.com/go-go-golems/glazed/pkg/help/cmd"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var rootCmd = &cobra.Command{
	Use:   "simple-inference",
	Short: "Simple inference example with Engine-first architecture",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		err := logging.InitLoggerFromViper()
		if err != nil {
			return err
		}
		return nil
	},
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

	engine, err := factory.NewEngineFromParsedLayers(parsedLayers)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create engine")
		return errors.Wrap(err, "failed to create engine")
	}

	if s.WithLogging {
		loggingMiddleware := func(next middleware.HandlerFunc) middleware.HandlerFunc {
			return func(ctx context.Context, t *turns.Turn) (*turns.Turn, error) {
				logger := log.With().Int("block_count", len(t.Blocks)).Logger()
				logger.Info().Msg("Starting inference")

				result, err := next(ctx, t)
				if err != nil {
					logger.Error().Err(err).Msg("Inference failed")
				} else {
					logger.Info().Int("result_block_count", len(result.Blocks)).Msg("Inference completed")
				}
				return result, err
			}
		}
		engine = middleware.NewEngineWithMiddleware(engine, loggingMiddleware)
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
	// Initialize a Turn from the initial conversation (system + user prompt)
	initialTurn := &turns.Turn{}
	for _, msg := range conversation_ {
		if chatMsg, ok := msg.Content.(*conversation.ChatMessageContent); ok {
			kind := turns.BlockKindOther
			switch chatMsg.Role {
			case conversation.RoleSystem:
				kind = turns.BlockKindSystem
			case conversation.RoleUser:
				kind = turns.BlockKindUser
			case conversation.RoleAssistant:
				kind = turns.BlockKindLLMText
			case conversation.RoleTool:
				kind = turns.BlockKindOther
			}
			switch kind {
			case turns.BlockKindUser:
				turns.AppendBlock(initialTurn, turns.NewUserTextBlock(chatMsg.Text))
			case turns.BlockKindLLMText:
				turns.AppendBlock(initialTurn, turns.NewAssistantTextBlock(chatMsg.Text))
			case turns.BlockKindSystem:
				turns.AppendBlock(initialTurn, turns.NewSystemTextBlock(chatMsg.Text))
			default:
				turns.AppendBlock(initialTurn, turns.NewUserTextBlock(chatMsg.Text))
			}
		}
	}

	updatedTurn, err := engine.RunInference(ctx, initialTurn)
	if err != nil {
		log.Error().Err(err).Msg("Inference failed")
		return fmt.Errorf("inference failed: %w", err)
	}

	// Convert updated Turn back to conversation for display
	messages := turns.BuildConversationFromTurn(updatedTurn)

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

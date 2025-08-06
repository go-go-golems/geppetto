package main

import (
	"context"
	"fmt"
	"io"
	"strings"

	clay "github.com/go-go-golems/clay/pkg"
	"github.com/go-go-golems/geppetto/pkg/conversation"
	"github.com/go-go-golems/geppetto/pkg/conversation/builder"
	"github.com/go-go-golems/geppetto/pkg/inference/engine/factory"
	"github.com/go-go-golems/geppetto/pkg/inference/middleware"
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
	"os"
)

var rootCmd = &cobra.Command{
	Use:   "middleware-inference",
	Short: "A demo command that shows how to use middleware with inference engines",
	Long:  "This command demonstrates how to use logging and uppercase text transformation middleware with inference engines.",
}

type MiddlewareInferenceCommand struct {
	*cmds.CommandDescription
}

type MiddlewareInferenceSettings struct {
	PinocchioProfile string `glazed.parameter:"pinocchio-profile"`
	Debug            bool   `glazed.parameter:"debug"`
	WithLogging      bool   `glazed.parameter:"with-logging"`
	WithUppercase    bool   `glazed.parameter:"with-uppercase"`
	Prompt           string `glazed.parameter:"prompt"`
}

func NewMiddlewareInferenceCommand() (*MiddlewareInferenceCommand, error) {
	geppettoLayers, err := geppettolayers.CreateGeppettoLayers()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create geppetto layers")
	}

	description := cmds.NewCommandDescription(
		"middleware-inference",
		cmds.WithShort("Demonstrates middleware usage with engines"),
		cmds.WithLong("A demo command that shows how to use logging and uppercase text transformation middleware with inference engines."),
		cmds.WithArguments(
			parameters.NewParameterDefinition("prompt",
				parameters.ParameterTypeString,
				parameters.WithHelp("The prompt to send to the AI"),
				parameters.WithRequired(true),
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
			parameters.NewParameterDefinition("with-uppercase",
				parameters.ParameterTypeBool,
				parameters.WithHelp("Enable uppercase text transformation middleware"),
				parameters.WithDefault(false),
			),
		),
		cmds.WithLayersList(
			geppettoLayers...,
		),
	)

	return &MiddlewareInferenceCommand{
		CommandDescription: description,
	}, nil
}

func (c *MiddlewareInferenceCommand) RunIntoWriter(ctx context.Context, parsedLayers *layers.ParsedLayers, w io.Writer) error {
	log.Info().Msg("Starting middleware inference command")

	s := &MiddlewareInferenceSettings{}
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

	// Create base engine
	engine, err := factory.NewEngineFromParsedLayers(parsedLayers)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create engine")
		return errors.Wrap(err, "failed to create engine")
	}

	// Collect middleware
	var middlewares []middleware.Middleware

	// Add logging middleware if requested
	if s.WithLogging {
		loggingMiddleware := func(next middleware.HandlerFunc) middleware.HandlerFunc {
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
		middlewares = append(middlewares, loggingMiddleware)
	}

	// Add uppercase middleware if requested
	if s.WithUppercase {
		uppercaseMiddleware := func(next middleware.HandlerFunc) middleware.HandlerFunc {
			return func(ctx context.Context, messages conversation.Conversation) (conversation.Conversation, error) {
				result, err := next(ctx, messages)
				if err != nil {
					return result, err
				}

				// Transform the last message (AI response) to uppercase
				if len(result) > len(messages) {
					// Find new messages
					for i := len(messages); i < len(result); i++ {
						if chatContent, ok := result[i].Content.(*conversation.ChatMessageContent); ok {
							// Create a new message with uppercase text
							newContent := conversation.NewChatMessageContent(
								chatContent.Role,
								strings.ToUpper(chatContent.Text),
								chatContent.Images,
							)
							// Copy the message with metadata properly
							originalMsg := result[i]
							result[i] = &conversation.Message{
								ParentID:           originalMsg.ParentID,
								ID:                 originalMsg.ID,
								Time:               originalMsg.Time,
								LastUpdate:         originalMsg.LastUpdate,
								Content:            newContent,
								Metadata:           originalMsg.Metadata,
								LLMMessageMetadata: originalMsg.LLMMessageMetadata,
								Children:           originalMsg.Children,
							}
						}
					}
				}

				return result, nil
			}
		}
		middlewares = append(middlewares, uppercaseMiddleware)
	}

	// Wrap engine with middleware if any are provided
	if len(middlewares) > 0 {
		engine = middleware.NewEngineWithMiddleware(engine, middlewares...)
	}

	// Build conversation manager
	b := builder.NewManagerBuilder().
		WithSystemPrompt("You are a helpful assistant. Answer the question in a short and concise manner.").
		WithPrompt(s.Prompt)

	manager, err := b.Build()
	if err != nil {
		log.Error().Err(err).Msg("Failed to build conversation manager")
		return err
	}

	conversation_ := manager.GetConversation()

	// Run inference
	updatedConversation, err := engine.RunInference(ctx, conversation_)
	if err != nil {
		log.Error().Err(err).Msg("Inference failed")
		return fmt.Errorf("inference failed: %w", err)
	}

	// Extract new messages from the updated conversation
	newMessages := updatedConversation[len(conversation_):]
	for _, msg := range newMessages {
		if err := manager.AppendMessages(msg); err != nil {
			log.Error().Err(err).Msg("Failed to append message to conversation")
			return fmt.Errorf("failed to append message: %w", err)
		}
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

	middlewareList := []string{}
	if s.WithLogging {
		middlewareList = append(middlewareList, "logging")
	}
	if s.WithUppercase {
		middlewareList = append(middlewareList, "uppercase")
	}
	if len(middlewareList) > 0 {
		fmt.Fprintf(w, "\nApplied middleware: %s\n", strings.Join(middlewareList, ", "))
	} else {
		fmt.Fprintln(w, "\nNo middleware applied")
	}

	log.Info().Int("total_messages", len(messages)).Msg("Middleware inference command completed successfully")
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

	middlewareCmd, err := NewMiddlewareInferenceCommand()
	cobra.CheckErr(err)

	command, err := cli.BuildCobraCommand(middlewareCmd,
		cli.WithCobraMiddlewaresFunc(geppettolayers.GetCobraCommandGeppettoMiddlewares),
	)
	cobra.CheckErr(err)
	rootCmd.AddCommand(command)

	cobra.CheckErr(rootCmd.Execute())
}

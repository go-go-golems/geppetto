package main

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	clay "github.com/go-go-golems/clay/pkg"
	"github.com/go-go-golems/geppetto/cmd/examples/internal/examplebuilder"
	"github.com/go-go-golems/geppetto/pkg/conversation"
	"github.com/go-go-golems/geppetto/pkg/conversation/builder"
	enginepkg "github.com/go-go-golems/geppetto/pkg/inference/engine"
	"github.com/go-go-golems/geppetto/pkg/inference/middleware"
	"github.com/go-go-golems/geppetto/pkg/inference/session"
	"github.com/go-go-golems/geppetto/pkg/inference/toolcontext"
	"github.com/go-go-golems/geppetto/pkg/inference/tools"
	geppettolayers "github.com/go-go-golems/geppetto/pkg/layers"
	"github.com/go-go-golems/geppetto/pkg/turns"
	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/cmds/logging"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
	"github.com/go-go-golems/glazed/pkg/help"
	help_cmd "github.com/go-go-golems/glazed/pkg/help/cmd"
	"github.com/google/uuid"
	"github.com/invopop/jsonschema"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var rootCmd = &cobra.Command{
	Use:   "middleware-inference",
	Short: "A demo command that shows how to use middleware with inference engines",
	Long:  "This command demonstrates how to use logging and uppercase text transformation middleware with inference engines.",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return logging.InitLoggerFromCobra(cmd)
	},
}

type MiddlewareInferenceCommand struct {
	*cmds.CommandDescription
}

type MiddlewareInferenceSettings struct {
	PinocchioProfile string `glazed.parameter:"pinocchio-profile"`
	Debug            bool   `glazed.parameter:"debug"`
	WithLogging      bool   `glazed.parameter:"with-logging"`
	WithUppercase    bool   `glazed.parameter:"with-uppercase"`
	WithTools        bool   `glazed.parameter:"with-tools"`
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
			parameters.NewParameterDefinition("with-tools",
				parameters.ParameterTypeBool,
				parameters.WithHelp("Enable tool-calling middleware (expects provider to emit tool_call blocks)"),
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
	engBuilder := examplebuilder.NewParsedLayersEngineBuilder(parsedLayers, nil)
	engine, _, err := engBuilder.Build("", s.PinocchioProfile, nil)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create engine")
		return errors.Wrap(err, "failed to create engine")
	}

	// Collect middleware
	var middlewares []middleware.Middleware

	// Add logging middleware if requested
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
		middlewares = append(middlewares, loggingMiddleware)
	}

	// Add uppercase middleware if requested
	if s.WithUppercase {
		uppercaseMiddleware := func(next middleware.HandlerFunc) middleware.HandlerFunc {
			return func(ctx context.Context, t *turns.Turn) (*turns.Turn, error) {
				result, err := next(ctx, t)
				if err != nil {
					return result, err
				}
				// Uppercase any newly appended assistant LLMText blocks
				for i := range result.Blocks {
					b := &result.Blocks[i]
					if b.Kind == turns.BlockKindLLMText {
						if txt, ok := b.Payload[turns.PayloadKeyText].(string); ok {
							b.Payload[turns.PayloadKeyText] = strings.ToUpper(txt)
						}
					}
				}
				return result, nil
			}
		}
		middlewares = append(middlewares, uppercaseMiddleware)
	}

	if s.WithTools {
		// Minimal toolbox with a demo tool
		tb := middleware.NewMockToolbox()
		tb.RegisterTool("echo", "Echo back the input text", map[string]interface{}{
			"text": map[string]interface{}{"type": "string"},
		}, func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
			if v, ok := args["text"].(string); ok {
				return v, nil
			}
			return "", nil
		})
		toolMw := middleware.NewToolMiddleware(tb, middleware.ToolConfig{MaxIterations: 5, Timeout: 30 * time.Second})
		middlewares = append(middlewares, toolMw)
		// Attach registry and minimal tool config to Turn at seeding time below
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
	// Seed a Turn from the initial conversation
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
			case turns.BlockKindToolCall:
				turns.AppendBlock(initialTurn, turns.NewUserTextBlock(chatMsg.Text))
			case turns.BlockKindToolUse:
				turns.AppendBlock(initialTurn, turns.NewUserTextBlock(chatMsg.Text))
			case turns.BlockKindReasoning:
				// Current middleware example has no conversation representation for reasoning; skip.
			case turns.BlockKindOther:
				// Preserve Other/tool role as-is
				turns.AppendBlock(initialTurn, turns.NewUserTextBlock(chatMsg.Text))
			}
		}
	}

	// If tools enabled, attach registry to context and a minimal engine ToolConfig via Turn.Data
	if s.WithTools {
		echoSchema := &jsonschema.Schema{Type: "object"}
		props := jsonschema.NewProperties()
		props.Set("text", &jsonschema.Schema{Type: "string"})
		echoSchema.Properties = props
		echoSchema.Required = []string{"text"}
		// Build a lightweight registry the engine can read
		reg := tools.NewInMemoryToolRegistry()
		_ = reg.RegisterTool("echo", tools.ToolDefinition{
			Name:        "echo",
			Description: "Echo back the input text",
			Parameters:  echoSchema,
			Tags:        []string{"demo"},
			Version:     "1.0",
		})
		ctx = toolcontext.WithRegistry(ctx, reg)
		if err := enginepkg.KeyToolConfig.Set(&initialTurn.Data, enginepkg.ToolConfig{
			Enabled:          true,
			ToolChoice:       enginepkg.ToolChoiceAuto,
			MaxIterations:    5,
			ExecutionTimeout: 30 * time.Second,
			MaxParallelTools: 1,
		}); err != nil {
			return fmt.Errorf("set tool config: %w", err)
		}
	}

	// Run inference
	runID := uuid.NewString()
	initialTurn.RunID = runID
	sess := &session.Session{
		SessionID: runID,
		Builder:   &session.ToolLoopEngineBuilder{Base: engine},
		Turns:     []*turns.Turn{initialTurn},
	}
	handle, err := sess.StartInference(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Failed to start inference")
		return fmt.Errorf("failed to start inference: %w", err)
	}
	updatedTurn, err := handle.Wait()
	if err != nil {
		log.Error().Err(err).Msg("Inference failed")
		return fmt.Errorf("inference failed: %w", err)
	}

	// Display final result with the PrettyPrinter
	fmt.Fprintln(w, "\n=== Final Blocks ===")
	turns.FprintfTurn(w, updatedTurn,
		turns.WithIDs(false),
		turns.WithRoles(true),
		turns.WithToolDetail(true),
		turns.WithIndent(0),
		turns.WithMaxTextLines(0),
	)

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

	log.Info().Int("total_blocks", len(updatedTurn.Blocks)).Msg("Middleware inference command completed successfully")
	return nil
}

func main() {
	err := clay.InitGlazed("pinocchio", rootCmd)
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

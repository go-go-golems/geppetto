package main

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	clay "github.com/go-go-golems/clay/pkg"
	"github.com/go-go-golems/geppetto/pkg/inference/engine/factory"
	"github.com/go-go-golems/geppetto/pkg/inference/middleware"
	"github.com/go-go-golems/geppetto/pkg/inference/session"
	"github.com/go-go-golems/geppetto/pkg/inference/toolloop"
	"github.com/go-go-golems/geppetto/pkg/inference/toolloop/enginebuilder"
	"github.com/go-go-golems/geppetto/pkg/inference/tools"
	geppettosections "github.com/go-go-golems/geppetto/pkg/sections"
	"github.com/go-go-golems/geppetto/pkg/turns"
	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/logging"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/glazed/pkg/help"
	help_cmd "github.com/go-go-golems/glazed/pkg/help/cmd"
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
	PinocchioProfile string `glazed:"pinocchio-profile"`
	Debug            bool   `glazed:"debug"`
	WithLogging      bool   `glazed:"with-logging"`
	WithUppercase    bool   `glazed:"with-uppercase"`
	WithTools        bool   `glazed:"with-tools"`
	Prompt           string `glazed:"prompt"`
}

func NewMiddlewareInferenceCommand() (*MiddlewareInferenceCommand, error) {
	geppettoSections, err := geppettosections.CreateGeppettoSections()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create geppetto layers")
	}

	description := cmds.NewCommandDescription(
		"middleware-inference",
		cmds.WithShort("Demonstrates middleware usage with engines"),
		cmds.WithLong("A demo command that shows how to use logging and uppercase text transformation middleware with inference engines."),
		cmds.WithArguments(
			fields.New("prompt",
				fields.TypeString,
				fields.WithHelp("The prompt to send to the AI"),
				fields.WithRequired(true),
			),
		),
		cmds.WithFlags(
			fields.New("pinocchio-profile",
				fields.TypeString,
				fields.WithHelp("Pinocchio profile"),
				fields.WithDefault("4o-mini"),
			),
			fields.New("debug",
				fields.TypeBool,
				fields.WithHelp("Debug mode - show parsed layers"),
				fields.WithDefault(false),
			),
			fields.New("with-logging",
				fields.TypeBool,
				fields.WithHelp("Enable logging middleware"),
				fields.WithDefault(false),
			),
			fields.New("with-uppercase",
				fields.TypeBool,
				fields.WithHelp("Enable uppercase text transformation middleware"),
				fields.WithDefault(false),
			),
			fields.New("with-tools",
				fields.TypeBool,
				fields.WithHelp("Enable tool-calling middleware (expects provider to emit tool_call blocks)"),
				fields.WithDefault(false),
			),
		),
		cmds.WithSections(
			geppettoSections...,
		),
	)

	return &MiddlewareInferenceCommand{
		CommandDescription: description,
	}, nil
}

func (c *MiddlewareInferenceCommand) RunIntoWriter(ctx context.Context, parsedValues *values.Values, w io.Writer) error {
	log.Info().Msg("Starting middleware inference command")

	s := &MiddlewareInferenceSettings{}
	err := parsedValues.DecodeSectionInto(values.DefaultSlug, s)
	if err != nil {
		return errors.Wrap(err, "failed to initialize settings")
	}

	if s.Debug {
		b, err := yaml.Marshal(parsedValues)
		if err != nil {
			return err
		}
		fmt.Fprintln(w, "=== Parsed Layers Debug ===")
		fmt.Fprintln(w, string(b))
		fmt.Fprintln(w, "=========================")
		return nil
	}

	// Create base engine
	engine, err := factory.NewEngineFromParsedValues(parsedValues)
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

	if s.WithTools && s.Prompt == "" {
		s.Prompt = "Use the echo tool with text 'hello' and return the result."
	}

	initialTurn := &turns.Turn{}
	turns.AppendBlock(initialTurn, turns.NewSystemTextBlock(
		"You are a helpful assistant. Answer the question in a short and concise manner.",
	))
	turns.AppendBlock(initialTurn, turns.NewUserTextBlock(s.Prompt))

	var toolLoopRegistry tools.ToolRegistry
	var toolLoopLoopCfg toolloop.LoopConfig
	var toolLoopToolCfg tools.ToolConfig
	toolLoopEnabled := false
	if s.WithTools {
		type echoIn struct {
			Text string `json:"text" jsonschema:"required,description=The text to echo back"`
		}
		echoDef, err := tools.NewToolFromFunc("echo", "Echo back the input text", func(in echoIn) (map[string]any, error) {
			return map[string]any{"text": in.Text}, nil
		})
		if err != nil {
			return errors.Wrap(err, "failed to create echo tool")
		}

		reg := tools.NewInMemoryToolRegistry()
		if err := reg.RegisterTool("echo", *echoDef); err != nil {
			return errors.Wrap(err, "failed to register echo tool")
		}

		toolLoopRegistry = reg
		toolLoopLoopCfg = toolloop.NewLoopConfig().WithMaxIterations(5)
		toolLoopToolCfg = tools.DefaultToolConfig().
			WithExecutionTimeout(30 * time.Second).
			WithMaxParallelTools(1).
			WithToolChoice(tools.ToolChoiceAuto).
			WithToolErrorHandling(tools.ToolErrorContinue)
		toolLoopEnabled = true
	}

	// Run inference
	sess := session.NewSession()
	builderOpts := []enginebuilder.Option{
		enginebuilder.WithBase(engine),
		enginebuilder.WithMiddlewares(middlewares...),
	}
	if toolLoopEnabled {
		builderOpts = append(builderOpts,
			enginebuilder.WithToolRegistry(toolLoopRegistry),
			enginebuilder.WithLoopConfig(toolLoopLoopCfg),
			enginebuilder.WithToolConfig(toolLoopToolCfg),
		)
	}
	sess.Builder = enginebuilder.New(builderOpts...)
	sess.Append(initialTurn)
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
		cli.WithCobraMiddlewaresFunc(geppettosections.GetCobraCommandGeppettoMiddlewares),
	)
	cobra.CheckErr(err)
	rootCmd.AddCommand(command)

	cobra.CheckErr(rootCmd.Execute())
}

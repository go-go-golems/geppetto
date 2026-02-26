package main

import (
	"context"
	"fmt"
	"io"

	clay "github.com/go-go-golems/clay/pkg"
	"github.com/go-go-golems/geppetto/pkg/inference/engine/factory"
	"github.com/go-go-golems/geppetto/pkg/inference/middleware"
	"github.com/go-go-golems/geppetto/pkg/inference/session"
	"github.com/go-go-golems/geppetto/pkg/inference/toolloop/enginebuilder"
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
	Use:   "simple-inference",
	Short: "Simple inference example with Engine-first architecture",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return logging.InitLoggerFromCobra(cmd)
	},
}

type SimpleInferenceCommand struct {
	*cmds.CommandDescription
}

var _ cmds.WriterCommand = (*SimpleInferenceCommand)(nil)

type SimpleInferenceSettings struct {
	PinocchioProfile string `glazed:"pinocchio-profile"`
	Debug            bool   `glazed:"debug"`
	WithLogging      bool   `glazed:"with-logging"`
	Prompt           string `glazed:"prompt"`
}

func NewSimpleInferenceCommand() (*SimpleInferenceCommand, error) {
	geppettoSections, err := geppettosections.CreateGeppettoSections()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create geppetto parameter layer")
	}
	description := cmds.NewCommandDescription(
		"simple-inference",
		cmds.WithShort("Simple inference with Engine-first architecture"),
		cmds.WithArguments(
			fields.New(
				"prompt",
				fields.TypeString,
				fields.WithHelp("Prompt to run"),
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
		),
		cmds.WithSections(
			geppettoSections...,
		),
	)

	return &SimpleInferenceCommand{
		CommandDescription: description,
	}, nil
}

func (c *SimpleInferenceCommand) RunIntoWriter(ctx context.Context, parsedValues *values.Values, w io.Writer) error {
	log.Info().Msg("Starting simple inference command")

	s := &SimpleInferenceSettings{}
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

	engine, err := factory.NewEngineFromParsedValues(parsedValues)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create engine")
		return errors.Wrap(err, "failed to create engine")
	}

	var mws []middleware.Middleware
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
		mws = append(mws, loggingMiddleware)
	}

	// Seed initial Turn with Blocks (no conversation manager)
	initialTurn := turns.NewTurnBuilder().
		WithSystemPrompt("You are a helpful assistant. Answer the question in a short and concise manner. ").
		WithUserPrompt(s.Prompt).
		Build()

	sess := session.NewSession()
	sess.Builder = enginebuilder.New(
		enginebuilder.WithBase(engine),
		enginebuilder.WithMiddlewares(mws...),
	)
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

	fmt.Fprintln(w, "\n=== Final Turn ===")
	if updatedTurn != nil {
		turns.FprintTurn(w, updatedTurn)
	}

	log.Info().Int("total_blocks", len(updatedTurn.Blocks)).Msg("Simple inference command completed successfully")
	return nil
}

func main() {
	err := clay.InitGlazed("pinocchio", rootCmd)
	cobra.CheckErr(err)

	helpSystem := help.NewHelpSystem()
	help_cmd.SetupCobraRootCommand(helpSystem, rootCmd)
	cobra.CheckErr(err)

	simpleCmd, err := NewSimpleInferenceCommand()
	cobra.CheckErr(err)

	command, err := cli.BuildCobraCommand(simpleCmd,
		cli.WithCobraMiddlewaresFunc(geppettosections.GetCobraCommandGeppettoMiddlewares),
	)
	cobra.CheckErr(err)
	rootCmd.AddCommand(command)

	cobra.CheckErr(rootCmd.Execute())
}

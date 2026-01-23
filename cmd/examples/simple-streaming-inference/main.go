package main

import (
	"context"
	"fmt"
	"io"

	"github.com/go-go-golems/geppetto/pkg/events"
	"github.com/go-go-golems/geppetto/pkg/inference/engine/factory"
	"github.com/go-go-golems/geppetto/pkg/inference/middleware"
	"github.com/go-go-golems/geppetto/pkg/inference/session"
	"github.com/go-go-golems/geppetto/pkg/inference/toolloop"

	"github.com/go-go-golems/geppetto/pkg/turns"

	clay "github.com/go-go-golems/clay/pkg"
	geppettolayers "github.com/go-go-golems/geppetto/pkg/layers"
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
	"golang.org/x/sync/errgroup"
	"gopkg.in/yaml.v3"
)

var rootCmd = &cobra.Command{
	Use:   "simple-streaming-inference",
	Short: "Simple streaming inference example with Engine-first architecture",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		err := logging.InitLoggerFromCobra(cmd)
		if err != nil {
			return err
		}
		return nil
	},
}

type SimpleStreamingInferenceCommand struct {
	*cmds.CommandDescription
}

var _ cmds.WriterCommand = (*SimpleStreamingInferenceCommand)(nil)

type SimpleStreamingInferenceSettings struct {
	PinocchioProfile string `glazed.parameter:"pinocchio-profile"`
	Debug            bool   `glazed.parameter:"debug"`
	WithLogging      bool   `glazed.parameter:"with-logging"`
	Prompt           string `glazed.parameter:"prompt"`
	OutputFormat     string `glazed.parameter:"output-format"`
	WithMetadata     bool   `glazed.parameter:"with-metadata"`
	FullOutput       bool   `glazed.parameter:"full-output"`
	Verbose          bool   `glazed.parameter:"verbose"`
}

// TurnBuilder moved to turns package

func NewSimpleStreamingInferenceCommand() (*SimpleStreamingInferenceCommand, error) {
	geppettoLayers, err := geppettolayers.CreateGeppettoLayers()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create geppetto parameter layer")
	}
	description := cmds.NewCommandDescription(
		"simple-streaming-inference",
		cmds.WithShort("Simple streaming inference with Engine-first architecture"),
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
			parameters.NewParameterDefinition("output-format",
				parameters.ParameterTypeString,
				parameters.WithHelp("Output format (text, json, yaml)"),
				parameters.WithDefault("text"),
			),
			parameters.NewParameterDefinition("with-metadata",
				parameters.ParameterTypeBool,
				parameters.WithHelp("Include metadata in output"),
				parameters.WithDefault(false),
			),
			parameters.NewParameterDefinition("full-output",
				parameters.ParameterTypeBool,
				parameters.WithHelp("Include full output details"),
				parameters.WithDefault(false),
			),
			parameters.NewParameterDefinition("verbose",
				parameters.ParameterTypeBool,
				parameters.WithHelp("Verbose event router logging"),
				parameters.WithDefault(false),
			),
		),
		cmds.WithLayersList(
			geppettoLayers...,
		),
	)

	return &SimpleStreamingInferenceCommand{
		CommandDescription: description,
	}, nil
}

func (c *SimpleStreamingInferenceCommand) RunIntoWriter(ctx context.Context, parsedLayers *layers.ParsedLayers, w io.Writer) error {
	log.Info().Msg("Starting simple streaming inference command")

	s := &SimpleStreamingInferenceSettings{}
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

	// 1. Create event router
	routerOptions := []events.EventRouterOption{}
	if s.Verbose {
		routerOptions = append(routerOptions, events.WithVerbose(true))
	}

	router, err := events.NewEventRouter(routerOptions...)
	if err != nil {
		return errors.Wrap(err, "failed to create event router")
	}
	defer func() {
		if router != nil {
			_ = router.Close()
		}
	}()

	// 2. Create watermill sink
	watermillSink := middleware.NewWatermillSink(router.Publisher, "chat")

	// 3. Add printer handler based on output format
	if s.OutputFormat == "" {
		router.AddHandler("chat", "chat", events.StepPrinterFunc("", w))
	} else {
		printer := events.NewStructuredPrinter(w, events.PrinterOptions{
			Format:          events.PrinterFormat(s.OutputFormat),
			Name:            "",
			IncludeMetadata: s.WithMetadata,
			Full:            s.FullOutput,
		})
		router.AddHandler("chat", "chat", printer)
	}

	eng, err := factory.NewEngineFromParsedLayers(parsedLayers)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create engine")
		return errors.Wrap(err, "failed to create engine")
	}
	sink := watermillSink

	var mws []middleware.Middleware
	// Add logging middleware if requested
	if s.WithLogging {
		mws = append(mws, middleware.NewTurnLoggingMiddleware(log.Logger))
	}

	// Build initial Turn with Blocks (no conversation manager)
	seed := turns.NewTurnBuilder().
		WithSystemPrompt("You are a helpful assistant. Answer the question in a short and concise manner. ").
		WithUserPrompt(s.Prompt).
		Build()

	// 5. Start router and run inference in parallel
	eg := errgroup.Group{}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	eg.Go(func() error {
		defer cancel()
		return router.Run(ctx)
	})

	var finalTurn *turns.Turn
	eg.Go(func() error {
		defer cancel()
		<-router.Running()

		sess := session.NewSession()
		sess.Builder = toolloop.NewEngineBuilder(
			toolloop.WithBase(eng),
			toolloop.WithMiddlewares(mws...),
			toolloop.WithEventSinks(sink),
		)
		sess.Append(seed)
		handle, err := sess.StartInference(ctx)
		if err != nil {
			return fmt.Errorf("failed to start inference: %w", err)
		}
		updatedTurn, err := handle.Wait()
		if err != nil {
			log.Error().Err(err).Msg("Inference failed")
			return fmt.Errorf("inference failed: %w", err)
		}
		finalTurn = updatedTurn
		return nil
	})

	err = eg.Wait()
	if err != nil {
		return err
	}

	fmt.Fprintln(w, "\n=== Final Turn ===")
	if finalTurn != nil {
		turns.FprintTurn(w, finalTurn)
	}

	log.Info().Msg("Simple streaming inference command completed successfully")
	return nil
}

func main() {
	err := clay.InitGlazed("pinocchio", rootCmd)
	cobra.CheckErr(err)

	helpSystem := help.NewHelpSystem()
	help_cmd.SetupCobraRootCommand(helpSystem, rootCmd)
	cobra.CheckErr(err)

	simpleCmd, err := NewSimpleStreamingInferenceCommand()
	cobra.CheckErr(err)

	command, err := cli.BuildCobraCommand(simpleCmd,
		cli.WithCobraMiddlewaresFunc(geppettolayers.GetCobraCommandGeppettoMiddlewares),
	)
	cobra.CheckErr(err)
	rootCmd.AddCommand(command)

	cobra.CheckErr(rootCmd.Execute())
}

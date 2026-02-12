package main

import (
	"context"
	"fmt"
	"io"

	"github.com/go-go-golems/geppetto/pkg/events"
	"github.com/go-go-golems/geppetto/pkg/inference/engine/factory"
	"github.com/go-go-golems/geppetto/pkg/inference/middleware"
	"github.com/go-go-golems/geppetto/pkg/inference/session"
	"github.com/go-go-golems/geppetto/pkg/inference/toolloop/enginebuilder"

	"github.com/go-go-golems/geppetto/pkg/turns"

	clay "github.com/go-go-golems/clay/pkg"
	geppettosections "github.com/go-go-golems/geppetto/pkg/sections"
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
	PinocchioProfile string `glazed:"pinocchio-profile"`
	Debug            bool   `glazed:"debug"`
	WithLogging      bool   `glazed:"with-logging"`
	Prompt           string `glazed:"prompt"`
	OutputFormat     string `glazed:"output-format"`
	WithMetadata     bool   `glazed:"with-metadata"`
	FullOutput       bool   `glazed:"full-output"`
	Verbose          bool   `glazed:"verbose"`
}

// TurnBuilder moved to turns package

func NewSimpleStreamingInferenceCommand() (*SimpleStreamingInferenceCommand, error) {
	geppettoSections, err := geppettosections.CreateGeppettoSections()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create geppetto parameter layer")
	}
	description := cmds.NewCommandDescription(
		"simple-streaming-inference",
		cmds.WithShort("Simple streaming inference with Engine-first architecture"),
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
			fields.New("output-format",
				fields.TypeString,
				fields.WithHelp("Output format (text, json, yaml)"),
				fields.WithDefault("text"),
			),
			fields.New("with-metadata",
				fields.TypeBool,
				fields.WithHelp("Include metadata in output"),
				fields.WithDefault(false),
			),
			fields.New("full-output",
				fields.TypeBool,
				fields.WithHelp("Include full output details"),
				fields.WithDefault(false),
			),
			fields.New("verbose",
				fields.TypeBool,
				fields.WithHelp("Verbose event router logging"),
				fields.WithDefault(false),
			),
		),
		cmds.WithSections(
			geppettoSections...,
		),
	)

	return &SimpleStreamingInferenceCommand{
		CommandDescription: description,
	}, nil
}

func (c *SimpleStreamingInferenceCommand) RunIntoWriter(ctx context.Context, parsedValues *values.Values, w io.Writer) error {
	log.Info().Msg("Starting simple streaming inference command")

	s := &SimpleStreamingInferenceSettings{}
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

	eng, err := factory.NewEngineFromParsedValues(parsedValues)
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
		sess.Builder = enginebuilder.New(
			enginebuilder.WithBase(eng),
			enginebuilder.WithMiddlewares(mws...),
			enginebuilder.WithEventSinks(sink),
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
		cli.WithCobraMiddlewaresFunc(geppettosections.GetCobraCommandGeppettoMiddlewares),
	)
	cobra.CheckErr(err)
	rootCmd.AddCommand(command)

	cobra.CheckErr(rootCmd.Execute())
}

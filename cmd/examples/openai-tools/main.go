package main

import (
	"context"
	"fmt"
	"io"

	"github.com/ThreeDotsLabs/watermill/message"
	clay "github.com/go-go-golems/clay/pkg"
	"github.com/go-go-golems/geppetto/pkg/events"
	"github.com/go-go-golems/geppetto/pkg/inference/engine/factory"
	"github.com/go-go-golems/geppetto/pkg/inference/middleware"
	"github.com/go-go-golems/geppetto/pkg/inference/session"
	"github.com/go-go-golems/geppetto/pkg/inference/toolloop"
	"github.com/go-go-golems/geppetto/pkg/inference/toolloop/enginebuilder"
	"github.com/go-go-golems/geppetto/pkg/inference/tools"
	geppettolayers "github.com/go-go-golems/geppetto/pkg/layers"
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
	"golang.org/x/sync/errgroup"
)

// WeatherRequest represents the input for the weather tool
type WeatherRequest struct {
	Location string `json:"location" jsonschema:"required,description=The city and country to get weather for"`
	Units    string `json:"units,omitempty" jsonschema:"description=Temperature units (celsius or fahrenheit),default=celsius,enum=celsius,enum=fahrenheit"`
}

// WeatherResponse represents the weather tool's response
type WeatherResponse struct {
	Location    string  `json:"location"`
	Temperature float64 `json:"temperature"`
	Conditions  string  `json:"conditions"`
	Humidity    int     `json:"humidity"`
	Units       string  `json:"units"`
}

// weatherTool is a mock weather tool that returns fake data
func weatherTool(req WeatherRequest) WeatherResponse {
	log.Info().Str("location", req.Location).Str("units", req.Units).Msg("Weather tool called!")

	return WeatherResponse{
		Location:    req.Location,
		Temperature: 22.5,
		Conditions:  "Sunny",
		Humidity:    60,
		Units:       req.Units,
	}
}

// CalculatorRequest represents the input for a calculator tool
type CalculatorRequest struct {
	Expression string `json:"expression" jsonschema:"required,description=An arithmetic expression to evaluate,example=2*(3+4)-5"`
}

// calculatorTool evaluates a simple arithmetic expression (very basic)
func calculatorTool(req CalculatorRequest) (float64, error) {
	// Minimal, safe-ish parser: support + - * / and parentheses via Go's eval is not available; implement tiny stack-based or defer to strconv for numbers only
	// For demo, handle a limited set: a+b, a-b, a*b, a/b with optional spaces
	expr := req.Expression
	// try space-separated: "A op B"
	var a, b float64
	var op string
	n, _ := fmt.Sscanf(expr, "%f %s %f", &a, &op, &b)
	if n == 3 {
		switch op {
		case "+":
			return a + b, nil
		case "-":
			return a - b, nil
		case "*", "x":
			return a * b, nil
		case "/":
			if b != 0 {
				return a / b, nil
			}
			return 0, fmt.Errorf("division by zero")
		}
	}
	return 0, fmt.Errorf("unsupported expression: %s", expr)
}

var rootCmd = &cobra.Command{
	Use:   "test-openai-tools",
	Short: "Test OpenAI tools integration with debug logging",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return logging.InitLoggerFromCobra(cmd)
	},
}

type TestOpenAIToolsCommand struct {
	*cmds.CommandDescription
}

var _ cmds.WriterCommand = (*TestOpenAIToolsCommand)(nil)

type TestOpenAIToolsSettings struct {
	Debug        bool   `glazed:"debug"`
	OutputFormat string `glazed:"output-format"`
	WithMetadata bool   `glazed:"with-metadata"`
	FullOutput   bool   `glazed:"full-output"`
	Verbose      bool   `glazed:"verbose"`
	Mode         string `glazed:"mode"`
	Prompt       string `glazed:"prompt"`
}

func NewTestOpenAIToolsCommand() (*TestOpenAIToolsCommand, error) {
	geppettoLayers, err := geppettolayers.CreateGeppettoLayers()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create geppetto parameter layer")
	}

	description := cmds.NewCommandDescription(
		"test-openai-tools",
		cmds.WithShort("Test OpenAI tools integration with debug logging"),
		cmds.WithFlags(
			fields.New("debug",
				fields.TypeBool,
				fields.WithHelp("Enable debug logging"),
				fields.WithDefault(true),
			),
			fields.New("output-format",
				fields.TypeString,
				fields.WithHelp("Printer format (text, json, yaml)"),
				fields.WithDefault("text"),
			),
			fields.New("with-metadata",
				fields.TypeBool,
				fields.WithHelp("Include metadata in printed events"),
				fields.WithDefault(false),
			),
			fields.New("full-output",
				fields.TypeBool,
				fields.WithHelp("Include full event details"),
				fields.WithDefault(false),
			),
			fields.New("verbose",
				fields.TypeBool,
				fields.WithHelp("Verbose router logging (SSE debug)"),
				fields.WithDefault(false),
			),
			fields.New("mode",
				fields.TypeChoice,
				fields.WithChoices("tools", "thinking", "parallel-tools", "server-tools"),
				fields.WithHelp("Modes: tools (function calling), thinking (no tools), parallel-tools (multiple calls), server-tools (enable server-side tools)"),
				fields.WithDefault("tools"),
			),
			fields.New("prompt",
				fields.TypeString,
				fields.WithHelp("Override the default prompt for the selected mode"),
				fields.WithDefault(""),
			),
		),
		cmds.WithSections(
			geppettoLayers...,
		),
	)

	return &TestOpenAIToolsCommand{
		CommandDescription: description,
	}, nil
}

func (c *TestOpenAIToolsCommand) RunIntoWriter(ctx context.Context, parsedLayers *values.Values, w io.Writer) error {
	log.Debug().Msg("Debug logging enabled for tool testing")

	s := &TestOpenAIToolsSettings{}
	err := parsedLayers.DecodeSectionInto(values.DefaultSlug, s)
	if err != nil {
		return errors.Wrap(err, "failed to initialize settings")
	}

	// Create event router and printer to display streaming SSE events
	routerOpts := []events.EventRouterOption{}
	if s.Verbose {
		routerOpts = append(routerOpts, events.WithVerbose(true))
	}
	router, err := events.NewEventRouter(routerOpts...)
	if err != nil {
		return errors.Wrap(err, "failed to create event router")
	}
	defer func() {
		if closeErr := router.Close(); closeErr != nil {
			log.Error().Err(closeErr).Msg("failed to close event router")
		}
	}()
	if s.OutputFormat == "" || s.OutputFormat == "text" {
		router.AddHandler("chat-printer", "chat", events.StepPrinterFunc("", w))
	} else {
		printer := events.NewStructuredPrinter(w, events.PrinterOptions{
			Format:          events.PrinterFormat(s.OutputFormat),
			Name:            "",
			IncludeMetadata: s.WithMetadata,
			Full:            s.FullOutput,
		})
		router.AddHandler("chat-printer", "chat", printer)
	}

	// Thinking tokens printer: pretty summary on final events without requiring trace logs
	router.AddHandler("thinking-printer", "chat", func(msg *message.Message) error {
		ev, err := events.NewEventFromJson(msg.Payload)
		if err != nil {
			msg.Ack()
			return nil
		}
		meta := ev.Metadata()
		// Stream reasoning summary boundaries; thinking deltas come as EventPartialThinking
		if string(ev.Type()) == string(events.EventTypeInfo) {
			if ie, ok := events.ToTypedEvent[events.EventInfo](ev); ok && ie != nil {
				switch ie.Message {
				case "reasoning-summary-started":
					fmt.Fprintln(w, "\n--- Reasoning summary started ---")
				case "reasoning-summary-ended":
					fmt.Fprintln(w, "\n--- Reasoning summary ended ---")
				case "thinking-started":
					fmt.Fprintln(w, "\n--- Thinking started ---")
				case "thinking-ended":
					fmt.Fprintln(w, "\n--- Thinking ended ---")
				case "output-started":
					fmt.Fprintln(w, "\n--- Output started ---")
				case "output-ended":
					fmt.Fprintln(w, "\n--- Output ended ---")
				}
			}
		}
		// Print thinking partials as raw text
		if string(ev.Type()) == string(events.EventTypePartialThinking) {
			if tp, ok := events.ToTypedEvent[events.EventThinkingPartial](ev); ok && tp != nil {
				if tp.Delta != "" {
					if _, writeErr := fmt.Fprint(w, tp.Delta); writeErr != nil {
						log.Error().Err(writeErr).Msg("failed to write thinking delta")
					}
				}
			}
		}
		if string(ev.Type()) == "final" {
			extra := meta.Extra
			var rt any
			if extra != nil {
				if v, ok := extra["reasoning_tokens"]; ok {
					rt = v
				}
			}
			it := 0
			ot := 0
			if meta.Usage != nil {
				it = meta.Usage.InputTokens
				ot = meta.Usage.OutputTokens
			}
			// Pretty summary line separating reasoning vs normal streaming tokens
			if rt != nil {
				fmt.Fprintf(w, "\n🧠 Reasoning tokens: %v  |  📝 Output tokens: %d  |  📥 Input tokens: %d\n", rt, ot, it)
			} else {
				fmt.Fprintf(w, "\n📝 Output tokens: %d  |  📥 Input tokens: %d\n", ot, it)
			}
		}
		msg.Ack()
		return nil
	})

	// Create engine and wire an event sink to publish streaming events
	watermillSink := middleware.NewWatermillSink(router.Publisher, "chat")
	engineInstance, err := factory.NewEngineFromParsedLayers(parsedLayers)
	if err != nil {
		return errors.Wrap(err, "failed to create engine from parsed layers")
	}
	sink := watermillSink

	// Run the router concurrently
	eg := errgroup.Group{}
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	eg.Go(func() error { return router.Run(runCtx) })

	// Build mode-specific setup
	var (
		turn             *turns.Turn
		toolLoopRegistry tools.ToolRegistry
		toolLoopLoopCfg  toolloop.LoopConfig
		toolLoopToolCfg  tools.ToolConfig
		toolLoopEnabled  bool
	)

	switch s.Mode {
	case "thinking":
		p := s.Prompt
		if p == "" {
			p = "Think step-by-step and answer concisely: What is 23*17 + 55?"
		}
		turn = &turns.Turn{}
		turns.AppendBlock(turn, turns.NewUserTextBlock(p))
	case "server-tools":
		turn = &turns.Turn{}
		serverTools := []any{
			map[string]any{"type": "web_search"},
		}
		if err := turns.KeyResponsesServerTools.Set(&turn.Data, serverTools); err != nil {
			return errors.Wrap(err, "set responses server tools")
		}
		turns.AppendBlock(turn, turns.NewSystemTextBlock("You have access to the server-side web_search tool. Use it where appropriate."))
		userPrompt := s.Prompt
		if userPrompt == "" {
			userPrompt = "Use web_search to find information about the OpenAI Responses API reasoning items and summarize briefly."
		}
		turns.AppendBlock(turn, turns.NewUserTextBlock(userPrompt))
	default:
		weatherToolDef, err := tools.NewToolFromFunc(
			"get_weather",
			"Get current weather information for a specific location",
			weatherTool,
		)
		if err != nil {
			return errors.Wrap(err, "failed to create weather tool")
		}
		if weatherToolDef.Parameters != nil {
			fmt.Fprintf(w, "Tool schema type: %v\n", weatherToolDef.Parameters.Type)
			if weatherToolDef.Parameters.Properties != nil {
				fmt.Fprintf(w, "Tool schema properties count: %d\n", weatherToolDef.Parameters.Properties.Len())
				for pair := weatherToolDef.Parameters.Properties.Oldest(); pair != nil; pair = pair.Next() {
					fmt.Fprintf(w, "  - %s: type=%s\n", pair.Key, pair.Value.Type)
				}
			} else {
				fmt.Fprintln(w, "Tool schema properties: nil")
			}
		} else {
			fmt.Fprintln(w, "Warning: Tool schema is nil")
		}

		calcDef, err := tools.NewToolFromFunc(
			"calculator",
			"Evaluate a simple arithmetic expression (format: 'A op B')",
			calculatorTool,
		)
		if err != nil {
			return errors.Wrap(err, "failed to create calculator tool")
		}

		reg := tools.NewInMemoryToolRegistry()
		_ = reg.RegisterTool("get_weather", *weatherToolDef)
		_ = reg.RegisterTool("calculator", *calcDef)

		maxPar := 1
		if s.Mode == "parallel-tools" {
			maxPar = 2
		}

		turn = &turns.Turn{}
		toolLoopRegistry = reg
		toolLoopLoopCfg = toolloop.NewLoopConfig().WithMaxIterations(3)
		toolLoopToolCfg = tools.DefaultToolConfig().
			WithMaxParallelTools(maxPar).
			WithToolChoice(tools.ToolChoiceAuto).
			WithToolErrorHandling(tools.ToolErrorContinue)
		toolLoopEnabled = true

		userPrompt := s.Prompt
		if userPrompt == "" {
			if s.Mode == "parallel-tools" {
				userPrompt = "Use both tools: 1) get_weather for San Francisco (celsius); 2) calculator to compute 12 * 7."
			} else {
				userPrompt = "Please use get_weather to check the weather in San Francisco, in celsius."
			}
		}
		turns.AppendBlock(turn, turns.NewUserTextBlock(userPrompt))
	}

	// No explicit stateless toggle: encrypted reasoning is requested by default in the engine helper.
	sess := session.NewSession()
	builderOpts := []enginebuilder.Option{
		enginebuilder.WithBase(engineInstance),
		enginebuilder.WithEventSinks(sink),
	}
	if toolLoopEnabled {
		builderOpts = append(builderOpts,
			enginebuilder.WithToolRegistry(toolLoopRegistry),
			enginebuilder.WithLoopConfig(toolLoopLoopCfg),
			enginebuilder.WithToolConfig(toolLoopToolCfg),
		)
	}
	sess.Builder = enginebuilder.New(builderOpts...)
	sess.Append(turn)
	handle, err := sess.StartInference(runCtx)
	if err != nil {
		return errors.Wrap(err, "failed to start inference")
	}
	updatedTurn, err := handle.Wait()
	if err != nil {
		return errors.Wrap(err, "inference with tools failed")
	}

	// Render final result with the PrettyPrinter
	fmt.Fprintln(w, "\nWorkflow completed. Blocks:")
	turns.FprintfTurn(w, updatedTurn,
		turns.WithIDs(false),
		turns.WithRoles(true),
		turns.WithToolDetail(true),
		turns.WithIndent(0),
		turns.WithMaxTextLines(0),
	)

	cancel()
	_ = eg.Wait()
	return nil
}

func main() {
	// Initialize zerolog with pretty console output
	err := clay.InitGlazed("pinocchio", rootCmd)
	cobra.CheckErr(err)

	helpSystem := help.NewHelpSystem()
	help_cmd.SetupCobraRootCommand(helpSystem, rootCmd)
	// logging flags are already added globally by the framework; no need to re-add here

	testCmd, err := NewTestOpenAIToolsCommand()
	cobra.CheckErr(err)

	command, err := cli.BuildCobraCommand(testCmd,
		cli.WithCobraMiddlewaresFunc(geppettolayers.GetCobraCommandGeppettoMiddlewares),
	)
	cobra.CheckErr(err)
	rootCmd.AddCommand(command)

	cobra.CheckErr(rootCmd.Execute())
}

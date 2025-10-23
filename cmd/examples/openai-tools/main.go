package main

import (
	"context"
	"fmt"
	"io"

	"github.com/ThreeDotsLabs/watermill/message"
	clay "github.com/go-go-golems/clay/pkg"
	"github.com/go-go-golems/geppetto/pkg/events"
	"github.com/go-go-golems/geppetto/pkg/inference/engine"
	"github.com/go-go-golems/geppetto/pkg/inference/engine/factory"
	"github.com/go-go-golems/geppetto/pkg/inference/middleware"
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
		return logging.InitLoggerFromViper()
	},
}

type TestOpenAIToolsCommand struct {
	*cmds.CommandDescription
}

var _ cmds.WriterCommand = (*TestOpenAIToolsCommand)(nil)

type TestOpenAIToolsSettings struct {
	Debug        bool   `glazed.parameter:"debug"`
	OutputFormat string `glazed.parameter:"output-format"`
	WithMetadata bool   `glazed.parameter:"with-metadata"`
	FullOutput   bool   `glazed.parameter:"full-output"`
	Verbose      bool   `glazed.parameter:"verbose"`
	Mode         string `glazed.parameter:"mode"`
	Prompt       string `glazed.parameter:"prompt"`
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
			parameters.NewParameterDefinition("debug",
				parameters.ParameterTypeBool,
				parameters.WithHelp("Enable debug logging"),
				parameters.WithDefault(true),
			),
			parameters.NewParameterDefinition("output-format",
				parameters.ParameterTypeString,
				parameters.WithHelp("Printer format (text, json, yaml)"),
				parameters.WithDefault("text"),
			),
			parameters.NewParameterDefinition("with-metadata",
				parameters.ParameterTypeBool,
				parameters.WithHelp("Include metadata in printed events"),
				parameters.WithDefault(false),
			),
			parameters.NewParameterDefinition("full-output",
				parameters.ParameterTypeBool,
				parameters.WithHelp("Include full event details"),
				parameters.WithDefault(false),
			),
			parameters.NewParameterDefinition("verbose",
				parameters.ParameterTypeBool,
				parameters.WithHelp("Verbose router logging (SSE debug)"),
				parameters.WithDefault(false),
			),
			parameters.NewParameterDefinition("mode",
				parameters.ParameterTypeChoice,
				parameters.WithChoices("tools", "thinking", "parallel-tools"),
				parameters.WithHelp("Modes: tools (function calling), thinking (no tools), parallel-tools (multiple calls)"),
				parameters.WithDefault("tools"),
			),
			parameters.NewParameterDefinition("prompt",
				parameters.ParameterTypeString,
				parameters.WithHelp("Override the default prompt for the selected mode"),
				parameters.WithDefault(""),
			),
		),
		cmds.WithLayersList(
			geppettoLayers...,
		),
	)

	return &TestOpenAIToolsCommand{
		CommandDescription: description,
	}, nil
}

func (c *TestOpenAIToolsCommand) RunIntoWriter(ctx context.Context, parsedLayers *layers.ParsedLayers, w io.Writer) error {
	log.Debug().Msg("Debug logging enabled for tool testing")

	s := &TestOpenAIToolsSettings{}
	err := parsedLayers.InitializeStruct(layers.DefaultSlug, s)
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

	// Create engine using factory with an event sink to publish streaming events
	watermillSink := middleware.NewWatermillSink(router.Publisher, "chat")
	engineInstance, err := factory.NewEngineFromParsedLayers(parsedLayers, engine.WithSink(watermillSink))
	if err != nil {
		return errors.Wrap(err, "failed to create engine from parsed layers")
	}

	// Run the router concurrently
	eg := errgroup.Group{}
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	eg.Go(func() error { return router.Run(runCtx) })

	// Build mode-specific setup
	var turn *turns.Turn
	if s.Mode == "thinking" {
		// No tools; reasoning-focused prompt
		p := s.Prompt
		if p == "" {
			p = "Think step-by-step and answer concisely: What is 23*17 + 55?"
		}
		turn = &turns.Turn{Data: map[string]any{}}
		turns.AppendBlock(turn, turns.NewUserTextBlock(p))
	} else {
		// Tools mode (default)
		weatherToolDef, err := tools.NewToolFromFunc(
			"get_weather",
			"Get current weather information for a specific location",
			weatherTool,
		)
		if err != nil {
			return errors.Wrap(err, "failed to create weather tool")
		}
		// Debug print schema
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
		// Add calculator tool
		calcDef, err := tools.NewToolFromFunc(
			"calculator",
			"Evaluate a simple arithmetic expression (format: 'A op B')",
			calculatorTool,
		)
		if err != nil {
			return errors.Wrap(err, "failed to create calculator tool")
		}

		// registry + config
		reg := tools.NewInMemoryToolRegistry()
		_ = reg.RegisterTool("get_weather", *weatherToolDef)
		_ = reg.RegisterTool("calculator", *calcDef)
		maxPar := 1
		if s.Mode == "parallel-tools" {
			maxPar = 2
		}
		turn = &turns.Turn{Data: map[string]any{turns.DataKeyToolRegistry: reg, turns.DataKeyToolConfig: engine.ToolConfig{Enabled: true, ToolChoice: engine.ToolChoiceAuto, MaxIterations: 3, MaxParallelTools: maxPar, ToolErrorHandling: engine.ToolErrorContinue}}}
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

	// Prepare a toolbox and register executable implementation
	tb := middleware.NewMockToolbox()
	tb.RegisterTool("get_weather", "Get current weather information for a specific location", map[string]any{
		"location": map[string]any{"type": "string"},
		"units":    map[string]any{"type": "string"},
	}, func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
		// Map args to WeatherRequest
		req := WeatherRequest{Units: "celsius"}
		if v, ok := args["location"].(string); ok {
			req.Location = v
		}
		if v, ok := args["units"].(string); ok && v != "" {
			req.Units = v
		}
		resp := weatherTool(req)
		return resp, nil
	})

	// Register calculator tool implementation
	tb.RegisterTool("calculator", "Evaluate a simple arithmetic expression (format: 'A op B')", map[string]any{
		"expression": map[string]any{"type": "string"},
	}, func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
		// Map args to CalculatorRequest
		req := CalculatorRequest{}
		if v, ok := args["expression"].(string); ok {
			req.Expression = v
		}
		res, err := calculatorTool(req)
		if err != nil {
			return nil, err
		}
		return map[string]any{"result": res}, nil
	})

	// Wrap engine with tool middleware
	mw := middleware.NewToolMiddleware(tb, middleware.ToolConfig{MaxIterations: 3})
	wrapped := middleware.NewEngineWithMiddleware(engineInstance, mw)

	// Run inference with middleware-managed tool execution
	updatedTurn, err := wrapped.RunInference(runCtx, turn)
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
	err := clay.InitViper("pinocchio", rootCmd)
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

package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/go-go-golems/geppetto/pkg/conversation"
	"github.com/go-go-golems/geppetto/pkg/conversation/builder"
	"github.com/go-go-golems/geppetto/pkg/events"
	"github.com/go-go-golems/geppetto/pkg/inference/engine"
	"github.com/go-go-golems/geppetto/pkg/inference/engine/factory"
	"github.com/go-go-golems/geppetto/pkg/inference/middleware"
	"github.com/go-go-golems/geppetto/pkg/inference/tools"

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
	"golang.org/x/sync/errgroup"
	"gopkg.in/yaml.v3"
)

var rootCmd = &cobra.Command{
	Use:   "tool-calling",
	Short: "Tool calling example with weather tool",
}

type ToolCallingCommand struct {
	*cmds.CommandDescription
}

var _ cmds.WriterCommand = (*ToolCallingCommand)(nil)

type ToolCallingSettings struct {
	PinocchioProfile string `glazed.parameter:"pinocchio-profile"`
	Debug            bool   `glazed.parameter:"debug"`
	WithLogging      bool   `glazed.parameter:"with-logging"`
	Prompt           string `glazed.parameter:"prompt"`
	OutputFormat     string `glazed.parameter:"output-format"`
	WithMetadata     bool   `glazed.parameter:"with-metadata"`
	FullOutput       bool   `glazed.parameter:"full-output"`
	Verbose          bool   `glazed.parameter:"verbose"`
	
	// Tool configuration
	MaxIterations    int    `glazed.parameter:"max-iterations"`
	ToolChoice       string `glazed.parameter:"tool-choice"`
	MaxParallelTools int    `glazed.parameter:"max-parallel-tools"`
	ToolsEnabled     bool   `glazed.parameter:"tools-enabled"`
}

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

// CustomCalculatorRequest represents input for the calculator tool
type CustomCalculatorRequest struct {
	Expression string `json:"expression" jsonschema:"required,description=Mathematical expression to evaluate"`
}

// CustomCalculatorResponse represents the calculator tool's response
type CustomCalculatorResponse struct {
	Expression string  `json:"expression"`
	Result     float64 `json:"result"`
	Message    string  `json:"message"`
}

// customCalculator is a very basic calculator tool that's clearly custom
func customCalculator(req CustomCalculatorRequest) CustomCalculatorResponse {
	log.Info().Str("expression", req.Expression).Msg("CUSTOM CALCULATOR TOOL CALLED!")
	
	// Very simple calculator - just handle basic cases for demo
	var result float64
	var message string
	
	switch req.Expression {
	case "2+2", "2 + 2":
		result = 4
		message = "Simple addition performed by custom tool"
	case "10*5", "10 * 5":
		result = 50
		message = "Simple multiplication performed by custom tool"
	case "100/4", "100 / 4":
		result = 25
		message = "Simple division performed by custom tool"
	default:
		result = 42
		message = "Default answer from custom calculator tool (this proves it's our custom implementation!)"
	}
	
	response := CustomCalculatorResponse{
		Expression: req.Expression,
		Result:     result,
		Message:    message,
	}
	
	log.Info().Interface("response", response).Msg("CUSTOM CALCULATOR TOOL RETURNING RESPONSE")
	return response
}

// weatherTool is a mock weather tool that returns fake data
func weatherTool(req WeatherRequest) WeatherResponse {
	log.Info().Str("location", req.Location).Str("units", req.Units).Msg("Weather tool called!")
	
	// Mock weather data based on location
	var temp float64
	var conditions string
	var humidity int
	
	switch req.Location {
	case "San Francisco", "san francisco":
		temp = 18.0
		conditions = "Partly cloudy"
		humidity = 75
	case "New York", "new york":
		temp = 22.0
		conditions = "Sunny"
		humidity = 60
	case "London", "london":
		temp = 12.0
		conditions = "Rainy"
		humidity = 85
	case "Tokyo", "tokyo":
		temp = 25.0
		conditions = "Clear"
		humidity = 55
	default:
		temp = 20.0
		conditions = "Partly cloudy"
		humidity = 70
	}
	
	// Convert to Fahrenheit if requested
	if req.Units == "fahrenheit" {
		temp = temp*9/5 + 32
	}
	
	response := WeatherResponse{
		Location:    req.Location,
		Temperature: temp,
		Conditions:  conditions,
		Humidity:    humidity,
		Units:       req.Units,
	}
	
	log.Info().Interface("response", response).Msg("Weather tool returning response")
	return response
}

func NewToolCallingCommand() (*ToolCallingCommand, error) {
	geppettoLayers, err := geppettolayers.CreateGeppettoLayers()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create geppetto parameter layer")
	}
	
	description := cmds.NewCommandDescription(
		"tool-calling",
		cmds.WithShort("Tool calling example with weather tool"),
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
			// Tool configuration parameters
			parameters.NewParameterDefinition("max-iterations",
				parameters.ParameterTypeInteger,
				parameters.WithHelp("Maximum tool calling iterations"),
				parameters.WithDefault(5),
			),
			parameters.NewParameterDefinition("tool-choice",
				parameters.ParameterTypeString,
				parameters.WithHelp("Tool choice strategy (auto, none, required)"),
				parameters.WithDefault("auto"),
			),
			parameters.NewParameterDefinition("max-parallel-tools",
				parameters.ParameterTypeInteger,
				parameters.WithHelp("Maximum parallel tool executions"),
				parameters.WithDefault(3),
			),
			parameters.NewParameterDefinition("tools-enabled",
				parameters.ParameterTypeBool,
				parameters.WithHelp("Enable tool calling"),
				parameters.WithDefault(true),
			),
		),
		cmds.WithLayersList(
			geppettoLayers...,
		),
	)

	return &ToolCallingCommand{
		CommandDescription: description,
	}, nil
}

func (c *ToolCallingCommand) RunIntoWriter(ctx context.Context, parsedLayers *layers.ParsedLayers, w io.Writer) error {
	log.Info().Msg("Starting tool calling command")

	s := &ToolCallingSettings{}
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

	// 4. Create base engine with sink
	engineOptions := []engine.Option{
		engine.WithSink(watermillSink),
	}

	baseEngine, err := factory.NewEngineFromParsedLayers(parsedLayers, engineOptions...)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create engine")
		return errors.Wrap(err, "failed to create engine")
	}

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
		baseEngine = middleware.NewEngineWithMiddleware(baseEngine, loggingMiddleware)
	}

	// 5. Create tool registry and register weather tool
	registry := tools.NewInMemoryToolRegistry()
	
	weatherToolDef, err := tools.NewToolFromFunc(
		"get_weather",
		"Get current weather information for a specific location",
		weatherTool,
	)
	if err != nil {
		return errors.Wrap(err, "failed to create weather tool")
	}
	
	err = registry.RegisterTool("get_weather", *weatherToolDef)
	if err != nil {
		return errors.Wrap(err, "failed to register weather tool")
	}

	// Register calculator tool with unique name
	calculatorToolDef, err := tools.NewToolFromFunc(
		"geppetto_custom_math_42",
		"A totally unique calculator that proves we're using custom tools - returns 42 by default",
		customCalculator,
	)
	if err != nil {
		return errors.Wrap(err, "failed to create calculator tool")
	}
	
	err = registry.RegisterTool("geppetto_custom_math_42", *calculatorToolDef)
	if err != nil {
		return errors.Wrap(err, "failed to register calculator tool")
	}
	
	log.Info().Int("registered_tools", registry.Count()).Msg("Tool registry initialized")
	for _, tool := range registry.ListTools() {
		log.Info().Str("tool_name", tool.Name).Str("description", tool.Description).Msg("Registered tool")
	}

	// 6. Create tool configuration
	toolChoice := tools.ToolChoiceAuto
	switch s.ToolChoice {
	case "none":
		toolChoice = tools.ToolChoiceNone
	case "required":
		toolChoice = tools.ToolChoiceRequired
	case "auto":
		toolChoice = tools.ToolChoiceAuto
	}

	toolConfig := tools.ToolConfig{
		Enabled:           s.ToolsEnabled,
		ToolChoice:        toolChoice,
		MaxIterations:     s.MaxIterations,
		ExecutionTimeout:  30 * time.Second,
		MaxParallelTools:  s.MaxParallelTools,
		AllowedTools:      nil, // Allow all tools
		ToolErrorHandling: tools.ToolErrorContinue,
		RetryConfig: tools.RetryConfig{
			MaxRetries:    2,
			BackoffBase:   time.Second,
			BackoffFactor: 2.0,
		},
	}

	// 7. Create engine wrapper that forces orchestrator tool usage
	engineWrapper := tools.NewEngineWrapper(baseEngine, registry, toolConfig)
	
	log.Info().Msg("Created engine wrapper with orchestrator tool support")

	// 8. Build conversation
	b := builder.NewManagerBuilder().
		WithSystemPrompt("You are a helpful assistant with access to tools. Use get_weather for weather queries and custom_calculator for math problems.").
		WithPrompt(s.Prompt)

	manager, err := b.Build()
	if err != nil {
		log.Error().Err(err).Msg("Failed to build conversation manager")
		return err
	}

	conversation_ := manager.GetConversation()

	// 9. Start router and run inference in parallel
	eg := errgroup.Group{}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	eg.Go(func() error {
		defer cancel()
		return router.Run(ctx)
	})

	eg.Go(func() error {
		defer cancel()
		<-router.Running()

		// Run inference with tool support
		updatedConversation, err := engineWrapper.RunInference(ctx, conversation_)
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

		return nil
	})

	err = eg.Wait()
	if err != nil {
		return err
	}

	messages := manager.GetConversation()

	fmt.Fprintln(w, "\n=== Final Conversation ===")
	for _, msg := range messages {
		switch content := msg.Content.(type) {
		case *conversation.ChatMessageContent:
			fmt.Fprintf(w, "%s: %s\n", content.Role, content.Text)
		case *conversation.ToolUseContent:
			fmt.Fprintf(w, "Tool Call (%s): %s\n", content.Name, string(content.Input))
		case *conversation.ToolResultContent:
			fmt.Fprintf(w, "Tool Result (%s): %s\n", content.ToolID, content.Result)
		default:
			fmt.Fprintf(w, "%s: %s\n", msg.Content.ContentType(), msg.Content.String())
		}
	}

	log.Info().Int("total_messages", len(messages)).Msg("Tool calling command completed successfully")
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

	toolCallingCmd, err := NewToolCallingCommand()
	cobra.CheckErr(err)

	command, err := cli.BuildCobraCommand(toolCallingCmd,
		cli.WithCobraMiddlewaresFunc(geppettolayers.GetCobraCommandGeppettoMiddlewares),
	)
	cobra.CheckErr(err)
	rootCmd.AddCommand(command)

	cobra.CheckErr(rootCmd.Execute())
}

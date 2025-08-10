package main

import (
	"context"
	"fmt"
	"io"

	clay "github.com/go-go-golems/clay/pkg"
	"github.com/go-go-golems/geppetto/pkg/inference/engine"
	"github.com/go-go-golems/geppetto/pkg/inference/engine/factory"
	"github.com/go-go-golems/geppetto/pkg/inference/middleware"
	"github.com/go-go-golems/geppetto/pkg/inference/tools"
	geppettolayers "github.com/go-go-golems/geppetto/pkg/layers"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/claude"
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

var rootCmd = &cobra.Command{
	Use:   "test-claude-tools",
	Short: "Test Claude tools integration with debug logging",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		err := logging.InitLoggerFromViper()
		if err != nil {
			return err
		}
		return nil
	},
}

type TestClaudeToolsCommand struct {
	*cmds.CommandDescription
}

var _ cmds.WriterCommand = (*TestClaudeToolsCommand)(nil)

type TestClaudeToolsSettings struct {
	Debug bool `glazed.parameter:"debug"`
}

func NewTestClaudeToolsCommand() (*TestClaudeToolsCommand, error) {
	geppettoLayers, err := geppettolayers.CreateGeppettoLayers()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create geppetto parameter layer")
	}

	description := cmds.NewCommandDescription(
		"test-claude-tools",
		cmds.WithShort("Test Claude tools integration with debug logging"),
		cmds.WithFlags(
			parameters.NewParameterDefinition("debug",
				parameters.ParameterTypeBool,
				parameters.WithHelp("Enable debug logging"),
				parameters.WithDefault(true),
			),
		),
		cmds.WithLayersList(
			geppettoLayers...,
		),
	)

	return &TestClaudeToolsCommand{
		CommandDescription: description,
	}, nil
}

func (c *TestClaudeToolsCommand) RunIntoWriter(ctx context.Context, parsedLayers *layers.ParsedLayers, w io.Writer) error {
	log.Debug().Msg("Debug logging enabled for tool testing")

	s := &TestClaudeToolsSettings{}
	err := parsedLayers.InitializeStruct(layers.DefaultSlug, s)
	if err != nil {
		return errors.Wrap(err, "failed to initialize settings")
	}

	// Create engine using factory with ParsedLayers
	engineInstance, err := factory.NewEngineFromParsedLayers(parsedLayers)
	if err != nil {
		return errors.Wrap(err, "failed to create engine from parsed layers")
	}

	// Create tool definition using NewToolFromFunc which handles schema generation
	weatherToolDef, err := tools.NewToolFromFunc(
		"get_weather",
		"Get current weather information for a specific location",
		weatherTool,
	)
	if err != nil {
		return errors.Wrap(err, "failed to create weather tool")
	}

	// Debug: Print the schema to see what's being generated
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

	// Convert to engine tool definition
	engineTool := engine.ToolDefinition{
		Name:        weatherToolDef.Name,
		Description: weatherToolDef.Description,
		Parameters:  weatherToolDef.Parameters,
		Examples:    []engine.ToolExample{},
		Tags:        []string{"weather"},
		Version:     "1.0",
	}

	toolConfig := engine.ToolConfig{
		Enabled:           true,
		ToolChoice:        engine.ToolChoiceAuto,
		MaxIterations:     3,
		MaxParallelTools:  1, // Claude doesn't support parallel tools
		AllowedTools:      []string{"get_weather"},
		ToolErrorHandling: engine.ToolErrorContinue,
	}

	// Check if engine is Claude engine and configure tools
	if claudeEngine, ok := engineInstance.(*claude.ClaudeEngine); ok {
		claudeEngine.ConfigureTools([]engine.ToolDefinition{engineTool}, toolConfig)
		fmt.Fprintln(w, "Claude engine found - configured weather tool")
	} else {
		fmt.Fprintln(w, "Warning: Engine is not Claude engine, cannot configure tools directly")
		fmt.Fprintf(w, "Engine type: %T\n", engineInstance)
	}

	// Build a Turn seeded with a user prompt that asks to use the tool
    turn := &turns.Turn{}
    turns.AppendBlock(turn, turns.NewUserTextBlock("Use get_weather to check the weather in Paris, France. Return the result."))

	// Prepare a toolbox and register executable implementation
	tb := middleware.NewMockToolbox()
	tb.RegisterTool("get_weather", "Get current weather information for a specific location", map[string]any{
		"location": map[string]any{"type": "string"},
		"units":    map[string]any{"type": "string"},
	}, func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
		req := WeatherRequest{Units: "celsius"}
		if v, ok := args["location"].(string); ok {
			req.Location = v
		}
		if v, ok := args["units"].(string); ok && v != "" {
			req.Units = v
		}
		return weatherTool(req), nil
	})

	// Wrap engine with tool middleware
	mw := middleware.NewToolMiddleware(tb, middleware.ToolConfig{MaxIterations: 3})
	wrapped := middleware.NewEngineWithMiddleware(engineInstance, mw)

	// Run inference with middleware-managed tool execution
	updatedTurn, err := wrapped.RunInference(ctx, turn)
	if err != nil {
		return errors.Wrap(err, "inference with tools failed")
	}

	// Render final conversation from Turn
	msgs := turns.BuildConversationFromTurn(updatedTurn)
	fmt.Fprintf(w, "\nWorkflow completed. Result has %d messages\n", len(msgs))
	for i, msg := range msgs {
		fmt.Fprintf(w, "Message %d: %s\n", i, msg.Content.String())
	}

	return nil
}

func main() {
	// Initialize zerolog with pretty console output
	err := clay.InitViper("pinocchio", rootCmd)
	cobra.CheckErr(err)

	helpSystem := help.NewHelpSystem()
	help_cmd.SetupCobraRootCommand(helpSystem, rootCmd)
	cobra.CheckErr(err)

	testCmd, err := NewTestClaudeToolsCommand()
	cobra.CheckErr(err)

	command, err := cli.BuildCobraCommand(testCmd,
		cli.WithCobraMiddlewaresFunc(geppettolayers.GetCobraCommandGeppettoMiddlewares),
	)
	cobra.CheckErr(err)
	rootCmd.AddCommand(command)

	cobra.CheckErr(rootCmd.Execute())
}

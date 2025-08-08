package main

import (
	"context"
	"fmt"
	"io"

	"github.com/go-go-golems/geppetto/pkg/conversation"
	"github.com/go-go-golems/geppetto/pkg/inference/engine"
	"github.com/go-go-golems/geppetto/pkg/inference/engine/factory"
    "github.com/go-go-golems/geppetto/pkg/inference/toolhelpers"
	"github.com/go-go-golems/geppetto/pkg/inference/tools"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/openai"

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
	Use:   "test-openai-tools",
	Short: "Test OpenAI tools integration with debug logging",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		err := logging.InitLoggerFromViper()
		if err != nil {
			return err
		}
		return nil
	},
}

type TestOpenAIToolsCommand struct {
	*cmds.CommandDescription
}

var _ cmds.WriterCommand = (*TestOpenAIToolsCommand)(nil)

type TestOpenAIToolsSettings struct {
	Debug bool `glazed.parameter:"debug"`
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
		MaxParallelTools:  1,
		AllowedTools:      []string{"get_weather"},
		ToolErrorHandling: engine.ToolErrorContinue,
	}

	// Check if engine is OpenAI engine and configure tools
	if openaiEngine, ok := engineInstance.(*openai.OpenAIEngine); ok {
		openaiEngine.ConfigureTools([]engine.ToolDefinition{engineTool}, toolConfig)
		fmt.Fprintln(w, "OpenAI engine found - configured weather tool")
	} else {
		fmt.Fprintln(w, "Warning: Engine is not OpenAI engine, cannot configure tools directly")
		fmt.Fprintf(w, "Engine type: %T\n", engineInstance)
	}

    // Create a simple conversation
    conversation := conversation.Conversation{
		conversation.NewMessage(
			conversation.NewChatMessageContent(
				conversation.RoleUser,
				"What's the weather like in San Francisco?",
				nil,
			),
		),
	}

    // Prepare registry and register our tool for execution
    registry := tools.NewInMemoryToolRegistry()
    if err := registry.RegisterTool("get_weather", *weatherToolDef); err != nil {
        return errors.Wrap(err, "failed to register weather tool")
    }

    fmt.Fprintln(w, "=== Testing OpenAI Engine With Tool Calling Helper ===")
    fmt.Fprintf(w, "Conversation has %d messages\n", len(conversation))
    fmt.Fprintln(w, "Running full tool-calling loop (max 2 iterations)...")
    fmt.Fprintln(w)

    // Configure helper to allow the model to call the tool and then respond
    helperConfig := toolhelpers.NewToolConfig().
        WithMaxIterations(2)

    // Run the automated tool calling loop
    result, err := toolhelpers.RunToolCallingLoop(ctx, engineInstance, conversation, registry, helperConfig)
    if err != nil {
        log.Error().Err(err).Msg("Tool calling workflow failed")
        return errors.Wrap(err, "tool calling workflow failed")
    }

    fmt.Fprintf(w, "\nWorkflow completed. Result has %d messages\n", len(result))
    for i, msg := range result {
        fmt.Fprintf(w, "Message %d: type=%s text=%q\n", i, msg.Content.ContentType(), msg.Content.String())
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

	testCmd, err := NewTestOpenAIToolsCommand()
	cobra.CheckErr(err)

	command, err := cli.BuildCobraCommand(testCmd,
		cli.WithCobraMiddlewaresFunc(geppettolayers.GetCobraCommandGeppettoMiddlewares),
	)
	cobra.CheckErr(err)
	rootCmd.AddCommand(command)

	cobra.CheckErr(rootCmd.Execute())
}

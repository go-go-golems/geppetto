package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-go-golems/geppetto/pkg/events"
	"github.com/go-go-golems/geppetto/pkg/inference/engine"
	"github.com/go-go-golems/geppetto/pkg/inference/engine/factory"
	"github.com/go-go-golems/geppetto/pkg/inference/middleware"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/openai"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/types"
	go_openai "github.com/sashabaranov/go-openai"
	"io"
	"os"
	"strings"

	"github.com/go-go-golems/geppetto/pkg/conversation"
	"github.com/go-go-golems/geppetto/pkg/conversation/builder"

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
	Short: "Tool calling example with Engine-first architecture",
}

type ToolCallingInferenceCommand struct {
	*cmds.CommandDescription
}

var _ cmds.WriterCommand = (*ToolCallingInferenceCommand)(nil)

type ToolCallingInferenceSettings struct {
	PinocchioProfile string `glazed.parameter:"pinocchio-profile"`
	Debug            bool   `glazed.parameter:"debug"`
	WithLogging      bool   `glazed.parameter:"with-logging"`
	Prompt           string `glazed.parameter:"prompt"`
	OutputFormat     string `glazed.parameter:"output-format"`
	WithMetadata     bool   `glazed.parameter:"with-metadata"`
	FullOutput       bool   `glazed.parameter:"full-output"`
	Verbose          bool   `glazed.parameter:"verbose"`
	MaxIterations    int    `glazed.parameter:"max-iterations"`
}

// WeatherTool provides a simple mock weather service
type WeatherTool struct{}

// WeatherResult represents the result of a weather query
type WeatherResult struct {
	City        string `json:"city"`
	Temperature int    `json:"temperature"`
	Condition   string `json:"condition"`
	Description string `json:"description"`
}

// GetWeather returns mock weather data for a given city
func (w *WeatherTool) GetWeather(location string) (*WeatherResult, error) {
	log.Debug().Str("function", "GetWeather").Str("location", location).Msg("=== WEATHER TOOL CALLED ===")
	
	if location == "" {
		log.Error().Msg("Weather tool called with empty location")
		return nil, fmt.Errorf("location is required")
	}

	// Simple mock weather data
	location = strings.TrimSpace(location)
	result := &WeatherResult{
		City:        location,
		Temperature: 75,
		Condition:   "sunny",
		Description: fmt.Sprintf("The weather in %s is sunny, 75°F", location),
	}

	log.Info().
		Str("location", location).
		Str("result", result.Description).
		Msg("=== WEATHER TOOL RESPONSE GENERATED ===")
	return result, nil
}

// OpenAIWithToolsEngine is a custom engine that extends OpenAI with tool calling capabilities
type OpenAIWithToolsEngine struct {
	baseEngine   engine.Engine
	tools        []go_openai.Tool
	toolFunctions map[string]interface{}
}

// NewOpenAIWithToolsEngine creates a new OpenAI engine with tool support
func NewOpenAIWithToolsEngine(baseEngine engine.Engine, toolFunctions map[string]interface{}) (*OpenAIWithToolsEngine, error) {
	tools := []go_openai.Tool{}
	
	// Create OpenAI tool definitions from tool functions
	for name, _ := range toolFunctions {
		// Use helpers to get JSON schema for the function
		jsonSchema, err := getBasicFunctionSchema(name, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create schema for tool %s: %w", name, err)
		}
		
		schemaBytes, _ := json.Marshal(jsonSchema)
		tools = append(tools, go_openai.Tool{
			Type: "function",
			Function: &go_openai.FunctionDefinition{
				Name:        name,
				Description: jsonSchema["description"].(string),
				Parameters:  json.RawMessage(schemaBytes),
			},
		})
	}
	
	return &OpenAIWithToolsEngine{
		baseEngine:    baseEngine,
		tools:         tools,
		toolFunctions: toolFunctions,
	}, nil
}

// getBasicFunctionSchema creates a basic schema for our weather tool
func getBasicFunctionSchema(name string, toolFunc interface{}) (map[string]interface{}, error) {
	if name == "get_weather" {
		return map[string]interface{}{
			"type":        "object",
			"description": "Get the current weather for a specified location",
			"properties": map[string]interface{}{
				"location": map[string]interface{}{
					"type":        "string",
					"description": "The city or location to get weather for",
				},
			},
			"required": []string{"location"},
		}, nil
	}
	return nil, fmt.Errorf("unknown tool: %s", name)
}

// RunInference implements the Engine interface with tool calling support
func (e *OpenAIWithToolsEngine) RunInference(ctx context.Context, messages conversation.Conversation) (conversation.Conversation, error) {
	log.Debug().Int("num_messages", len(messages)).Int("num_tools", len(e.tools)).Msg("=== TOOL ENGINE RunInference started ===")
	
	// For now, let's directly create an OpenAI request with tools
	// We need to extract the settings from somewhere or create a basic configuration
	baseSettings := &settings.StepSettings{
		API: &settings.APISettings{},
		Chat: &settings.ChatSettings{
			Stream: true, // Enable streaming
		},
	}
	
	// Try to use OpenAI
	if baseSettings.Chat.ApiType == nil {
		apiType := types.ApiTypeOpenAI
		baseSettings.Chat.ApiType = &apiType
	}
	if baseSettings.Chat.Engine == nil {
		engineName := "gpt-4o-mini"
		baseSettings.Chat.Engine = &engineName
	}
	
	client, err := openai.MakeClient(baseSettings.API, *baseSettings.Chat.ApiType)
	if err != nil {
		return nil, fmt.Errorf("failed to create OpenAI client: %w", err)
	}
	
	req, err := openai.MakeCompletionRequest(baseSettings, messages)
	if err != nil {
		return nil, fmt.Errorf("failed to create completion request: %w", err)
	}
	
	// Add tools to the request
	req.Tools = e.tools
	log.Debug().Int("tools_count", len(req.Tools)).Msg("=== ADDED TOOLS TO REQUEST ===")
	
	// Make the API call
	resp, err := client.CreateChatCompletion(ctx, *req)
	if err != nil {
		return nil, fmt.Errorf("OpenAI API call failed: %w", err)
	}
	
	result := make(conversation.Conversation, len(messages))
	copy(result, messages)
	
	// Process the response
	choice := resp.Choices[0]
	assistantMessage := choice.Message
	
	// Create assistant message
	messageContent := conversation.NewChatMessageContent(conversation.RoleAssistant, assistantMessage.Content, nil)
	assistantMsg := conversation.NewMessage(messageContent)
	result = append(result, assistantMsg)
	
	log.Debug().Str("content", assistantMessage.Content).Int("tool_calls", len(assistantMessage.ToolCalls)).Msg("=== RECEIVED RESPONSE FROM OPENAI ===")
	
	// Handle tool calls if present
	if len(assistantMessage.ToolCalls) > 0 {
		log.Info().Int("tool_calls_count", len(assistantMessage.ToolCalls)).Msg("=== PROCESSING TOOL CALLS ===")
		
		for _, toolCall := range assistantMessage.ToolCalls {
			if toolCall.Type != "function" {
				log.Warn().Str("type", string(toolCall.Type)).Msg("=== SKIPPING NON-FUNCTION TOOL CALL ===")
				continue
			}
			
			log.Debug().Str("tool_name", toolCall.Function.Name).Str("tool_id", toolCall.ID).Msg("=== PROCESSING TOOL CALL ===")
			
			// Create tool use message
			toolInput := json.RawMessage(toolCall.Function.Arguments)
			toolUseContent := &conversation.ToolUseContent{
				ToolID: toolCall.ID,
				Name:   toolCall.Function.Name,
				Input:  toolInput,
				Type:   "function",
			}
			toolUseMsg := conversation.NewMessage(toolUseContent)
			result = append(result, toolUseMsg)
			
			// Execute the tool
			toolResult, err := e.executeToolCall(toolCall)
			if err != nil {
				log.Error().Err(err).Str("tool_name", toolCall.Function.Name).Msg("=== TOOL EXECUTION FAILED ===")
				toolResult = fmt.Sprintf("Error: %s", err.Error())
			}
			
			// Create tool result message
			toolResultContent := &conversation.ToolResultContent{
				ToolID: toolCall.ID,
				Result: toolResult,
			}
			toolResultMsg := conversation.NewMessage(toolResultContent)
			result = append(result, toolResultMsg)
			
			log.Info().Str("tool_result", toolResult).Msg("=== TOOL RESULT ADDED ===")
		}
	}
	
	log.Debug().Int("original_length", len(messages)).Int("final_length", len(result)).Msg("=== TOOL ENGINE RunInference completed ===")
	return result, nil
}

// executeToolCall executes a specific tool call
func (e *OpenAIWithToolsEngine) executeToolCall(toolCall go_openai.ToolCall) (string, error) {
	toolFunc, exists := e.toolFunctions[toolCall.Function.Name]
	if !exists {
		return "", fmt.Errorf("tool function not found: %s", toolCall.Function.Name)
	}
	
	// Parse arguments
	var args map[string]interface{}
	if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
		return "", fmt.Errorf("failed to parse tool arguments: %w", err)
	}
	
	// Call the specific tool
	if toolCall.Function.Name == "get_weather" {
		if weatherFunc, ok := toolFunc.(*WeatherTool); ok {
			location, _ := args["location"].(string)
			result, err := weatherFunc.GetWeather(location)
			if err != nil {
				return "", err
			}
			
			resultJSON, err := json.Marshal(result)
			if err != nil {
				return "", fmt.Errorf("failed to marshal result: %w", err)
			}
			
			return string(resultJSON), nil
		}
	}
	
	return "", fmt.Errorf("unsupported tool: %s", toolCall.Function.Name)
}

func NewToolCallingInferenceCommand() (*ToolCallingInferenceCommand, error) {
	geppettoLayers, err := geppettolayers.CreateGeppettoLayers()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create geppetto parameter layer")
	}
	description := cmds.NewCommandDescription(
		"tool-calling",
		cmds.WithShort("Tool calling inference with Engine-first architecture"),
		cmds.WithArguments(
			parameters.NewParameterDefinition(
				"prompt",
				parameters.ParameterTypeString,
				parameters.WithHelp("Prompt to run"),
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
			parameters.NewParameterDefinition("max-iterations",
				parameters.ParameterTypeInteger,
				parameters.WithHelp("Maximum tool calling iterations"),
				parameters.WithDefault(5),
			),
		),
		cmds.WithLayersList(
			geppettoLayers...,
		),
	)

	return &ToolCallingInferenceCommand{
		CommandDescription: description,
	}, nil
}

func (c *ToolCallingInferenceCommand) RunIntoWriter(ctx context.Context, parsedLayers *layers.ParsedLayers, w io.Writer) error {
	log.Info().Msg("Starting tool calling inference command")

	s := &ToolCallingInferenceSettings{}
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

	// 4. Create weather tool and tool functions map
	log.Debug().Msg("=== CREATING WEATHER TOOL ===")
	weatherTool := &WeatherTool{}
	toolFunctions := map[string]interface{}{
		"get_weather": weatherTool,
	}
	log.Info().Msg("=== WEATHER TOOL CREATED ===")

	// 5. Create base engine first
	engineOptions := []engine.Option{
		engine.WithSink(watermillSink),
	}

	baseEngine, err := factory.NewEngineFromParsedLayers(parsedLayers, engineOptions...)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create base engine")
		return errors.Wrap(err, "failed to create base engine")
	}

	// 6. Create custom tool-enabled engine
	log.Debug().Msg("=== CREATING TOOL-ENABLED ENGINE ===")
	toolEngine, err := NewOpenAIWithToolsEngine(baseEngine, toolFunctions)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create tool engine")
		return errors.Wrap(err, "failed to create tool engine")
	}
	log.Info().Msg("=== TOOL-ENABLED ENGINE CREATED ===")

	// 7. Apply logging middleware if requested
	finalEngine := engine.Engine(toolEngine)
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
		finalEngine = middleware.NewEngineWithMiddleware(finalEngine, loggingMiddleware)
	}

	// 8. Create conversation builder with system prompt that encourages tool use
	b := builder.NewManagerBuilder().
		WithSystemPrompt("You are a helpful assistant. When asked about weather, you should use the get_weather tool to provide accurate information. Always call the weather tool when users ask about weather conditions in any location.").
		WithPrompt(s.Prompt)

	manager, err := b.Build()
	if err != nil {
		log.Error().Err(err).Msg("Failed to build conversation manager")
		return err
	}

	fmt.Fprintln(w, "=== Tool Calling Example ===")
	fmt.Fprintln(w, "Available tools: get_weather")
	fmt.Fprintln(w, "Try asking about weather in different cities!")
	fmt.Fprintln(w, "============================")

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

		log.Debug().Msg("=== STARTING TOOL CALLING INFERENCE ===")
		
		// Get current conversation
		currentConversation := manager.GetConversation()
		
		// Run inference with tool calling (the tool engine handles everything)
		updatedConversation, err := finalEngine.RunInference(ctx, currentConversation)
		if err != nil {
			log.Error().Err(err).Msg("Tool calling inference failed")
			return fmt.Errorf("tool calling inference failed: %w", err)
		}

		log.Debug().Int("original_length", len(currentConversation)).Int("updated_length", len(updatedConversation)).Msg("=== TOOL CALLING INFERENCE COMPLETED ===")
		
		// Extract new messages from the updated conversation
		newMessages := updatedConversation[len(currentConversation):]
		log.Debug().Int("new_messages_count", len(newMessages)).Msg("=== EXTRACTING NEW MESSAGES ===")
		
		// Add new messages to manager
		for i, msg := range newMessages {
			log.Debug().Int("message_index", i).Str("content_type", string(msg.Content.ContentType())).Msg("=== PROCESSING NEW MESSAGE ===")
			if err := manager.AppendMessages(msg); err != nil {
				log.Error().Err(err).Msg("Failed to append message to conversation")
				return fmt.Errorf("failed to append message: %w", err)
			}
		}

		log.Info().Msg("=== TOOL CALLING INFERENCE COMPLETED SUCCESSFULLY ===")
		return nil
	})

	err = eg.Wait()
	if err != nil {
		return err
	}

	messages := manager.GetConversation()

	fmt.Fprintln(w, "\n=== Final Conversation ===")
	for _, msg := range messages {
		if chatMsg, ok := msg.Content.(*conversation.ChatMessageContent); ok {
			fmt.Fprintf(w, "%s: %s\n", chatMsg.Role, chatMsg.Text)
		} else {
			fmt.Fprintf(w, "%s: %s\n", msg.Content.ContentType(), msg.Content.String())
		}
	}

	log.Info().Int("total_messages", len(messages)).Msg("Tool calling inference command completed successfully")
	return nil
}

func main() {
	// Initialize zerolog with pretty console output and debug level
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	err := clay.InitViper("pinocchio", rootCmd)
	cobra.CheckErr(err)

	helpSystem := help.NewHelpSystem()
	help_cmd.SetupCobraRootCommand(helpSystem, rootCmd)
	cobra.CheckErr(err)

	toolCmd, err := NewToolCallingInferenceCommand()
	cobra.CheckErr(err)

	command, err := cli.BuildCobraCommand(toolCmd,
		cli.WithCobraMiddlewaresFunc(geppettolayers.GetCobraCommandGeppettoMiddlewares),
	)
	cobra.CheckErr(err)
	rootCmd.AddCommand(command)

	cobra.CheckErr(rootCmd.Execute())
}

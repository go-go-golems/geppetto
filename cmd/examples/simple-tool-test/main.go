package main

import (
	"context"
	"fmt"
	"os"

	"github.com/go-go-golems/geppetto/pkg/conversation/builder"
	"github.com/go-go-golems/geppetto/pkg/inference/engine"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/openai"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/types"
	"github.com/invopop/jsonschema"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	// Set up debug logging to see the request payloads
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr}).Level(zerolog.DebugLevel)

	ctx := context.Background()

	// Create basic settings for OpenAI
	stepSettings := &settings.StepSettings{
		API: &settings.APISettings{
			APIKeys: map[string]string{
				"openai-api-key": os.Getenv("OPENAI_API_KEY"),
			},
			BaseUrls: map[string]string{
				"openai-base-url": "https://api.openai.com/v1",
			},
		},
		Chat: &settings.ChatSettings{
			Engine:             &[]string{"gpt-4o-mini"}[0],
			Temperature:        &[]float64{0.7}[0],
			MaxResponseTokens:  &[]int{1000}[0],
			Stream:             false,
			ApiType:            &[]types.ApiType{types.ApiTypeOpenAI}[0],
		},
	}

	// Create engine
	openaiEngine, err := openai.NewOpenAIEngine(stepSettings)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create OpenAI engine")
	}

	// Create simple tool definitions
	weatherToolParams := &jsonschema.Schema{
		Type: "object",
		Properties: func() *orderedmap.OrderedMap[string, *jsonschema.Schema] {
			props := orderedmap.New[string, *jsonschema.Schema]()
			props.Set("location", &jsonschema.Schema{
				Type:        "string",
				Description: "Location to get weather for",
			})
			props.Set("units", &jsonschema.Schema{
				Type:        "string",
				Description: "Temperature units (celsius or fahrenheit)",
				Default:     "celsius",
			})
			return props
		}(),
		Required: []string{"location"},
	}

	// Manually create tool definitions using the correct types
	engineTool := engine.ToolDefinition{
		Name:        "get_weather",
		Description: "Get current weather for a location",
		Parameters:  weatherToolParams,
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

	// Configure tools on the engine
	openaiEngine.ConfigureTools([]engine.ToolDefinition{engineTool}, toolConfig)

	// Create a conversation with a weather question
	b := builder.NewManagerBuilder().
		WithSystemPrompt("You are a helpful assistant that can provide weather information.").
		WithPrompt("What's the weather like in San Francisco?")

	manager, err := b.Build()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to build conversation manager")
	}

	conversation := manager.GetConversation()

	fmt.Println("=== Testing Tool Calling with Request Logging ===")
	fmt.Printf("Conversation has %d messages\n", len(conversation))
	fmt.Println("Running inference with tools configured...")
	fmt.Println("Check the debug logs to see the OpenAI request payload with tools!")
	fmt.Println()

	// Run inference - this should log the request with tools
	result, err := openaiEngine.RunInference(ctx, conversation)
	if err != nil {
		log.Error().Err(err).Msg("Inference failed")
		return
	}

	fmt.Printf("\nInference completed. Result has %d messages\n", len(result))
	
	// Print the conversation
	for i, msg := range result {
		if chatContent, ok := msg.Content.(*conversation.ChatMessageContent); ok {
			fmt.Printf("Message %d [%s]: %s\n", i, chatContent.Role, chatContent.Text)
		} else {
			fmt.Printf("Message %d [%s]: %s\n", i, msg.Content.ContentType(), msg.Content.String())
		}
	}
}

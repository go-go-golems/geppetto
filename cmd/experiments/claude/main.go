package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-go-golems/bobatea/pkg/conversation"
	"github.com/go-go-golems/clay/pkg"
	"github.com/go-go-golems/geppetto/pkg/cmds"
	"github.com/go-go-golems/geppetto/pkg/steps"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/chat"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/claude"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/claude/api"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
	"strings"
)

func testSimple() {
	events, _, done := streamTestRequest()
	if done {
		return
	}

	// Process the streaming events
	fmt.Println("Streaming response:")
	var response string
	for event := range events {
		fmt.Println("Type: ", event.Type)
		switch event.Type {
		case api.PingType:
			fmt.Println("Ping")
		case api.MessageStartType:
			if event.Message == nil {
				fmt.Println("Error: message is nil")
				continue
			}
			s, err := json.MarshalIndent(event.Message, "", "  ")
			if err != nil {
				fmt.Printf("Error: failed to marshal message: %v\n", err)
				continue
			}
			fmt.Println(string(s))
		case api.ContentBlockStartType:
			if event.ContentBlock == nil {
				fmt.Println("Error: content block is nil")
				continue
			}
			s, err := json.MarshalIndent(event.ContentBlock, "", "  ")
			if err != nil {
				fmt.Printf("Error: failed to marshal content block: %v\n", err)
				continue
			}
			fmt.Println(string(s))

		case api.ContentBlockDeltaType:
			if event.Delta == nil {
				fmt.Println("Error: delta is nil")
				continue
			}
			delta := event.Delta
			deltaType := event.Delta.Type
			switch deltaType {
			case api.TextDeltaType:
				fmt.Println(delta.Text)
				response += delta.Text
			case api.InputJSONDeltaType:
				fmt.Println(delta.PartialJSON)
				response += delta.PartialJSON
			default:
				fmt.Printf("Unknown delta type: %s\n", deltaType)
			}
		case api.ContentBlockStopType:
			fmt.Println()
		case api.MessageDeltaType:
			messageDelta := make(map[string]interface{})
			messageDelta["type"] = "message_delta"
			messageDelta["delta"] = event.Delta
			if event.Message != nil {
				messageDelta["usage"] = event.Message.Usage
			}
			jsonBytes, err := json.MarshalIndent(messageDelta, "", "  ")
			if err != nil {
				fmt.Printf("Error: failed to marshal message_delta: %v\n", err)
				continue
			}
			fmt.Printf("\n%s\n", string(jsonBytes))
		case api.MessageStopType:
			fmt.Println()
		case api.ErrorType:
			fmt.Println("Error: ", event.Error)
		}
		fmt.Println("---")
	}
	fmt.Println()

	// Print the full response
	fmt.Println("Full response:")
	fmt.Println(strings.TrimSpace(response))
}

func streamTestRequest() (<-chan api.StreamingEvent, *settings.StepSettings, bool) {
	err := pkg.InitViperWithAppName("pinocchio", "")
	if err != nil {
		fmt.Printf("Error initializing viper: %v\n", err)
		return nil, nil, true
	}

	// Set up the Claude API client
	settings_, err := cmds.LoadConfig()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		return nil, nil, true
	}
	apiKey, ok := settings_.API.APIKeys[settings.ApiTypeClaude+"-api-key"]
	if !ok {
		fmt.Printf("Error: Claude API key not found in settings\n")
		return nil, nil, true
	}

	baseURL := "https://api.anthropic.com"
	client := api.NewClient(apiKey, baseURL)

	// Prepare the message request
	req := &api.MessageRequest{
		Model: "claude-3-sonnet-20240229",
		Messages: []api.Message{
			{
				Role: "user",
				// TODO(manuel, 2024-06-03) Find a better way to ensure that this is always a list (even though it can be a string in the response? maybe we don't need to care at all)
				Content: []api.Content{api.NewTextContent("Hello, Claude! Please check my spelling on this sentiment filled sentence.")},
			},
		},
		Tools: []api.Tool{
			{
				Name: "spellcheck",
				InputSchema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"text": map[string]interface{}{
							"type": "string",
						},
					},
					"required": []string{"text"},
				},
			},
			{
				Name: "sentiment",
				InputSchema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"text": map[string]interface{}{
							"type": "string",
						},
					},
					"required": []string{"text"},
				},
			},
		},
		MaxTokens: 100,
		Stream:    true,
	}

	// Send the message request and receive the streaming response
	ctx := context.Background()
	events, err := client.StreamMessage(ctx, req)
	if err != nil {
		fmt.Printf("Error streaming message: %v\n", err)
		return nil, nil, true
	}
	return events, settings_, false
}

func testBlockMerger() {
	events, stepSettings, done := streamTestRequest()
	if done {
		fmt.Println("Error streaming message")
		return
	}

	metadata := chat.EventMetadata{
		ID:       conversation.NewNodeID(),
		ParentID: conversation.NewNodeID(),
	}
	stepMetadata := &steps.StepMetadata{
		StepID:     uuid.New(),
		Type:       "claude-messages",
		InputType:  "conversation.Conversation",
		OutputType: "api.MessageResponse",
		Metadata: map[string]interface{}{
			steps.MetadataSettingsSlug: stepSettings.GetMetadata(),
		},
	}
	completionMerger := claude.NewContentBlockMerger(metadata, stepMetadata)

	for {
		select {
		case event, ok := <-events:
			if !ok {
				response := completionMerger.Response()
				fmt.Printf("\n\n\n")
				for _, v := range response.Content {
					switch v.Type() {
					case api.ContentTypeText:
						fmt.Printf("TEXT: %s\n", v.(api.TextContent).Text)
					case api.ContentTypeImage:
						fmt.Println("IMAGE")
					case api.ContentTypeToolUse:
						v_ := v.(api.ToolUseContent)
						fmt.Printf("TOOL_USE: name %s input %s\n", v_.Name, string(v_.Input))

					case api.ContentTypeToolResult:
						v_ := v.(api.ToolResultContent)
						fmt.Printf("TOOL_RESULT: %s\n", v_.Content)
					}
				}
				return
			}
			completions, err := completionMerger.Add(event)
			if err != nil {
				fmt.Println("Error adding event to completionMerger:", err)
				return
			}

			for _, partialCompletion := range completions {
				v_, _ := yaml.Marshal(partialCompletion)
				fmt.Println(v_)
			}

		}
	}
}
func main() {
	//testSimple()
	testBlockMerger()
}

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/claude/api"
	"os"
	"strings"
)

func main() {
	// Set up the Claude API client
	apiKey := os.Getenv("CLAUDE_API_KEY")
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
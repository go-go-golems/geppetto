package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/claude"
	"os"
	"strings"
)

func main() {
	// Set up the Claude API client
	apiKey := os.Getenv("CLAUDE_API_KEY")
	baseURL := "https://api.anthropic.com"
	client := claude.NewClient(apiKey, baseURL)

	// Prepare the message request
	req := &claude.MessageRequest{
		Model: "claude-3-sonnet-20240229",
		Messages: []claude.Message{
			{
				Role: "user",
				// TODO(manuel, 2024-06-03) Find a better way to ensure that this is always a list (even though it can be a string in the response? maybe we don't need to care at all)
				Content: []claude.Content{claude.NewTextContent("Hello, Claude! How are you doing today?")},
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
		switch event.Type {
		case "message_start":
			fmt.Println("Assistant: ")
		case "content_block_start":
			fmt.Println()
		case "content_block_delta":
			delta, ok := event.Delta.(map[string]interface{})
			if !ok {
				fmt.Printf("Error: invalid delta type: %T\n", event.Delta)
				continue
			}
			deltaType, ok := delta["type"].(string)
			if !ok {
				fmt.Printf("Error: invalid delta type: %v\n", delta["type"])
				continue
			}
			switch deltaType {
			case "text_delta":
				text, ok := delta["text"].(string)
				if !ok {
					fmt.Printf("Error: invalid text delta: %v\n", delta["text"])
					continue
				}
				fmt.Print(text)
				response += text
			case "input_json_delta":
				partialJSON, ok := delta["partial_json"].(string)
				if !ok {
					fmt.Printf("Error: invalid input JSON delta: %v\n", delta["partial_json"])
					continue
				}
				fmt.Print(partialJSON)
				response += partialJSON
			default:
				fmt.Printf("Unknown delta type: %s\n", deltaType)
			}
		case "content_block_stop":
			contentBlockStop := make(map[string]interface{})
			contentBlockStop["type"] = "content_block_stop"
			contentBlockStop["index"] = event.Index
			jsonBytes, err := json.MarshalIndent(contentBlockStop, "", "  ")
			if err != nil {
				fmt.Printf("Error: failed to marshal content_block_stop: %v\n", err)
				continue
			}
			fmt.Printf("\n%s\n", string(jsonBytes))
		case "message_delta":
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
		case "message_stop":
			fmt.Println()
		default:
			fmt.Printf("Unknown event type: %s\n", event.Type)
		}
	}
	fmt.Println()

	// Print the full response
	fmt.Println("Full response:")
	fmt.Println(strings.TrimSpace(response))
}

package main

import (
	"encoding/json"
	"fmt"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/claude"
	"log"
	"strings"
	"time"
)

const key = "FOOBAR"

func main() {
	mainStream()
}

func mainNonStream() {
	// Create a new Claude API client
	client := claude.NewClient(key)

	// Define the request parameters
	req := &claude.Request{
		Model:             "claude-2",
		Prompt:            "\n\nHuman: Translate the following English text to French: 'Hello, how are you?'\n\nAssistant: ",
		MaxTokensToSample: 512,
		Stream:            false,
	}

	// Send the request
	resp, err := client.Complete(req)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	// Print the completion response
	fmt.Println("Completion:", resp.Completion)
	fmt.Println("StopReason:", resp.StopReason)
	fmt.Println("Model:", resp.Model)
}

func mainStream() {
	// Create a new Claude API client
	client := claude.NewClient(key)

	// Define the request parameters
	req := &claude.Request{
		Model:             "claude-2",
		Prompt:            "\n\nHuman: Translate the following English text to French: 'Hello, how are you?'\n\nAssistant:\n",
		MaxTokensToSample: 50,
		Stream:            true,
	}

	// Send the streaming request
	events, err := client.StreamComplete(req)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	timeout := time.After(10 * time.Second) // Optional timeout to stop listening after a while

	isFirstEvent := true
	// Listen to the events channel for incoming events
	for {
		select {
		case event, ok := <-events:
			if !ok {
				return
			}
			decoded := map[string]interface{}{}
			err = json.Unmarshal(([]byte)(event.Data), &decoded)
			if err != nil {
				log.Fatalf("Error: %v", err)
			}
			if decoded["completion"] != nil {
				completion := decoded["completion"].(string)
				if isFirstEvent {
					completion = strings.TrimLeft(completion, " ")
					isFirstEvent = false
				}
				fmt.Print(completion)
			}
		case <-timeout:
			fmt.Println("Timeout reached. Stopping the stream listener.")
			return
		}
	}
}

//go:build ignore
// +build ignore

package main

import (
	"fmt"
	"os"

	"github.com/go-go-golems/geppetto/pkg/inference/engine/factory"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/types"
)

func ptr[T any](v T) *T { return &v }

func main() {
	ss, err := settings.NewStepSettings()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create step settings: %v\n", err)
		os.Exit(1)
	}
	ss.API.APIKeys["openai-api-key"] = "test"
	ss.Chat.Engine = ptr("gpt-5-mini")

	e1, err := factory.NewEngineFromStepSettings(ss)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create engine for default api-type: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("default (ai-api-type unset) engine=%T\n", e1)

	apiType := types.ApiTypeOpenAIResponses
	ss.Chat.ApiType = &apiType
	e2, err := factory.NewEngineFromStepSettings(ss)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create engine for openai-responses: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("explicit ai-api-type=openai-responses engine=%T\n", e2)
}

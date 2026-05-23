package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/go-go-golems/geppetto/cmd/examples/internal/runnerexample"
	"github.com/go-go-golems/geppetto/pkg/inference/runner"
	"github.com/go-go-golems/geppetto/pkg/turns"
)

type CalculatorRequest struct {
	A  float64 `json:"a"`
	B  float64 `json:"b"`
	Op string  `json:"op"`
}

type CalculatorResponse struct {
	Result float64 `json:"result"`
}

func calculatorTool(req CalculatorRequest) (CalculatorResponse, error) {
	switch strings.TrimSpace(req.Op) {
	case "", "add":
		return CalculatorResponse{Result: req.A + req.B}, nil
	case "sub":
		return CalculatorResponse{Result: req.A - req.B}, nil
	case "mul":
		return CalculatorResponse{Result: req.A * req.B}, nil
	case "div":
		if req.B == 0 {
			return CalculatorResponse{}, fmt.Errorf("division by zero")
		}
		return CalculatorResponse{Result: req.A / req.B}, nil
	default:
		return CalculatorResponse{}, fmt.Errorf("unsupported op %q", req.Op)
	}
}

func main() {
	var (
		profile           = flag.String("profile", "gpt-5-nano-low", "engine profile slug")
		profileRegistries = flag.String("profile-registries", runnerexample.PinocchioProfileRegistryPath(), "comma-separated engine profile registry paths")
		prompt            = flag.String("prompt", "Use the calculator tool to multiply 17 by 23, then explain the answer briefly.", "prompt to run")
	)
	flag.Parse()

	stepSettings, closeProfiles, err := runnerexample.OpenAIInferenceSettingsFromProfiles(context.Background(), *profileRegistries, *profile, true)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer func() { _ = closeProfiles() }()

	r := runner.New(
		runner.WithFuncTool("calculator", "Basic arithmetic calculator", calculatorTool),
	)
	_, out, err := r.Run(context.Background(), runner.StartRequest{
		Prompt: *prompt,
		Runtime: runner.Runtime{
			InferenceSettings: stepSettings,
			SystemPrompt:      "You are a concise assistant that uses tools when needed.",
			ToolNames:         []string{"calculator"},
		},
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	turns.FprintTurn(os.Stdout, out)
}

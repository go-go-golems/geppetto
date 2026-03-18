//go:build ignore

// Sketch only.
//
// This file is intentionally not buildable today. It records the proposed shape
// of the opinionated Geppetto runner API described in the GP-40 design doc.

package main

import (
	"context"
	"log"

	// Placeholder package path and names for the proposed API.
	oprunner "github.com/go-go-golems/geppetto/pkg/inference/opinionated"
)

type CalcRequest struct {
	A  float64 `json:"a"`
	B  float64 `json:"b"`
	Op string  `json:"op"`
}

type CalcResponse struct {
	Result float64 `json:"result"`
}

func calculatorTool(in CalcRequest) (CalcResponse, error) {
	switch in.Op {
	case "", "add":
		return CalcResponse{Result: in.A + in.B}, nil
	case "sub":
		return CalcResponse{Result: in.A - in.B}, nil
	case "mul":
		return CalcResponse{Result: in.A * in.B}, nil
	case "div":
		return CalcResponse{Result: in.A / in.B}, nil
	default:
		return CalcResponse{}, nil
	}
}

func simpleCLI(ctx context.Context) error {
	runner := oprunner.Must(
		oprunner.WithSystemPrompt("You are a concise inventory assistant."),
		oprunner.WithFuncTool("calc", "A basic calculator", calculatorTool),
	)

	_, turn, err := runner.Run(ctx, oprunner.StartRequest{
		Prompt: "Use calc to compute 17 * 23 and explain the result briefly.",
	})
	if err != nil {
		return err
	}

	log.Printf("final turn: %+v", turn)
	return nil
}

func profileDrivenChat(ctx context.Context, composer oprunner.RuntimeComposer, regs ...oprunner.ToolRegistrar) error {
	runner := oprunner.Must(
		oprunner.WithRuntimeComposer(composer),
		oprunner.WithToolRegistrars(regs...),
	)

	prep, handle, err := runner.Start(ctx, oprunner.StartRequest{
		SessionID: "conv-123",
		Prompt:    "Show me the latest coin inventory anomalies.",
		Runtime: oprunner.RuntimeRequest{
			RuntimeKey: "app:coinvault::inference:team/analyst",
		},
	})
	if err != nil {
		return err
	}

	log.Printf("prepared runtime fingerprint: %s", prep.Runtime.RuntimeFingerprint)
	_, err = handle.Wait()
	return err
}

func customOuterLoop(ctx context.Context, runner *oprunner.Runner) error {
	prep, err := runner.Prepare(ctx, oprunner.StartRequest{
		SessionID: "extract-run-001",
		Prompt:    "Extract the first pass of relationships.",
	})
	if err != nil {
		return err
	}

	for i := 0; i < 3; i++ {
		handle, err := prep.Session.StartInference(ctx)
		if err != nil {
			return err
		}
		turn, err := handle.Wait()
		if err != nil {
			return err
		}
		if oprunner.ShouldStop(turn) {
			return nil
		}
		if _, err := prep.Session.AppendNewTurnFromUserPrompt("Continue from the previous partial extraction."); err != nil {
			return err
		}
	}
	return nil
}


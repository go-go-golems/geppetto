package runner

import (
	"context"
	"errors"
	"testing"
)

type echoInput struct {
	Text string `json:"text"`
}

type echoOutput struct {
	Text string `json:"text"`
}

func echoTool(in echoInput) (echoOutput, error) {
	return echoOutput(in), nil
}

func upperTool(in echoInput) (echoOutput, error) {
	return echoOutput(in), nil
}

func TestFuncToolRegistersDefinition(t *testing.T) {
	reg, err := buildRegistry(context.Background(), []ToolRegistrar{
		FuncTool("echo", "Echo the provided text", echoTool),
	}, nil)
	if err != nil {
		t.Fatalf("buildRegistry: %v", err)
	}
	if reg == nil {
		t.Fatal("expected non-nil registry")
	}

	def, err := reg.GetTool("echo")
	if err != nil {
		t.Fatalf("GetTool: %v", err)
	}
	if def.Name != "echo" {
		t.Fatalf("unexpected tool name: %s", def.Name)
	}
	if def.Description != "Echo the provided text" {
		t.Fatalf("unexpected description: %s", def.Description)
	}
}

func TestBuildRegistryReturnsNilWithoutRegistrars(t *testing.T) {
	reg, err := buildRegistry(context.Background(), nil, nil)
	if err != nil {
		t.Fatalf("buildRegistry: %v", err)
	}
	if reg != nil {
		t.Fatalf("expected nil registry, got %#v", reg)
	}
}

func TestBuildRegistryFiltersRequestedTools(t *testing.T) {
	reg, err := buildRegistry(context.Background(), []ToolRegistrar{
		FuncTool("echo", "Echo the provided text", echoTool),
		FuncTool("upper", "Upper-case the provided text", upperTool),
	}, []string{"upper"})
	if err != nil {
		t.Fatalf("buildRegistry: %v", err)
	}

	tools := reg.ListTools()
	if len(tools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(tools))
	}
	if tools[0].Name != "upper" {
		t.Fatalf("unexpected tool name: %s", tools[0].Name)
	}
}

func TestBuildRegistryErrorsOnMissingRequestedTool(t *testing.T) {
	_, err := buildRegistry(context.Background(), []ToolRegistrar{
		FuncTool("echo", "Echo the provided text", echoTool),
	}, []string{"missing"})
	if !errors.Is(err, ErrRequestedToolMissing) {
		t.Fatalf("expected ErrRequestedToolMissing, got %v", err)
	}
}

func TestBuildRegistrySkipsNilRegistrars(t *testing.T) {
	reg, err := buildRegistry(context.Background(), []ToolRegistrar{
		nil,
		FuncTool("echo", "Echo the provided text", echoTool),
	}, nil)
	if err != nil {
		t.Fatalf("buildRegistry: %v", err)
	}
	if reg == nil {
		t.Fatal("expected non-nil registry")
	}
	if _, err := reg.GetTool("echo"); err != nil {
		t.Fatalf("GetTool: %v", err)
	}
}

package tools

import (
	"context"
	"testing"
)

func TestAdvertisedToolDefinitionsFromContext_UsesLiveRegistryOnly(t *testing.T) {
	t.Parallel()

	reg := NewInMemoryToolRegistry()
	type echoIn struct {
		Text string `json:"text"`
	}
	echoTool, err := NewToolFromFunc("runtime_echo", "Echo runtime text", func(in echoIn) (map[string]any, error) {
		return map[string]any{"echo": in.Text}, nil
	})
	if err != nil {
		t.Fatalf("NewToolFromFunc: %v", err)
	}
	if err := reg.RegisterTool("runtime_echo", *echoTool); err != nil {
		t.Fatalf("RegisterTool: %v", err)
	}

	defs := AdvertisedToolDefinitionsFromContext(WithRegistry(context.Background(), reg))
	if len(defs) != 1 {
		t.Fatalf("expected one advertised tool definition, got %d", len(defs))
	}
	if defs[0].Name != "runtime_echo" {
		t.Fatalf("expected runtime tool name, got %q", defs[0].Name)
	}
	if defs[0].Description != "Echo runtime text" {
		t.Fatalf("expected runtime tool description, got %q", defs[0].Description)
	}
	if defs[0].Parameters == nil {
		t.Fatalf("expected runtime tool parameters to be present")
	}
}

func TestAdvertisedToolDefinitionsFromContext_NoRegistryReturnsNil(t *testing.T) {
	t.Parallel()

	defs := AdvertisedToolDefinitionsFromContext(context.Background())
	if defs != nil {
		t.Fatalf("expected nil advertised tool definitions without a live registry, got %#v", defs)
	}
}

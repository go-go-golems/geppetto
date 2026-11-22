package claude

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/go-go-golems/geppetto/pkg/inference/engine"
	"github.com/go-go-golems/geppetto/pkg/inference/tools"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/claude/api"
	"github.com/go-go-golems/geppetto/pkg/turns"
)

func echoToolContext(t *testing.T) context.Context {
	t.Helper()
	reg := tools.NewInMemoryToolRegistry()
	type in struct {
		Text string `json:"text"`
	}
	tool, err := tools.NewToolFromFunc("echo", "Echo text", func(v in) (map[string]any, error) { return map[string]any{"echo": v.Text}, nil })
	if err != nil {
		t.Fatalf("NewToolFromFunc: %v", err)
	}
	if err := reg.RegisterTool("echo", *tool); err != nil {
		t.Fatalf("RegisterTool: %v", err)
	}
	return tools.WithRegistry(context.Background(), reg)
}

// Tool choice "none" (a tool loop's final answer) is sent as
// tool_choice {type: none}, with the tools still listed (the history may hold
// tool_use blocks, which the API only accepts with tool definitions).
func TestAttachToolsToClaudeRequest_ToolChoiceNone(t *testing.T) {
	ctx := echoToolContext(t)
	turn := &turns.Turn{}
	if err := engine.KeyToolConfig.Set(&turn.Data, engine.ToolConfig{Enabled: true, ToolChoice: engine.ToolChoiceNone}); err != nil {
		t.Fatalf("set tool config: %v", err)
	}
	req := &api.MessageRequest{}
	attachToolsToClaudeRequest(ctx, turn, req)
	if len(req.Tools) != 1 || req.ToolChoice == nil || req.ToolChoice.Type != "none" {
		t.Fatalf("want 1 tool and tool_choice none, got %d tools and %#v", len(req.Tools), req.ToolChoice)
	}
	b, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(b), `"tool_choice":{"type":"none"}`) {
		t.Fatalf("wire shape: %s", b)
	}
}

func TestAttachToolsToClaudeRequest_DefaultChoiceIsOmitted(t *testing.T) {
	ctx := echoToolContext(t)
	for _, turn := range []*turns.Turn{{}, nil} {
		req := &api.MessageRequest{}
		attachToolsToClaudeRequest(ctx, turn, req)
		if len(req.Tools) != 1 || req.ToolChoice != nil {
			t.Fatalf("want 1 tool and no tool_choice, got %d tools and %#v", len(req.Tools), req.ToolChoice)
		}
		b, _ := json.Marshal(req)
		if strings.Contains(string(b), "tool_choice") {
			t.Fatalf("tool_choice must be omitted: %s", b)
		}
	}
}

package openai_responses

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/go-go-golems/geppetto/pkg/inference/engine"
	inferencetools "github.com/go-go-golems/geppetto/pkg/inference/tools"
	"github.com/go-go-golems/geppetto/pkg/turns"
)

// wireItems marshals the input items the engine would send and decodes them
// generically, so the tests see exactly the JSON a provider receives.
func wireItems(t *testing.T, turn *turns.Turn) []map[string]any {
	t.Helper()
	b, err := json.Marshal(buildInputItemsFromTurn(turn))
	if err != nil {
		t.Fatalf("marshal input items: %v", err)
	}
	var out []map[string]any
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal input items: %v", err)
	}
	return out
}

// Assistant history without a provider item id (a transcript the caller
// supplies) must not be sent as an output_text list: that is only valid on
// output items, which carry an id and a status, and schema-validating
// providers reject it ("Input should be 'input_text'").
func TestWireShape_AssistantHistoryWithoutItemIDIsStringContent(t *testing.T) {
	turn := &turns.Turn{Blocks: []turns.Block{
		turns.NewUserTextBlock("Say ping."),
		turns.NewAssistantTextBlock("ping"),
		turns.NewUserTextBlock("Now say pong."),
	}}
	items := wireItems(t, turn)
	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %d: %#v", len(items), items)
	}
	as := items[1]
	if as["role"] != "assistant" || as["content"] != "ping" {
		t.Fatalf("assistant history should be {role: assistant, content: \"ping\"}, got %#v", as)
	}
	if _, ok := as["id"]; ok {
		t.Fatalf("assistant history must not invent an id: %#v", as)
	}
	// User content keeps the input_text list form.
	if parts, ok := items[2]["content"].([]any); !ok || len(parts) != 1 || parts[0].(map[string]any)["type"] != "input_text" {
		t.Fatalf("user content should stay an input_text list, got %#v", items[2]["content"])
	}
}

// The same holds after a reasoning item when the text has no item id.
func TestWireShape_ReasoningFollowerWithoutItemIDIsStringContent(t *testing.T) {
	rs := turns.Block{Kind: turns.BlockKindReasoning, ID: "rs_1", Payload: map[string]any{
		turns.PayloadKeyItemID:           "rs_1",
		turns.PayloadKeyEncryptedContent: "gAAAAA...",
	}}
	turn := &turns.Turn{Blocks: []turns.Block{
		turns.NewUserTextBlock("Question"),
		rs,
		turns.NewAssistantTextBlock("Answer"),
		turns.NewUserTextBlock("Follow-up"),
	}}
	items := wireItems(t, turn)
	if len(items) != 4 {
		t.Fatalf("expected 4 items, got %d: %#v", len(items), items)
	}
	if items[1]["type"] != "reasoning" {
		t.Fatalf("item 2 should be the reasoning item, got %#v", items[1])
	}
	if items[2]["role"] != "assistant" || items[2]["content"] != "Answer" {
		t.Fatalf("assistant follower should be string content, got %#v", items[2])
	}
}

// Text replayed with the provider's item id is a complete output item: type
// message, the id, status completed, and annotations on its output_text part.
func TestWireShape_AssistantOutputItemWithItemIDIsComplete(t *testing.T) {
	rs := turns.Block{Kind: turns.BlockKindReasoning, ID: "rs_1", Payload: map[string]any{
		turns.PayloadKeyItemID:           "rs_1",
		turns.PayloadKeyEncryptedContent: "gAAAAA...",
	}}
	as := turns.NewAssistantTextBlock("Answer")
	as.Payload[turns.PayloadKeyItemID] = "msg_1"
	turn := &turns.Turn{Blocks: []turns.Block{turns.NewUserTextBlock("Question"), rs, as}}
	items := wireItems(t, turn)
	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %d: %#v", len(items), items)
	}
	msg := items[2]
	if msg["type"] != "message" || msg["role"] != "assistant" || msg["id"] != "msg_1" || msg["status"] != "completed" {
		t.Fatalf("output item should carry type, role, id and status, got %#v", msg)
	}
	parts, ok := msg["content"].([]any)
	if !ok || len(parts) != 1 {
		t.Fatalf("output item content should be one part, got %#v", msg["content"])
	}
	part := parts[0].(map[string]any)
	if part["type"] != "output_text" || part["text"] != "Answer" {
		t.Fatalf("unexpected output part %#v", part)
	}
	if ann, ok := part["annotations"].([]any); !ok || len(ann) != 0 {
		t.Fatalf("output_text should carry annotations: [], got %#v", part["annotations"])
	}
}

func echoRegistry(t *testing.T) inferencetools.ToolRegistry {
	t.Helper()
	reg := inferencetools.NewInMemoryToolRegistry()
	type in struct {
		Text string `json:"text"`
	}
	tool, err := inferencetools.NewToolFromFunc("echo", "Echo text", func(v in) (map[string]any, error) {
		return map[string]any{"echo": v.Text}, nil
	})
	if err != nil {
		t.Fatalf("NewToolFromFunc: %v", err)
	}
	if err := reg.RegisterTool("echo", *tool); err != nil {
		t.Fatalf("RegisterTool: %v", err)
	}
	return reg
}

// Tools in the context are sent when the turn has no tool config, as the
// chat-completions, Claude and Gemini engines do. Before, they were dropped
// silently and models wrote their tool calls as plain text.
func TestAttachToolsToResponsesRequest_SendsAdvertisedToolsWithoutToolConfig(t *testing.T) {
	eng := &Engine{}
	reqBody := &responsesRequest{}
	ctx := inferencetools.WithRegistry(context.Background(), echoRegistry(t))
	if err := eng.attachToolsToResponsesRequest(ctx, &turns.Turn{}, reqBody); err != nil {
		t.Fatalf("attachToolsToResponsesRequest: %v", err)
	}
	if len(reqBody.Tools) != 1 {
		t.Fatalf("expected the advertised tool, got %#v", reqBody.Tools)
	}
	if tool, _ := reqBody.Tools[0].(map[string]any); tool["name"] != "echo" || tool["type"] != "function" {
		t.Fatalf("unexpected tool payload %#v", reqBody.Tools[0])
	}
}

// An explicit Enabled: false still keeps them out.
func TestAttachToolsToResponsesRequest_RespectsDisabledToolConfig(t *testing.T) {
	eng := &Engine{}
	turn := &turns.Turn{}
	if err := engine.KeyToolConfig.Set(&turn.Data, engine.ToolConfig{Enabled: false}); err != nil {
		t.Fatalf("set tool config: %v", err)
	}
	reqBody := &responsesRequest{}
	ctx := inferencetools.WithRegistry(context.Background(), echoRegistry(t))
	if err := eng.attachToolsToResponsesRequest(ctx, turn, reqBody); err != nil {
		t.Fatalf("attachToolsToResponsesRequest: %v", err)
	}
	if len(reqBody.Tools) != 0 {
		t.Fatalf("expected no tools with Enabled: false, got %#v", reqBody.Tools)
	}
}

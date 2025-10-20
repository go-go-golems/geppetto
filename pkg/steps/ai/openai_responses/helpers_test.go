package openai_responses

import (
    "testing"

    "github.com/go-go-golems/geppetto/pkg/turns"
)

func typesOf(parts []responsesContentPart) []string {
    ts := make([]string, 0, len(parts))
    for _, p := range parts {
        ts = append(ts, p.Type)
    }
    return ts
}

func TestBuildInputItemsFromTurn_PlainChat(t *testing.T) {
    turn := &turns.Turn{Blocks: []turns.Block{
        turns.NewSystemTextBlock("You are a LLM."),
        turns.NewUserTextBlock("Hello"),
    }}

    got := buildInputItemsFromTurn(turn)
    if len(got) != 2 {
        t.Fatalf("expected 2 items, got %d", len(got))
    }
    if got[0].Type != "" || got[0].Role != "system" {
        t.Fatalf("first item must be role-based system, got type=%q role=%q", got[0].Type, got[0].Role)
    }
    if got[1].Type != "" || got[1].Role != "user" {
        t.Fatalf("second item must be role-based user, got type=%q role=%q", got[1].Type, got[1].Role)
    }
    if len(got[0].Content) != 1 || typesOf(got[0].Content)[0] != "input_text" {
        t.Fatalf("system content must have single input_text part")
    }
    if len(got[1].Content) != 1 || typesOf(got[1].Content)[0] != "input_text" {
        t.Fatalf("user content must have single input_text part")
    }
}

func TestBuildInputItemsFromTurn_ReasoningWithAssistantFollower(t *testing.T) {
    rs := turns.Block{Kind: turns.BlockKindReasoning, ID: "rs_1", Payload: map[string]any{
        turns.PayloadKeyEncryptedContent: "gAAAAA...",
    }}
    as := turns.NewAssistantTextBlock("Answer")
    turn := &turns.Turn{Blocks: []turns.Block{
        turns.NewSystemTextBlock("You are a LLM."),
        turns.NewUserTextBlock("Question"),
        rs,
        as,
    }}

    got := buildInputItemsFromTurn(turn)
    if len(got) != 4 {
        t.Fatalf("expected 4 items, got %d", len(got))
    }
    // Pre-context role-based items
    if got[0].Type != "" || got[0].Role != "system" { t.Fatalf("pre system wrong shape") }
    if got[1].Type != "" || got[1].Role != "user" { t.Fatalf("pre user wrong shape") }
    // Reasoning then item-based message follower
    if got[2].Type != "reasoning" { t.Fatalf("item 3 must be reasoning, got %q", got[2].Type) }
    if got[2].ID != "rs_1" { t.Fatalf("reasoning id mismatch: %q", got[2].ID) }
    if got[3].Type != "message" || got[3].Role != "assistant" { t.Fatalf("item 4 must be item-based assistant message") }
    if len(got[3].Content) != 1 || typesOf(got[3].Content)[0] != "output_text" { t.Fatalf("assistant item content must be output_text") }
}



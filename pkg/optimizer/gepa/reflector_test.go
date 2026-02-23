package gepa

import (
	"context"
	"testing"

	"github.com/go-go-golems/geppetto/pkg/turns"
)

type staticEngine struct {
	text string
}

func (e *staticEngine) RunInference(_ context.Context, _ *turns.Turn) (*turns.Turn, error) {
	return &turns.Turn{
		Blocks: []turns.Block{
			turns.NewAssistantTextBlock(e.text),
		},
	}, nil
}

func TestExtractTripleBacktickBlockPreservesFirstWordForPlainFence(t *testing.T) {
	in := "```Answer with OPT and be concise.```"
	got := extractTripleBacktickBlock(in)
	want := "Answer with OPT and be concise."
	if got != want {
		t.Fatalf("unexpected fenced extraction: got %q want %q", got, want)
	}
}

func TestExtractTripleBacktickBlockStripsKnownLanguageLine(t *testing.T) {
	in := "```markdown\nAnswer with OPT and be concise.\n```"
	got := extractTripleBacktickBlock(in)
	want := "Answer with OPT and be concise."
	if got != want {
		t.Fatalf("unexpected fenced extraction: got %q want %q", got, want)
	}
}

func TestReflectorProposeUsesFenceExtraction(t *testing.T) {
	r := &Reflector{
		Engine: &staticEngine{
			text: "```markdown\nAnswer with OPT and be concise.\n```",
		},
	}

	proposed, raw, err := r.Propose(context.Background(), "seed", "side-info")
	if err != nil {
		t.Fatalf("Propose returned error: %v", err)
	}
	if raw == "" {
		t.Fatalf("expected raw reflection text")
	}
	if proposed != "Answer with OPT and be concise." {
		t.Fatalf("unexpected proposed prompt: %q", proposed)
	}
}

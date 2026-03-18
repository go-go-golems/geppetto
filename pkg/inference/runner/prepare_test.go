package runner

import (
	"context"
	"errors"
	"testing"

	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/go-go-golems/geppetto/pkg/turns"
)

func newTestStepSettings(t *testing.T) *settings.StepSettings {
	t.Helper()

	ss, err := settings.NewStepSettings()
	if err != nil {
		t.Fatalf("NewStepSettings: %v", err)
	}
	ss.API.APIKeys["openai-api-key"] = "test-key"
	return ss
}

func TestPreparePromptOnlyBuildsSessionAndRegistry(t *testing.T) {
	r := New(
		WithFuncTool("echo", "Echo the provided text", echoTool),
	)

	prepared, err := r.Prepare(context.Background(), StartRequest{
		SessionID: "session-123",
		Prompt:    "hello",
		Runtime: Runtime{
			StepSettings: newTestStepSettings(t),
			ToolNames:    []string{"echo"},
		},
	})
	if err != nil {
		t.Fatalf("Prepare: %v", err)
	}

	if prepared.Session == nil {
		t.Fatal("expected session")
	}
	if prepared.Session.SessionID != "session-123" {
		t.Fatalf("unexpected session id: %s", prepared.Session.SessionID)
	}
	if prepared.Turn == nil || len(prepared.Turn.Blocks) != 1 {
		t.Fatalf("expected one seeded block, got %#v", prepared.Turn)
	}
	if prepared.Turn.Blocks[0].Kind != turns.BlockKindUser {
		t.Fatalf("expected user block, got %s", prepared.Turn.Blocks[0].Kind)
	}
	if prepared.Registry == nil {
		t.Fatal("expected tool registry")
	}
	if _, err := prepared.Registry.GetTool("echo"); err != nil {
		t.Fatalf("GetTool: %v", err)
	}
	if prepared.Engine == nil {
		t.Fatal("expected engine")
	}
	if prepared.Session.Builder == nil {
		t.Fatal("expected session builder")
	}
}

func TestPrepareClonesSeedTurnAndAppendsPrompt(t *testing.T) {
	r := New()

	seed := &turns.Turn{}
	turns.AppendBlock(seed, turns.NewUserTextBlock("seed"))

	prepared, err := r.Prepare(context.Background(), StartRequest{
		SessionID: "seeded",
		Prompt:    "follow up",
		SeedTurn:  seed,
		Runtime: Runtime{
			StepSettings: newTestStepSettings(t),
		},
	})
	if err != nil {
		t.Fatalf("Prepare: %v", err)
	}

	if prepared.Turn == seed {
		t.Fatal("expected seed turn to be cloned")
	}
	if len(seed.Blocks) != 1 {
		t.Fatalf("expected original seed to stay unchanged, got %d blocks", len(seed.Blocks))
	}
	if len(prepared.Turn.Blocks) != 2 {
		t.Fatalf("expected cloned turn to contain original and appended prompt, got %d blocks", len(prepared.Turn.Blocks))
	}
	if sid, ok, err := turns.KeyTurnMetaSessionID.Get(prepared.Turn.Metadata); err != nil || !ok || sid != "seeded" {
		t.Fatalf("expected session metadata on prepared turn, got sid=%q ok=%v err=%v", sid, ok, err)
	}
}

func TestPrepareRejectsEmptyPromptAndSeed(t *testing.T) {
	r := New()

	_, err := r.Prepare(context.Background(), StartRequest{
		Runtime: Runtime{
			StepSettings: newTestStepSettings(t),
		},
	})
	if !errors.Is(err, ErrPromptAndSeedEmpty) {
		t.Fatalf("expected ErrPromptAndSeedEmpty, got %v", err)
	}
}

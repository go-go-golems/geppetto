package runner

import (
	"context"
	"errors"
	"testing"

	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/go-go-golems/geppetto/pkg/turns"
)

func newTestInferenceSettings(t *testing.T) *settings.InferenceSettings {
	t.Helper()

	ss, err := settings.NewInferenceSettings()
	if err != nil {
		t.Fatalf("NewInferenceSettings: %v", err)
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
			InferenceSettings: newTestInferenceSettings(t),
			ToolNames:         []string{"echo"},
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

	seed := &turns.Turn{ID: "existing-turn-id"}
	turns.AppendBlock(seed, turns.NewUserTextBlock("seed"))

	prepared, err := r.Prepare(context.Background(), StartRequest{
		SessionID: "seeded",
		Prompt:    "follow up",
		SeedTurn:  seed,
		Runtime: Runtime{
			InferenceSettings: newTestInferenceSettings(t),
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
	if prepared.Turn.ID != "" {
		t.Fatalf("expected prepared turn id to be cleared, got %q", prepared.Turn.ID)
	}
	if seed.ID != "existing-turn-id" {
		t.Fatalf("expected original seed id to stay unchanged, got %q", seed.ID)
	}
	if sid, ok, err := turns.KeyTurnMetaSessionID.Get(prepared.Turn.Metadata); err != nil || !ok || sid != "seeded" {
		t.Fatalf("expected session metadata on prepared turn, got sid=%q ok=%v err=%v", sid, ok, err)
	}
}

func TestPrepareStampsRuntimeMetadataOnPreparedTurn(t *testing.T) {
	r := New()

	prepared, err := r.Prepare(context.Background(), StartRequest{
		Prompt: "hello",
		Runtime: Runtime{
			InferenceSettings:  newTestInferenceSettings(t),
			RuntimeKey:         "assistant@v1",
			RuntimeFingerprint: "fp-123",
			ProfileVersion:     7,
		},
	})
	if err != nil {
		t.Fatalf("Prepare: %v", err)
	}

	raw, ok, err := turns.KeyTurnMetaRuntime.Get(prepared.Turn.Metadata)
	if err != nil || !ok {
		t.Fatalf("expected runtime metadata, got ok=%v err=%v", ok, err)
	}
	attrib, ok := raw.(map[string]any)
	if !ok {
		t.Fatalf("expected runtime metadata map, got %T", raw)
	}
	if got := attrib["runtime_key"]; got != "assistant@v1" {
		t.Fatalf("unexpected runtime_key: %#v", got)
	}
	if got := attrib["runtime_fingerprint"]; got != "fp-123" {
		t.Fatalf("unexpected runtime_fingerprint: %#v", got)
	}
	if got := attrib["profile.version"]; got != uint64(7) {
		t.Fatalf("unexpected profile.version: %#v", got)
	}
}

func TestPreparePreservesExistingRuntimeMetadataFields(t *testing.T) {
	r := New()

	seed := &turns.Turn{}
	_ = turns.KeyTurnMetaRuntime.Set(&seed.Metadata, map[string]any{
		"profile.slug":     "assistant",
		"profile.registry": "team",
	})
	turns.AppendBlock(seed, turns.NewUserTextBlock("seed"))

	prepared, err := r.Prepare(context.Background(), StartRequest{
		SeedTurn: seed,
		Runtime: Runtime{
			InferenceSettings:  newTestInferenceSettings(t),
			RuntimeKey:         "assistant@v1",
			RuntimeFingerprint: "fp-123",
		},
	})
	if err != nil {
		t.Fatalf("Prepare: %v", err)
	}

	raw, ok, err := turns.KeyTurnMetaRuntime.Get(prepared.Turn.Metadata)
	if err != nil || !ok {
		t.Fatalf("expected runtime metadata, got ok=%v err=%v", ok, err)
	}
	attrib := raw.(map[string]any)
	if got := attrib["profile.slug"]; got != "assistant" {
		t.Fatalf("unexpected profile.slug: %#v", got)
	}
	if got := attrib["profile.registry"]; got != "team" {
		t.Fatalf("unexpected profile.registry: %#v", got)
	}
	if got := attrib["runtime_key"]; got != "assistant@v1" {
		t.Fatalf("unexpected runtime_key: %#v", got)
	}
}

func TestPrepareRejectsEmptyPromptAndSeed(t *testing.T) {
	r := New()

	_, err := r.Prepare(context.Background(), StartRequest{
		Runtime: Runtime{
			InferenceSettings: newTestInferenceSettings(t),
		},
	})
	if !errors.Is(err, ErrPromptAndSeedEmpty) {
		t.Fatalf("expected ErrPromptAndSeedEmpty, got %v", err)
	}
}

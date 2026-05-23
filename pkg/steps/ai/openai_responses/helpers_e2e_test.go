//go:build responses_e2e

package openai_responses

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dnaeon/go-vcr/recorder"
	profiles "github.com/go-go-golems/geppetto/pkg/engineprofiles"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	openaisettings "github.com/go-go-golems/geppetto/pkg/steps/ai/settings/openai"
	"github.com/go-go-golems/geppetto/pkg/turns"
)

func p[T any](v T) *T { return &v }

func resolvePinocchioResponsesSettings(ctx context.Context) (*settings.InferenceSettings, error) {
	registryPath := filepath.Join(os.Getenv("HOME"), ".config", "pinocchio", "profiles.yaml")
	specs, err := profiles.ParseRegistrySourceSpecs([]string{registryPath})
	if err != nil {
		return nil, err
	}
	chain, err := profiles.NewChainedRegistryFromSourceSpecs(ctx, specs)
	if err != nil {
		return nil, err
	}
	defer chain.Close()

	resolved, err := chain.ResolveEngineProfile(ctx, profiles.ResolveInput{EngineProfileSlug: profiles.MustEngineProfileSlug("gpt-5-nano-low")})
	if err != nil {
		return nil, err
	}
	return resolved.InferenceSettings, nil
}

func TestResponses_E2E_VCR_Simple(t *testing.T) {
	if os.Getenv("RUN_E2E") != "1" {
		t.Skip("Set RUN_E2E=1 to run live/VCR tests")
	}

	cassette := filepath.Join("testdata", "vcr", "responses_simple.yaml")
	mode := recorder.ModeReplaying
	if os.Getenv("RESPONSES_RECORD") == "1" {
		mode = recorder.ModeRecording
	}
	r, err := recorder.NewAsMode(cassette, mode, nil)
	if err != nil {
		t.Fatalf("recorder: %v", err)
	}
	defer r.Stop()

	// Override default client used by engine via environment; engine uses http.DefaultClient.
	// We temporarily swap the default transport.
	origTransport := http.DefaultTransport
	http.DefaultTransport = r
	defer func() { http.DefaultTransport = origTransport }()

	ss, err := resolvePinocchioResponsesSettings(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	ss.Chat.Engine = p("o4-mini")
	ss.Chat.Stream = true
	ss.Chat.MaxResponseTokens = p(512)
	ss.OpenAI = &openaisettings.Settings{
		ReasoningEffort:  p("medium"),
		ReasoningSummary: p("detailed"),
	}

	eng, err := NewEngine(ss)
	if err != nil {
		t.Fatal(err)
	}

	turn := &turns.Turn{Blocks: []turns.Block{
		turns.NewSystemTextBlock("You are a LLM."),
		turns.NewUserTextBlock("Say hi."),
	}}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if _, err := eng.RunInference(ctx, turn); err != nil {
		t.Fatalf("RunInference failed: %v", err)
	}
}

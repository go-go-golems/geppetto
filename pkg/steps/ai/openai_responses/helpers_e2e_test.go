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
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	openaisettings "github.com/go-go-golems/geppetto/pkg/steps/ai/settings/openai"
	"github.com/go-go-golems/geppetto/pkg/turns"
)

func p[T any](v T) *T { return &v }

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

	eng, err := NewEngine(&settings.StepSettings{
		API: &settings.APISettings{
			APIKeys:  map[string]string{"openai-api-key": os.Getenv("OPENAI_API_KEY")},
			BaseUrls: map[string]string{"openai-base-url": "https://api.openai.com/v1"},
		},
		Chat: &settings.ChatSettings{
			Engine:            p("o4-mini"),
			Stream:            true,
			MaxResponseTokens: p(512),
		},
		OpenAI: &openaisettings.Settings{
			ReasoningEffort:  p("medium"),
			ReasoningSummary: p("detailed"),
		},
	})
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

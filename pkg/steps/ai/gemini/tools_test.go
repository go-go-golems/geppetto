package gemini

import (
	"context"
	"testing"

	"github.com/go-go-golems/geppetto/pkg/inference/engine"
	"github.com/go-go-golems/geppetto/pkg/inference/tools"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/go-go-golems/geppetto/pkg/turns"
	"github.com/stretchr/testify/require"
	moderngenai "google.golang.org/genai"
)

// Tool choice "none" (a tool loop's final answer) turns function calling off
// while keeping the declarations; other choices keep AUTO.
func TestModernConfigHonoursToolChoiceNone(t *testing.T) {
	reg := tools.NewInMemoryToolRegistry()
	type in struct {
		Text string `json:"text"`
	}
	tool, err := tools.NewToolFromFunc("echo", "Echo text", func(v in) (map[string]any, error) { return map[string]any{"echo": v.Text}, nil })
	require.NoError(t, err)
	require.NoError(t, reg.RegisterTool("echo", *tool))
	ctx := tools.WithRegistry(context.Background(), reg)
	e := &GeminiEngine{settings: &settings.InferenceSettings{Chat: &settings.ChatSettings{}}}

	for choice, want := range map[engine.ToolChoice]moderngenai.FunctionCallingConfigMode{
		engine.ToolChoiceNone: moderngenai.FunctionCallingConfigModeNone,
		engine.ToolChoiceAuto: moderngenai.FunctionCallingConfigModeAuto,
	} {
		turn := &turns.Turn{}
		require.NoError(t, engine.KeyToolConfig.Set(&turn.Data, engine.ToolConfig{Enabled: true, ToolChoice: choice}))
		cfg, err := e.buildModernGenerateContentConfig(ctx, turn)
		require.NoError(t, err)
		require.Len(t, cfg.Tools, 1, "declarations stay for %q", choice)
		require.Equal(t, want, cfg.ToolConfig.FunctionCallingConfig.Mode, "choice %q", choice)
	}
}

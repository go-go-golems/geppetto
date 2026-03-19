package engineprofiles

import (
	"testing"

	"github.com/go-go-golems/geppetto/pkg/inference/engine"
	aitypes "github.com/go-go-golems/geppetto/pkg/steps/ai/types"
)

func TestMergeInferenceSettings_ExportedHelperMatchesEngineProfileMergeSemantics(t *testing.T) {
	base := mustTestInferenceSettings(t, aitypes.ApiTypeOpenAI, "gpt-4o-mini")
	baseModel := "gpt-4o-mini"
	base.Chat.Engine = &baseModel
	baseTemp := 0.2
	base.Chat.Temperature = &baseTemp

	overlay := mustTestInferenceSettings(t, aitypes.ApiTypeOpenAIResponses, "gpt-5-mini")
	overlayModel := "gpt-5-mini"
	overlay.Chat.Engine = &overlayModel
	overlay.Chat.Temperature = nil
	reasoning := "medium"
	overlay.Inference = &engine.InferenceConfig{}
	overlay.Inference.ReasoningEffort = &reasoning

	merged, err := MergeInferenceSettings(base, overlay)
	if err != nil {
		t.Fatalf("MergeInferenceSettings failed: %v", err)
	}
	if merged == nil || merged.Chat == nil || merged.Chat.Engine == nil {
		t.Fatal("expected merged settings with chat engine")
	}
	if got := *merged.Chat.Engine; got != "gpt-5-mini" {
		t.Fatalf("expected overlay engine, got %q", got)
	}
	if merged.Chat.Temperature == nil || *merged.Chat.Temperature != 0.2 {
		t.Fatalf("expected base temperature to remain, got %#v", merged.Chat.Temperature)
	}
	if merged.Inference == nil || merged.Inference.ReasoningEffort == nil || *merged.Inference.ReasoningEffort != "medium" {
		t.Fatalf("expected overlay reasoning effort, got %#v", merged.Inference)
	}
}

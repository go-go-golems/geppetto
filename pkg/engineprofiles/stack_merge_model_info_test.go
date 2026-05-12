package engineprofiles

import (
	"context"
	"testing"

	aistepssettings "github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	aitypes "github.com/go-go-golems/geppetto/pkg/steps/ai/types"
)

func TestDecodeEngineProfileYAML_ModelInfo(t *testing.T) {
	data := []byte(`slug: test-provider
profiles:
  default:
    slug: default
    inference_settings:
      chat:
        api_type: openai
        engine: gpt-4o-mini
      model_info:
        id: gpt-4o-mini
        name: GPT-4o Mini
        reasoning: false
        input:
          - text
          - image
        context_window: 128000
        quality_high_watermark: 64000
        max_output_tokens: 16384
        cost:
          input: 0.15
          output: 0.60
          cache_read: 0.075
          cache_write: 0.30
        metadata:
          fine_tunable: true
`)
	reg, err := DecodeEngineProfileYAMLSingleRegistry(data)
	if err != nil {
		t.Fatalf("DecodeEngineProfileYAMLSingleRegistry: %v", err)
	}
	profile := reg.Profiles[MustEngineProfileSlug("default")]
	info := profile.InferenceSettings.ModelInfo
	if info == nil {
		t.Fatal("expected model info")
	}
	if got := *info.ID; got != "gpt-4o-mini" {
		t.Fatalf("id = %q", got)
	}
	if got := info.EffectiveContextLimit(); got != 64000 {
		t.Fatalf("effective context = %d", got)
	}
	if got := info.Cost.CacheWrite; got != 0.30 {
		t.Fatalf("cache write = %f", got)
	}
}

func TestMergeEngineProfileStackLayers_ModelInfo(t *testing.T) {
	baseSS := mustTestInferenceSettings(t, aitypes.ApiTypeOpenAI, "gpt-4o-mini")
	baseSS.ModelInfo = &aistepssettings.ModelInfo{
		ID:                   strPtrEP("gpt-4o-mini"),
		Name:                 strPtrEP("GPT-4o Mini"),
		Reasoning:            boolPtrEP(false),
		Input:                []aistepssettings.InputModality{aistepssettings.InputModalityText},
		ContextWindow:        intPtrEP(128000),
		QualityHighWatermark: intPtrEP(128000),
		MaxOutputTokens:      intPtrEP(16384),
		Cost:                 &aistepssettings.ModelCost{Input: 0.15, Output: 0.60, CacheRead: 0.075, CacheWrite: 0.30},
		Metadata: map[string]any{
			"nested": map[string]any{"base": "yes", "override": "base"},
		},
	}

	leafSS := mustTestInferenceSettings(t, aitypes.ApiTypeOpenAI, "gpt-5")
	leafSS.ModelInfo = &aistepssettings.ModelInfo{
		ID:            strPtrEP("gpt-5"),
		Reasoning:     boolPtrEP(true),
		ContextWindow: intPtrEP(1000000),
		Cost:          &aistepssettings.ModelCost{Input: 2.50},
		Metadata: map[string]any{
			"nested": map[string]any{"override": "leaf"},
		},
	}

	merged, err := MergeEngineProfileStackLayers([]EngineProfileStackLayer{
		{EngineProfile: &EngineProfile{Slug: MustEngineProfileSlug("base"), InferenceSettings: baseSS}},
		{EngineProfile: &EngineProfile{Slug: MustEngineProfileSlug("leaf"), InferenceSettings: leafSS}},
	})
	if err != nil {
		t.Fatalf("MergeEngineProfileStackLayers: %v", err)
	}
	info := merged.InferenceSettings.ModelInfo
	if info == nil {
		t.Fatal("expected merged model info")
	}
	if got := *info.ID; got != "gpt-5" {
		t.Fatalf("id = %q", got)
	}
	if got := *info.Name; got != "GPT-4o Mini" {
		t.Fatalf("name fallback = %q", got)
	}
	if got := *info.ContextWindow; got != 1000000 {
		t.Fatalf("context = %d", got)
	}
	if got := info.Cost.Output; got != 0 {
		t.Fatalf("cost should be wholesale replacement, output = %f", got)
	}
	if got := info.Metadata["nested"].(map[string]any)["base"]; got != "yes" {
		t.Fatalf("nested base = %v", got)
	}
	if got := info.Metadata["nested"].(map[string]any)["override"]; got != "leaf" {
		t.Fatalf("nested override = %v", got)
	}
}

func TestResolveEngineProfile_ModelInfo(t *testing.T) {
	apiType := aitypes.ApiTypeOpenAI
	model := "gpt-4o-mini"
	store := NewInMemoryEngineProfileStore()
	mustUpsertRegistry(t, store, &EngineProfileRegistry{
		Slug:                     MustRegistrySlug("default"),
		DefaultEngineProfileSlug: MustEngineProfileSlug("default"),
		Profiles: map[EngineProfileSlug]*EngineProfile{
			MustEngineProfileSlug("default"): {
				Slug: MustEngineProfileSlug("default"),
				InferenceSettings: &aistepssettings.InferenceSettings{
					Chat: &aistepssettings.ChatSettings{ApiType: &apiType, Engine: &model},
					ModelInfo: &aistepssettings.ModelInfo{
						ID:            strPtrEP(model),
						ContextWindow: intPtrEP(128000),
					},
				},
			},
		},
	})
	registry := mustNewStoreRegistry(t, store)
	resolved, err := registry.ResolveEngineProfile(context.Background(), ResolveInput{})
	if err != nil {
		t.Fatalf("ResolveEngineProfile: %v", err)
	}
	if resolved.InferenceSettings.ModelInfo == nil || resolved.InferenceSettings.ModelInfo.ContextWindow == nil {
		t.Fatalf("expected resolved model info: %#v", resolved.InferenceSettings.ModelInfo)
	}
}

func strPtrEP(v string) *string { return &v }
func boolPtrEP(v bool) *bool    { return &v }
func intPtrEP(v int) *int       { return &v }

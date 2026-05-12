package settings

import (
	"math"
	"testing"

	"github.com/go-go-golems/geppetto/pkg/turns"
)

func strPtr(v string) *string { return &v }
func boolPtr(v bool) *bool    { return &v }
func intPtr(v int) *int       { return &v }

func TestModelInfoValidate(t *testing.T) {
	t.Run("quality watermark cannot exceed context", func(t *testing.T) {
		mi := &ModelInfo{ContextWindow: intPtr(100), QualityHighWatermark: intPtr(101)}
		if err := mi.Validate(); err == nil {
			t.Fatal("expected validation error")
		}
	})

	t.Run("cost cannot be negative", func(t *testing.T) {
		mi := &ModelInfo{Cost: &ModelCost{Input: -1}}
		if err := mi.Validate(); err == nil {
			t.Fatal("expected validation error")
		}
	})

	t.Run("valid", func(t *testing.T) {
		mi := &ModelInfo{ContextWindow: intPtr(100), QualityHighWatermark: intPtr(90), Cost: &ModelCost{Input: 1}}
		if err := mi.Validate(); err != nil {
			t.Fatalf("Validate: %v", err)
		}
	})
}

func TestModelInfoContextLimits(t *testing.T) {
	if got := (*ModelInfo)(nil).EffectiveContextLimit(); got != 0 {
		t.Fatalf("nil effective limit: %d", got)
	}
	mi := &ModelInfo{ContextWindow: intPtr(1000)}
	if got := mi.EffectiveContextLimit(); got != 1000 {
		t.Fatalf("effective without high watermark: %d", got)
	}
	if got := mi.HardContextLimit(); got != 1000 {
		t.Fatalf("hard limit: %d", got)
	}
	mi.QualityHighWatermark = intPtr(800)
	if got := mi.EffectiveContextLimit(); got != 800 {
		t.Fatalf("effective with high watermark: %d", got)
	}
}

func TestModelInfoCloneDeepCopies(t *testing.T) {
	mi := &ModelInfo{
		ID:            strPtr("model-a"),
		Reasoning:     boolPtr(true),
		Input:         []InputModality{InputModalityText},
		ContextWindow: intPtr(1000),
		Cost:          &ModelCost{Input: 1, Output: 2},
		Metadata: map[string]any{
			"nested": map[string]any{"a": "b"},
		},
	}
	clone := mi.Clone()
	*clone.ID = "model-b"
	clone.Input[0] = InputModalityImage
	clone.Cost.Input = 10
	clone.Metadata["nested"].(map[string]any)["a"] = "changed"

	if *mi.ID != "model-a" || mi.Input[0] != InputModalityText || mi.Cost.Input != 1 {
		t.Fatalf("clone mutated source: %#v", mi)
	}
	if got := mi.Metadata["nested"].(map[string]any)["a"]; got != "b" {
		t.Fatalf("metadata clone mutated source: %v", got)
	}
}

func TestMergeModelInfo(t *testing.T) {
	base := &ModelInfo{
		ID:                   strPtr("base"),
		Name:                 strPtr("Base"),
		Reasoning:            boolPtr(false),
		Input:                []InputModality{InputModalityText},
		ContextWindow:        intPtr(1000),
		QualityHighWatermark: intPtr(900),
		MaxOutputTokens:      intPtr(100),
		Cost:                 &ModelCost{Input: 1, Output: 2, CacheRead: 3, CacheWrite: 4},
		Metadata: map[string]any{
			"a":      "base",
			"nested": map[string]any{"x": "base", "keep": "yes"},
		},
	}
	overlay := &ModelInfo{
		ID:            strPtr("overlay"),
		Input:         []InputModality{InputModalityImage},
		ContextWindow: intPtr(2000),
		Cost:          &ModelCost{Input: 10},
		Metadata: map[string]any{
			"nested": map[string]any{"x": "overlay"},
			"b":      "overlay",
		},
	}
	merged := MergeModelInfo(base, overlay)
	if *merged.ID != "overlay" {
		t.Fatalf("ID = %q", *merged.ID)
	}
	if *merged.Name != "Base" {
		t.Fatalf("Name fallback = %q", *merged.Name)
	}
	if merged.Input[0] != InputModalityImage {
		t.Fatalf("Input = %#v", merged.Input)
	}
	if merged.Cost.Output != 0 || merged.Cost.Input != 10 {
		t.Fatalf("cost should be replaced wholesale: %#v", merged.Cost)
	}
	if got := merged.Metadata["nested"].(map[string]any)["keep"]; got != "yes" {
		t.Fatalf("nested keep = %v", got)
	}
	if got := merged.Metadata["nested"].(map[string]any)["x"]; got != "overlay" {
		t.Fatalf("nested x = %v", got)
	}
}

func TestModelInfoComputeCost(t *testing.T) {
	mi := &ModelInfo{Cost: &ModelCost{Input: 2, Output: 10, CacheRead: 1, CacheWrite: 4}}
	usage := &turns.InferenceUsage{
		InputTokens:              1000,
		OutputTokens:             2000,
		CacheReadInputTokens:     3000,
		CacheCreationInputTokens: 4000,
	}
	cost := mi.ComputeCost(usage)
	if cost == nil {
		t.Fatal("expected cost")
	}
	want := 2*1000.0/1_000_000 + 10*2000.0/1_000_000 + 1*3000.0/1_000_000 + 4*4000.0/1_000_000
	if math.Abs(*cost-want) > 1e-12 {
		t.Fatalf("cost = %f, want %f", *cost, want)
	}
	if got := (*ModelInfo)(nil).ComputeCost(usage); got != nil {
		t.Fatalf("nil info cost = %#v", got)
	}
}

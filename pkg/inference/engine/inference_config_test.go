package engine

import (
	"testing"
)

func intPtr(v int) *int             { return &v }
func float64Ptr(v float64) *float64 { return &v }
func strPtr(v string) *string       { return &v }

// --- MergeInferenceConfig ---

func TestMergeInferenceConfig_BothNil(t *testing.T) {
	if got := MergeInferenceConfig(nil, nil); got != nil {
		t.Errorf("expected nil, got %+v", got)
	}
}

func TestMergeInferenceConfig_TurnNil(t *testing.T) {
	def := &InferenceConfig{Temperature: float64Ptr(0.7)}
	got := MergeInferenceConfig(nil, def)
	if got != def {
		t.Errorf("expected pointer to default, got different pointer")
	}
}

func TestMergeInferenceConfig_DefaultNil(t *testing.T) {
	turn := &InferenceConfig{TopP: float64Ptr(0.9)}
	got := MergeInferenceConfig(turn, nil)
	if got != turn {
		t.Errorf("expected pointer to turn config, got different pointer")
	}
}

func TestMergeInferenceConfig_TurnFieldsOverrideDefaults(t *testing.T) {
	def := &InferenceConfig{
		Temperature:       float64Ptr(0.7),
		TopP:              float64Ptr(0.9),
		MaxResponseTokens: intPtr(1024),
		ThinkingBudget:    intPtr(4096),
		Stop:              []string{"<END>"},
		Seed:              intPtr(42),
	}
	turn := &InferenceConfig{
		Temperature: float64Ptr(0.3),
		Stop:        []string{"STOP"},
	}
	got := MergeInferenceConfig(turn, def)

	// Turn overrides
	if *got.Temperature != 0.3 {
		t.Errorf("Temperature: want 0.3, got %v", *got.Temperature)
	}
	if len(got.Stop) != 1 || got.Stop[0] != "STOP" {
		t.Errorf("Stop: want [STOP], got %v", got.Stop)
	}
	// Default fallbacks
	if *got.TopP != 0.9 {
		t.Errorf("TopP: want 0.9 (default), got %v", *got.TopP)
	}
	if *got.MaxResponseTokens != 1024 {
		t.Errorf("MaxResponseTokens: want 1024 (default), got %v", *got.MaxResponseTokens)
	}
	if *got.ThinkingBudget != 4096 {
		t.Errorf("ThinkingBudget: want 4096 (default), got %v", *got.ThinkingBudget)
	}
	if *got.Seed != 42 {
		t.Errorf("Seed: want 42 (default), got %v", *got.Seed)
	}
}

func TestMergeInferenceConfig_AllTurnFieldsSet(t *testing.T) {
	def := &InferenceConfig{
		Temperature:       float64Ptr(0.7),
		TopP:              float64Ptr(0.9),
		MaxResponseTokens: intPtr(1024),
	}
	turn := &InferenceConfig{
		ThinkingBudget:    intPtr(8192),
		ReasoningEffort:   strPtr("high"),
		ReasoningSummary:  strPtr("concise"),
		Temperature:       float64Ptr(1.0),
		TopP:              float64Ptr(0.5),
		MaxResponseTokens: intPtr(2048),
		Stop:              []string{"<END>"},
		Seed:              intPtr(99),
	}
	got := MergeInferenceConfig(turn, def)

	if *got.ThinkingBudget != 8192 {
		t.Errorf("ThinkingBudget: want 8192, got %v", *got.ThinkingBudget)
	}
	if *got.ReasoningEffort != "high" {
		t.Errorf("ReasoningEffort: want high, got %v", *got.ReasoningEffort)
	}
	if *got.ReasoningSummary != "concise" {
		t.Errorf("ReasoningSummary: want concise, got %v", *got.ReasoningSummary)
	}
	if *got.Temperature != 1.0 {
		t.Errorf("Temperature: want 1.0, got %v", *got.Temperature)
	}
	if *got.TopP != 0.5 {
		t.Errorf("TopP: want 0.5, got %v", *got.TopP)
	}
	if *got.MaxResponseTokens != 2048 {
		t.Errorf("MaxResponseTokens: want 2048, got %v", *got.MaxResponseTokens)
	}
	if *got.Seed != 99 {
		t.Errorf("Seed: want 99, got %v", *got.Seed)
	}
}

func TestMergeInferenceConfig_DoesNotMutateInputs(t *testing.T) {
	def := &InferenceConfig{Temperature: float64Ptr(0.7), TopP: float64Ptr(0.9)}
	turn := &InferenceConfig{Temperature: float64Ptr(0.3)}
	got := MergeInferenceConfig(turn, def)

	// Mutate the result and verify originals are untouched
	*got.Temperature = 999.0
	if *def.Temperature != 0.7 {
		t.Errorf("def.Temperature mutated: want 0.7, got %v", *def.Temperature)
	}
	if *turn.Temperature != 0.3 {
		t.Errorf("turn.Temperature mutated: want 0.3, got %v", *turn.Temperature)
	}
}

func TestMergeInferenceConfig_EmptyStopClearsDefault(t *testing.T) {
	def := &InferenceConfig{
		Stop: []string{"<END>", "STOP"},
	}
	turn := &InferenceConfig{
		Stop: []string{}, // explicitly empty — should clear inherited stops
	}
	got := MergeInferenceConfig(turn, def)

	if got.Stop == nil {
		t.Fatal("Stop should be non-nil empty slice, got nil")
	}
	if len(got.Stop) != 0 {
		t.Errorf("Stop should be empty, got %v", got.Stop)
	}
}

func TestMergeInferenceConfig_NilStopPreservesDefault(t *testing.T) {
	def := &InferenceConfig{
		Stop: []string{"<END>"},
	}
	turn := &InferenceConfig{
		Temperature: float64Ptr(0.5),
		// Stop is nil — should preserve default
	}
	got := MergeInferenceConfig(turn, def)

	if len(got.Stop) != 1 || got.Stop[0] != "<END>" {
		t.Errorf("Stop should be preserved from default, got %v", got.Stop)
	}
}

// --- SanitizeForReasoningModel ---

func TestSanitizeForReasoningModel_Nil(t *testing.T) {
	if got := SanitizeForReasoningModel(nil); got != nil {
		t.Errorf("expected nil, got %+v", got)
	}
}

func TestSanitizeForReasoningModel_ClearsSamplingFields(t *testing.T) {
	cfg := &InferenceConfig{
		Temperature:       float64Ptr(0.7),
		TopP:              float64Ptr(0.9),
		MaxResponseTokens: intPtr(1024),
		ThinkingBudget:    intPtr(4096),
		Seed:              intPtr(42),
	}
	got := SanitizeForReasoningModel(cfg)

	if got.Temperature != nil {
		t.Errorf("Temperature should be nil, got %v", *got.Temperature)
	}
	if got.TopP != nil {
		t.Errorf("TopP should be nil, got %v", *got.TopP)
	}
	// Non-sampling fields preserved
	if *got.MaxResponseTokens != 1024 {
		t.Errorf("MaxResponseTokens should be preserved: want 1024, got %v", *got.MaxResponseTokens)
	}
	if *got.ThinkingBudget != 4096 {
		t.Errorf("ThinkingBudget should be preserved: want 4096, got %v", *got.ThinkingBudget)
	}
	if *got.Seed != 42 {
		t.Errorf("Seed should be preserved: want 42, got %v", *got.Seed)
	}
	// Original untouched
	if cfg.Temperature == nil {
		t.Error("original Temperature was mutated to nil")
	}
}

// --- SanitizeOpenAIForReasoningModel ---

func TestSanitizeOpenAIForReasoningModel_Nil(t *testing.T) {
	if got := SanitizeOpenAIForReasoningModel(nil); got != nil {
		t.Errorf("expected nil, got %+v", got)
	}
}

func TestSanitizeOpenAIForReasoningModel_ClearsFields(t *testing.T) {
	cfg := &OpenAIInferenceConfig{
		N:                intPtr(3),
		PresencePenalty:  float64Ptr(0.5),
		FrequencyPenalty: float64Ptr(0.3),
		Store:            boolPtr(true),
		ServiceTier:      strPtr("default"),
	}
	got := SanitizeOpenAIForReasoningModel(cfg)

	if got.N != nil {
		t.Errorf("N should be nil, got %v", *got.N)
	}
	if got.PresencePenalty != nil {
		t.Errorf("PresencePenalty should be nil, got %v", *got.PresencePenalty)
	}
	if got.FrequencyPenalty != nil {
		t.Errorf("FrequencyPenalty should be nil, got %v", *got.FrequencyPenalty)
	}
	// Non-affected fields preserved
	if got.Store == nil || !*got.Store {
		t.Error("Store should be preserved as true")
	}
	if got.ServiceTier == nil || *got.ServiceTier != "default" {
		t.Error("ServiceTier should be preserved as 'default'")
	}
	// Original untouched
	if cfg.PresencePenalty == nil {
		t.Error("original PresencePenalty was mutated to nil")
	}
}

func boolPtr(v bool) *bool { return &v }

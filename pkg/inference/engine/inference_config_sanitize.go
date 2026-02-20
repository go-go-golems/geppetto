package engine

// SanitizeForReasoningModel returns a copy of cfg with sampling fields cleared.
// Reasoning models (e.g., o1/o3/o4/gpt-5) reject temperature and top_p.
// The caller is responsible for determining whether the model is a reasoning model.
func SanitizeForReasoningModel(cfg *InferenceConfig) *InferenceConfig {
	if cfg == nil {
		return nil
	}
	sanitized := *cfg
	sanitized.Temperature = nil
	sanitized.TopP = nil
	return &sanitized
}

// SanitizeOpenAIForReasoningModel returns a copy of cfg with fields cleared
// that reasoning models reject: PresencePenalty, FrequencyPenalty, and N.
// The caller is responsible for determining whether the model is a reasoning model.
func SanitizeOpenAIForReasoningModel(cfg *OpenAIInferenceConfig) *OpenAIInferenceConfig {
	if cfg == nil {
		return nil
	}
	sanitized := *cfg
	sanitized.PresencePenalty = nil
	sanitized.FrequencyPenalty = nil
	sanitized.N = nil
	return &sanitized
}

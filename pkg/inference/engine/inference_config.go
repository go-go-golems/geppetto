package engine

import "github.com/go-go-golems/geppetto/pkg/turns"

// InferenceConfig provides per-turn overrides for inference parameters.
// When set on Turn.Data via KeyInferenceConfig, these take precedence
// over StepSettings defaults.
//
// Can also be set on StepSettings.Inference as engine-level defaults.
// Resolution order: Turn.Data > StepSettings.Inference > StepSettings.Chat fields.
//
// Fields use pointer types so that nil means "not set, use default".
// This follows the pattern established by ChatSettings.
type InferenceConfig struct {
	// ThinkingBudget sets the token budget for model reasoning/thinking.
	// Maps to Claude thinking.budget_tokens, OpenAI Responses reasoning.max_tokens.
	ThinkingBudget *int `json:"thinking_budget,omitempty"`

	// ReasoningEffort controls reasoning depth: "low", "medium", "high".
	// Maps to OpenAI Responses reasoning.effort.
	ReasoningEffort *string `json:"reasoning_effort,omitempty"`

	// ReasoningSummary controls reasoning summary generation.
	// Maps to OpenAI Responses reasoning.summary ("auto", "concise", "detailed").
	ReasoningSummary *string `json:"reasoning_summary,omitempty"`

	// Temperature overrides the sampling temperature for this turn.
	Temperature *float64 `json:"temperature,omitempty"`

	// TopP overrides the top-p (nucleus) sampling for this turn.
	TopP *float64 `json:"top_p,omitempty"`

	// MaxResponseTokens overrides the max output tokens for this turn.
	MaxResponseTokens *int `json:"max_response_tokens,omitempty"`

	// Stop overrides stop sequences for this turn.
	Stop []string `json:"stop,omitempty"`

	// Seed for reproducibility (OpenAI Chat Completions).
	Seed *int `json:"seed,omitempty"`
}

// ClaudeInferenceConfig holds Claude-specific per-turn overrides.
// Set on Turn.Data via KeyClaudeInferenceConfig.
type ClaudeInferenceConfig struct {
	// UserID for Claude metadata tracking.
	UserID *string `json:"user_id,omitempty"`

	// TopK sampling parameter specific to Claude.
	TopK *int `json:"top_k,omitempty"`
}

// OpenAIInferenceConfig holds OpenAI-specific per-turn overrides.
// Set on Turn.Data via KeyOpenAIInferenceConfig.
type OpenAIInferenceConfig struct {
	// N number of choices to generate.
	N *int `json:"n,omitempty"`

	// PresencePenalty override.
	PresencePenalty *float64 `json:"presence_penalty,omitempty"`

	// FrequencyPenalty override.
	FrequencyPenalty *float64 `json:"frequency_penalty,omitempty"`

	// Store whether to store the response (Responses API).
	Store *bool `json:"store,omitempty"`

	// ServiceTier rate limit hint (Responses API).
	ServiceTier *string `json:"service_tier,omitempty"`
}

// ResolveInferenceConfig returns the effective InferenceConfig by checking
// Turn.Data first, then falling back to the engine-level default.
// Returns nil if neither source has a config.
func ResolveInferenceConfig(t *turns.Turn, engineDefault *InferenceConfig) *InferenceConfig {
	if t != nil {
		if cfg, ok, err := KeyInferenceConfig.Get(t.Data); err == nil && ok {
			return &cfg
		}
	}
	return engineDefault
}

// ResolveClaudeInferenceConfig returns the per-turn Claude config from Turn.Data, or nil.
func ResolveClaudeInferenceConfig(t *turns.Turn) *ClaudeInferenceConfig {
	if t != nil {
		if cfg, ok, err := KeyClaudeInferenceConfig.Get(t.Data); err == nil && ok {
			return &cfg
		}
	}
	return nil
}

// ResolveOpenAIInferenceConfig returns the per-turn OpenAI config from Turn.Data, or nil.
func ResolveOpenAIInferenceConfig(t *turns.Turn) *OpenAIInferenceConfig {
	if t != nil {
		if cfg, ok, err := KeyOpenAIInferenceConfig.Get(t.Data); err == nil && ok {
			return &cfg
		}
	}
	return nil
}

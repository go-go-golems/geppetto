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
	ThinkingBudget *int `json:"thinking_budget,omitempty" yaml:"thinking_budget,omitempty" glazed:"inference-thinking-budget"`

	// ReasoningEffort controls reasoning depth: "low", "medium", "high".
	// Maps to OpenAI Responses reasoning.effort.
	ReasoningEffort *string `json:"reasoning_effort,omitempty" yaml:"reasoning_effort,omitempty" glazed:"inference-reasoning-effort"`

	// ReasoningSummary controls reasoning summary generation.
	// Maps to OpenAI Responses reasoning.summary ("auto", "concise", "detailed").
	ReasoningSummary *string `json:"reasoning_summary,omitempty" yaml:"reasoning_summary,omitempty" glazed:"inference-reasoning-summary"`

	// Temperature overrides the sampling temperature for this turn.
	Temperature *float64 `json:"temperature,omitempty" yaml:"temperature,omitempty" glazed:"inference-temperature"`

	// TopP overrides the top-p (nucleus) sampling for this turn.
	TopP *float64 `json:"top_p,omitempty" yaml:"top_p,omitempty" glazed:"inference-top-p"`

	// MaxResponseTokens overrides the max output tokens for this turn.
	MaxResponseTokens *int `json:"max_response_tokens,omitempty" yaml:"max_response_tokens,omitempty" glazed:"inference-max-response-tokens"`

	// Stop overrides stop sequences for this turn.
	Stop []string `json:"stop,omitempty" yaml:"stop,omitempty" glazed:"inference-stop"`

	// Seed for reproducibility (OpenAI Chat Completions).
	Seed *int `json:"seed,omitempty" yaml:"seed,omitempty" glazed:"inference-seed"`
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

// MergeInferenceConfig returns a new InferenceConfig where turnCfg fields
// take precedence over engineDefault fields. Nil fields in turnCfg fall
// back to the corresponding engineDefault field.
func MergeInferenceConfig(turnCfg, engineDefault *InferenceConfig) *InferenceConfig {
	if turnCfg == nil {
		return engineDefault
	}
	if engineDefault == nil {
		return turnCfg
	}
	merged := *engineDefault // shallow copy of defaults
	if turnCfg.ThinkingBudget != nil {
		v := *turnCfg.ThinkingBudget
		merged.ThinkingBudget = &v
	}
	if turnCfg.ReasoningEffort != nil {
		v := *turnCfg.ReasoningEffort
		merged.ReasoningEffort = &v
	}
	if turnCfg.ReasoningSummary != nil {
		v := *turnCfg.ReasoningSummary
		merged.ReasoningSummary = &v
	}
	if turnCfg.Temperature != nil {
		v := *turnCfg.Temperature
		merged.Temperature = &v
	}
	if turnCfg.TopP != nil {
		v := *turnCfg.TopP
		merged.TopP = &v
	}
	if turnCfg.MaxResponseTokens != nil {
		v := *turnCfg.MaxResponseTokens
		merged.MaxResponseTokens = &v
	}
	if len(turnCfg.Stop) > 0 {
		merged.Stop = append([]string(nil), turnCfg.Stop...)
	}
	if turnCfg.Seed != nil {
		v := *turnCfg.Seed
		merged.Seed = &v
	}
	return &merged
}

// ResolveInferenceConfig returns the effective InferenceConfig by merging
// Turn.Data config with the engine-level default. Turn-level fields take
// precedence; nil fields fall back to the engine default.
// Returns nil if neither source has a config.
func ResolveInferenceConfig(t *turns.Turn, engineDefault *InferenceConfig) *InferenceConfig {
	var turnCfg *InferenceConfig
	if t != nil {
		if cfg, ok, err := KeyInferenceConfig.Get(t.Data); err == nil && ok {
			turnCfg = &cfg
		}
	}
	return MergeInferenceConfig(turnCfg, engineDefault)
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

package engine

import "github.com/go-go-golems/geppetto/pkg/turns"

// KeyToolConfig is a typed key for storing engine.ToolConfig in Turn.Data.
//
// This key lives in the engine package (not in turns) to avoid an import cycle:
// turns -> engine (ToolConfig type) -> turns (Engine interface uses *turns.Turn)
var KeyToolConfig = turns.DataK[ToolConfig](turns.GeppettoNamespaceKey, turns.ToolConfigValueKey, 1)

// KeyStructuredOutputConfig is a typed key for storing engine.StructuredOutputConfig in Turn.Data.
var KeyStructuredOutputConfig = turns.DataK[StructuredOutputConfig](turns.GeppettoNamespaceKey, turns.StructuredOutputConfigValueKey, 1)

// KeyInferenceConfig is a typed key for storing engine.InferenceConfig in Turn.Data.
// When set, per-turn InferenceConfig values override engine-level defaults from StepSettings.Inference.
var KeyInferenceConfig = turns.DataK[InferenceConfig](turns.GeppettoNamespaceKey, turns.InferenceConfigValueKey, 1)

// KeyClaudeInferenceConfig is a typed key for storing Claude-specific per-turn overrides in Turn.Data.
var KeyClaudeInferenceConfig = turns.DataK[ClaudeInferenceConfig](turns.GeppettoNamespaceKey, turns.ClaudeInferenceConfigValueKey, 1)

// KeyOpenAIInferenceConfig is a typed key for storing OpenAI-specific per-turn overrides in Turn.Data.
var KeyOpenAIInferenceConfig = turns.DataK[OpenAIInferenceConfig](turns.GeppettoNamespaceKey, turns.OpenAIInferenceConfigValueKey, 1)

package engine

import "github.com/go-go-golems/geppetto/pkg/turns"

// KeyToolConfig is a typed key for storing engine.ToolConfig in Turn.Data.
//
// This key lives in the engine package (not in turns) to avoid an import cycle:
// turns -> engine (ToolConfig type) -> turns (Engine interface uses *turns.Turn)
var KeyToolConfig = turns.DataK[ToolConfig](turns.GeppettoNamespaceKey, turns.ToolConfigValueKey, 1)

// KeyStructuredOutputConfig is a typed key for storing engine.StructuredOutputConfig in Turn.Data.
var KeyStructuredOutputConfig = turns.DataK[StructuredOutputConfig](turns.GeppettoNamespaceKey, turns.StructuredOutputConfigValueKey, 1)

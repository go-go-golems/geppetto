package builder

import (
	"github.com/go-go-golems/geppetto/pkg/events"
	"github.com/go-go-golems/geppetto/pkg/inference/engine"
)

// EngineConfig is an opaque, comparable fingerprint of a composed engine.
//
// Callers use it to decide when to recompose an engine (e.g., profile/override changes).
// The concrete content is application-specific.
type EngineConfig interface {
	Signature() string
}

// EngineBuilder centralizes engine + sink composition so callers stay lean and
// recomposition happens deterministically.
//
// This is a geppetto-level interface (no lifecycle or ConversationManager injection).
// Applications can implement it with their own profile/override logic.
type EngineBuilder interface {
	Build(convID, profileSlug string, overrides map[string]any) (engine.Engine, events.EventSink, EngineConfig, error)
	BuildConfig(profileSlug string, overrides map[string]any) (EngineConfig, error)
	BuildFromConfig(convID string, config EngineConfig) (engine.Engine, events.EventSink, error)
}

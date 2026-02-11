package session

import (
	"context"

	"github.com/go-go-golems/geppetto/pkg/turns"
)

// EngineBuilder builds an inference runner for a session.
//
// The builder is responsible for wiring sinks, tools, middleware, snapshot hooks,
// persistence, and provider engine construction policy.
type EngineBuilder interface {
	Build(ctx context.Context, sessionID string) (InferenceRunner, error)
}

// InferenceRunner performs a blocking inference step: input Turn -> output Turn.
type InferenceRunner interface {
	RunInference(ctx context.Context, t *turns.Turn) (*turns.Turn, error)
}

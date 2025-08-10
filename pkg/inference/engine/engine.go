package engine

import (
    "context"

    "github.com/go-go-golems/geppetto/pkg/turns"
)

// Engine represents an AI inference engine that processes a Turn and returns an updated Turn.
// All provider-specific engines implement this.
type Engine interface {
    // RunInference processes a Turn and returns the updated Turn.
    // The engine handles provider-specific API calls but does NOT handle tool orchestration.
    // Tool calls in the response should be preserved as blocks for helper processing.
    RunInference(ctx context.Context, t *turns.Turn) (*turns.Turn, error)
}

package session

import (
	"context"
	"errors"

	"github.com/go-go-golems/geppetto/pkg/events"
	"github.com/go-go-golems/geppetto/pkg/inference/engine"
	"github.com/go-go-golems/geppetto/pkg/inference/middleware"
	"github.com/go-go-golems/geppetto/pkg/inference/toolhelpers"
	"github.com/go-go-golems/geppetto/pkg/inference/tools"
	"github.com/go-go-golems/geppetto/pkg/turns"
)

var (
	ErrToolLoopEngineBuilderNil     = errors.New("tool loop engine builder is nil")
	ErrToolLoopEngineBuilderBaseNil = errors.New("tool loop engine builder base engine is nil")
)

// TurnPersister persists a completed turn for a session (run).
//
// Turn identity is derived from the Turn itself (t.ID). Session correlation is
// expected to be present in Turn.Metadata (see turns.KeyTurnMetaSessionID).
type TurnPersister interface {
	PersistTurn(ctx context.Context, t *turns.Turn) error
}

// ToolLoopEngineBuilder builds a runner that:
// - wraps a base engine with middleware
// - injects sinks and snapshot hooks via context
// - runs either a single-pass inference or the tool-calling loop
// - best-effort persists the final turn
//
// This is the standard builder used by chat-style applications.
type ToolLoopEngineBuilder struct {
	// Base is the provider engine implementation (OpenAI/Claude/etc).
	Base engine.Engine

	// Middlewares are applied in-order around Base.
	Middlewares []middleware.Middleware

	// Registry enables tool calling. If nil, the runner performs a single-pass inference.
	Registry tools.ToolRegistry

	// ToolConfig configures tool-loop behavior when Registry is set.
	ToolConfig *toolhelpers.ToolConfig

	// EventSinks are attached to the run context for streaming/logging.
	EventSinks []events.EventSink

	// SnapshotHook is attached to the run context for capturing intermediate turns.
	SnapshotHook toolhelpers.SnapshotHook

	// Persister is invoked on successful completion (when err == nil and an updated turn exists).
	Persister TurnPersister
}

func (b *ToolLoopEngineBuilder) Build(ctx context.Context, sessionID string) (InferenceRunner, error) {
	if b == nil {
		return nil, ErrToolLoopEngineBuilderNil
	}
	if b.Base == nil {
		return nil, ErrToolLoopEngineBuilderBaseNil
	}

	eng := b.Base
	if len(b.Middlewares) > 0 {
		eng = middleware.NewEngineWithMiddleware(eng, b.Middlewares...)
	}

	cfg := toolhelpers.NewToolConfig()
	if b.ToolConfig != nil {
		cfg = *b.ToolConfig
	}

	return &toolLoopRunner{
		sessionID:    sessionID,
		eng:          eng,
		registry:     b.Registry,
		toolConfig:   cfg,
		eventSinks:   b.EventSinks,
		snapshotHook: b.SnapshotHook,
		persister:    b.Persister,
	}, nil
}

type toolLoopRunner struct {
	sessionID string

	eng      engine.Engine
	registry tools.ToolRegistry

	toolConfig toolhelpers.ToolConfig

	eventSinks   []events.EventSink
	snapshotHook toolhelpers.SnapshotHook

	persister TurnPersister
}

func (r *toolLoopRunner) RunInference(ctx context.Context, t *turns.Turn) (*turns.Turn, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	runCtx := ctx
	if len(r.eventSinks) > 0 {
		runCtx = events.WithEventSinks(runCtx, r.eventSinks...)
	}
	if r.snapshotHook != nil {
		runCtx = toolhelpers.WithTurnSnapshotHook(runCtx, r.snapshotHook)
	}

	if t == nil {
		t = &turns.Turn{}
	}
	if r.sessionID != "" {
		if _, ok, err := turns.KeyTurnMetaSessionID.Get(t.Metadata); err != nil || !ok {
			_ = turns.KeyTurnMetaSessionID.Set(&t.Metadata, r.sessionID)
		}
	}

	var (
		updated *turns.Turn
		err     error
	)
	if r.registry == nil {
		updated, err = r.eng.RunInference(runCtx, t)
	} else {
		updated, err = toolhelpers.RunToolCallingLoop(runCtx, r.eng, t, r.registry, r.toolConfig)
	}

	if updated != nil && r.sessionID != "" {
		if _, ok, err := turns.KeyTurnMetaSessionID.Get(updated.Metadata); err != nil || !ok {
			_ = turns.KeyTurnMetaSessionID.Set(&updated.Metadata, r.sessionID)
		}
	}

	if err == nil && r.persister != nil && updated != nil {
		_ = r.persister.PersistTurn(runCtx, updated)
	}

	return updated, err
}

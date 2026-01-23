package toolloop

import (
	"context"
	"errors"
	"time"

	"github.com/go-go-golems/geppetto/pkg/events"
	"github.com/go-go-golems/geppetto/pkg/inference/engine"
	"github.com/go-go-golems/geppetto/pkg/inference/middleware"
	"github.com/go-go-golems/geppetto/pkg/inference/session"
	"github.com/go-go-golems/geppetto/pkg/inference/tools"
	"github.com/go-go-golems/geppetto/pkg/turns"
	"github.com/google/uuid"
)

var (
	ErrEngineBuilderNil     = errors.New("engine builder is nil")
	ErrEngineBuilderBaseNil = errors.New("engine builder base engine is nil")
)

// TurnPersister persists a completed turn for a session (run).
//
// Turn identity is derived from the Turn itself (t.ID). Session correlation is
// expected to be present in Turn.Metadata (see turns.KeyTurnMetaSessionID).
type TurnPersister interface {
	PersistTurn(ctx context.Context, t *turns.Turn) error
}

// EngineBuilder builds a runner that:
// - wraps a base engine with middleware
// - injects sinks and snapshot hooks via context
// - runs either a single-pass inference or the tool-calling loop
// - best-effort persists the final turn
//
// This is the standard builder used by chat-style applications.
type EngineBuilder struct {
	// Base is the provider engine implementation (OpenAI/Claude/etc).
	Base engine.Engine

	// Middlewares are applied in-order around Base.
	Middlewares []middleware.Middleware

	// Registry enables tool calling. If nil, the runner performs a single-pass inference.
	Registry tools.ToolRegistry

	// ToolConfig configures tool-loop behavior when Registry is set.
	ToolConfig *ToolConfig

	// EventSinks are attached to the run context for streaming/logging.
	EventSinks []events.EventSink

	// SnapshotHook is attached to the run context for capturing intermediate turns.
	SnapshotHook SnapshotHook

	// StepController enables step-mode pauses in the tool loop when non-nil.
	// It is expected to be owned by the application/web layer (not by session).
	StepController *StepController

	// StepPauseTimeout is the duration to wait at each pause before auto-continuing.
	// If zero, the loop default is used.
	StepPauseTimeout time.Duration

	// Persister is invoked on successful completion (when err == nil and an updated turn exists).
	Persister TurnPersister
}

var _ session.EngineBuilder = (*EngineBuilder)(nil)

func (b *EngineBuilder) Build(ctx context.Context, sessionID string) (session.InferenceRunner, error) {
	if b == nil {
		return nil, ErrEngineBuilderNil
	}
	if b.Base == nil {
		return nil, ErrEngineBuilderBaseNil
	}

	eng := b.Base
	eng = newEngineWithMiddlewares(eng, b.Middlewares)

	cfg := NewToolConfig()
	if b.ToolConfig != nil {
		cfg = *b.ToolConfig
	}

	return &runner{
		sessionID:        sessionID,
		eng:              eng,
		registry:         b.Registry,
		toolConfig:       cfg,
		eventSinks:       b.EventSinks,
		snapshotHook:     b.SnapshotHook,
		stepController:   b.StepController,
		stepPauseTimeout: b.StepPauseTimeout,
		persister:        b.Persister,
	}, nil
}

type runner struct {
	sessionID string

	eng      engine.Engine
	registry tools.ToolRegistry

	toolConfig ToolConfig

	eventSinks   []events.EventSink
	snapshotHook SnapshotHook

	stepController   *StepController
	stepPauseTimeout time.Duration

	persister TurnPersister
}

var _ session.InferenceRunner = (*runner)(nil)

type engineWithMiddlewares struct {
	handler middleware.HandlerFunc
}

func newEngineWithMiddlewares(eng engine.Engine, mws []middleware.Middleware) engine.Engine {
	if len(mws) == 0 {
		return eng
	}

	handler := func(ctx context.Context, t *turns.Turn) (*turns.Turn, error) {
		return eng.RunInference(ctx, t)
	}

	return &engineWithMiddlewares{
		handler: middleware.Chain(handler, mws...),
	}
}

func (e *engineWithMiddlewares) RunInference(ctx context.Context, t *turns.Turn) (*turns.Turn, error) {
	return e.handler(ctx, t)
}

func (r *runner) RunInference(ctx context.Context, t *turns.Turn) (*turns.Turn, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	runCtx := ctx
	if len(r.eventSinks) > 0 {
		runCtx = events.WithEventSinks(runCtx, r.eventSinks...)
	}
	if r.snapshotHook != nil {
		runCtx = WithTurnSnapshotHook(runCtx, r.snapshotHook)
	}

	if t == nil {
		t = &turns.Turn{}
	}
	if t.ID == "" {
		t.ID = uuid.NewString()
	}
	if r.sessionID != "" {
		if _, ok, err := turns.KeyTurnMetaSessionID.Get(t.Metadata); err != nil || !ok {
			_ = turns.KeyTurnMetaSessionID.Set(&t.Metadata, r.sessionID)
		}
	}
	if _, ok, err := turns.KeyTurnMetaInferenceID.Get(t.Metadata); err != nil || !ok {
		_ = turns.KeyTurnMetaInferenceID.Set(&t.Metadata, uuid.NewString())
	}

	var (
		updated *turns.Turn
		err     error
	)
	if r.registry == nil {
		updated, err = r.eng.RunInference(runCtx, t)
	} else {
		opts := []Option{
			WithEngine(r.eng),
			WithRegistry(r.registry),
			WithConfig(r.toolConfig),
			WithStepController(r.stepController),
		}
		if r.stepPauseTimeout > 0 {
			opts = append(opts, WithPauseTimeout(r.stepPauseTimeout))
		}
		loop := New(opts...)
		updated, err = loop.RunLoop(runCtx, t)
	}

	if updated != nil && r.sessionID != "" {
		if _, ok, err := turns.KeyTurnMetaSessionID.Get(updated.Metadata); err != nil || !ok {
			_ = turns.KeyTurnMetaSessionID.Set(&updated.Metadata, r.sessionID)
		}
	}
	if updated != nil {
		if updated.ID == "" {
			updated.ID = t.ID
		}
		if iid, ok, err := turns.KeyTurnMetaInferenceID.Get(t.Metadata); err == nil && ok && iid != "" {
			if _, ok2, err2 := turns.KeyTurnMetaInferenceID.Get(updated.Metadata); err2 != nil || !ok2 {
				_ = turns.KeyTurnMetaInferenceID.Set(&updated.Metadata, iid)
			}
		}
	}

	if err == nil && r.persister != nil && updated != nil {
		_ = r.persister.PersistTurn(runCtx, updated)
	}

	return updated, err
}

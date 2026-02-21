package enginebuilder

import (
	"context"
	"errors"
	"time"

	"github.com/go-go-golems/geppetto/pkg/events"
	"github.com/go-go-golems/geppetto/pkg/inference/engine"
	"github.com/go-go-golems/geppetto/pkg/inference/middleware"
	"github.com/go-go-golems/geppetto/pkg/inference/session"
	"github.com/go-go-golems/geppetto/pkg/inference/toolloop"
	"github.com/go-go-golems/geppetto/pkg/inference/tools"
	"github.com/go-go-golems/geppetto/pkg/turns"
	"github.com/google/uuid"
)

var (
	ErrBuilderNil     = errors.New("engine builder is nil")
	ErrBuilderBaseNil = errors.New("engine builder base engine is nil")
)

// TurnPersister persists a completed turn for a session (run).
//
// Turn identity is derived from the Turn itself (t.ID). Session correlation is
// expected to be present in Turn.Metadata (see turns.KeyTurnMetaSessionID).
type TurnPersister interface {
	PersistTurn(ctx context.Context, t *turns.Turn) error
}

// Builder builds a runner that:
// - wraps a base engine with middleware
// - injects sinks and snapshot hooks via context
// - runs either a single-pass inference or the tool-calling loop
// - best-effort persists the final turn
//
// This is the standard builder used by chat-style applications.
type Builder struct {
	// Base is the provider engine implementation (OpenAI/Claude/etc).
	Base engine.Engine

	// Middlewares are applied in-order around Base.
	Middlewares []middleware.Middleware

	// Registry enables tool calling. If nil, the runner performs a single-pass inference.
	Registry tools.ToolRegistry

	// LoopConfig configures loop orchestration (e.g. MaxIterations).
	LoopConfig *toolloop.LoopConfig

	// ToolConfig configures tool advertisement and execution policy.
	ToolConfig *tools.ToolConfig

	// ToolExecutor allows overriding tool execution behavior (hooks, policies).
	// If nil, the default executor is used.
	ToolExecutor tools.ToolExecutor

	// EventSinks are attached to the run context for streaming/logging.
	EventSinks []events.EventSink

	// SnapshotHook is attached to the run context for capturing intermediate turns.
	SnapshotHook toolloop.SnapshotHook

	// StepController enables step-mode pauses in the tool loop when non-nil.
	// It is expected to be owned by the application/web layer (not by session).
	StepController *toolloop.StepController

	// StepPauseTimeout is the duration to wait at each pause before auto-continuing.
	// If zero, the loop default is used.
	StepPauseTimeout time.Duration

	// Persister is invoked on successful completion (when err == nil and an updated turn exists).
	Persister TurnPersister
}

var _ session.EngineBuilder = (*Builder)(nil)

func (b *Builder) Build(ctx context.Context, sessionID string) (session.InferenceRunner, error) {
	if b == nil {
		return nil, ErrBuilderNil
	}
	if b.Base == nil {
		return nil, ErrBuilderBaseNil
	}

	eng := b.Base
	eng = newEngineWithMiddlewares(eng, b.Middlewares)

	loopCfg := toolloop.NewLoopConfig()
	if b.LoopConfig != nil {
		loopCfg = *b.LoopConfig
	}

	toolCfg := tools.DefaultToolConfig()
	if b.ToolConfig != nil {
		toolCfg = *b.ToolConfig
	}

	return &runner{
		sessionID:        sessionID,
		eng:              eng,
		registry:         b.Registry,
		loopCfg:          loopCfg,
		toolCfg:          toolCfg,
		toolExecutor:     b.ToolExecutor,
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

	loopCfg toolloop.LoopConfig
	toolCfg tools.ToolConfig

	toolExecutor tools.ToolExecutor

	eventSinks   []events.EventSink
	snapshotHook toolloop.SnapshotHook

	stepController   *toolloop.StepController
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
		runCtx = toolloop.WithTurnSnapshotHook(runCtx, r.snapshotHook)
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
		opts := []toolloop.Option{
			toolloop.WithEngine(r.eng),
			toolloop.WithRegistry(r.registry),
			toolloop.WithLoopConfig(r.loopCfg),
			toolloop.WithToolConfig(r.toolCfg),
			toolloop.WithStepController(r.stepController),
		}
		if r.toolExecutor != nil {
			opts = append(opts, toolloop.WithExecutor(r.toolExecutor))
		}
		if r.stepPauseTimeout > 0 {
			opts = append(opts, toolloop.WithPauseTimeout(r.stepPauseTimeout))
		}
		loop := toolloop.New(opts...)
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

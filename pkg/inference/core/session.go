package core

import (
	"context"
	"errors"

	"github.com/go-go-golems/geppetto/pkg/events"
	"github.com/go-go-golems/geppetto/pkg/inference/state"
	"github.com/go-go-golems/geppetto/pkg/inference/toolhelpers"
	geptools "github.com/go-go-golems/geppetto/pkg/inference/tools"
	"github.com/go-go-golems/geppetto/pkg/turns"
)

// Runner is the minimal inference interface UIs should depend on.
type Runner interface {
	RunInference(ctx context.Context, seed *turns.Turn) (*turns.Turn, error)
}

// TurnPersister persists a completed turn for a run.
//
// The persister receives the run identifier explicitly. Turn identity is derived
// from the Turn itself (t.ID). runID may also be present in t.RunID.
type TurnPersister interface {
	PersistTurn(ctx context.Context, runID string, t *turns.Turn) error
}

// Session captures stable inference dependencies (state, tool registry, config,
// event sinks) and exposes a minimal RunInference method.
type Session struct {
	State *state.InferenceState

	// Registry enables tool calling. If nil, the Session executes a single-pass inference.
	Registry geptools.ToolRegistry

	// ToolConfig configures tool-loop behavior when Registry is set.
	ToolConfig *toolhelpers.ToolConfig

	// EventSinks are attached to the run context for streaming/logging.
	EventSinks []events.EventSink

	// SnapshotHook is attached to the run context for snapshotting intermediate turns.
	SnapshotHook toolhelpers.SnapshotHook

	// Persister is invoked on successful completion (when err == nil and an updated turn exists).
	Persister TurnPersister
}

// RunInference executes one inference run. It is safe to call from a goroutine;
// cancellation is exposed via s.State.CancelRun().
func (s *Session) RunInference(ctx context.Context, seed *turns.Turn) (*turns.Turn, error) {
	if s == nil || s.State == nil {
		return nil, errors.New("session/state is nil")
	}
	if s.State.Eng == nil {
		return nil, errors.New("engine is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	if err := s.State.StartRun(); err != nil {
		return nil, err
	}
	runCtx, cancel := context.WithCancel(ctx)
	s.State.SetCancel(cancel)
	defer func() {
		cancel()
		s.State.FinishRun()
	}()

	if len(s.EventSinks) > 0 {
		runCtx = events.WithEventSinks(runCtx, s.EventSinks...)
	}
	if s.SnapshotHook != nil {
		runCtx = toolhelpers.WithTurnSnapshotHook(runCtx, s.SnapshotHook)
	}

	if seed == nil {
		seed = s.State.Turn
	}
	if seed == nil {
		seed = &turns.Turn{}
	}

	var (
		updated *turns.Turn
		err     error
	)
	if s.Registry == nil {
		updated, err = s.State.Eng.RunInference(runCtx, seed)
	} else {
		cfg := toolhelpers.NewToolConfig()
		if s.ToolConfig != nil {
			cfg = *s.ToolConfig
		}
		updated, err = toolhelpers.RunToolCallingLoop(runCtx, s.State.Eng, seed, s.Registry, cfg)
	}

	if updated != nil {
		s.State.Turn = updated
		if updated.RunID != "" {
			s.State.RunID = updated.RunID
		}
	}

	if err == nil && s.Persister != nil && updated != nil {
		runID := s.State.RunID
		if updated.RunID != "" {
			runID = updated.RunID
		}
		_ = s.Persister.PersistTurn(runCtx, runID, updated)
	}

	return updated, err
}

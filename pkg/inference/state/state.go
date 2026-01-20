package state

import (
	"context"
	"errors"
	"sync"

	"github.com/go-go-golems/geppetto/pkg/inference/engine"
	"github.com/go-go-golems/geppetto/pkg/turns"
)

var (
	// ErrInferenceRunning indicates a run is already in progress.
	ErrInferenceRunning = errors.New("inference already running")
	// ErrInferenceNotRunning indicates no run is currently active.
	ErrInferenceNotRunning = errors.New("inference not running")
	// ErrInferenceStateNil indicates the state struct hasn't been initialized.
	ErrInferenceStateNil = errors.New("inference state is nil")
)

// InferenceState holds per-session inference data (engine, current turn, run flags).
//
// It is intentionally UI-agnostic and can be shared across TUI, CLI, and webchat.
type InferenceState struct {
	RunID string
	Turn  *turns.Turn
	Eng   engine.Engine

	mu      sync.Mutex
	running bool
	cancel  context.CancelFunc
}

// NewInferenceState builds a state holder with the provided run metadata.
func NewInferenceState(runID string, turn *turns.Turn, eng engine.Engine) *InferenceState {
	return &InferenceState{
		RunID: runID,
		Turn:  turn,
		Eng:   eng,
	}
}

// StartRun marks the inference loop as running. Returns ErrInferenceRunning if already active.
func (is *InferenceState) StartRun() error {
	if is == nil {
		return ErrInferenceStateNil
	}
	is.mu.Lock()
	defer is.mu.Unlock()
	if is.running {
		return ErrInferenceRunning
	}
	is.running = true
	return nil
}

// FinishRun clears the running flag and cancel handle.
func (is *InferenceState) FinishRun() {
	if is == nil {
		return
	}
	is.mu.Lock()
	is.running = false
	is.cancel = nil
	is.mu.Unlock()
}

// IsRunning reports whether the inference loop is active.
func (is *InferenceState) IsRunning() bool {
	if is == nil {
		return false
	}
	is.mu.Lock()
	defer is.mu.Unlock()
	return is.running
}

// SetCancel stores the cancel function associated with the current run.
func (is *InferenceState) SetCancel(cancel context.CancelFunc) {
	if is == nil {
		return
	}
	is.mu.Lock()
	is.cancel = cancel
	is.mu.Unlock()
}

// CancelRun triggers the stored cancel function if a run is active.
func (is *InferenceState) CancelRun() error {
	if is == nil {
		return ErrInferenceStateNil
	}
	is.mu.Lock()
	cancel := is.cancel
	running := is.running
	is.mu.Unlock()
	if !running || cancel == nil {
		return ErrInferenceNotRunning
	}
	cancel()
	return nil
}

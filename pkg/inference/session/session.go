package session

import (
	"context"
	"errors"
	"sync"

	"github.com/go-go-golems/geppetto/pkg/turns"
	"github.com/google/uuid"
)

var (
	ErrSessionNil           = errors.New("session is nil")
	ErrSessionBuilderNil    = errors.New("session builder is nil")
	ErrSessionAlreadyActive = errors.New("session already has an active inference")
	ErrSessionNoActive      = errors.New("session has no active inference")
)

// Session represents a long-lived, multi-turn interaction.
//
// It owns:
// - a stable SessionID
// - the session turn history (append-only snapshots)
// - the invariant that only one inference is active at a time
type Session struct {
	SessionID string
	Turns     []*turns.Turn

	Builder EngineBuilder

	mu     sync.Mutex
	active *ExecutionHandle
}

// IsRunning reports whether the session currently has an active inference.
func (s *Session) IsRunning() bool {
	if s == nil {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.active != nil && s.active.IsRunning()
}

func (s *Session) Latest() *turns.Turn {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.Turns) == 0 {
		return nil
	}
	return s.Turns[len(s.Turns)-1]
}

// Append appends a turn snapshot to the session history.
func (s *Session) Append(t *turns.Turn) {
	if s == nil || t == nil {
		return
	}
	s.mu.Lock()
	// Ensure RunID matches SessionID (chat-session semantics).
	if s.SessionID != "" && t.RunID == "" {
		t.RunID = s.SessionID
	}
	s.Turns = append(s.Turns, t)
	s.mu.Unlock()
}

// StartInference starts an inference asynchronously and returns an ExecutionHandle.
//
// The builder is invoked to produce a blocking runner (RunInference). The runner is
// executed in a goroutine, and the result is appended to the Session as a new Turn
// snapshot on success.
func (s *Session) StartInference(ctx context.Context) (*ExecutionHandle, error) {
	if s == nil {
		return nil, ErrSessionNil
	}
	if s.Builder == nil {
		return nil, ErrSessionBuilderNil
	}
	if ctx == nil {
		ctx = context.Background()
	}

	s.mu.Lock()
	if s.active != nil && s.active.IsRunning() {
		s.mu.Unlock()
		return nil, ErrSessionAlreadyActive
	}
	var input *turns.Turn
	if len(s.Turns) > 0 {
		input = s.Turns[len(s.Turns)-1]
	}
	if input == nil {
		// Allow starting from an empty turn if the caller hasn't seeded yet.
		input = &turns.Turn{RunID: s.SessionID}
		s.Turns = append(s.Turns, input)
	}
	// Ensure session id is threaded into the input turn.
	if s.SessionID != "" && input.RunID == "" {
		input.RunID = s.SessionID
	}
	s.mu.Unlock()

	runner, err := s.Builder.Build(ctx, s.SessionID)
	if err != nil {
		return nil, err
	}

	runCtx, cancel := context.WithCancel(ctx)
	handle := newExecutionHandle(s.SessionID, uuid.NewString(), input, cancel)

	s.mu.Lock()
	// Re-check after build: another goroutine may have started a run while we were building.
	if s.active != nil && s.active.IsRunning() {
		s.mu.Unlock()
		cancel()
		return nil, ErrSessionAlreadyActive
	}
	s.active = handle
	s.mu.Unlock()

	go func() {
		defer func() {
			s.mu.Lock()
			s.active = nil
			s.mu.Unlock()
		}()

		out, err := runner.RunInference(runCtx, input)
		if err == nil && out != nil {
			s.Append(out)
		}
		handle.setResult(out, err)
	}()

	return handle, nil
}

// CancelActive cancels the current active inference, if any.
func (s *Session) CancelActive() error {
	if s == nil {
		return ErrSessionNil
	}
	s.mu.Lock()
	h := s.active
	s.mu.Unlock()
	if h == nil || !h.IsRunning() {
		return ErrSessionNoActive
	}
	h.Cancel()
	return nil
}

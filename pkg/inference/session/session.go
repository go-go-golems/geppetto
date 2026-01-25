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
	ErrSessionEmptyTurn     = errors.New("session has no seed turn (or seed turn is empty)")
	ErrSessionIDEmpty       = errors.New("session has empty SessionID")
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

// NewSession constructs a Session with a generated SessionID.
func NewSession() *Session {
	return &Session{
		SessionID: uuid.NewString(),
	}
}

// AppendNewTurnFromUserPrompt clones the latest session turn (if any), appends a user block
// containing the prompt (if non-empty), assigns a Turn.ID if missing, appends it to the session
// history, and returns the appended Turn.
//
// This is the preferred API for “next prompt” creation in UIs: it prevents callers from mutating
// the historical latest turn in-place.
func (s *Session) AppendNewTurnFromUserPrompt(prompt string) (*turns.Turn, error) {
	return s.AppendNewTurnFromUserPrompts(prompt)
}

// AppendNewTurnFromUserPrompts is like AppendNewTurnFromUserPrompt, but appends one user block per
// non-empty prompt to the newly appended turn.
func (s *Session) AppendNewTurnFromUserPrompts(prompts ...string) (*turns.Turn, error) {
	if s == nil {
		return nil, ErrSessionNil
	}
	if s.SessionID == "" {
		return nil, ErrSessionIDEmpty
	}

	var base *turns.Turn
	s.mu.Lock()
	if s.active != nil && s.active.IsRunning() {
		s.mu.Unlock()
		return nil, ErrSessionAlreadyActive
	}
	if len(s.Turns) > 0 {
		base = s.Turns[len(s.Turns)-1]
	}
	s.mu.Unlock()

	seed := &turns.Turn{}
	if base != nil {
		seed = base.Clone()
		// Each appended user prompt is a new turn. The latest turn is cloned to preserve
		// conversation context, but the new turn must not retain the previous Turn.ID.
		// Otherwise hydration/persistence keyed by Turn.ID will overwrite prior turns.
		seed.ID = ""
	}
	for _, prompt := range prompts {
		if prompt == "" {
			continue
		}
		turns.AppendBlock(seed, turns.NewUserTextBlock(prompt))
	}
	if seed.ID == "" {
		seed.ID = uuid.NewString()
	}
	s.Append(seed)
	return seed, nil
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
	if s.SessionID != "" {
		if _, ok, err := turns.KeyTurnMetaSessionID.Get(t.Metadata); err != nil || !ok {
			_ = turns.KeyTurnMetaSessionID.Set(&t.Metadata, s.SessionID)
		}
	}
	s.Turns = append(s.Turns, t)
	s.mu.Unlock()
}

// StartInference starts an inference asynchronously and returns an ExecutionHandle.
//
// The builder is invoked to produce a blocking runner (RunInference). The runner is
// executed in a goroutine against the latest appended Turn, which is intentionally
// mutated in-place (middlewares may modify it).
func (s *Session) StartInference(ctx context.Context) (*ExecutionHandle, error) {
	if s == nil {
		return nil, ErrSessionNil
	}
	if s.SessionID == "" {
		return nil, ErrSessionIDEmpty
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
	if input == nil || len(input.Blocks) == 0 {
		s.mu.Unlock()
		return nil, ErrSessionEmptyTurn
	}

	// Inference runs against the latest appended turn in-place. This allows middlewares
	// to intentionally mutate the turn so the updated version becomes the next seed base.
	if input.ID == "" {
		input.ID = uuid.NewString()
	}
	_ = turns.KeyTurnMetaSessionID.Set(&input.Metadata, s.SessionID)
	inferenceID := uuid.NewString()
	_ = turns.KeyTurnMetaInferenceID.Set(&input.Metadata, inferenceID)
	s.mu.Unlock()

	runner, err := s.Builder.Build(ctx, s.SessionID)
	if err != nil {
		return nil, err
	}

	runCtx, cancel := context.WithCancel(ctx)
	handle := newExecutionHandle(s.SessionID, inferenceID, input, cancel)

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
		if err == nil {
			if out == nil {
				out = input
			}
			if out.ID == "" {
				out.ID = input.ID
			}
			_ = turns.KeyTurnMetaSessionID.Set(&out.Metadata, s.SessionID)
			_ = turns.KeyTurnMetaInferenceID.Set(&out.Metadata, inferenceID)

			// Keep the session's latest turn as the canonical result, even if the runner
			// returns a different pointer.
			if out != input {
				s.mu.Lock()
				*input = *out
				s.mu.Unlock()
				out = input
			}
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

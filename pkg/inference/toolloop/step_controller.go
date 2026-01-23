package toolloop

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
)

type StepPhase string

const (
	StepPhaseAfterInference StepPhase = "after_inference"
	StepPhaseAfterTools     StepPhase = "after_tools"
)

// StepScope is application-owned metadata used for routing/authorization of continue requests.
// The tool loop treats these values as opaque.
type StepScope struct {
	SessionID       string
	ConversationID  string
	Owner           string
	AdditionalScope map[string]any
}

// PauseMeta describes a single pause and is returned to the HTTP layer for authorization/routing.
type PauseMeta struct {
	PauseID    string
	Phase      StepPhase
	Summary    string
	DeadlineMs int64

	SessionID   string
	InferenceID string
	TurnID      string

	Scope StepScope
	Extra map[string]any
}

type pauseWaiter struct {
	meta PauseMeta
	ch   chan struct{}
}

// StepController coordinates pause/continue across multiple in-flight executions.
//
// It is intended to be owned by the application/web layer (not by session.Session or by a conversation object),
// so HTTP handlers can continue a pause by pause_id directly.
type StepController struct {
	mu sync.Mutex

	enabled map[string]StepScope    // keyed by SessionID
	waiters map[string]*pauseWaiter // keyed by PauseID
}

func NewStepController() *StepController {
	return &StepController{
		enabled: make(map[string]StepScope),
		waiters: make(map[string]*pauseWaiter),
	}
}

func (s *StepController) Enable(scope StepScope) {
	if s == nil || scope.SessionID == "" {
		return
	}
	s.mu.Lock()
	s.enabled[scope.SessionID] = scope
	s.mu.Unlock()
}

// DisableSession disables step mode for a session and drains any active pauses for that session.
func (s *StepController) DisableSession(sessionID string) {
	if s == nil || sessionID == "" {
		return
	}
	var toClose []chan struct{}
	s.mu.Lock()
	delete(s.enabled, sessionID)
	for id, w := range s.waiters {
		if w != nil && w.meta.SessionID == sessionID {
			toClose = append(toClose, w.ch)
			delete(s.waiters, id)
		}
	}
	s.mu.Unlock()
	for _, ch := range toClose {
		close(ch)
	}
}

func (s *StepController) IsEnabled(sessionID string) (StepScope, bool) {
	if s == nil || sessionID == "" {
		return StepScope{}, false
	}
	s.mu.Lock()
	scope, ok := s.enabled[sessionID]
	s.mu.Unlock()
	return scope, ok
}

// Pause registers a pause if step mode is enabled for meta.SessionID.
// It returns the stored meta and whether the pause was registered.
func (s *StepController) Pause(meta PauseMeta) (PauseMeta, bool) {
	if s == nil || meta.SessionID == "" {
		return meta, false
	}
	scope, ok := s.IsEnabled(meta.SessionID)
	if !ok {
		return meta, false
	}

	if meta.PauseID == "" {
		meta.PauseID = uuid.NewString()
	}
	if meta.DeadlineMs == 0 {
		meta.DeadlineMs = time.Now().Add(30 * time.Second).UnixMilli()
	}
	meta.Scope = mergeScope(scope, meta.Scope)
	if meta.Extra == nil {
		meta.Extra = map[string]any{}
	}

	s.mu.Lock()
	ch := make(chan struct{})
	s.waiters[meta.PauseID] = &pauseWaiter{meta: meta, ch: ch}
	s.mu.Unlock()
	return meta, true
}

func mergeScope(base StepScope, override StepScope) StepScope {
	out := base
	if override.SessionID != "" {
		out.SessionID = override.SessionID
	}
	if override.ConversationID != "" {
		out.ConversationID = override.ConversationID
	}
	if override.Owner != "" {
		out.Owner = override.Owner
	}
	if override.AdditionalScope != nil {
		if out.AdditionalScope == nil {
			out.AdditionalScope = map[string]any{}
		}
		for k, v := range override.AdditionalScope {
			out.AdditionalScope[k] = v
		}
	}
	return out
}

// Lookup returns the pause metadata without continuing it.
func (s *StepController) Lookup(pauseID string) (PauseMeta, bool) {
	if s == nil || pauseID == "" {
		return PauseMeta{}, false
	}
	s.mu.Lock()
	w := s.waiters[pauseID]
	s.mu.Unlock()
	if w == nil {
		return PauseMeta{}, false
	}
	return w.meta, true
}

// Wait blocks until Continue(pauseID) is called, timeout elapses (auto-continue), or ctx is canceled.
func (s *StepController) Wait(ctx context.Context, pauseID string, timeout time.Duration) error {
	if s == nil || pauseID == "" {
		return nil
	}
	s.mu.Lock()
	w := s.waiters[pauseID]
	s.mu.Unlock()
	if w == nil {
		return nil
	}

	select {
	case <-w.ch:
		return nil
	case <-time.After(timeout):
		s.Continue(pauseID)
		return nil
	case <-ctx.Done():
		s.Continue(pauseID)
		return ctx.Err()
	}
}

// Continue resumes a paused run and returns the pause metadata if the pause existed.
func (s *StepController) Continue(pauseID string) (PauseMeta, bool) {
	if s == nil || pauseID == "" {
		return PauseMeta{}, false
	}
	s.mu.Lock()
	w := s.waiters[pauseID]
	delete(s.waiters, pauseID)
	s.mu.Unlock()
	if w == nil {
		return PauseMeta{}, false
	}
	close(w.ch)
	return w.meta, true
}

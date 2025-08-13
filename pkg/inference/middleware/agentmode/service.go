package agentmode

import (
	"context"
	"strings"
	"time"
)

type Err string

func (e Err) Error() string { return string(e) }

var ErrUnknownMode = Err("unknown agent mode")

// Service merges resolving modes and recording changes.
type Service interface {
	GetMode(ctx context.Context, name string) (*AgentMode, error)
	GetCurrentMode(ctx context.Context, runID string) (string, error)
	RecordModeChange(ctx context.Context, change ModeChange) error
}

// StaticService implements Service purely in-memory.
type StaticService struct {
	modes   map[string]*AgentMode // keyed by lower-case name
	current map[string]string     // last per runID
}

func NewStaticService(modes []*AgentMode) *StaticService {
	mm := make(map[string]*AgentMode, len(modes))
	for _, m := range modes {
		if m != nil && m.Name != "" {
			mm[strings.ToLower(m.Name)] = m
		}
	}
	return &StaticService{modes: mm, current: map[string]string{}}
}

func (s *StaticService) GetMode(ctx context.Context, name string) (*AgentMode, error) {
	if name == "" {
		return nil, ErrUnknownMode
	}
	if m, ok := s.modes[strings.ToLower(name)]; ok {
		return m, nil
	}
	return nil, ErrUnknownMode
}
func (s *StaticService) GetCurrentMode(ctx context.Context, runID string) (string, error) {
	return s.current[runID], nil
}
func (s *StaticService) RecordModeChange(ctx context.Context, change ModeChange) error {
	s.current[change.RunID] = change.ToMode
	return nil
}

// SQLiteService wraps SQLiteStore with an in-memory mode catalog.
type SQLiteService struct {
	modes map[string]*AgentMode
	s     *SQLiteStore
}

func NewSQLiteService(store *SQLiteStore, modes []*AgentMode) *SQLiteService {
	mm := make(map[string]*AgentMode, len(modes))
	for _, m := range modes {
		if m != nil && m.Name != "" {
			mm[strings.ToLower(m.Name)] = m
		}
	}
	return &SQLiteService{modes: mm, s: store}
}

func (s *SQLiteService) GetMode(ctx context.Context, name string) (*AgentMode, error) {
	if name == "" {
		return nil, ErrUnknownMode
	}
	if m, ok := s.modes[strings.ToLower(name)]; ok {
		return m, nil
	}
	return nil, ErrUnknownMode
}
func (s *SQLiteService) GetCurrentMode(ctx context.Context, runID string) (string, error) {
	return s.s.GetCurrentMode(ctx, runID)
}
func (s *SQLiteService) RecordModeChange(ctx context.Context, change ModeChange) error {
	return s.s.RecordModeChange(ctx, change)
}

// Helper to stamp a change
func NewChange(runID, turnID, from, to, analysis string) ModeChange {
	return ModeChange{RunID: runID, TurnID: turnID, FromMode: from, ToMode: to, Analysis: analysis, At: time.Now()}
}

package agentmode

import (
	"context"
	"database/sql"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// SQLiteStore implements Store using a SQLite database.
type SQLiteStore struct{ db *sql.DB }

func NewSQLiteStore(dsn string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}
	s := &SQLiteStore{db: db}
	if err := s.migrate(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *SQLiteStore) migrate() error {
	_, err := s.db.Exec(`
CREATE TABLE IF NOT EXISTS agent_mode_changes (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  run_id TEXT,
  turn_id TEXT,
  from_mode TEXT,
  to_mode TEXT,
  analysis TEXT,
  at TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_agent_mode_changes_run_id_at ON agent_mode_changes(run_id, at);
`)
	return err
}

func (s *SQLiteStore) GetCurrentMode(ctx context.Context, runID string) (string, error) {
	row := s.db.QueryRowContext(ctx, `SELECT to_mode FROM agent_mode_changes WHERE run_id = ? ORDER BY at DESC, id DESC LIMIT 1`, runID)
	var mode string
	switch err := row.Scan(&mode); err {
	case nil:
		return mode, nil
	case sql.ErrNoRows:
		return "", nil
	default:
		return "", err
	}
}

func (s *SQLiteStore) RecordModeChange(ctx context.Context, change ModeChange) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO agent_mode_changes (run_id, turn_id, from_mode, to_mode, analysis, at) VALUES (?, ?, ?, ?, ?, ?)`,
		change.RunID, change.TurnID, change.FromMode, change.ToMode, change.Analysis, change.At.Format(time.RFC3339Nano))
	return err
}

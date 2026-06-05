package provider

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	geppettomodule "github.com/go-go-golems/geppetto/pkg/js/modules/geppetto"
	"github.com/go-go-golems/geppetto/pkg/turns"
	"github.com/go-go-golems/geppetto/pkg/turns/serde"
	_ "github.com/mattn/go-sqlite3"
)

const sqliteTurnStoreSchema = `
CREATE TABLE IF NOT EXISTS geppetto_turns (
	conv_id TEXT NOT NULL,
	session_id TEXT NOT NULL,
	turn_id TEXT NOT NULL,
	phase TEXT NOT NULL,
	runtime_key TEXT NOT NULL DEFAULT '',
	inference_id TEXT NOT NULL DEFAULT '',
	created_at_ms INTEGER NOT NULL,
	payload TEXT NOT NULL,
	PRIMARY KEY (conv_id, session_id, turn_id, phase)
);
CREATE INDEX IF NOT EXISTS geppetto_turns_by_session ON geppetto_turns(session_id, phase, created_at_ms DESC);
CREATE INDEX IF NOT EXISTS geppetto_turns_by_conv ON geppetto_turns(conv_id, phase, created_at_ms DESC);
`

type sqliteTurnStore struct {
	db *sql.DB
}

var _ geppettomodule.TurnStore = (*sqliteTurnStore)(nil)

func openSQLiteTurnStore(turnsDSN, turnsDB string) (*sqliteTurnStore, error) {
	dsn := strings.TrimSpace(turnsDSN)
	if dsn == "" {
		path := strings.TrimSpace(turnsDB)
		if path == "" {
			return nil, nil
		}
		if dir := filepath.Dir(path); dir != "" && dir != "." {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return nil, fmt.Errorf("geppetto provider create turns db dir: %w", err)
			}
		}
		dsn = sqliteTurnStoreDSNForFile(path)
	}
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("geppetto provider open turns sqlite: %w", err)
	}
	store := &sqliteTurnStore{db: db}
	if err := store.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return store, nil
}

func sqliteTurnStoreDSNForFile(path string) string {
	return fmt.Sprintf("file:%s?_busy_timeout=5000&_journal_mode=WAL", path)
}

func (s *sqliteTurnStore) migrate() error {
	if s == nil || s.db == nil {
		return fmt.Errorf("geppetto provider turns sqlite store is nil")
	}
	if _, err := s.db.Exec(sqliteTurnStoreSchema); err != nil {
		return fmt.Errorf("geppetto provider migrate turns sqlite: %w", err)
	}
	return nil
}

func (s *sqliteTurnStore) PersistTurn(ctx context.Context, t *turns.Turn) error {
	if s == nil || s.db == nil || t == nil {
		return nil
	}
	sessionID := turnSessionID(t)
	if sessionID == "" {
		return fmt.Errorf("geppetto provider turns sqlite store: sessionID is empty")
	}
	turnID := strings.TrimSpace(t.ID)
	if turnID == "" {
		turnID = "turn"
	}
	payload, err := serde.ToYAML(t, serde.Options{})
	if err != nil {
		return fmt.Errorf("geppetto provider turns sqlite store serialize turn: %w", err)
	}
	createdAtMs := time.Now().UnixMilli()
	_, err = s.db.ExecContext(ctx, `
INSERT INTO geppetto_turns (conv_id, session_id, turn_id, phase, runtime_key, inference_id, created_at_ms, payload)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(conv_id, session_id, turn_id, phase) DO UPDATE SET
	runtime_key=excluded.runtime_key,
	inference_id=excluded.inference_id,
	created_at_ms=excluded.created_at_ms,
	payload=excluded.payload
`, sessionID, sessionID, turnID, "final", turnRuntimeKey(t), turnInferenceID(t), createdAtMs, string(payload))
	if err != nil {
		return fmt.Errorf("geppetto provider turns sqlite store persist turn: %w", err)
	}
	return nil
}

func (s *sqliteTurnStore) ListTurns(ctx context.Context, q geppettomodule.TurnStoreQuery) ([]geppettomodule.TurnStoreSnapshot, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("geppetto provider turns sqlite store is nil")
	}
	convID := strings.TrimSpace(q.ConvID)
	sessionID := strings.TrimSpace(q.SessionID)
	phase := strings.TrimSpace(q.Phase)
	sinceMs := q.SinceMs
	limit := q.Limit
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT conv_id, session_id, turn_id, phase, runtime_key, inference_id, created_at_ms, payload
FROM geppetto_turns
WHERE (? = '' OR conv_id = ?)
  AND (? = '' OR session_id = ?)
  AND (? = '' OR phase = ?)
  AND (? <= 0 OR created_at_ms >= ?)
ORDER BY created_at_ms DESC
LIMIT ?`, convID, convID, sessionID, sessionID, phase, phase, sinceMs, sinceMs, limit)
	if err != nil {
		return nil, fmt.Errorf("geppetto provider turns sqlite store list turns: %w", err)
	}
	defer func() { _ = rows.Close() }()
	out := []geppettomodule.TurnStoreSnapshot{}
	for rows.Next() {
		snapshot, err := scanTurnSnapshot(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, snapshot)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *sqliteTurnStore) LoadLatestTurn(ctx context.Context, q geppettomodule.TurnStoreQuery) (*geppettomodule.TurnStoreSnapshot, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("geppetto provider turns sqlite store is nil")
	}
	convID := strings.TrimSpace(q.ConvID)
	sessionID := strings.TrimSpace(q.SessionID)
	if convID == "" && sessionID == "" {
		return nil, fmt.Errorf("geppetto provider turns sqlite store: convId or sessionId required")
	}
	phase := strings.TrimSpace(q.Phase)
	if phase == "" {
		phase = "final"
	}
	row := s.db.QueryRowContext(ctx, `
SELECT conv_id, session_id, turn_id, phase, runtime_key, inference_id, created_at_ms, payload
FROM geppetto_turns
WHERE (? = '' OR conv_id = ?)
  AND (? = '' OR session_id = ?)
  AND phase = ?
ORDER BY created_at_ms DESC
LIMIT 1`, convID, convID, sessionID, sessionID, phase)
	snapshot, err := scanTurnSnapshot(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &snapshot, nil
}

func (s *sqliteTurnStore) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanTurnSnapshot(row rowScanner) (geppettomodule.TurnStoreSnapshot, error) {
	var snapshot geppettomodule.TurnStoreSnapshot
	var payload string
	if err := row.Scan(&snapshot.ConvID, &snapshot.SessionID, &snapshot.TurnID, &snapshot.Phase, &snapshot.RuntimeKey, &snapshot.InferenceID, &snapshot.CreatedAtMs, &payload); err != nil {
		return geppettomodule.TurnStoreSnapshot{}, err
	}
	if strings.TrimSpace(payload) != "" {
		decoded, err := serde.FromYAML([]byte(payload))
		if err != nil {
			return geppettomodule.TurnStoreSnapshot{}, fmt.Errorf("geppetto provider turns sqlite store decode turn: %w", err)
		}
		snapshot.Turn = decoded
	}
	return snapshot, nil
}

func turnSessionID(t *turns.Turn) string {
	if t == nil {
		return ""
	}
	if v, ok, err := turns.KeyTurnMetaSessionID.Get(t.Metadata); err == nil && ok {
		return strings.TrimSpace(v)
	}
	return ""
}

func turnRuntimeKey(t *turns.Turn) string {
	if t == nil {
		return ""
	}
	if v, ok, err := turns.KeyTurnMetaRuntime.Get(t.Metadata); err == nil && ok {
		if s, ok := v.(string); ok {
			return strings.TrimSpace(s)
		}
	}
	return ""
}

func turnInferenceID(t *turns.Turn) string {
	if t == nil {
		return ""
	}
	if v, ok, err := turns.KeyTurnMetaInferenceID.Get(t.Metadata); err == nil && ok {
		return strings.TrimSpace(v)
	}
	return ""
}

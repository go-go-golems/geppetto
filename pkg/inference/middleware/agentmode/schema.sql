-- Example schema for Agent Mode middleware (SQLite)
-- This file defines a minimal schema for:
-- 1) agent_modes: declarative catalog of modes (name, prompt, allowed tools)
-- 2) agent_mode_changes: audit log of mode transitions per run/turn

PRAGMA foreign_keys = ON;

-- 1) Declarative catalog of modes
CREATE TABLE IF NOT EXISTS agent_modes (
  name TEXT PRIMARY KEY,
  prompt TEXT NOT NULL,
  -- Comma-separated list of tool names; can be swapped for JSON if preferred
  allowed_tools TEXT NOT NULL
);

-- 2) Audit log of changes
CREATE TABLE IF NOT EXISTS agent_mode_changes (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  run_id TEXT NOT NULL,
  turn_id TEXT NOT NULL,
  from_mode TEXT,
  to_mode TEXT NOT NULL,
  analysis TEXT,
  at TIMESTAMP NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_agent_mode_changes_run_id_at
  ON agent_mode_changes(run_id, at);



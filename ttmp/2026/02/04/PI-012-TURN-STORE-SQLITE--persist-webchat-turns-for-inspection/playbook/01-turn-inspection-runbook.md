---
Title: Turn Inspection Runbook
Ticket: PI-012-TURN-STORE-SQLITE
Status: active
Topics:
  - backend
  - webchat
  - sqlite
  - debugging
DocType: playbook
Intent: long-term
Owners: []
RelatedFiles: []
---

# Turn Inspection Runbook

## Purpose

Capture the **exact** LLM input blocks by enabling webchat turn snapshots and (optionally) querying a SQLite turn store.

## Environment Assumptions

- Backend is launched via `web-agent-example` (or another Pinocchio webchat server).
- You can restart the backend (tmux window `webagent:0`).
- You know the `conv_id` you want to inspect.

## Commands

### 1) Enable file snapshots

Set the snapshot output directory and restart the backend:

```bash
# choose a directory for snapshots (example uses the ticket workspace)
export PINOCCHIO_WEBCHAT_TURN_SNAPSHOTS_DIR=/home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/ttmp/2026/02/04/PI-012-TURN-STORE-SQLITE--persist-webchat-turns-for-inspection/various/turn-snapshots

# restart backend (example)
cd /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/web-agent-example
PINOCCHIO_WEBCHAT_TURN_SNAPSHOTS_DIR="$PINOCCHIO_WEBCHAT_TURN_SNAPSHOTS_DIR" \
  go run ./cmd/web-agent-example serve --addr :8080 --log-level debug
```

### 2) Trigger a request

Send a message in the UI for the target conversation (`conv_id`).

### 3) Inspect saved snapshots

```bash
ls -la "$PINOCCHIO_WEBCHAT_TURN_SNAPSHOTS_DIR/<conv_id>/<run_id>"

# open the newest file
ls -t "$PINOCCHIO_WEBCHAT_TURN_SNAPSHOTS_DIR/<conv_id>/<run_id>" | head -n 1 | xargs -I{} sed -n '1,200p' "$PINOCCHIO_WEBCHAT_TURN_SNAPSHOTS_DIR/<conv_id>/<run_id>/{}"
```

### 4) (Optional) Query SQLite turn store

If `turns-dsn`/`turns-db` is configured:

```bash
sqlite3 /path/to/turns.db "SELECT conv_id, run_id, turn_id, phase, created_at_ms FROM turns WHERE conv_id = '<conv_id>' ORDER BY created_at_ms DESC LIMIT 5;"
```

## Exit Criteria

- A YAML file exists under `<conv_id>/<run_id>` and includes system blocks + metadata.
- (Optional) SQLite query returns rows for the selected conversation.

## Failure Modes

- **No snapshots**: env var not set or backend not restarted.
- **Empty blocks**: turn may not have been written for the requested phase.
- **No SQLite rows**: store not configured or turn persister not wired.

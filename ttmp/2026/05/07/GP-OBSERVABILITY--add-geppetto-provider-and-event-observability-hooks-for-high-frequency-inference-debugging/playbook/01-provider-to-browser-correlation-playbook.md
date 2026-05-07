---
Title: Provider to Browser Correlation Playbook
Ticket: GP-OBSERVABILITY
Status: active
Topics:
    - events
    - inference
    - streaming
    - openai
    - glazed
    - sqlite
DocType: playbook
Intent: long-term
Owners:
    - manuel
RelatedFiles:
    - Path: ../../../../../../../pinocchio/cmd/web-chat/app/debug_reconcile_db.go
      Note: Defines geppetto_reasoning_to_frontend and Geppetto SQLite tables/views
    - Path: ttmp/2026/05/07/GP-OBSERVABILITY--add-geppetto-provider-and-event-observability-hooks-for-high-frequency-inference-debugging/scripts/01-meta-and-counts.sql
      Note: First SQL check for meta and record counts
    - Path: ttmp/2026/05/07/GP-OBSERVABILITY--add-geppetto-provider-and-event-observability-hooks-for-high-frequency-inference-debugging/scripts/03-provider-to-browser-correlation.sql
      Note: Manual provider-to-browser correlation query
    - Path: ttmp/2026/05/07/GP-OBSERVABILITY--add-geppetto-provider-and-event-observability-hooks-for-high-frequency-inference-debugging/scripts/06-geppetto-reasoning-to-frontend-view.sql
      Note: Built-in view inspection query
ExternalSources: []
Summary: Repeatable browser-driven validation procedure and SQL query sequence for correlating OpenAI Responses provider events to Geppetto events, Sessionstream backend records, frontend parsed frames, UI mutations, and timeline entities.
LastUpdated: 2026-05-07T12:42:55.83140453-04:00
WhatFor: Run and review end-to-end provider-to-browser observability validation for GP-OBSERVABILITY.
WhenToUse: Use after changing Geppetto observability, Pinocchio debug recording, SQLite export, or frontend stream debug handling.
---


# Provider to Browser Correlation Playbook

## Purpose

This playbook validates that a browser-visible reasoning event can be traced back to the decoded OpenAI Responses provider event that caused it. It uses the Pinocchio web-chat UI, frontend stream debug recording, Geppetto provider/event records, and the SQLite reconcile export.

The key question is:

> Can we start from a browser `ChatReasoningAppended` event and walk back to the provider `response.reasoning_summary_text.delta` record that produced it?

The current answer is yes, using ordered reasoning deltas and exact Geppetto-published chunk matching. Direct joins by provider item ID require a future `ReasoningUpdate` schema extension.

## Environment Assumptions

- Working directory root: `/home/manuel/workspaces/2026-05-02/use-sessionstream-coinvault`.
- `pinocchio` and `geppetto` are sibling Git repositories in that workspace.
- Web-chat can resolve a profile that uses OpenAI Responses, for example `gpt-5-nano`.
- Provider credentials/config are already available through the existing Pinocchio/Geppetto config stack.
- `sqlite3`, `curl`, Go, and Playwright tooling are available.
- Frontend stream debug is enabled in the browser with `localStorage['pinocchio.debugStream']='1'` before submitting the prompt.

## Commands

### 1. Start web-chat with Geppetto provider tracing

```bash
cd /home/manuel/workspaces/2026-05-02/use-sessionstream-coinvault/pinocchio

go run ./cmd/web-chat web-chat \
  --addr 127.0.0.1:18082 \
  --debug-api \
  --geppetto-trace-level provider
```

Expected config probe:

```bash
curl -fsS http://127.0.0.1:18082/app-config.js
```

Expected output includes:

```javascript
window.__PINOCCHIO_WEBCHAT_CONFIG__ = {"basePrefix":"","debugApiEnabled":true};
```

### 2. Run browser chat with frontend debug enabled

In Playwright or browser devtools:

```javascript
localStorage.setItem('pinocchio.debugStream', '1');
location.reload();
```

Then use the UI:

1. Type a prompt such as:

   ```text
   Use brief reasoning if available, then answer in one sentence: what is 3+4?
   ```

2. Click **Send**.
3. Wait for the status to become `finished`.
4. Verify the **Stream Debug** badge has a non-zero count.

### 3. Capture frontend debug entries and export SQLite

In browser devtools or Playwright:

```javascript
const sessionId = new URLSearchParams(location.search).get('sessionId');
const entries = window.__pinocchioStreamDebug.entries();
console.log({ sessionId, frontendEntries: entries.length });
```

Save the frontend entries locally for reproducibility:

```javascript
JSON.stringify({ sessionId, records: entries }, null, 2)
```

Upload the same browser-collected entries to the reconcile endpoint:

```bash
SESSION='<session-id-from-browser>'
curl -fsS \
  -X POST "http://127.0.0.1:18082/api/debug/sessions/${SESSION}/reconcile/upload" \
  -H 'content-type: application/json' \
  --data-binary @/tmp/browser-chat-frontend-upload.json \
  -o /tmp/browser-chat-e2e.sqlite
```

Expected file signature:

```bash
sqlite3 /tmp/browser-chat-e2e.sqlite 'SELECT sqlite_version();'
```

### 4. Run the ticket SQL scripts

The ticket keeps the exact SQL used for the investigation under `scripts/` with numerical prefixes:

```bash
TICKET=/home/manuel/workspaces/2026-05-02/use-sessionstream-coinvault/geppetto/ttmp/2026/05/07/GP-OBSERVABILITY--add-geppetto-provider-and-event-observability-hooks-for-high-frequency-inference-debugging
DB=/tmp/browser-chat-e2e.sqlite

sqlite3 "$DB" < "$TICKET/scripts/01-meta-and-counts.sql"
sqlite3 "$DB" < "$TICKET/scripts/02-geppetto-reasoning-sequence.sql"
sqlite3 "$DB" < "$TICKET/scripts/03-provider-to-browser-correlation.sql"
sqlite3 "$DB" < "$TICKET/scripts/04-correlation-quality-checks.sql"
sqlite3 "$DB" < "$TICKET/scripts/05-delivery-chain-and-timeline.sql"
```

## SQL Script Inventory

| Script | Purpose |
|---|---|
| `01-meta-and-counts.sql` | Confirms frontend/backend/Geppetto record counts and meta values. |
| `02-geppetto-reasoning-sequence.sql` | Inspects provider and emitted Geppetto reasoning records in sequence. |
| `03-provider-to-browser-correlation.sql` | Produces the provider -> Geppetto -> backend -> frontend -> timeline correlation rows. |
| `04-correlation-quality-checks.sql` | Summarizes exact delta/chunk matches and lists provider/frontend mismatches. |
| `05-delivery-chain-and-timeline.sql` | Shows Sessionstream delivery-chain rows and persisted thinking timeline entities. |

## Exit Criteria

A successful run should show:

- `frontend_record_count > 0`.
- `backend_record_count > 0`.
- `geppetto_record_count > 0`.
- `geppetto_provider_events > 0`.
- `geppetto_emitted_events > 0`.
- `geppetto_summary_without_item_id = 0` for the observed reasoning summary stream.
- `geppetto_publish_errors = 0`.
- Provider normalize-delta count equals backend `ChatReasoningDelta` count equals frontend `ChatReasoningAppended` count.
- `geppetto_to_frontend` exact matches equal total pairs in `04-correlation-quality-checks.sql`.
- `delivery_chain` rows show `transport_fanout=yes` and `frontend_parsed=yes` for early chat/reasoning events.
- `timeline_entities` contains the browser-visible thinking entity, such as `chat-msg-1:thinking:1`.

## Known Limitation

Provider IDs are currently present in Geppetto records but not directly in frontend `ReasoningUpdate` payloads. This means the current provider-to-browser query uses ordered reasoning deltas and exact chunk matching. A future task should add provider response/item/output/summary fields to `ReasoningUpdate`, allowing direct joins by provider identity.

## Failure Modes

- If `/api/debug/sessions/{id}/geppetto` returns zero records, check that web-chat was started with `--debug-api --geppetto-trace-level provider` and that the selected profile uses OpenAI Responses.
- If frontend records are zero, verify `localStorage['pinocchio.debugStream']='1'` was set before submitting the prompt.
- If `geppetto_summary_without_item_id` is non-zero, inspect `object_json` for whether the provider omitted `item_id` or Geppetto failed to propagate it.
- If `geppetto_to_frontend` mismatches are non-zero, inspect Geppetto normalization and Pinocchio/Sessionstream translation.
- If provider-to-frontend mismatches exist but Geppetto-to-frontend matches are exact, the difference is likely expected Geppetto normalization.

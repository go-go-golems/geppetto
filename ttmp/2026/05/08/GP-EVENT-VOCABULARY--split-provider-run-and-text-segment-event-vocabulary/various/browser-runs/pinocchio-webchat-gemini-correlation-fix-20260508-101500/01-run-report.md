---
Title: Pinocchio web-chat Gemini segment correlation fix validation
Ticket: GP-EVENT-VOCABULARY
Date: 2026-05-08
DocType: validation-report
Summary: Follow-up browser/SQLite validation after fixing duplicated generic segment correlation keys.
---

# Pinocchio web-chat Gemini segment correlation fix validation

## Environment

- Repo: `/home/manuel/workspaces/2026-05-02/use-sessionstream-coinvault/pinocchio`
- Dev profile: `web-chat-observe`
- Profile: `gemini-2.5-flash`
- Automation: `/home/manuel/workspaces/2026-05-02/use-sessionstream-coinvault/.playwright-mcp/pinocchio_webchat_e2e.py`

## Result

- Session: `6dad55bf-18ad-4a88-9890-38920cb41016`
- UI terminal check: pass
- Upload status: `200`
- SQLite size: `827,392` bytes
- `geppetto_records`: `31`
- `geppetto_provider_events`: `6`
- `geppetto_emitted_events`: `25`
- `geppetto_inference_results`: `1`
- `geppetto_segments`: `8`
- non-empty Geppetto correlation rows: `31`
- non-empty backend correlation rows: `26`
- non-empty frontend correlation rows: `13`

## Key assertion

The generic segment correlation key now matches the segment ID without duplicating the provider-call prefix:

`gemini:59cd2ab8-c506-4f03-ae5b-28a1b695caad:provider-call:0:segment:0:text`

The same value appears consistently on segment start, deltas, and finish rows.

## Artifacts

- `browser-result.json`
- `frontend-records.json`
- `debug.sqlite`
- `sqlite-analysis.json`
- `sqlite-counts.txt`
- `final-ui.png`
- `browser-run-summary.json`

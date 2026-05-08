---
Title: Pinocchio web-chat canonical correlation browser run
Ticket: GP-EVENT-VOCABULARY
Date: 2026-05-08
DocType: validation-report
Summary: Browser/SQLite run proving canonical Geppetto events and typed correlation through Pinocchio web-chat for OpenAI Responses, Claude, Gemini, and OpenAI-compatible Chat Completions profiles.
---

# Pinocchio web-chat canonical correlation browser run

## Environment

- Repo: `/home/manuel/workspaces/2026-05-02/use-sessionstream-coinvault/pinocchio`
- Dev profile: `web-chat-observe`
- Backend: `http://127.0.0.1:8092`
- Vite: `http://127.0.0.1:5174`
- Trace level: `provider`
- Frontend debug: `window.__pinocchioStreamDebug.enable()`
- Automation: `/home/manuel/workspaces/2026-05-02/use-sessionstream-coinvault/.playwright-mcp/pinocchio_webchat_e2e.py`

## Prompt

`Reply with exactly one short sentence containing {profile} and CORRELATION-SMOKE. Do not use markdown.`

## Results

| Profile | API family | Terminal UI check | SQLite bytes | Geppetto records | Provider events | Emitted canonical events | Inference rows | Segment rows | Geppetto corr rows | Backend corr rows | Frontend corr rows |
| --- | --- | --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| `gpt-5-nano` | OpenAI Responses | pass | 1,617,920 | 96 | 35 | 0 | 1 | 29 | 96 | 66 | 33 |
| `haiku` | Claude | pass | 585,728 | 35 | 13 | 0 | 1 | 9 | 22 | 26 | 13 |
| `gemini-2.5-flash` | Gemini | pass | 823,296 | 31 | 6 | 25 | 1 | 8 | 31 | 26 | 13 |
| `wafer-qwen3.5-397b` | OpenAI-compatible Chat Completions | body-timeout for final UI predicate; SQLite complete | 314,347,520 | 5,483 | 2,732 | 0 | 1 | 1,373 | 5,481 | 2,758 | 1,215 |

## Findings

- OpenAI Responses, Claude, Gemini, and OpenAI-compatible Chat Completions all produced `geppetto_inference_results` rows.
- Each family produced `geppetto_segments` rows with canonical lifecycle stages such as `segment_started`, `segment_updated`, and `segment_finished`.
- Correlation survived into backend and frontend records via canonical `CorrelationInfo`/correlation payloads.
- The first run exposed two observability gaps:
  - Pinocchio web-chat only wired debug observers for OpenAI Responses and Claude.
  - Gemini had canonical runtime events but no Geppetto observer integration.
- Fixes were committed as:
  - Geppetto `2e7f6c8 Add Gemini observability hooks`.
  - Pinocchio `8ba04fc Wire web chat observers for all providers`.
- The run also exposed a duplicated generic segment correlation key for Gemini (`provider-call:0:provider-call:0:segment...`). This was fixed in Geppetto `e1be7f2 Avoid nested segment correlation duplication` and validated by the follow-up Gemini artifact in `../pinocchio-webchat-gemini-correlation-fix-20260508-101500`.

## Artifacts

Per profile directories contain:

- `browser-result.json`
- `frontend-records.json`
- `debug.sqlite`
- `sqlite-analysis.json`
- `sqlite-counts.txt`
- `final-ui.png`

Summary artifact:

- `browser-run-summary.json`

## Caveats

- The `wafer-qwen3.5-397b` prompt triggered a very long reasoning trace and the automation timed out waiting for the UI terminal state; the SQLite export nevertheless contains a complete provider-call result row with `finish_class=completed`, a response ID, and canonical reasoning/text segment lifecycle rows.
- This report covers Pinocchio web-chat first. CoinVault full-trace browser/tool-use validation remains a separate Phase 13 task.

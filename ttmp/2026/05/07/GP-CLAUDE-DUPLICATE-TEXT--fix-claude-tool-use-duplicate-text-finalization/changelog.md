---
Title: Changelog
Ticket: GP-CLAUDE-DUPLICATE-TEXT
Status: active
Topics:
  - geppetto
  - claude
  - streaming
DocType: changelog
Intent: short-term
Owners:
  - manuel
Summary: Changelog for Claude tool-use duplicate text finalization fix.
LastUpdated: 2026-05-08T00:15:00-04:00
---

# Changelog

## 2026-05-08

- Fixed follow-up metadata regression from suppressing tool-use final events: `ClaudeEngine` now syncs merger metadata after every provider event, and the merger mirrors `message_delta` stop/usage metadata into the reconstructed response.
- Added regression coverage for metadata-only Claude tool-use events so persisted turn `inference_result` keeps `stop_reason=tool_use`, usage, cache-token fields, and duration without emitting duplicate transcript text.
- Verified the stronger Claude fix end-to-end in CoinVault with `PROFILE_SLUG=haiku devctl up --profile full-trace`; session `62ae60c5-e27a-4569-a2ed-4dd18bae0a80` produced `debug.sqlite`, `frontend-records.json`, and `final-ui.png` under the CoinVault observability browser-runs directory.
- Confirmed via SQLite that three `message_stop` events after `stop_reason=tool_use` produced no immediate Geppetto `final` publish; only the normal final `end_turn` produced `final`.
- Confirmed via backend pipeline SQLite that every `ChatInferenceFinished` had preceding real `ChatTokensDelta` content and none occurred directly after a tool event.
- Uploaded the bug-analysis and provider-to-chatapp event-semantics guides to reMarkable at `/ai/2026/05/07/GP-CLAUDE-DUPLICATE-TEXT` as `GP-CLAUDE-DUPLICATE-TEXT - event semantics guide`.
- Added `design/02-provider-to-chatapp-event-semantics-guide.md`, a detailed intern-facing analysis of correct event semantics and current mappings for OpenAI Chat Completions, OpenAI Responses, Anthropic Claude, Geppetto events, Pinocchio chatapp events, tool/reasoning plugins, and documentation gaps.
- Captured line-numbered source evidence under `sources/` for Geppetto provider engines, Pinocchio runtime sink/projections/plugins, and existing Pinocchio docs.
- Created `GP-CLAUDE-DUPLICATE-TEXT` docmgr ticket with a bug analysis and implementation guide.
- Added regression coverage for the Anthropic/Claude sequence `text block -> tool_use block -> message_delta stop_reason=tool_use -> message_stop`.
- Changed `ContentBlockMerger` so `message_delta` updates stop/usage metadata without publishing a textual partial.
- Changed `ContentBlockMerger` so `message_stop` emits no final event when the stop reason is `tool_use`, preventing downstream consumers from finalizing cached text as a duplicate assistant segment.
- Validated with:
  - `go test ./pkg/steps/ai/claude -run 'TestContentBlockMerger' -count=1`
  - `go test ./pkg/steps/ai/claude -count=1`
  - `go test ./pkg/steps/ai/... -count=1`

## 2026-05-07

- Initial workspace created.

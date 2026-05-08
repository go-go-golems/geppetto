---
Title: Tasks
Ticket: GP-CLAUDE-DUPLICATE-TEXT
Status: active
Topics:
  - geppetto
  - claude
  - streaming
DocType: tasks
Intent: short-term
Owners:
  - manuel
Summary: Task list for Claude tool-use duplicate text finalization fix.
LastUpdated: 2026-05-08T00:15:00-04:00
---

# Tasks

## Done

- [x] Upload bug-analysis and event-semantics guide bundle to reMarkable.
- [x] Write provider-to-Geppetto-to-Pinocchio event semantics intern guide.
- [x] Capture source evidence snippets for Geppetto and Pinocchio mapping code/docs.
- [x] Create docmgr ticket workspace and bug analysis guide.
- [x] Correlate CoinVault Haiku duplicate through provider, backend, transport, frontend, and timeline records.
- [x] Add regression coverage for Claude `tool_use` stop finalization.
- [x] Change Claude `message_delta` handling to metadata-only.
- [x] Change Claude `message_stop` with `stop_reason=tool_use` to emit no final event.
- [x] Run `go test ./pkg/steps/ai/claude -run 'TestContentBlockMerger' -count=1`.
- [x] Run `go test ./pkg/steps/ai/claude -count=1`.
- [x] Run `go test ./pkg/steps/ai/... -count=1`.

## TODO

- [ ] Re-run CoinVault Haiku browser smoke after Geppetto dependency/workspace integration picks up this fix.
- [ ] Consider a separate Anthropic provider correlation-key ticket; the Haiku artifact had provider records but no non-empty normalized `correlation_key` values.

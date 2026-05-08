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

---
Title: Changelog
Ticket: GP-EVENT-VOCABULARY
Status: active
Topics:
  - geppetto
  - pinocchio
  - streaming
  - observability
  - events
DocType: changelog
Intent: short-term
Owners:
  - manuel
Summary: Changelog for the provider/run/text segment vocabulary design ticket.
LastUpdated: 2026-05-08T05:55:00-04:00
---

# Changelog

## 2026-05-08

- Created `GP-EVENT-VOCABULARY` ticket workspace.
- Added `design-doc/01-provider-run-and-text-segment-event-vocabulary-design-guide.md`, a detailed intern-facing design guide for splitting chat run, provider call, text segment, reasoning segment, and tool lifecycles.
- Added a typed correlation envelope proposal covering `session_id`, `run_id`, `inference_id`, `turn_id`, `provider_call_id`, provider-native response/item IDs, indexes, segment IDs, tool IDs, and normalized `correlation_key` values.
- Captured line-numbered source evidence under `sources/` for Geppetto event definitions, provider mappings, observability records, Pinocchio runtime/protobuf/plugin code, and SQLite reconcile code.
- Added `reference/01-investigation-diary.md` documenting the ticket setup, source evidence, and design decisions.
- Validated the ticket with `docmgr doctor --ticket GP-EVENT-VOCABULARY --root ttmp --stale-after 30`.
- Uploaded the design bundle to reMarkable at `/ai/2026/05/08/GP-EVENT-VOCABULARY` as `GP-EVENT-VOCABULARY - event vocabulary design`.

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
LastUpdated: 2026-05-08T10:25:00-04:00
---

# Changelog

## 2026-05-08

- Ran Pinocchio web-chat browser/SQLite correlation validation across `gpt-5-nano`, `haiku`, `gemini-2.5-flash`, and `wafer-qwen3.5-397b`; archived artifacts and reports under `various/browser-runs/pinocchio-webchat-correlation-20260508-095442` and the Gemini follow-up `pinocchio-webchat-gemini-correlation-fix-20260508-101500`.
- Fixed the gaps found by the browser run: Geppetto `2e7f6c8` adds Gemini observer records, Pinocchio `8ba04fc` wires web-chat observers for OpenAI-compatible Chat Completions and Gemini, and Geppetto `e1be7f2` removes duplicated generic segment correlation keys.
- Committed Pinocchio SQLite debug export update in `0c5d343`: added canonical Geppetto provider-call result and segment lifecycle tables/views, extended stored correlation fields, and updated debug reconcile tests.
- Committed Pinocchio full canonical chatapp/UI cutover in `95fb755`: removed active `ChatInference*`/`ChatTokensDelta` payloads and UI wrappers, forwarded canonical backend payloads to the browser, preserved nested `CorrelationInfo` in timeline entities, and migrated legacy TUI/simple-chat consumers to canonical Geppetto text/reasoning/tool events.
- Cleaned up post-cutover Geppetto event leftovers: removed unused `ChatEventHandler` and `ToolEventAggregator`, renamed `text_events.go` to `error_events.go`, keyed structured-sink streams by canonical segment correlation, and refreshed stale comments.
- Deleted Geppetto legacy text/tool event structs, constants, constructors, and JSON decoder branches from active code; migrated structured sink, printers, tool aggregator, event-router interface, examples, and ttmp probe scripts to canonical events; saved validation under `various/legacy-event-deletion-validation.log`.
- Migrated Gemini and the local tool executor to canonical provider-call/text/tool lifecycle events, removed the remaining OpenAI Responses wrapper `NewStartEvent` emission, updated JS event collector handling for canonical tool execution/result events, added targeted tests, and saved validation output under `various/gemini-canonical-migration-validation.log`.
- Added a dedicated Gemini canonical event migration analysis, source evidence, and Phase 5B task list after the deletion scan found active Gemini and local tool-executor legacy event emissions.
- Migrated Pinocchio reasoning/tool plugins to canonical backend events with typed `CorrelationInfo`, including tool argument deltas.
- Migrated Pinocchio core chat runtime/projections to canonical run/provider-call/text events and removed active `ChatInference*`/`ChatTokensDelta` emission from `pkg/chatapp`.
- Added Pinocchio canonical `CorrelationInfo` and run/provider-call/text/reasoning/tool chatapp protobuf payloads, regenerated Go/TS bindings, and registered non-overlapping canonical base event schemas.
- Added canonical observability record kinds/stages for provider-call results and segment lifecycle rows, with typed correlation enrichment across Claude/OpenAI providers.
- Migrated OpenAI-compatible Chat Completions streaming to canonical provider-call/text/reasoning/tool events and updated observability/tests.
- Migrated OpenAI Responses streaming and non-streaming paths to canonical provider-call/text/reasoning/tool events and updated tests/observability.
- Migrated Claude content-block merger to canonical provider-call/text-segment/tool events and updated Claude observability/tests for typed correlation.
- Started Phase 2 by centralizing provider-call, segment, Chat Completions, Responses, and Claude correlation builders; OpenAI observability helpers now delegate to shared builders.
- Added `ValidateCanonicalEvent` to enforce canonical typed-correlation invariants for provider-call, segment, and tool events.
- Began Phase 1 by adding canonical Geppetto `Correlation` and run/provider-call/text/reasoning/tool event structs with JSON round-trip tests; `go test ./pkg/events/... -count=1` passed.
- Started Phase 0 implementation work: captured a legacy symbol/correlation inventory under `various/phase-0/legacy-symbol-inventory.txt` and saved baseline validation output under `various/phase-0/baseline-validation.log`; all baseline Geppetto, Pinocchio, CoinVault backend, and targeted frontend checks passed.
- Replaced the short TODO section in `tasks.md` with a detailed phase-by-phase hard-cutover migration checklist covering Geppetto, Pinocchio, CoinVault, SQLite export, trace browser updates, validation gates, commit strategy, and final acceptance criteria.
- Revised the primary design guide to assume a hard cutover instead of a compatibility migration: old `EventFinal`/`EventPartialCompletion`/`ChatInferenceFinished` vocabulary is removed, not aliased, and typed `Correlation` / `CorrelationInfo` is mandatory for all canonical events.
- Uploaded a new hard-cutover reMarkable copy named `GP-EVENT-VOCABULARY - hard cutover event vocabulary design`.
- Created `GP-EVENT-VOCABULARY` ticket workspace.
- Added `design-doc/01-provider-run-and-text-segment-event-vocabulary-design-guide.md`, a detailed intern-facing design guide for splitting chat run, provider call, text segment, reasoning segment, and tool lifecycles.
- Added a typed correlation envelope proposal covering `session_id`, `run_id`, `inference_id`, `turn_id`, `provider_call_id`, provider-native response/item IDs, indexes, segment IDs, tool IDs, and normalized `correlation_key` values.
- Captured line-numbered source evidence under `sources/` for Geppetto event definitions, provider mappings, observability records, Pinocchio runtime/protobuf/plugin code, and SQLite reconcile code.
- Added `reference/01-investigation-diary.md` documenting the ticket setup, source evidence, and design decisions.
- Validated the ticket with `docmgr doctor --ticket GP-EVENT-VOCABULARY --root ttmp --stale-after 30`.
- Uploaded the design bundle to reMarkable at `/ai/2026/05/08/GP-EVENT-VOCABULARY` as `GP-EVENT-VOCABULARY - event vocabulary design`.

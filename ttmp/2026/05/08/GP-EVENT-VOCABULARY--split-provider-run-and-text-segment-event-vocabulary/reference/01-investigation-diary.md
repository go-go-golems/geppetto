---
Title: Investigation diary
Ticket: GP-EVENT-VOCABULARY
Status: active
Topics:
  - geppetto
  - pinocchio
  - streaming
  - observability
  - events
DocType: reference
Intent: long-term
Owners:
  - manuel
RelatedFiles: []
ExternalSources: []
Summary: Chronological investigation and delivery notes for the provider/run/text segment event vocabulary design ticket.
LastUpdated: 2026-05-08T05:55:00-04:00
WhatFor: Preserve how the vocabulary design was researched, written, validated, and delivered.
WhenToUse: Read before implementing or updating GP-EVENT-VOCABULARY.
---

# Investigation diary

## 2026-05-08 05:20 — Created ticket and gathered source evidence

### What happened

Created a new docmgr ticket for the event vocabulary cleanup:

```bash
cd /home/manuel/workspaces/2026-05-02/use-sessionstream-coinvault/geppetto
docmgr ticket create-ticket \
  --ticket GP-EVENT-VOCABULARY \
  --title "Split provider, run, and text segment event vocabulary" \
  --topics geppetto,pinocchio,streaming,observability,events

docmgr doc add --ticket GP-EVENT-VOCABULARY \
  --doc-type design-doc \
  --title "Provider run and text segment event vocabulary design guide"

docmgr doc add --ticket GP-EVENT-VOCABULARY \
  --doc-type reference \
  --title "Investigation diary"
```

Ticket path:

```text
ttmp/2026/05/08/GP-EVENT-VOCABULARY--split-provider-run-and-text-segment-event-vocabulary/
```

I then captured line-numbered evidence excerpts under:

```text
ttmp/2026/05/08/GP-EVENT-VOCABULARY--split-provider-run-and-text-segment-event-vocabulary/sources/
```

The evidence covers:

- Geppetto event constants and text events;
- Geppetto observability records;
- Claude content-block merger and engine metadata syncing;
- OpenAI Chat Completions correlation-key code;
- OpenAI Responses correlation-key code;
- Pinocchio chatapp runtime sink and runtime inference;
- Pinocchio protobuf payloads and current correlation fields;
- Pinocchio reasoning/tool-call correlation helpers;
- Pinocchio debug reconcile SQLite schema/views.

### Why it matters

The user explicitly called out that the new design must pay attention to correlation IDs so that future code does not reconstruct relationships through metadata heuristics. The captured sources show that many identity fields already exist in traces and protobufs, but they are not yet a first-class event contract.

### What was tricky

The system already has pieces of the right answer in different layers:

- `EventMetadata` carries `SessionID`, `InferenceID`, and `TurnID`.
- OpenAI Chat Completions and Responses already build normalized `correlation_key` values.
- Pinocchio protobufs already carry provider fields on message/reasoning/tool updates.
- SQLite reconcile already indexes `correlation_key`.

The gap is that Geppetto transcript events still use broad names like `EventFinal`, and Pinocchio still extracts provider identity from `metadata.Extra` maps in several places.

## 2026-05-08 05:55 — Wrote intern-facing design guide

### What happened

Wrote the primary design document:

```text
design-doc/01-provider-run-and-text-segment-event-vocabulary-design-guide.md
```

The guide explains the current bug class and proposes a clean vocabulary:

```text
Chat run lifecycle
Provider call lifecycle
Text segment lifecycle
Reasoning segment lifecycle
Tool lifecycle
```

The guide pays special attention to typed correlation. It proposes a `Correlation` / `CorrelationInfo` envelope that carries:

```text
session_id
run_id / inference_id
turn_id
provider_call_id
provider_call_index
provider
model
response_id
item_id
output_index
summary_index
choice_index
content_block_index
segment_id
segment_index
segment_type
stream_kind
tool_call_id
tool_call_index
correlation_key
parent_correlation_key
```

### What worked

The Claude stream gives the clearest teaching example because Anthropic already separates provider envelope events from content block events:

```text
message_start / message_delta / message_stop = provider call lifecycle
content_block_start/delta/stop text = text segment lifecycle
content_block_start/delta/stop tool_use = tool lifecycle
```

That made it possible to state the core rule clearly:

```text
Provider envelope events are not transcript events.
```

### What didn't work

The current names `EventFinal` and `ChatInferenceFinished` cannot be made clear through comments alone. The design therefore recommends new explicit event names and treats old names as compatibility aliases.

### Code review instructions

When implementing this ticket, reviewers should look for these properties:

1. New provider-call events never create or finish text segments.
2. New text segment events always carry typed correlation, including `segment_id` and `correlation_key`.
3. New tool events always carry `provider_call_id`, `tool_call_id`, and a normalized `correlation_key`.
4. Pinocchio no longer needs to guess provider identity from `metadata.Extra` for new events.
5. Legacy `EventFinal` remains guarded so lifecycle-only finals cannot manufacture text rows.

## 2026-05-08 06:05 — Validated ticket and uploaded design bundle to reMarkable

### Validation

Ran docmgr doctor:

```bash
docmgr doctor --ticket GP-EVENT-VOCABULARY --root ttmp --stale-after 30
```

Result:

```text
GP-EVENT-VOCABULARY — All checks passed
```

### reMarkable upload

Verified remarquee and account state:

```bash
remarquee status
remarquee cloud account --non-interactive
```

Dry-ran and uploaded a bundle containing:

```text
index.md
design-doc/01-provider-run-and-text-segment-event-vocabulary-design-guide.md
reference/01-investigation-diary.md
tasks.md
changelog.md
```

Upload command shape:

```bash
remarquee upload bundle \
  index.md \
  design-doc/01-provider-run-and-text-segment-event-vocabulary-design-guide.md \
  reference/01-investigation-diary.md \
  tasks.md \
  changelog.md \
  --name "GP-EVENT-VOCABULARY - event vocabulary design" \
  --remote-dir "/ai/2026/05/08/GP-EVENT-VOCABULARY" \
  --toc-depth 2
```

Verified remote listing:

```text
[f]	GP-EVENT-VOCABULARY - event vocabulary design
```

## 2026-05-08 06:30 — Revised design for hard cutover and removed legacy compatibility framing

### What changed

After discussion, we decided the design should assume a hard cutover rather than a compatibility migration. I rewrote the primary design guide accordingly:

```text
design-doc/01-provider-run-and-text-segment-event-vocabulary-design-guide.md
```

The revised document now states that old names are removed rather than aliased:

```text
EventFinal              -> removed
EventPartialCompletion  -> removed
ChatInferenceStarted    -> removed
ChatTokensDelta         -> removed
ChatInferenceFinished   -> removed
```

The replacement vocabulary is canonical and mandatory:

```text
ChatRunStarted / ChatRunFinished
ProviderCallStarted / ProviderCallMetadataUpdated / ProviderCallFinished
TextSegmentStarted / TextDelta / TextSegmentFinished
ReasoningSegmentStarted / ReasoningDelta / ReasoningSegmentFinished
ToolCallStarted / ToolCallArgumentsDelta / ToolCallRequested / ToolResultReady
```

The correlation design was also made stricter. The revised guide now says every canonical event must carry typed `Correlation` / `CorrelationInfo`, and that new runtime logic must not route through `metadata.Extra`.

### Why

A compatibility migration would preserve the ambiguous model in runtime code. Since the old vocabulary caused the bug class, keeping it as an alias would make the design harder to reason about and harder to test. The hard cutover document is shorter, clearer, and makes the deletion checklist explicit.

### Upload note

The original reMarkable copy remains as the earlier migration-style design. I will upload a new copy with a different name so both versions are available for comparison.

## 2026-05-08 06:35 — Uploaded hard-cutover reMarkable copy

### Commands

```bash
remarquee upload bundle --dry-run \
  index.md \
  design-doc/01-provider-run-and-text-segment-event-vocabulary-design-guide.md \
  reference/01-investigation-diary.md \
  tasks.md \
  changelog.md \
  --name "GP-EVENT-VOCABULARY - hard cutover event vocabulary design" \
  --remote-dir "/ai/2026/05/08/GP-EVENT-VOCABULARY" \
  --toc-depth 2

remarquee upload bundle ...

remarquee cloud ls /ai/2026/05/08/GP-EVENT-VOCABULARY --long --non-interactive
```

### Result

```text
OK: uploaded GP-EVENT-VOCABULARY - hard cutover event vocabulary design.pdf -> /ai/2026/05/08/GP-EVENT-VOCABULARY
[f]	GP-EVENT-VOCABULARY - event vocabulary design
[f]	GP-EVENT-VOCABULARY - hard cutover event vocabulary design
```

## 2026-05-08 06:45 — Expanded tasks into phased hard-cutover migration checklist

### What changed

I rewrote `tasks.md` from a short TODO list into a detailed phase-by-phase hard-cutover migration checklist. The checklist now covers the full cross-repo path:

1. workspace and baseline gates;
2. Geppetto canonical event/correlation contracts;
3. correlation builders and invariants;
4. Claude migration;
5. OpenAI Responses migration;
6. OpenAI-compatible Chat Completions migration;
7. Geppetto inference-result and segment observability;
8. Pinocchio protobuf replacement;
9. Pinocchio runtime/projection replacement;
10. Pinocchio SQLite export updates;
11. CoinVault protobuf mirror/frontend parser updates;
12. trace browser and debug script updates;
13. deletion gates for old vocabulary;
14. browser/SQLite validation matrix;
15. documentation/reMarkable delivery;
16. suggested commit strategy;
17. final acceptance criteria.

### Important hard-cutover constraint

The tasks explicitly remove the old compatibility item that said to keep aliases for `EventFinal` and `ChatInferenceFinished`. The new task list states that the migration is complete only when no active runtime code emits or consumes the old vocabulary.

### Correlation emphasis

The checklist makes typed correlation mandatory at each layer. It includes specific checks for `Correlation` / `CorrelationInfo`, provider-call IDs, segment IDs, tool-call IDs, normalized `correlation_key`, and removal of routing through `metadata.Extra`.

## 2026-05-08 05:45 — Phase 0 baseline and legacy inventory

### What happened

Started the hard-cutover implementation task-by-task. The workspace was clean in Geppetto and Pinocchio. CoinVault still has unrelated existing working-tree state:

```text
M  ttmp/2026/05/07/SQLITE-TRACE-VERBS--design-sqlite-trace-inspection-verbs/scripts/serve/trace_browser_app.js
?? ttmp/2026/05/07/COINVAULT-OBSERVABILITY--add-observer-correlation-export-for-coinvault-web-chat/various/
```

Confirmed `go.work` points at the local repos needed for the migration:

```text
./2026-03-16--gec-rag
./geppetto
./glazed
./pinocchio
./sessionstream
```

Saved a legacy symbol/correlation inventory to:

```text
various/phase-0/legacy-symbol-inventory.txt
```

The inventory has 1252 lines and includes references for:

```text
EventFinal
EventPartialCompletion
EventTypeFinal
EventTypePartialCompletion
ChatInferenceStarted
ChatInferenceFinished
ChatTokensDelta
ChatInferenceStopped
response_id
correlation_key
metadata.Extra
choice_index
tool_call_index
```

### Baseline validation

Saved baseline validation output to:

```text
various/phase-0/baseline-validation.log
```

Commands and results:

```bash
cd geppetto && go test ./pkg/steps/ai/... -count=1
# ok

cd pinocchio && go test ./pkg/chatapp/... -count=1
# ok

cd 2026-03-16--gec-rag && go test ./internal/webchat ./cmd/coinvault/cmds -count=1
# ok

cd 2026-03-16--gec-rag/web && pnpm run typecheck
# ok

cd 2026-03-16--gec-rag/web && pnpm run test:unit -- src/ws/parsing.test.ts src/ws/wsManager.test.ts
# 8 files, 29 tests passed
```

### Next

Begin Phase 1 in Geppetto by adding the canonical `Correlation` envelope and new run/provider/text/reasoning/tool event structs before touching provider adapters.

## 2026-05-08 06:05 — Phase 1 started: canonical Geppetto event and correlation types

### What changed

Added the first canonical Geppetto event contracts:

```text
pkg/events/correlation.go
pkg/events/canonical_events.go
pkg/events/canonical_tool_events.go
pkg/events/canonical_events_test.go
```

`events.Correlation` now carries typed join identity for runtime, provider-call, provider item/block, transcript segment, tool, and normalized correlation-key scope. This is the first step toward removing routing through `metadata.Extra`.

Added canonical event types and constructors for:

```text
EventRunStarted
EventRunFinished
EventRunStopped
EventRunFailed
EventProviderCallStarted
EventProviderCallMetadataUpdated
EventProviderCallFinished
EventTextSegmentStarted
EventTextDelta
EventTextSegmentFinished
EventReasoningSegmentStarted
EventReasoningDelta
EventReasoningSegmentFinished
EventToolCallStarted
EventToolCallArgumentsDelta
EventToolCallRequested
EventToolExecutionStarted
EventToolResultReady
EventToolCallFinished
```

Also added canonical event type constants and `NewEventFromJson` decoding support.

### Validation

```bash
go test ./pkg/events -count=1
go test ./pkg/events/... -count=1
```

Both passed.

### Notes

This commit intentionally adds canonical types before removing legacy event producers/consumers. The hard cutover remains the final target; old events still exist until provider adapters and downstream consumers are migrated in later phases.

## 2026-05-08 06:25 — Phase 1 guard: canonical correlation validation

### What changed

Added `pkg/events/correlation_validation.go` with `ValidateCanonicalEvent(event Event) error`.

The validation guard currently enforces:

- every canonical event implements `CorrelatedEvent`;
- every canonical event has a typed `correlation_key`;
- provider-call events have `provider_call_id`;
- text/reasoning segment events have `segment_id`;
- tool lifecycle events have `tool_call_id`.

This gives provider adapters and downstream bridges a single guard to call during hard cutover and documents the invariant in executable tests.

### Validation

```bash
go test ./pkg/events -count=1
```

Passed.

## 2026-05-08 06:55 — Phase 2 started: centralized correlation builders

### What changed

Added centralized correlation builders in `pkg/events/correlation_builders.go`:

- `BuildProviderCallCorrelation` for response-ID-independent provider-call identity;
- `BuildSegmentCorrelation` for transcript segment identity nested under a provider call;
- `BuildChatCompletionsCorrelation` for OpenAI-compatible Chat Completions content/reasoning/tool streams;
- `BuildResponsesCorrelation` for OpenAI Responses response/item/output/summary streams;
- `BuildClaudeProviderCallCorrelation` and `BuildClaudeSegmentCorrelation` for Claude envelope/content-block identity.

Updated the existing OpenAI Chat Completions and OpenAI Responses observability correlation-key helpers to delegate to these shared builders rather than duplicating string construction locally.

### RunID decision

For the initial hard cutover implementation, `RunID` remains optional and the generic provider-call builder uses `RunID` when present, otherwise `InferenceID`. This keeps provider-call IDs stable before provider response IDs are known while avoiding a premature run-ID generator change. If a later runtime layer introduces a first-class run ID, the builder API already has a `runID` slot.

### Validation

```bash
go test ./pkg/events ./pkg/steps/ai/openai ./pkg/steps/ai/openai_responses -count=1
```

Passed.

### Metadata.Extra routing inventory

Captured a Phase 2 inventory at:

```text
various/phase-2/metadata-extra-routing-inventory.txt
```

The canonical `pkg/events` package has no `metadata.Extra` references. Remaining provider adapter references are legacy/debug metadata and will be retired as providers migrate to canonical events.

## 2026-05-08 07:20 — Phase 3 started: Claude emits canonical provider/segment/tool events

### What changed

Migrated `pkg/steps/ai/claude/content-block-merger.go` from legacy transcript events to canonical vocabulary:

- `message_start` now emits `EventProviderCallStarted`;
- `message_delta` now emits `EventProviderCallMetadataUpdated`;
- `message_stop` now emits `EventProviderCallFinished` for both `end_turn` and `tool_use`;
- text content-block start/delta/stop now emit `EventTextSegmentStarted`, `EventTextDelta`, and `EventTextSegmentFinished`;
- tool-use block start/input-json/stop now emit `EventToolCallStarted`, `EventToolCallArgumentsDelta`, and `EventToolCallRequested`.

The previous Claude duplicate-text guard is now represented semantically: a `tool_use` `message_stop` finishes the provider call, not a text segment. No empty text finalizer is manufactured.

Updated Claude observability so publish-started records read typed `events.Correlation` from canonical events rather than relying on `metadata.Extra` for correlation fields.

### Validation

```bash
go test ./pkg/steps/ai/claude -count=1
go test ./pkg/steps/ai/... -count=1
```

Both passed.

### Caveat

This is a Geppetto-side hard-cutover step. Pinocchio still needs its protobuf/runtime migration before browser consumers can process the new Claude event vocabulary end-to-end.

## 2026-05-08 08:15 — Phase 4: OpenAI Responses emits canonical provider/segment/tool events

### What changed

Migrated `pkg/steps/ai/openai_responses/streaming.go` to canonical event vocabulary:

- starts each provider call with `EventProviderCallStarted` using a response-ID-independent provider-call correlation;
- maps message output items to `EventTextSegmentStarted`, `EventTextDelta`, and `EventTextSegmentFinished`;
- maps reasoning output items and reasoning summaries/text to `EventReasoningSegmentStarted`, `EventReasoningDelta`, and `EventReasoningSegmentFinished`;
- maps function-call lifecycle to `EventToolCallStarted`, `EventToolCallArgumentsDelta`, and `EventToolCallRequested`;
- maps `response.completed` / end-of-stream completion to `EventProviderCallFinished` only;
- removed streaming use of legacy `EventPartialCompletion`, `EventThinkingPartial`, `EventToolCall`, and `EventFinal`.

Also migrated `pkg/steps/ai/openai_responses/nonstreaming.go` so non-streaming Responses outputs publish canonical provider-call, text, and reasoning events rather than a legacy final event.

Updated `pkg/steps/ai/openai_responses/observability.go` so publish records populate debug correlation fields from typed `events.Correlation` on canonical events.

### Validation

```bash
go test ./pkg/steps/ai/openai_responses -count=1
go test ./pkg/steps/ai/... -count=1
```

Both passed.

### Notes

OpenAI Responses still persists assistant/reasoning/tool blocks into the returned `turns.Turn` as before. The event stream, however, now separates provider-call lifecycle from transcript text/reasoning/tool segment lifecycle.

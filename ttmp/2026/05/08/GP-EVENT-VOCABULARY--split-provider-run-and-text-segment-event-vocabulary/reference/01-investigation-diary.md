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

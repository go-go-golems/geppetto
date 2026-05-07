---
Title: 'Textbook Report: Geppetto Provider/Event Observability Design Assessment'
Ticket: GP-OBSERVABILITY
Status: active
Topics:
    - events
    - inference
    - streaming
    - openai
    - glazed
    - sqlite
DocType: analysis
Intent: long-term
Owners:
    - manuel
RelatedFiles:
    - Path: ../../../../../../../pinocchio/cmd/web-chat/app/debug_reconcile_db.go
      Note: SQLite reconcile export boundary discussed in the report
    - Path: ../../../../../../../pinocchio/cmd/web-chat/app/debug_recorder.go
      Note: Pinocchio app-owned recorder boundary discussed in the report
    - Path: pkg/events/chat-events.go
      Note: EventInfo and EventMetadata concepts explained in the report
    - Path: pkg/steps/ai/openai_responses/engine.go
      Note: Provider stream routing and Geppetto event emission examples discussed in the report
ExternalSources: []
Summary: 'Textbook-style explanation and assessment of the GP-OBSERVABILITY design: concepts, strengths, overengineering risks, future failure modes, and recommended implementation slices.'
LastUpdated: 2026-05-07T09:41:07.737671509-04:00
WhatFor: Teach the base concepts behind Geppetto provider/event observability and guide a scoped implementation that preserves useful evidence without overbuilding the first version.
WhenToUse: Read before implementing or reviewing GP-OBSERVABILITY, especially when deciding what belongs in Geppetto, Pinocchio, trace records, or SQLite exports.
---


# Textbook Report: Geppetto Provider/Event Observability Design Assessment

## 1. Purpose of this report

This report explains the `GP-OBSERVABILITY` design as a small textbook. The goal is not merely to restate the ticket. The goal is to make the design understandable enough that an engineer can implement it without copying every proposed detail blindly.

By the end, the reader should understand four things:

- Why Geppetto needs its own observability layer below Pinocchio and Sessionstream.
- Which parts of the design are structurally sound and should be preserved.
- Which parts are likely too broad for the first implementation and should be narrowed.
- Where future issues are likely to arise: performance, privacy, schema coupling, ordering, and retention.

The core judgment is simple: the design is directionally right, but the first implementation should be smaller than the end-state document suggests. We should first prove that provider identity survives from the OpenAI Responses stream into Geppetto events and observer records. Once that path is working, Pinocchio endpoints and SQLite views can be built on real evidence rather than guessed schemas.

## 2. The system in one page

Geppetto, Pinocchio, Sessionstream, and the browser debug layer form a pipeline. Each layer sees a different representation of the same inference.

```text
Provider stream
  -> Geppetto provider engine
  -> Geppetto events.Event
  -> Pinocchio chat plugins
  -> Sessionstream canonical events
  -> Sessionstream UI events and timeline entities
  -> WebSocket frames
  -> Browser state and rendered UI
```

A bug can happen at any boundary. The same reasoning summary may be present in the provider stream, lost in Geppetto metadata, translated incorrectly by Pinocchio, projected twice by Sessionstream, delivered twice over WebSocket, or rendered twice by the browser. Without observability at the right layer, we can only guess where the error entered.

The existing debug work already covers much of the lower half:

| Layer | Existing visibility | What it can answer | What it cannot answer |
|---|---|---|---|
| Sessionstream Hub | Pipeline records | Which canonical events were appended, projected, and fanned out | What decoded provider object caused them |
| WebSocket transport | Transport records | Which UI events were sent to which sockets | Whether Geppetto preserved provider IDs |
| Browser debug layer | Raw/parsed frames, UI mutations | What the browser received and rendered | Whether provider stream ordering was already wrong |
| SQLite reconcile export | Cross-layer artifact | Backend/frontend/timeline comparison | Provider-to-Geppetto correlation |

`GP-OBSERVABILITY` adds the missing upper half:

```text
Provider stream
  -> decoded provider event
  -> Geppetto normalization
  -> emitted Geppetto event
  -> downstream publish
```

This is the correct place to answer questions like:

- Did OpenAI send `item_id` for the reasoning summary?
- Did the provider send `response.reasoning_summary_text.delta` before or after `response.output_item.done`?
- Did Geppetto normalize a delta into an empty string?
- Did Geppetto emit `thinking-ended` before a late final `reasoning-summary`?
- Did Geppetto put provider IDs into `EventInfo.Data`, or were they already lost before Pinocchio saw the event?

## 3. Base concept: provider events are not application events

A provider event is a fact reported by an external API. A Geppetto event is a fact reported by our inference engine. They are related, but they are not the same thing.

An OpenAI Responses stream may send a payload shaped like this:

```json
{
  "type": "response.reasoning_summary_text.delta",
  "response_id": "resp_123",
  "item_id": "rs_456",
  "output_index": 0,
  "summary_index": 0,
  "delta": "The user is asking for..."
}
```

Geppetto may translate that into an event shaped conceptually like this:

```text
EventThinkingPartial
  message_id: <geppetto message uuid>
  session_id: <session id>
  inference_id: <inference id>
  turn_id: <turn id>
  delta: "The user is asking for..."
  text_so_far: "The user is asking for..."
```

That translation is useful. Pinocchio should not need to understand every provider's streaming protocol. But translation can also lose evidence. The provider event may carry `response_id`, `item_id`, `output_index`, and `summary_index`; the Geppetto event may only carry a message UUID and text. Once the provider identity is dropped, downstream layers cannot reconstruct it.

This is why the design has two related goals:

1. Emit observer records that show what happened at the Geppetto/provider boundary.
2. Preserve important provider IDs in durable Geppetto event data when those IDs have semantic value.

The first goal helps debugging when tracing is enabled. The second goal helps correctness and correlation even when tracing is disabled.

## 4. Base concept: observability records are evidence, not behavior

An observer record should describe what happened. It should not decide what happens next.

This distinction matters because high-frequency inference debugging is tempting. Once a trace record has all the fields needed to understand the provider stream, it is easy to let application behavior start depending on those records. That would be a mistake. Geppetto should still publish normal events. Pinocchio should still translate those events through its chat plugins. Sessionstream should still project canonical events into UI events and timeline entities.

The observer path should be a side channel:

```text
Normal path:
  provider event -> Geppetto event -> Pinocchio -> Sessionstream -> browser

Observer path:
  provider event -> Geppetto Record -> Pinocchio debug recorder -> debug endpoint / SQLite
```

The normal path must remain correct when the observer is nil, disabled, panicking, slow, or absent. That leads to two practical rules:

- Observer notifications must be panic-safe.
- The `off` trace level must have near-zero overhead.

A good observer implementation is like a flight recorder. It records the flight, but the plane does not need the recorder to fly.

## 5. Base concept: query columns and JSON evidence answer different questions

When debugging a streaming bug, the first instinct is to preserve the lower-level provider object. That instinct is correct: many of the hardest bugs are caused by missing fields, provider-specific shape differences, or a wrong interpretation of decoded provider data. If we only store normalized fields, we may accidentally preserve the mistake rather than the evidence needed to find it.

The lesson is to separate two kinds of evidence. Correlation fields are the queryable index. JSON evidence is the inspectable record of what each layer believed it was handling. For the first implementation, we do not need the raw SSE string. We do need the decoded provider object, the emitted Geppetto event, and the Geppetto metadata.

The most valuable first-class query data are correlation fields:

| Field | Why it matters |
|---|---|
| `session_id` | Connects records to the web-chat session. |
| `inference_id` | Connects records to one model invocation. |
| `turn_id` | Connects records to persisted turn state. |
| `message_id` | Connects records to Geppetto event metadata. |
| `response_id` | Connects records to one provider response. |
| `item_id` | Connects reasoning/message/tool output items across provider events. |
| `output_index` | Distinguishes provider output items in sequence. |
| `summary_index` | Distinguishes multiple reasoning summary parts. |
| `event_type` | Explains which provider or Geppetto event occurred. |
| `stage` | Explains where in the pipeline the record was captured. |

A JSON blob can answer many questions, but only after a human opens it. A stable `item_id` column can answer a first-pass question with a query:

```sql
SELECT ts, stage, event_type, item_id, summary_index
FROM geppetto_records
WHERE item_id = 'rs_456'
ORDER BY ts, id;
```

The design should therefore treat correlation fields as the stable query contract and JSON payloads as required diagnostic evidence when trace level asks for them. The JSON payloads should be capped and redacted deliberately; they should not be omitted merely because they are less convenient to query.

### 5.1 What `object_json`, `event_json`, and `metadata_json` mean

These fields represent different distances from the provider stream. They are not duplicates; they answer different debugging questions.

| Field | What it is | Why it matters |
|---|---|---|
| `object_json` | The decoded provider object after `json.Unmarshal`, usually marshaled back to JSON after optional redaction/capping. | Shows what Geppetto believed the provider sent after decoding. This is the main evidence for missing fields, nested shape mismatches, numeric conversion quirks, OpenAI-compatible provider differences, and fields ignored by typed extraction helpers. |
| `event_json` | The Geppetto event payload that was constructed or published, encoded as JSON. | Shows what Geppetto emitted downstream. Comparing `object_json` to `event_json` catches translation bugs, missing provider IDs, wrong event names, wrong data keys, and lost data. |
| `metadata_json` | The Geppetto `EventMetadata` attached to the event, encoded as JSON. | Shows session/inference/turn/message correlation, usage, stop reason, settings metadata, runtime attribution, and `Extra` fields. This catches bugs where the event payload is correct but metadata enrichment or correlation is missing. |

A useful trace record may contain normalized columns plus decoded provider evidence:

```json
{
  "stage": "provider_routed_event",
  "eventType": "response.reasoning_summary_text.delta",
  "responseId": "resp_123",
  "itemId": "rs_456",
  "summaryIndex": 0,
  "objectJson": {
    "type": "response.reasoning_summary_text.delta",
    "response_id": "resp_123",
    "item_id": "rs_456",
    "summary_index": 0,
    "delta": "The user is asking for..."
  }
}
```

A publish-stage record should preserve the downstream event and metadata as well:

```json
{
  "stage": "geppetto_publish_done",
  "eventType": "reasoning-summary",
  "responseId": "resp_123",
  "itemId": "rs_456",
  "eventJson": {
    "message": "reasoning-summary",
    "data": {
      "text": "The user is asking for...",
      "response_id": "resp_123",
      "item_id": "rs_456"
    }
  },
  "metadataJson": {
    "session_id": "session_1",
    "inference_id": "inference_1",
    "turn_id": "turn_1"
  }
}
```

The columns make the record findable. The JSON evidence makes it falsifiable.

## 6. The good parts of the design

### 6.1 The ownership split is correct

The design's strongest decision is the ownership split:

| Component | Owns | Should not own |
|---|---|---|
| Geppetto | Neutral observer API, provider/event instrumentation, typed trace config | Pinocchio debug endpoints or SQLite schema |
| Pinocchio | Debug recorder, HTTP endpoints, SQLite export, web-chat wiring | Provider-specific stream parsing inside app code |
| Sessionstream | Canonical event pipeline and transport observers | Provider stream correlation |
| Browser | Frontend debug capture and UI state evidence | Provider/event normalization logic |

This separation keeps Geppetto reusable. It also makes Pinocchio's debug export more honest: Pinocchio records what Geppetto reports; it does not reach into provider internals itself.

If this ownership boundary is preserved, the design can grow. Other apps can attach their own Geppetto observer without inheriting Pinocchio's debug API. Other providers can emit records without changing Sessionstream. SQLite export can evolve without contaminating Geppetto's core engine.

### 6.2 Glazed configuration is the right path

The design explicitly rejects raw `os.Getenv()` configuration. That is good.

Trace capture is runtime instrumentation. It should be visible in the command schema, typed into a settings struct, and decoded through the same configuration system as the rest of the application. A user should be able to discover it with command help or schema output.

The proposed settings are well chosen:

```text
geppetto-trace-level
geppetto-trace-max-records
geppetto-trace-max-payload-bytes
geppetto-trace-redact-provider-data
```

The trace level says how much to capture. The max records setting bounds retention. The max payload bytes setting bounds memory and export size. The redaction flag controls sensitivity. Together they define an operational envelope rather than a hidden debug switch.

### 6.3 Trace levels are easier to operate than many booleans

A single level is better than many separate flags. Operators should not need to know five internal instrumentation names to get useful traces.

| Level | Meaning | Expected cost |
|---|---|---|
| `off` | No records | Near zero |
| `events` | Geppetto-level events and publish stages | Low |
| `provider` | Provider event names and stable provider metadata | Medium |
| `raw` | Reserved for future raw stream previews; not needed for the first implementation | High and sensitive |

This hierarchy creates a natural escalation path. Start with `events`; move to `provider` when lower-level provider shape matters. Reserve `raw` for a future mode if decoded `object_json` is not enough to diagnose malformed frames or parser-level failures.

### 6.4 Provider IDs in `EventInfo.Data` are more important than debug records

The design recommends adding provider identity to events such as `thinking-ended` and `reasoning-summary`. This is exactly right.

Consider the final reasoning summary today:

```go
events.NewInfoEvent(metadata, "reasoning-summary", map[string]any{
    "text": summaryBuf.String(),
})
```

A better event carries provider identity when available:

```go
events.NewInfoEvent(metadata, "reasoning-summary", map[string]any{
    "text": summaryBuf.String(),
    "provider": "openai_responses",
    "response_id": currentResponseID,
    "item_id": lastReasoningItemID,
    "output_index": outputIndex,
    "summary_index": summaryIndex,
})
```

This is not merely observability. It is durable semantics. If downstream systems need to correlate a summary to a reasoning item, the event itself should carry that fact.

### 6.5 Panic-safe notification is the right failure model

The proposed `Notify` helper is small but important:

```go
func Notify(ctx context.Context, obs Observer, rec Record) {
    if obs == nil {
        return
    }
    if rec.Timestamp.IsZero() {
        rec.Timestamp = time.Now().UTC()
    }
    defer func() {
        _ = recover()
    }()
    obs.OnGeppettoRecord(ctx, rec)
}
```

This expresses the correct contract. Observer failures are debug failures, not inference failures. We may later choose to count observer panics somewhere, but we should not let them interrupt inference.

## 7. What is over-engineered or premature

The design is a strong end-state map. The risk is implementing the map all at once.

### 7.1 The proposed `Record` type is probably too wide for version one

The proposed record contains both stable fields and diagnostic blobs:

```text
Stable/queryable fields:
  timestamp, provider, model, session_id, inference_id, turn_id,
  message_id, stage, event_type, response_id, item_id,
  output_index, summary_index, delta_len, buffer_len, error

Diagnostic payloads:
  object_json, event_json, metadata_json
```

The stable fields should become the query contract. The diagnostic payloads should be captured in the appropriate trace modes, but downstream automated behavior should still avoid depending on arbitrary provider-specific JSON shape. Humans need the JSON for debugging; stable columns and explicit extracted fields should power joins and views.

A narrower first implementation might define the full type but only fill a small subset:

```text
provider, model, session_id, inference_id, turn_id, message_id,
stage, event_type, response_id, item_id, output_index, summary_index,
delta_len, normalized_delta_len, buffer_len, error
```

Then add JSON payload capture behind `provider`/publish instrumentation, with tests that prove capping and redaction. Do not add `raw_preview` in the first implementation.

### 7.2 The SQLite export plan should follow real traces

The proposed SQLite schema includes multiple tables and views. The direction is good, but views should not be designed entirely from imagination.

A minimal first export should answer three questions:

1. Are Geppetto records present for the session?
2. Do provider events and emitted Geppetto events share stable correlation IDs?
3. Did the export preserve enough provider/event/metadata JSON for manual inspection?

That suggests starting with:

```sql
CREATE TABLE geppetto_records (... stable columns ..., record_json TEXT NOT NULL);
```

Possibly add:

```sql
CREATE TABLE geppetto_provider_events (...);
CREATE TABLE geppetto_emitted_events (...);
```

But defer complex cross-layer views until we have real downloaded artifacts. The view `reasoning_provider_to_timeline` is desirable, but it depends on whether provider IDs actually propagate into Sessionstream payloads and timeline entities. If they do not, the view will be misleading.

### 7.3 Raw string capture is not needed for the first implementation

The design can diagnose the current class of bugs without storing the original SSE string. The important evidence is the decoded provider object that Geppetto actually routed. That `object_json` is enough to answer most missing-field and wrong-interpretation questions: did the decoded provider object contain the field, where was it nested, what type did it have, and did our extraction logic look in the right place?

For the first implementation, use these rules:

- `provider` level stores decoded provider `object_json` after redaction and capping.
- publish-stage records store `event_json` and `metadata_json`, also capped if needed.
- no `raw_preview` field is needed now.
- `raw` trace level may remain a reserved future concept, but should not drive first-slice implementation.
- SQLite exports should preserve `object_json`, `event_json`, and `metadata_json` because enrichment and translation can be buggy too.

### 7.4 Too many stages can create noise

The proposed stage list is comprehensive. A smaller first list is easier to understand and test:

```text
provider_routed_event
provider_normalize_delta
geppetto_publish_started
geppetto_publish_done
geppetto_publish_error
```

This set captures the essential path:

- provider event entered routing;
- delta normalization happened;
- Geppetto published an event;
- publishing succeeded or failed.

Stages such as `provider_raw_frame`, `provider_stream_connected`, and `geppetto_event_constructed` can be added later if a concrete debugging session needs them.

## 8. Future issues to expect

### 8.1 Performance and allocation pressure

High-frequency traces can be expensive even if each record looks small. A model may produce many text deltas, reasoning deltas, summary deltas, tool argument deltas, and output events. If each event causes map cloning, JSON marshaling, timestamp allocation, and recorder locking, debugging can change the behavior being debugged.

The disabled path should avoid work:

```go
if e.observer == nil || e.obsConfig.Level == TraceOff {
    return
}
```

The `events` path should avoid provider object marshaling. The `provider` path should cap payloads. The `raw` path should be explicitly expensive and bounded.

Future performance tests should measure at least:

- observer disabled;
- observer enabled at `events`;
- observer enabled at `provider` with payload capping;
- recorder behavior under thousands of records.

### 8.2 Redaction will be harder than key filtering

Simple key redaction is necessary but not sufficient. It is easy to redact obvious fields:

```text
authorization
api_key
encrypted_content
```

The harder problem is content. The fields `delta`, `text`, `arguments`, `input`, and `content` may contain sensitive user or model data. If `provider` level includes full provider objects by default, then the trace may still contain sensitive content even when `encrypted_content` is redacted.

A better privacy model is explicit:

| Data kind | Default behavior |
|---|---|
| Provider IDs and indexes | Keep |
| Event type and stage | Keep |
| Delta lengths and buffer lengths | Keep |
| Provider text deltas in `object_json` | Cap aggressively, with explicit truncation markers if truncated |
| Tool arguments | Redact or cap by default unless explicitly needed for the debugging session |
| Encrypted reasoning | Replace with marker and byte length |
| Authorization/API keys | Never store |

The report task added to the ticket should become a short policy document before provider object/event/metadata JSON is broadly exposed through debug endpoints or SQLite exports.

### 8.3 Retention may cause debug records to evict each other

Pinocchio's current recorder uses a bounded record list. Once Geppetto high-frequency records are added, that single cap may fill quickly. If provider delta records evict pipeline and transport records, the SQLite export will lose the downstream evidence needed for reconciliation.

This is a subtle failure mode. Debugging will appear to be enabled, but the artifact will be incomplete.

Possible mitigations:

- per-kind caps;
- per-session caps;
- dropped-record counters;
- summary records when detailed records are dropped;
- max records configured from `geppetto-trace-max-records` but applied carefully.

### 8.4 Timestamp order is not enough

The full pipeline has several ordering systems:

| Layer | Ordering mechanism |
|---|---|
| Provider stream | SSE order |
| Geppetto observer | timestamp and call order |
| Sessionstream | ordinals |
| WebSocket | send order and connection behavior |
| Browser | receive time and mutation order |

Timestamps are useful, but they are not authoritative across layers. Clock granularity, buffering, goroutine scheduling, and JSON export order can all confuse timestamp-based reasoning.

The design should prefer stable order/correlation fields where available:

- provider `output_index` and `summary_index`;
- Sessionstream ordinals;
- recorder append order;
- SQLite autoincrement IDs;
- explicit event names and IDs.

### 8.5 Provider generality may erode over time

The first target is OpenAI Responses. That is appropriate because the motivating bug involves reasoning summaries and OpenAI-style provider identity. But the observer package should not become an OpenAI Responses schema hidden inside a generic package.

Fields like `response_id`, `item_id`, `output_index`, and `summary_index` are acceptable because they are optional and broadly understandable. But if future work adds many provider-specific top-level fields, the API will become harder to use for Claude, Gemini, Ollama, or other providers.

A good rule is:

- Put common correlation fields at top level.
- Put provider-specific extra fields in a bounded provider metadata map or JSON attachment.
- Do not add a top-level field until at least two downstream queries need it.

### 8.6 Tests may require better engine seams

OpenAI Responses streaming tests may be awkward if the engine cannot easily use a fake HTTP client or fake stream. The current engine already has substantial logic in one SSE loop. Adding instrumentation directly inside that loop is practical, but tests need a way to feed deterministic events.

Good tests should not hit the real provider. They should feed events like:

```text
event: response.reasoning_summary_text.delta
data: {"type":"response.reasoning_summary_text.delta","response_id":"resp_1","item_id":"rs_1","summary_index":0,"delta":"hello"}
```

Then assert:

- the observer captured the provider event type;
- the observer captured `item_id`;
- the final `reasoning-summary` event data includes provider IDs;
- a panicking observer does not fail inference.

If that is hard to test, the implementation should first add a small internal seam rather than layering observability onto untestable code.

## 9. Recommended implementation sequence

The design guide lists six phases. I recommend executing them as narrower slices.

### Slice 1: Minimal Geppetto observability package

Build the neutral core first:

```text
geppetto/pkg/observability/observer.go
geppetto/pkg/observability/config.go
geppetto/pkg/observability/redact.go
```

Include:

- `TraceLevel`;
- `ParseTraceLevel`;
- `Config`;
- `Record`;
- `Observer`;
- `Notify`;
- redaction/capping helpers;
- unit tests.

Acceptance criteria:

- `ParseTraceLevel` accepts `off`, `events`, `provider`, and `raw`.
- invalid levels return useful errors.
- `Notify` handles nil observers.
- `Notify` recovers from observer panic.
- capping and redaction are deterministic and tested.

### Slice 2: Glazed observability section

Add:

```text
geppetto/pkg/cli/bootstrap/inference_observability.go
```

Expose:

```text
geppetto-trace-level
geppetto-trace-max-records
geppetto-trace-max-payload-bytes
geppetto-trace-redact-provider-data
```

Acceptance criteria:

- defaults are safe;
- trace level default is `off`;
- redaction default is true;
- max payload bytes has a reasonable bounded default;
- the settings struct decodes through Glazed.

### Slice 3: OpenAI Responses proof path

Instrument the smallest useful path:

- provider routed event;
- reasoning summary delta normalization;
- reasoning text delta normalization;
- `thinking-ended` event data;
- final `reasoning-summary` event data;
- publish start/done/error if the existing helper makes that easy.

Acceptance criteria:

- provider IDs are captured in observer records;
- provider IDs are preserved in `EventInfo.Data` when available;
- `off` records nothing;
- `events` does not include provider object JSON;
- `provider` includes stable provider metadata and redacted/capped payloads only if configured.

### Slice 4: Pinocchio recorder and endpoint

Only after Geppetto records exist, extend Pinocchio:

```text
pinocchio/cmd/web-chat/app/debug_recorder.go
pinocchio/cmd/web-chat/app/server_debug.go
pinocchio/cmd/web-chat/main.go
```

Acceptance criteria:

- `StreamDebugRecorder` implements `OnGeppettoRecord`;
- debug endpoint returns Geppetto records for a session;
- recorder respects max records;
- trace settings are decoded through Glazed, not environment variables.

### Slice 5: Minimal SQLite export

Start with storage before clever views:

- `geppetto_records` table;
- maybe provider/emitted event helper tables;
- counts in `meta`;
- basic indexes on `session_id`, `stage`, `event_type`, and `item_id`.

Acceptance criteria:

- downloaded SQLite contains Geppetto records;
- records can be joined manually by `session_id` and `item_id`;
- export remains useful even if no Geppetto trace is present.

### Slice 6: Views from real artifacts

After live validation, add views that answer proven questions:

- reasoning sequence;
- summaries without item IDs;
- publish errors;
- provider-to-timeline correlation if provider IDs propagate downstream.

This avoids freezing speculative views before the data shape is known.

## 10. What to review carefully

A reviewer should focus on the boundaries, not just the code compiles.

### 10.1 Geppetto API boundary

Ask:

- Is the observer package app-neutral?
- Are stable fields separate from diagnostic payloads?
- Does the observer path remain optional?
- Does `off` avoid unnecessary allocations?

### 10.2 Privacy boundary

Ask:

- Can sensitive user/model content enter `object_json`, `event_json`, or `metadata_json` by default?
- Are payloads capped before storage?
- Are sensitive keys redacted recursively?
- Does SQLite export make sensitivity obvious?

### 10.3 Runtime boundary

Ask:

- Can observer panic break inference?
- Can recorder lock contention slow streaming too much?
- Can high-frequency records evict lower-frequency pipeline/transport evidence?

### 10.4 Schema boundary

Ask:

- Are SQLite columns based on stable fields?
- Are JSON blobs treated as inspection aids rather than required schema?
- Do views depend on fields that are always present?
- Are missing provider IDs visible rather than silently ignored?

## 11. A concrete example: the duplicate thinking block

The motivating bug is the duplicate thinking block. The browser showed two thinking blocks because a late `reasoning-summary` was treated as a new reasoning segment after `thinking-ended` closed the first one.

Sessionstream could show this symptom:

```text
ChatMessage|chat-msg-1:thinking:1|created=4|last=700
ChatMessage|chat-msg-1:thinking:2|created=1659|last=1659
```

That proves two timeline entities existed. It does not prove why.

Geppetto observability should let us reconstruct the upper half:

```text
1. provider_routed_event
   event_type=response.reasoning_summary_text.delta
   response_id=resp_1 item_id=rs_1 summary_index=0

2. provider_normalize_delta
   event_type=response.reasoning_summary_text.delta
   item_id=rs_1 delta_len=45 normalized_delta_len=45 buffer_len=45

3. geppetto_publish_done
   event_type=thinking-partial
   message_id=... item_id=rs_1

4. provider_routed_event
   event_type=response.output_item.done
   item_id=rs_1 output_index=0

5. geppetto_publish_done
   event_type=info message=thinking-ended
   item_id=rs_1 output_index=0

6. geppetto_publish_done
   event_type=info message=reasoning-summary
   item_id=rs_1 summary_index=0
```

If records 5 and 6 share `item_id`, then Pinocchio should be able to attach the final summary to the same reasoning segment. If record 6 lacks `item_id`, then the loss happened in Geppetto or provider parsing. If Geppetto has `item_id` but Sessionstream does not, the loss happened in Pinocchio translation.

That is the value of this ticket: it turns a visual symptom into a boundary-by-boundary proof.

## 12. Key takeaways

- Geppetto observability fills the gap between provider streams and Pinocchio/Sessionstream debug records.
- Provider events and Geppetto events are different representations; useful provider identity must survive translation.
- Observer records should be evidence, not behavior. Inference must work correctly when observers are disabled or broken.
- Stable correlation fields make records queryable, while `object_json`, `event_json`, and `metadata_json` make records inspectable and falsifiable.
- The ownership split in the design is strong: Geppetto emits neutral records, Pinocchio records and exports them.
- The first implementation should be narrower than the full design. Prove the provider-to-Geppetto path before building all SQLite views.
- Raw string capture is not needed in the first implementation; decoded provider objects and emitted event/metadata JSON are the required diagnostic payloads for now.
- High-frequency traces can distort the system if disabled-path overhead, allocation behavior, retention, and recorder contention are not tested.
- SQLite views should be derived from real trace artifacts, not only from planned schemas.

## 13. Final recommendation

Implement the design, but implement it in vertical slices. The right first milestone is not a complete SQLite reconciliation universe. The right first milestone is a tested Geppetto observer path that answers one question reliably:

> For an OpenAI Responses reasoning stream, can we see the provider event, the provider IDs, the normalization step, and the emitted Geppetto event that carries those IDs forward?

If the answer is yes, then Pinocchio debug endpoints and SQLite export become straightforward. If the answer is no, the system has learned something more important than any table or view: the evidence is lost before it reaches the application.

That is the engineering principle for this ticket:

```text
Preserve evidence at each boundary.
Make the evidence cheap when disabled.
Make sensitive evidence explicit when enabled.
Build durable queries only after real traces prove the schema.
```

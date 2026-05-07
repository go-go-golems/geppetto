---
Title: Geppetto Code Review and Intern Onboarding Guide
Ticket: GP-CODE-REVIEW
Status: active
Topics:
    - code-review
    - observability
    - architecture
    - cleanup
    - intern-onboarding
DocType: design-doc
Intent: long-term
Owners:
    - manuel
RelatedFiles:
    - Path: ../../../../../../../pinocchio/cmd/web-chat/app/debug_reconcile_db.go
      Note: Downstream SQLite export/correlation boundary reviewed
    - Path: ../../../../../../../pinocchio/cmd/web-chat/app/debug_recorder.go
      Note: Downstream app-owned Geppetto observer adapter boundary
    - Path: README.md
      Note: High-level Geppetto architecture and package map used for intern orientation
    - Path: pkg/cli/bootstrap/inference_observability.go
      Note: Glazed observability settings and MaxRecords/API-boundary review
    - Path: pkg/events/chat-events.go
      Note: Event taxonomy
    - Path: pkg/inference/engine/engine.go
      Note: Minimal provider Engine interface explained in guide
    - Path: pkg/inference/engine/factory/factory.go
      Note: Provider factory and OpenAI Responses option injection seam
    - Path: pkg/inference/session/session.go
      Note: Session lifecycle and single-active-inference invariant explained in guide
    - Path: pkg/observability/config.go
      Note: Trace level and observability config API review
    - Path: pkg/observability/json.go
      Note: Evidence JSON redaction/capping review and sanitizer issue
    - Path: pkg/observability/observer.go
      Note: Neutral Record/Observer/Notify API review
    - Path: pkg/steps/ai/openai_responses/engine.go
      Note: Main OpenAI Responses stream engine and large-file refactor target
    - Path: pkg/steps/ai/openai_responses/observability.go
      Note: Provider/event observability helper review
    - Path: pkg/steps/ai/openai_responses/observability_test.go
      Note: Recent observability tests and coverage baseline
    - Path: pkg/turns/types.go
      Note: Canonical Turn/Block data model explained in guide
    - Path: ttmp/2026/05/07/GP-OBSERVABILITY--add-geppetto-provider-and-event-observability-hooks-for-high-frequency-inference-debugging/playbook/01-provider-to-browser-correlation-playbook.md
      Note: Provider-to-browser validation playbook referenced in testing strategy
    - Path: ttmp/2026/05/07/GP-OBSERVABILITY--add-geppetto-provider-and-event-observability-hooks-for-high-frequency-inference-debugging/reference/01-diary.md
      Note: Prior implementation diary evidence for smoke results
ExternalSources: []
Summary: Evidence-backed architecture review and intern onboarding guide for Geppetto, focused on recent observability/OpenAI Responses/runtime integration work plus broader package cleanup opportunities.
LastUpdated: 2026-05-07T13:03:35.097452117-04:00
WhatFor: Teach a new contributor how Geppetto is organized, how inference flows through the system, what recent observability code does, and where cleanup/refactor work is highest leverage.
WhenToUse: Read before reviewing, extending, or refactoring Geppetto provider engines, observability, events, runtime factory wiring, sessions, or downstream Pinocchio debug integration.
---


# Geppetto Code Review and Intern Onboarding Guide

## 1. Executive summary

Geppetto is the Go runtime core for building LLM applications. It supplies the provider-agnostic pieces that downstream applications such as Pinocchio use: turns and blocks, inference engines, middleware, tool loops, profile registries, event streams, sessions, and JavaScript bindings. The most recent work added a neutral observability package and OpenAI Responses instrumentation so provider-level records can be correlated with Geppetto events, Pinocchio debug records, Sessionstream delivery, and browser-visible reasoning output.

This review has two goals:

1. Onboard a new intern into the system with enough architecture, API, and flow knowledge to work safely.
2. Identify concrete cleanup opportunities, not only correctness bugs: unclear code, deprecated concepts, too-large files, overgrown packages, coupling, duplicated logic, overengineered pieces, and missing validation.

The main conclusion is that the recent observability architecture is directionally good: Geppetto emits neutral records, applications decide whether/how to store them, and Pinocchio owns debug endpoints and SQLite exports. The design preserves the correct ownership boundary. The highest-risk implementation details are not the public observer interface; they are the large OpenAI Responses stream loop, evidence JSON sanitization semantics, high-frequency retention volume, and cross-repository dependency alignment.

The most important cleanup recommendations are:

- Split `pkg/steps/ai/openai_responses/engine.go` into a small request runner plus a stream processor/state machine. At 1,283 lines, it currently mixes HTTP transport, SSE parsing, provider event normalization, reasoning state, tool-call accumulation, event publishing, debug taps, turn mutation, and observability.
- Fix `observability.MarshalEvidenceJSON` so struct payloads such as Geppetto events and metadata are actually sanitized recursively. The current helper sanitizes `map[string]any`, `[]any`, and `string`, but structs pass through the default branch before `json.Marshal`.
- Decide whether both `geppetto_publish_started` and `geppetto_publish_done` need full `event_json` and `metadata_json`. The latest browser smoke produced a 25 MB SQLite artifact for one small prompt, which is a clear retention/cost signal.
- Turn the existing direct-provider-ID follow-up into a real schema/API task: carry `response_id`, `item_id`, `output_index`, and `summary_index` into Pinocchio/Sessionstream `ReasoningUpdate` so SQL correlation no longer depends on row order and exact chunk matching.
- Review old event types and TODOs in `pkg/events/chat-events.go`. This 1,139-line file contains explicit comments such as “This might be possible to delete” and “This needs to deleted once we have a good way to do tool calling.” Some of those concepts should be deprecated, moved, or given stronger contracts.
- Add a dependency/release alignment plan for the multi-repo feature branch. Pinocchio currently imports local Geppetto observability and local Sessionstream observer APIs; its `GOWORK=off` lint hook cannot pass until those dependencies are published or replaced.

This document is intentionally written as both a review and a guide. Interns should first read the architecture sections, then the recent observability walkthrough, then the cleanup plan.

## 2. Scope and evidence

### 2.1 What was reviewed

The review focused on:

- Geppetto repository-level package layout.
- Core runtime concepts: turns, engines, events, sessions, middleware/tool-loop boundaries.
- Recent GP-OBSERVABILITY code:
  - `pkg/observability/*`
  - `pkg/steps/ai/openai_responses/engine.go`
  - `pkg/steps/ai/openai_responses/observability.go`
  - `pkg/cli/bootstrap/inference_observability.go`
  - `pkg/inference/engine/factory/factory.go`
- Prior GP-OBSERVABILITY diary/playbook/report evidence.
- Downstream Pinocchio integration points, because the new Geppetto API is consumed there:
  - `pinocchio/cmd/web-chat/app/debug_recorder.go`
  - `pinocchio/cmd/web-chat/app/debug_reconcile_db.go`
  - `pinocchio/cmd/web-chat/main.go`
  - `pinocchio/cmd/web-chat/runtime_composer.go`

This is not a line-by-line audit of every package in Geppetto. It is a package-level orientation plus a targeted review of the areas most affected by the recent work.

### 2.2 Commands used for evidence

Representative commands run from the workspace:

```bash
cd /home/manuel/workspaces/2026-05-02/use-sessionstream-coinvault

git -C geppetto status --short
git -C pinocchio status --short
git -C sessionstream status --short

cd geppetto
find pkg -mindepth 1 -maxdepth 2 -type d | sort
find pkg -name '*.go' -print0 | xargs -0 wc -l | sort -nr | head -35
rg -n "TODO|FIXME|Deprecated|deprecated|legacy|Legacy|HACK|XXX" pkg cmd -S
rg -n "observer|observability|publishEvent|observeProvider|reasoning-summary|response\.reasoning" \
  pkg/steps/ai/openai_responses/engine.go \
  pkg/steps/ai/openai_responses/observability.go \
  pkg/observability/*.go \
  pkg/cli/bootstrap/inference_observability.go \
  pkg/inference/engine/factory/factory.go
```

Key file-size evidence:

| File | Lines | Review note |
|---|---:|---|
| `pkg/js/modules/geppetto/module_test.go` | 1,975 | Very large JS module test suite; likely needs scenario grouping. |
| `pkg/steps/ai/openai_responses/engine_test.go` | 1,514 | Large but expected for provider compatibility; should track fixtures/scenarios. |
| `pkg/steps/ai/openai_responses/engine.go` | 1,283 | Main refactor candidate; does too many jobs. |
| `pkg/events/chat-events.go` | 1,139 | Event taxonomy and constructors are too centralized. |
| `pkg/inference/middlewarecfg/resolver.go` | 795 | Important config resolution code; needs docs but less directly tied to recent work. |
| `pkg/steps/ai/openai_responses/helpers.go` | 756 | Request/payload helpers; likely belongs next to provider-specific codec subpackage. |
| `pkg/cli/bootstrap/inference_debug.go` | 540 | Large CLI section code; possible section split. |
| `pkg/observability/*.go` | 336 total | Small package; good boundary, but sanitizer semantics need review. |

Package line totals from `pkg/`:

| Top-level package | Go lines | Go files | Interpretation |
|---|---:|---:|---|
| `steps` | 14,357 | 57 | Provider implementations dominate runtime complexity. |
| `inference` | 13,236 | 98 | Engine, session, middleware, tool-loop, tools. |
| `js` | 6,945 | 24 | Goja module surface, large tests. |
| `engineprofiles` | 5,468 | 33 | Registry-first profile system. |
| `events` | 4,155 | 18 | Event model, sinks, router. |
| `turns` | 1,759 | 17 | Canonical conversation data model. |
| `cli` | 1,654 | 10 | Glazed sections/bootstrap. |
| `observability` | 336 | 4 | New neutral observer package. |

## 3. Geppetto in one page for a new intern

Geppetto is a library. It does not own the full application UI. Its job is to take a conversation-like data structure, apply runtime configuration/middleware/tools, call a model provider, and produce a mutated conversation plus streaming events.

A minimal mental model is:

```text
Application
  owns config, UI, persistence, debug storage
  calls Geppetto runtime

Geppetto
  owns Turn/Block data model
  owns provider engines
  owns events emitted by engines/middleware/tool-loop
  owns middleware/tool-loop/session primitives
  emits optional observer records for debugging

Provider
  owns wire protocol and streaming event shape

Downstream apps such as Pinocchio
  translate Geppetto events into app/session/browser concepts
```

The main runtime path is:

```text
User prompt
  -> turns.Turn with user Block
  -> EngineFactory creates provider engine
  -> middleware chain wraps provider engine
  -> provider engine streams events.Event values
  -> tool-loop may execute tool calls and run another inference
  -> resulting Turn contains assistant/tool/reasoning blocks
  -> app persists/responds/renders
```

The recent observability path is a side channel:

```text
OpenAI Responses SSE frame
  -> decoded provider map[string]any
  -> geppetto observability.Record(stage=provider_routed_event, object_json=...)
  -> provider-specific normalization
  -> events.Event publish
  -> geppetto observability.Record(stage=geppetto_publish_done, event_json=..., metadata_json=...)
  -> Pinocchio StreamDebugRecorder
  -> /api/debug/sessions/{id}/geppetto
  -> SQLite reconcile export
```

Important rule: observability must not be required for inference correctness. It is evidence, not behavior.

## 4. Core concepts and API references

### 4.1 Turn and Block: the canonical conversation model

File reference: `pkg/turns/types.go`.

A `Turn` is the canonical conversation snapshot. A `Block` is one atomic piece inside a turn. Blocks carry a `Kind`, optional role, payload, and metadata.

Simplified API:

```go
type Block struct {
    ID       string
    Kind     BlockKind
    Role     string
    Payload  map[string]any
    Metadata BlockMetadata
}

type Turn struct {
    ID       string
    Blocks   []Block
    Metadata Metadata
    Data     Data
}
```

What interns should know:

- Provider engines mutate or append blocks on the turn.
- User text, assistant text, reasoning, tool calls, and tool results are all blocks.
- `Turn.Clone()` makes a new slice and shallow-copies payload maps. It does not deep-copy arbitrary nested values inside payloads.
- Typed keys are generated as strings like `namespace.value@vN` with helpers such as `NewTurnDataKey` and `NewBlockMetadataKey`.

Typical pseudocode:

```go
turn := &turns.Turn{}
turns.AppendBlock(turn, turns.NewUserTextBlock("hello"))

out, err := engine.RunInference(ctx, turn)
if err != nil { return err }

for _, block := range out.Blocks {
    switch block.Kind {
    case turns.BlockKindAssistant:
        // render assistant answer
    case turns.BlockKindReasoning:
        // render or persist reasoning/summary evidence
    }
}
```

Review note: `Turn.Clone()` is explicit about being shallow for nested payload values. That is acceptable for performance, but code that stores maps/slices inside payloads should avoid mutating nested values after cloning unless it owns them.

### 4.2 Engine: provider-neutral inference interface

File reference: `pkg/inference/engine/engine.go`.

The core provider interface is intentionally small:

```go
type Engine interface {
    RunInference(ctx context.Context, t *turns.Turn) (*turns.Turn, error)
}
```

This keeps provider engines composable. The engine should:

- read the input turn,
- call the provider,
- publish Geppetto events while streaming,
- append final blocks to the turn,
- return the updated turn.

Tool orchestration is not part of this interface. Tool orchestration is layered outside engines by tool-loop or enginebuilder components.

### 4.3 Events: streaming facts emitted by engines and middleware

File reference: `pkg/events/chat-events.go`.

`events.Event` is the streaming event contract:

```go
type Event interface {
    Type() EventType
    Metadata() EventMetadata
    Payload() []byte
}
```

Examples of event types:

- `start`
- `partial`
- `partial-thinking`
- `final`
- `tool-call`
- `tool-result`
- `error`
- `info`
- web-search, citation, file-search, code-interpreter, MCP, image-generation events

Intern rule of thumb:

- Use typed events for UI-visible model progress.
- Use `EventInfo` for named lifecycle facts when no dedicated event exists yet.
- Do not overload `EventInfo.Data` as an unbounded dump of provider state. It should carry stable provider IDs or small semantic fields.

Recent observability uses `EventInfo.Data` for provider IDs:

```go
providerData("openai_responses", responseID, itemID, outputIndex, summaryIndex)
```

This is useful because provider IDs survive normal event flow, not only trace flow.

### 4.4 Session: long-lived turn history and one-active-inference invariant

File reference: `pkg/inference/session/session.go`.

A `Session` owns a stable `SessionID`, append-only turn history, and the invariant that only one inference runs at a time.

Important API shape:

```go
type Session struct {
    SessionID string
    Turns     []*turns.Turn
    Builder   EngineBuilder

    mu     sync.Mutex
    active *ExecutionHandle
}
```

Key behaviors:

- `AppendNewTurnFromUserPrompt` clones the latest turn, clears the turn ID, appends a new user block, assigns a new ID, and stores it.
- `StartInference` uses the latest appended turn and runs the configured builder asynchronously.
- `TurnsSnapshot` clones turn snapshots for safe external reads.

Intern warning: session code protects its own `Turns` slice and active handle, but provider engines still mutate the active turn in-place during inference. Do not hand the same turn pointer to unrelated goroutines without a clear ownership contract.

### 4.5 EngineFactory: provider selection and app injection seam

File reference: `pkg/inference/engine/factory/factory.go`.

The factory maps `InferenceSettings` to provider engines. Recent work added a narrow app injection seam:

```go
type StandardEngineFactory struct {
    openAIResponsesOptions []openai_responses.EngineOption
}

func WithOpenAIResponsesOptions(opts ...openai_responses.EngineOption) StandardEngineFactoryOption
```

This lets Pinocchio attach `WithObserver(...)` and `WithObservabilityConfig(...)` only for OpenAI Responses engines without changing every caller.

Current flow:

```text
settings.Chat.ApiType
  -> StandardEngineFactory.CreateEngine
  -> openai.NewOpenAIEngine OR openai_responses.NewEngine OR claude.NewClaudeEngine OR gemini.NewGeminiEngine
```

Review note: the seam is small and pragmatic. If more providers gain observability hooks, consider a provider-agnostic `EngineOptions` struct instead of adding one option slice per provider.

### 4.6 Observability: neutral evidence records

Files:

- `pkg/observability/config.go`
- `pkg/observability/observer.go`
- `pkg/observability/json.go`

The new package defines:

```go
type TraceLevel string
const (
    TraceOff      TraceLevel = "off"
    TraceEvents   TraceLevel = "events"
    TraceProvider TraceLevel = "provider"
)

type Config struct {
    Level              TraceLevel
    MaxPayloadBytes    int
    RedactProviderData bool
}

type Record struct {
    Timestamp time.Time
    Provider  string
    Model     string

    SessionID   string
    InferenceID string
    TurnID      string
    MessageID   string

    Stage       Stage
    EventType   string
    InfoMessage string

    ResponseID   string
    ItemID       string
    OutputIndex  *int
    SummaryIndex *int

    ObjectJSON   json.RawMessage
    EventJSON    json.RawMessage
    MetadataJSON json.RawMessage

    DeltaLen           int
    NormalizedDeltaLen int
    BufferLen          int
    Error              string
}

type Observer interface {
    OnGeppettoRecord(ctx context.Context, rec Record)
}
```

Trace levels:

- `off`: no records.
- `events`: records Geppetto publish stages and event/metadata JSON.
- `provider`: records provider-routed/normalized decoded provider object JSON plus event records.
- `raw`: explicitly rejected for now; raw stream strings are reserved for future work.

Important invariant:

```go
func Notify(ctx context.Context, obs Observer, rec Record) {
    if obs == nil { return }
    if rec.Timestamp.IsZero() { rec.Timestamp = time.Now().UTC() }
    defer func() { _ = recover() }()
    obs.OnGeppettoRecord(ctx, rec)
}
```

This is the right failure model for a debug side channel: observer panics must not break inference.

## 5. Recent observability flow in detail

### 5.1 The normal event path

In `pkg/steps/ai/openai_responses/engine.go`, `publishEvent` is the central event outlet:

```go
func (e *Engine) publishEvent(ctx context.Context, event events.Event) {
    e.observePublish(ctx, event, geppettoobs.StageGeppettoPublishStarted, nil)
    events.PublishEventToContext(ctx, event)
    e.observePublish(ctx, event, geppettoobs.StageGeppettoPublishDone, nil)
}
```

This is a good instrumentation seam because every emitted Geppetto event passes through it. The downside is that high-frequency streams emit two observer records per event when tracing is enabled.

### 5.2 The provider event path

Inside the SSE loop, each decoded provider payload becomes a `map[string]any`. The code then normalizes the event name and emits a provider record:

```go
providerEventType := normalizeResponsesEventName(eventName)
if providerEventType == "" {
    if typ, ok := m["type"].(string); ok {
        providerEventType = normalizeResponsesEventName(typ)
    }
}
e.observeProviderEvent(ctx, metadata, reqBody.Model, currentResponseID, providerEventType, m)
```

Provider records carry:

- provider name (`openai_responses`),
- model,
- session/inference/turn/message IDs,
- provider event type,
- response/item/output/summary IDs when available,
- decoded `object_json`.

### 5.3 Reasoning summary correlation

For reasoning summary deltas, OpenAI Responses provider payloads such as `response.reasoning_summary_text.delta` are normalized and forwarded as Geppetto thinking partials:

```go
normalized := streamhelpers.NormalizeReasoningSummaryDelta(summaryBuf.String(), v)
e.observeProviderNormalizeDelta(ctx, metadata, reqBody.Model, currentResponseID,
    providerEventType, m, len(v), len(normalized), before+len(normalized))
summaryBuf.WriteString(normalized)
currentReasoningSummary.WriteString(normalized)
e.publishEvent(ctx, events.NewThinkingPartialEvent(metadata, normalized, summaryBuf.String()))
```

The browser smoke proved that Geppetto-published `event_json.delta` is the stable comparison point against frontend `payload.chunk`. Provider `object_json.delta` is pre-normalization and can intentionally differ.

### 5.4 Pinocchio integration boundary

Although this ticket lives in Geppetto, the recent feature is only useful because Pinocchio records and exports it.

Pinocchio adapter shape:

```go
func (r *StreamDebugRecorder) OnGeppettoRecord(ctx context.Context, rec geppettoobs.Record) {
    r.RecordGeppetto(ctx, rec)
}
```

Pinocchio stores records as `DebugRecordKindGeppetto`, exposes `/api/debug/sessions/{id}/geppetto`, and exports `geppetto_records`, `geppetto_provider_events`, and `geppetto_emitted_events` to SQLite.

Ownership is correct:

```text
Geppetto owns: Record shape, trace levels, provider/event instrumentation.
Pinocchio owns: retention, debug APIs, SQLite schema, browser correlation views.
```

Do not move SQLite or browser concepts into Geppetto.

## 6. Code-quality findings with cleanup sketches

### 6.1 OpenAI Responses engine is too large and too multi-purpose

Problem: `pkg/steps/ai/openai_responses/engine.go` is 1,283 lines and the streaming branch contains a very large nested `flush` closure. It handles HTTP request construction, SSE parsing, provider event name normalization, reasoning state, assistant text backfill, tool-call accumulation, web search, debug taps, turn mutation, usage parsing, event publishing, and observability.

Where to look:

- `pkg/steps/ai/openai_responses/engine.go:54` — `publishEvent` instrumentation seam.
- `pkg/steps/ai/openai_responses/engine.go:210` onward — streaming local state and SSE loop.
- `pkg/steps/ai/openai_responses/engine.go:397` onward — large switch over provider event types.
- `pkg/steps/ai/openai_responses/engine.go:870` onward — completion, metadata, final turn block assembly.

Example shape:

```go
flush := func() error {
    // decode JSON
    // update response ID
    // observe provider event
    // define append/backfill helpers
    // switch providerEventType
    // publish Geppetto events
    // mutate reasoning/tool/message state
    return nil
}
```

Why it matters:

- A single local variable can accidentally become shared state for multiple event handlers.
- Adding support for one provider event risks regressions in unrelated reasoning/tool/message behavior.
- Tests need to exercise the full engine to validate a small handler.
- Observability logic is mixed into core provider parsing, making it harder to reason about disabled-path overhead.

Cleanup sketch:

```text
pkg/steps/ai/openai_responses/
  engine.go                 # RunInference, request execution, high-level stream/non-stream choice
  stream_processor.go       # streamProcessor struct, ProcessFrame, Finalize
  stream_state.go           # reasoning/message/tool state structs
  stream_handlers.go        # map/switch handlers by normalized event type
  stream_observer.go        # provider/publish observability adapter calls
  sse_reader.go             # read event/data frames into decoded maps
```

Pseudocode:

```go
type streamProcessor struct {
    engine   *Engine
    metadata events.EventMetadata
    model    string
    state    streamState
    out      *turns.Turn
    tap      engine.DebugTap
}

func (p *streamProcessor) Process(frame providerFrame) error {
    p.state.updateResponseID(frame.Object)
    p.engine.observeProviderEvent(...)

    handler := responseHandlers[frame.Type]
    if handler == nil {
        return nil
    }
    return handler(p, frame)
}

func (p *streamProcessor) Finalize() (*turns.Turn, error) {
    p.appendReasoningSummaryInfoIfPresent()
    p.appendAssistantBlockIfPresent()
    p.appendToolCallBlocks()
    p.publishFinalEvent()
    return p.out, p.state.streamErr
}
```

Start with extraction only; do not change behavior. Move code into structs while preserving existing tests.

### 6.2 Evidence JSON sanitizer likely does not cap/redact struct payloads

Problem: `observability.MarshalEvidenceJSON` recursively sanitizes `map[string]any`, `[]any`, and `string`. For other values it returns the value unchanged and then calls `json.Marshal`. That means struct payloads such as Geppetto event structs and `events.EventMetadata` are not recursively capped/redacted before marshaling.

Where to look:

- `pkg/observability/json.go:21` — `MarshalEvidenceJSON`.
- `pkg/observability/json.go:31` — `sanitizeValue` cases.
- `pkg/steps/ai/openai_responses/observability.go:69` — `EventJSON: MarshalEvidenceJSON(event, ...)`.
- `pkg/steps/ai/openai_responses/observability.go:70` — `MetadataJSON: MarshalEvidenceJSON(metadata, ...)`.

Example:

```go
func sanitizeValue(v any, cfg Config) any {
    switch tv := v.(type) {
    case map[string]any:
        // recurse
    case []any:
        // recurse
    case string:
        return capString(tv, cfg.MaxPayloadBytes)
    default:
        return v
    }
}
```

Why it matters:

- Provider `object_json` is usually a decoded map and is sanitized as intended.
- `event_json` and `metadata_json` are passed as structs and may bypass string capping/redaction.
- This undermines the stated first-slice privacy/retention contract: object/event/metadata JSON should all be capped/redacted.

Cleanup sketch:

Option A: marshal first, decode to generic JSON, sanitize generic value, marshal again.

```go
func MarshalEvidenceJSON(v any, cfg Config) json.RawMessage {
    cfg = cfg.Normalized()

    b, err := json.Marshal(v)
    if err != nil { return marshalError(err) }

    var generic any
    if err := json.Unmarshal(b, &generic); err != nil { return marshalError(err) }

    clean := sanitizeValue(generic, cfg)
    out, err := json.Marshal(clean)
    if err != nil { return marshalError(err) }
    return out
}
```

Option B: implement reflection-based struct traversal. Option A is simpler and safer for debug payloads.

Add tests:

```go
func TestMarshalEvidenceJSONCapsEventStructStringFields(t *testing.T) { ... }
func TestMarshalEvidenceJSONRedactsNestedMetadataStructMaps(t *testing.T) { ... }
```

### 6.3 High-frequency publish-started and publish-done records double event JSON volume

Problem: `publishEvent` records both `geppetto_publish_started` and `geppetto_publish_done`, and `observePublish` attaches full `event_json` and `metadata_json` to both records. The GP-OBSERVABILITY diary records a 25 MB SQLite artifact for one small browser prompt with 1,158 frontend records and 1,539 Geppetto records.

Where to look:

- `pkg/steps/ai/openai_responses/engine.go:54` — both publish stages.
- `pkg/steps/ai/openai_responses/observability.go:69` and `:70` — full JSON evidence on publish records.
- GP-OBSERVABILITY diary Step 10 — latest smoke artifact size and counts.

Why it matters:

- Event JSON for high-frequency `partial-thinking` records can dominate artifact size.
- Doubling records may be useful for diagnosing publish errors, but most publishes are synchronous and do not fail.
- Retention pressure makes debug artifacts harder to share and inspect.

Cleanup sketch:

Introduce a stage payload policy:

```go
type PublishPayloadPolicy int
const (
    PublishPayloadDoneOnly PublishPayloadPolicy = iota
    PublishPayloadStartedAndDone
    PublishPayloadErrorsOnlyForStarted
)
```

Simpler first implementation:

```go
func (e *Engine) publishEvent(ctx context.Context, event events.Event) {
    if e.observabilityConfig.RecordPublishStarted() {
        e.observePublish(ctx, event, StageGeppettoPublishStarted, nil, PayloadSummaryOnly)
    }
    events.PublishEventToContext(ctx, event)
    e.observePublish(ctx, event, StageGeppettoPublishDone, nil, PayloadFull)
}
```

Acceptance criteria:

- Existing correlation SQL still works from `geppetto_publish_done`.
- Publish errors still have enough event context.
- A small reasoning prompt artifact shrinks measurably.

### 6.4 `MaxRecords` lives in CLI settings but not in Geppetto `Config`

Problem: `InferenceObservabilitySettings` includes `MaxRecords`, but `observability.Config` contains only `Level`, `MaxPayloadBytes`, and `RedactProviderData`. This is architecturally defensible because record retention is app-owned, but the API shape is easy to misread.

Where to look:

- `pkg/cli/bootstrap/inference_observability.go:11` — `MaxRecords` field.
- `pkg/cli/bootstrap/inference_observability.go:18` — `Config()` ignores `MaxRecords`.
- `pkg/observability/config.go:21` — no `MaxRecords` field.

Why it matters:

- A Geppetto caller may call `Config()` and assume all observability settings were captured.
- Pinocchio has to remember to use `MaxRecords` separately for recorder sizing.
- The field is app-retention policy, not provider-emission policy.

Cleanup sketch:

Make the split explicit:

```go
type InferenceObservabilitySettings struct {
    TraceLevel         string
    MaxRecords         int
    MaxPayloadBytes    int
    RedactProviderData bool
}

func (s InferenceObservabilitySettings) EmissionConfig() (observability.Config, error)
func (s InferenceObservabilitySettings) RecorderConfig() RecorderConfig

type RecorderConfig struct {
    MaxRecords int
}
```

Or rename the method:

```go
func (s InferenceObservabilitySettings) GeppettoConfig() (observability.Config, error)
```

### 6.5 Provider ID propagation stops before direct browser joins

Problem: Provider IDs are preserved in Geppetto records and some `EventInfo.Data`, but browser/frontend `ReasoningUpdate` payloads do not yet carry `response_id`, `item_id`, `output_index`, or `summary_index`. Current SQL correlation uses ordered reasoning deltas and exact chunk matching.

Where to look:

- `pkg/steps/ai/openai_responses/observability.go:113` — `providerData` helper.
- `pkg/steps/ai/openai_responses/engine.go:542`, `:578`, `:624`, `:949` — provider IDs added to info events.
- GP-OBSERVABILITY playbook “Known Limitation”.

Why it matters:

- Row-order correlation works for the observed prompt, but it is a diagnostic workaround.
- Parallel streams, retried events, duplicate chunks, or future chunk coalescing could make order/chunk joins ambiguous.
- Direct provider IDs should become first-class in downstream app schemas.

Cleanup sketch:

In downstream Pinocchio/Sessionstream schema:

```protobuf
message ReasoningUpdate {
  string message_id = 1;
  string chunk = 2;
  string text = 3;

  string provider = 10;
  string response_id = 11;
  string item_id = 12;
  optional int32 output_index = 13;
  optional int32 summary_index = 14;
}
```

In Pinocchio event translation:

```go
func reasoningUpdateFromGeppettoEvent(ev *events.EventThinkingPartial, info ProviderInfo) ReasoningUpdate {
    return ReasoningUpdate{
        MessageID: msgID,
        Chunk: ev.Delta,
        Text: ev.Text,
        Provider: info.Provider,
        ResponseID: info.ResponseID,
        ItemID: info.ItemID,
        OutputIndex: info.OutputIndex,
        SummaryIndex: info.SummaryIndex,
    }
}
```

Then add a SQLite view that joins directly by `item_id` instead of row number.

### 6.6 Event taxonomy is too centralized and contains stale TODO/deprecated concepts

Problem: `pkg/events/chat-events.go` is 1,139 lines. It defines many event types and constructors in one file, including TODOs that indicate uncertainty or potential deletion. Example markers include `EventText` “might be possible to delete,” tool-call TODOs, and comments about needing block stop types.

Where to look:

- `pkg/events/chat-events.go:21` — uncertain `EventTypeStatus`.
- `pkg/events/chat-events.go:24` — possible need for block stop event.
- `pkg/events/chat-events.go:211` — `EventText` might be possible to delete.
- `pkg/events/chat-events.go:238` — multiple tool-call TODO.
- `pkg/events/chat-events.go:364` — “needs to deleted once we have a good way to do tool calling.”

Why it matters:

- Event additions become easy but taxonomy cleanup becomes hard.
- Downstream apps may depend on unclear or transitional events.
- A single file mixes base event contracts, chat text events, tools, web search, file search, MCP, image generation, and code interpreter events.

Cleanup sketch:

Split by domain while preserving package API:

```text
pkg/events/
  event.go              # Event interface, EventImpl, metadata
  text_events.go        # start/partial/final/thinking/status
  tool_events.go        # tool-call/tool-result/tool execution/MCP
  search_events.go      # web/file search/citation
  media_events.go       # image generation/code interpreter
  info_events.go        # EventInfo and lifecycle helpers
  deprecated.go         # explicit deprecated compatibility types
```

For stale events:

```go
// Deprecated: EventText is retained for compatibility. Use EventPartialCompletion
// or typed Turn blocks instead. Planned removal: after <ticket>.
type EventText struct { ... }
```

If a concept is truly unused, remove it in a hard-cut ticket with search evidence.

### 6.7 Redaction keys are useful but incomplete as a privacy policy

Problem: `pkg/observability/json.go` redacts a fixed set of sensitive key names. This is a good start, but the project still needs a documented privacy/retention policy for provider traces.

Where to look:

- `pkg/observability/json.go:8` — `sensitiveKeys`.
- GP-OBSERVABILITY tasks include “Document provider trace privacy policy” but it is unchecked.

Why it matters:

- Provider object JSON may contain user content, reasoning content, tool arguments, URLs, file IDs, or vendor-specific sensitive fields that do not match the fixed key list.
- Capping is not the same as redaction.
- Users need to understand that `provider` trace level captures decoded model content.

Cleanup sketch:

Add a policy doc and code comments:

```text
Trace mode privacy policy:
- off: no debug payload capture.
- events: captures emitted Geppetto events/metadata; may include user/model content.
- provider: captures decoded provider objects; may include user/model/provider content.
- raw: not implemented.
- redaction: key-based secret redaction only; content is capped, not removed.
- retention: app-owned recorder cap; dropped-record count required.
```

Add tests for likely provider secret fields:

```go
api_key, api-key, authorization, bearer, token, access_token,
refresh_token, encrypted_content, client_secret, secret, password
```

### 6.8 Byte slicing can break UTF-8 in capped strings

Problem: `capString` slices by byte index: `s[:limit]`. If a multi-byte UTF-8 rune crosses the limit, the resulting JSON string can contain replacement characters after marshaling or display oddly.

Where to look:

- `pkg/observability/json.go:83` — `capString`.

Why it matters:

- Debug payloads can contain international text.
- Byte-based limits are good for storage; slicing must still preserve valid UTF-8.

Cleanup sketch:

```go
func capString(s string, limit int) string {
    if limit <= 0 || len(s) <= limit { return s }
    cut := limit
    for cut > 0 && !utf8.ValidString(s[:cut]) {
        cut--
    }
    if cut == 0 { return fmt.Sprintf("<truncated:%d bytes>", len(s)) }
    return fmt.Sprintf("%s<truncated:%d bytes>", s[:cut], len(s)-cut)
}
```

Add a test with emoji or non-Latin text at the boundary.

### 6.9 Numeric parsing should reject partial strings

Problem: `intFromAny` uses `fmt.Sscanf(tv, "%d", &i)` for string parsing. `fmt.Sscanf` can parse a leading integer from a string with trailing junk depending on format behavior. Provider indexes should be exact integers.

Where to look:

- `pkg/steps/ai/openai_responses/observability.go:181` — string handling in `intFromAny`.

Why it matters:

- Provider fields are evidence. Accepting malformed numeric strings silently could make a bad provider payload look valid.

Cleanup sketch:

```go
case string:
    i64, err := strconv.ParseInt(strings.TrimSpace(tv), 10, 32)
    if err != nil { return 0, false }
    return int(i64), true
```

Add tests for `"1"`, `" 1 "`, `"1x"`, `"x1"`, and empty strings.

### 6.10 Cross-repository dependency alignment is currently a release risk

Problem: The Pinocchio commit that consumes the new Geppetto observability package had to use `--no-verify` because Pinocchio `make lintmax` runs with `GOWORK=off`. In that mode, its pinned module graph does not yet contain `github.com/go-go-golems/geppetto/pkg/observability` or local Sessionstream observer APIs.

Where to look:

- GP-OBSERVABILITY diary Step 11.
- `pinocchio/cmd/web-chat/app/debug_recorder.go` imports `github.com/go-go-golems/geppetto/pkg/observability`.
- `pinocchio/cmd/web-chat/app/server.go` and `debug_recorder.go` depend on Sessionstream observer APIs.

Why it matters:

- Workspace tests can pass while release/CI/lint hooks fail.
- Reviewers need to know which repos must be published/tagged together.
- This is not a Geppetto API correctness problem, but it is a delivery problem.

Cleanup sketch:

Create a release coordination checklist:

```text
1. Merge/tag Sessionstream observer API.
2. Update Pinocchio go.mod to the Sessionstream version.
3. Merge/tag Geppetto observability package.
4. Update Pinocchio go.mod to the Geppetto version.
5. Run Pinocchio with GOWORK=off:
   - go test ./...
   - make lintmax
6. Remove any temporary replace directives before release.
```

## 7. What looks good and should be preserved

### 7.1 Neutral observability package

`pkg/observability` is small and does not import Pinocchio, Sessionstream, SQLite, or browser concepts. That is the correct ownership split.

Good properties:

- `Record` contains queryable scalar fields and optional JSON evidence.
- `Observer` is a single-method interface.
- `Notify` is panic-safe.
- `TraceOff` is the default.
- `raw` is rejected rather than misleadingly accepted.

### 7.2 Engine option seam preserves existing callers

`openai_responses.NewEngine(settings, opts...)` and `factory.WithOpenAIResponsesOptions(...)` avoid breaking existing engine construction while giving apps an explicit hook point.

This is the right kind of compatibility: small, opt-in, and provider-specific enough to avoid a generic abstraction before multiple providers need it.

### 7.3 Provider IDs are moving into durable event data

Putting provider IDs into `EventInfo.Data` for reasoning lifecycle events is important. Observer records are only present when tracing is enabled; durable event data can survive normal operation.

Keep this direction. Extend it carefully to downstream schemas.

### 7.4 Browser smoke produced actionable evidence

The GP-OBSERVABILITY playbook captured a real browser session, frontend debug records, Geppetto records, and SQLite queries. That is exactly the right validation style for high-frequency event-debugging work.

The best evidence from the diary:

- `geppetto_to_frontend`: 359/359 exact matches.
- `backend_to_frontend`: 359/359 exact matches.
- `provider_to_frontend`: 356/359 exact matches, with mismatches explained by normalization.
- `geppetto_summary_without_item_id`: 0.
- `geppetto_publish_errors`: 0.

## 8. Proposed cleanup roadmap

### Phase 0: Documentation and policy fixes

Low risk, high clarity.

Tasks:

- Add provider trace privacy/retention policy.
- Document that `MaxRecords` is app-recorder retention, not Geppetto emission config.
- Add release/dependency alignment notes for Geppetto + Pinocchio + Sessionstream.
- Mark stale/possibly-deleted event concepts with explicit `Deprecated:` comments or cleanup tickets.

Validation:

```bash
cd geppetto
go test ./pkg/observability ./pkg/cli/bootstrap ./pkg/steps/ai/openai_responses -count=1
```

### Phase 1: Fix evidence JSON sanitizer semantics

Risk: medium-low. Behavior changes only affect observability payload shape/size.

Tasks:

- Change `MarshalEvidenceJSON` to sanitize structs by JSON round-tripping into generic maps.
- Add tests for event structs, metadata structs, nested maps, UTF-8 capping, and redaction.
- Add a test proving `event_json` long strings are capped.

Pseudocode:

```go
func MarshalEvidenceJSON(v any, cfg Config) json.RawMessage {
    cfg = cfg.Normalized()
    b, err := json.Marshal(v)
    if err != nil { return marshalError(err) }
    var decoded any
    if err := json.Unmarshal(b, &decoded); err != nil { return marshalError(err) }
    return mustMarshal(sanitizeValue(decoded, cfg))
}
```

### Phase 2: Add payload retention knobs

Risk: medium. Requires deciding what evidence is required for correlation.

Tasks:

- Make publish-started records summary-only or disabled by default for full JSON payloads.
- Add dropped-record counts in app recorders or debug export meta.
- Add a benchmark or fixture-driven test that estimates records and bytes for a known provider stream.

Acceptance criteria:

```text
Given a fixed reasoning SSE fixture:
- trace=off emits 0 records.
- trace=events emits event records under expected byte budget.
- trace=provider emits provider + event records under expected byte budget.
- disabling publish-started full payload reduces bytes by a measurable percentage.
```

### Phase 3: Refactor OpenAI Responses stream processing without behavior change

Risk: medium-high because stream behavior is subtle.

Tasks:

1. Extract `streamState` without changing logic.
2. Extract SSE frame reading into `sse_reader.go`.
3. Extract provider handlers into methods on `streamProcessor`.
4. Keep existing tests green after each extraction.
5. Add one fixture per major provider stream category: text, reasoning summary, reasoning text, tool call, web search, provider error, non-streaming.

Suggested sequence:

```text
Commit 1: Add streamState struct; still used only inside engine.go.
Commit 2: Move helper closures to methods; keep switch in engine.go.
Commit 3: Move switch into stream_handlers.go.
Commit 4: Move SSE reading into sse_reader.go.
Commit 5: Add focused unit tests for streamProcessor without HTTP.
```

### Phase 4: Downstream direct provider ID propagation

Risk: medium because it crosses repos and schemas.

Tasks:

- Extend Pinocchio/Sessionstream `ReasoningUpdate` payloads with provider fields.
- Populate fields from Geppetto event data.
- Update frontend stream debug and SQLite views.
- Replace row-order correlation view with direct ID joins where possible.

### Phase 5: Event package cleanup

Risk: medium because downstream apps may rely on event types.

Tasks:

- Inventory event type usage with `rg "EventTypeText|NewTextEvent|EventTypeStatus|New.*Tool"`.
- Add explicit deprecation comments.
- Split files by event domain.
- Remove truly unused event types only after usage search and migration notes.

## 9. Intern review checklist

When an intern starts on this area, use this checklist.

### First day reading order

1. `README.md` — project direction and package map.
2. `pkg/turns/types.go` — Turn and Block model.
3. `pkg/inference/engine/engine.go` — minimal engine interface.
4. `pkg/events/chat-events.go` — streaming event model.
5. `pkg/inference/session/session.go` — session lifecycle.
6. `pkg/inference/engine/factory/factory.go` — provider selection.
7. `pkg/observability/*` — new observer contract.
8. `pkg/steps/ai/openai_responses/engine.go` — current provider stream implementation.
9. GP-OBSERVABILITY diary Step 10 and Step 11 — validation evidence and release caveat.
10. GP-OBSERVABILITY playbook — browser-to-provider SQL correlation.

### Things to avoid

- Do not add Pinocchio or SQLite imports to Geppetto observability.
- Do not make inference correctness depend on observer records.
- Do not store raw SSE strings unless a separate privacy/retention design is approved.
- Do not add another provider-specific option field to `StandardEngineFactory` without considering a generic options container.
- Do not refactor `openai_responses/engine.go` and change stream behavior in the same commit.

### Safe starter tasks

- Add tests for `MarshalEvidenceJSON` struct sanitization.
- Add UTF-8-safe string capping.
- Add exact integer parsing tests for provider indexes.
- Add documentation clarifying trace modes and privacy.
- Add file-splitting-only refactor for `pkg/events/chat-events.go` if public API remains unchanged.

## 10. Testing and validation strategy

### Unit tests

Run focused tests during observability work:

```bash
cd geppetto
go test ./pkg/observability ./pkg/cli/bootstrap ./pkg/steps/ai/openai_responses ./pkg/inference/engine/factory -count=1
```

Add or maintain tests for:

- trace-level parsing,
- observer panic safety,
- redaction and capping,
- no raw stream string capture,
- provider object JSON capture,
- event/metadata JSON capture,
- provider ID preservation in `EventInfo.Data`,
- trace-off behavior.

### Full Geppetto validation

```bash
cd geppetto
go test ./...
make lintmax
```

The recent Geppetto commit passed full pre-commit validation after lint fixes.

### Cross-repo validation

For Pinocchio integration in workspace mode:

```bash
cd pinocchio
go test ./cmd/web-chat ./cmd/web-chat/app -count=1
make test
```

For release readiness, after dependency versions are aligned:

```bash
cd pinocchio
GOWORK=off go test ./...
make lintmax
```

### Browser/SQLite validation

Use the GP-OBSERVABILITY playbook. The short version:

```bash
cd pinocchio
go run ./cmd/web-chat web-chat \
  --addr 127.0.0.1:18082 \
  --debug-api \
  --geppetto-trace-level provider
```

Then enable frontend stream debug in the browser:

```javascript
localStorage.setItem('pinocchio.debugStream', '1')
location.reload()
```

Submit a prompt, export/upload frontend records, download SQLite, then run the ticket SQL scripts.

Expected checks:

```sql
SELECT key,value FROM meta
WHERE key IN ('backend_record_count','frontend_record_count','geppetto_record_count');

SELECT COUNT(*) FROM geppetto_publish_errors;
SELECT COUNT(*) FROM geppetto_summary_without_item_id;
SELECT * FROM geppetto_reasoning_to_frontend LIMIT 10;
```

## 11. Open questions

1. Should `TraceEvents` include full event JSON for both publish-started and publish-done, or only publish-done?
2. Should `MaxRecords` remain in the Geppetto CLI section, or move to app-specific debug-recorder configuration?
3. How strict should provider object JSON redaction be: secret-key redaction only, or content-class redaction as well?
4. Should `pkg/events/chat-events.go` be split before or after downstream Pinocchio event schema cleanup?
5. What release order should be enforced for Sessionstream, Geppetto, and Pinocchio so `GOWORK=off` validation passes?
6. Should OpenAI Responses stream processing use a handler map or a typed state-machine switch? A handler map improves modularity; a switch keeps event order easier to audit in one place.

## 12. Reference map

### Geppetto source files

| File | Why it matters |
|---|---|
| `README.md` | Project direction and high-level package map. |
| `pkg/turns/types.go` | Canonical conversation data model. |
| `pkg/inference/engine/engine.go` | Minimal provider engine interface. |
| `pkg/inference/session/session.go` | Long-lived session lifecycle and active inference invariant. |
| `pkg/events/chat-events.go` | Streaming event taxonomy and constructors. |
| `pkg/inference/engine/factory/factory.go` | Provider selection and OpenAI Responses option injection. |
| `pkg/cli/bootstrap/inference_observability.go` | Glazed flags for trace level, payload size, record cap, redaction. |
| `pkg/observability/config.go` | Trace-level config and parsing. |
| `pkg/observability/observer.go` | Neutral record and observer interface. |
| `pkg/observability/json.go` | Evidence JSON sanitizer/capper. |
| `pkg/steps/ai/openai_responses/engine.go` | OpenAI Responses request/stream handling and event publishing. |
| `pkg/steps/ai/openai_responses/observability.go` | Provider/event observability helpers. |
| `pkg/steps/ai/openai_responses/observability_test.go` | Tests for provider object/event/metadata capture and provider IDs. |

### Prior ticket documents

| Document | Why it matters |
|---|---|
| `GP-OBSERVABILITY/reference/01-diary.md` | Chronological implementation diary with smoke evidence and commit caveats. |
| `GP-OBSERVABILITY/playbook/01-provider-to-browser-correlation-playbook.md` | Repeatable provider-to-browser validation procedure. |
| `GP-OBSERVABILITY/analysis/01-textbook-report-geppetto-provider-event-observability-design-assessment.md` | Design rationale and tradeoffs for observability. |
| `GP-OBSERVABILITY/tasks.md` | Remaining follow-ups: retention/privacy and direct provider IDs in `ReasoningUpdate`. |

### Downstream Pinocchio files

| File | Why it matters |
|---|---|
| `pinocchio/cmd/web-chat/app/debug_recorder.go` | App-owned recorder implementing `OnGeppettoRecord`. |
| `pinocchio/cmd/web-chat/app/debug_reconcile_db.go` | SQLite tables/views for Geppetto records and browser correlation. |
| `pinocchio/cmd/web-chat/main.go` | Mounts Geppetto observability settings and wires recorder/config. |
| `pinocchio/cmd/web-chat/runtime_composer.go` | Uses custom engine factory for observer-enabled OpenAI Responses engines. |

## 13. Final recommendation

The recent observability work should be kept, but it should be hardened before it becomes a template for more provider instrumentation. The next best engineering slice is not a new feature; it is a quality slice:

1. Fix evidence JSON sanitization for structs.
2. Clarify trace privacy/retention policy.
3. Reduce duplicated event JSON volume or add retention accounting.
4. Start extracting OpenAI Responses stream state into a testable processor.
5. Align downstream dependencies so Pinocchio validates with `GOWORK=off`.

That sequence improves correctness, privacy, runtime cost, and maintainability without changing the user-visible feature set.

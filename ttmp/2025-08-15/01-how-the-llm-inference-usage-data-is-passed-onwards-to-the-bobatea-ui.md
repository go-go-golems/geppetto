## How LLMInferenceData flows from engines to the Bobatea UI

This note explains how `LLMInferenceData` is populated at the engine level, transported via events, forwarded through messages, and finally rendered by the Bobatea timeline UI. It also lists log points we added to trace the path end-to-end.

### 1) Engines produce events with LLMInferenceData

- Files: `geppetto/pkg/steps/ai/openai/engine_openai.go`, `geppetto/pkg/steps/ai/claude/engine_claude.go`
- When an inference run starts, the engine constructs an `events.EventMetadata` with embedded `events.LLMInferenceData`:

```170:195:geppetto/pkg/steps/ai/openai/engine_openai.go
    metadata := events.EventMetadata{
        ID: uuid.New(),
        LLMInferenceData: events.LLMInferenceData{
            Model:       req.Model,
            Usage:       nil,
            StopReason:  nil,
            Temperature: e.settings.Chat.Temperature,
            TopP:        e.settings.Chat.TopP,
            MaxTokens:   e.settings.Chat.MaxResponseTokens,
        },
    }
```

- Engines now compute and set duration at the source. On errors, interrupts, and finalization, `metadata.DurationMs` is populated before publishing the corresponding event.
  - OpenAI: duration measured from function start; set on error/interrupt/final right before publishing.
  - Claude: duration measured inside `ContentBlockMerger`; set on final/error event emission.

### 2) JSON/log serialization of EventMetadata

- File: `geppetto/pkg/events/chat-events.go`
- The `EventMetadata` is logged via `MarshalZerologObject`. We log `model`, temperature/top_p/max_tokens, usage, stop_reason, duration_ms, and extra. The `engine` field was removed from `LLMInferenceData` and is no longer logged.

```307:343:geppetto/pkg/events/chat-events.go
func (em EventMetadata) MarshalZerologObject(e *zerolog.Event) {
    e.Str("message_id", em.ID.String())
    if em.RunID != "" { e.Str("run_id", em.RunID) }
    if em.TurnID != "" { e.Str("turn_id", em.TurnID) }
    if em.Model != "" { e.Str("model", em.Model) }
    if em.Temperature != nil { e.Float64("temperature", *em.Temperature) }
    if em.TopP != nil { e.Float64("top_p", *em.TopP) }
    if em.MaxTokens != nil { e.Int("max_tokens", *em.MaxTokens) }
    if em.StopReason != nil && *em.StopReason != "" { e.Str("stop_reason", *em.StopReason) }
    if em.Usage != nil { e.Int("input_tokens", em.Usage.InputTokens); e.Int("output_tokens", em.Usage.OutputTokens) }
    if em.DurationMs != nil { e.Int64("duration_ms", *em.DurationMs) }
    if len(em.Extra) > 0 { e.Dict("extra", zerolog.Dict().Fields(em.Extra)) }
}
```

### 3) How events are dispatched to the UI (Bubble Tea)

- File: `pinocchio/pkg/ui/backend.go`
- The backend listens to engine events and dispatches them as timeline UI lifecycle messages. We now pass typed `LLMInferenceData` through `Props`, `Patch`, and `Result`.

```212:241:pinocchio/pkg/ui/backend.go
md := e.Metadata()
entityID := md.ID.String()
...
switch e_ := e.(type) {
case *events.EventPartialCompletionStart:
    p.Send(timeline.UIEntityCreated{
        ID:        timeline.EntityID{LocalID: entityID, Kind: "llm_text"},
        Renderer:  timeline.RendererDescriptor{Kind: "llm_text"},
        Props:     map[string]any{"role": "assistant", "text": "", "metadata": md.LLMInferenceData},
        StartedAt: time.Now(),
    })
case *events.EventPartialCompletion:
    p.Send(timeline.UIEntityUpdated{
        ID:        timeline.EntityID{LocalID: entityID, Kind: "llm_text"},
        Patch:     map[string]any{"text": e_.Completion, "metadata": md.LLMInferenceData},
        Version:   time.Now().UnixNano(),
        UpdatedAt: time.Now(),
    })
case *events.EventFinal:
    p.Send(timeline.UIEntityCompleted{
        ID:     timeline.EntityID{LocalID: entityID, Kind: "llm_text"},
        Result: map[string]any{"text": e_.Text, "metadata": md.LLMInferenceData},
    })
}
```

- These `UIEntityCreated/Updated/Completed` are then consumed by `bobatea/pkg/timeline/controller.go`, which updates the store and forwards size/props updates to the renderer models.

### 4) Bobatea chat model path (direct message flow)

- File: `bobatea/pkg/chat/model.go`
- The chat model’s stream messages convert `events.EventMetadata` into `events.LLMInferenceData` without computing duration locally. Engines now set `DurationMs`.

```414:447:bobatea/pkg/chat/model.go
// StreamStartMsg/StreamCompletionMsg/StreamDoneMsg → OnCreated/OnUpdated/OnCompleted
// Props/Patch/Result include: {"metadata": toLLMInferenceData(v.EventMetadata)}
```

- The helper now maps directly and uses `em.DurationMs`:

```1175:1191:bobatea/pkg/chat/model.go
func toLLMInferenceData(em *geppetto_events.EventMetadata) *geppetto_events.LLMInferenceData {
    if em == nil { return nil }
    return &geppetto_events.LLMInferenceData{
        Model:       em.Model,
        Temperature: em.Temperature,
        TopP:        em.TopP,
        MaxTokens:   em.MaxTokens,
        StopReason:  em.StopReason,
        DurationMs:  em.DurationMs,
        Usage:       em.Usage,
    }
}
```

### 5) Timeline renderer formatting (typed metadata only) and tps

- File: `bobatea/pkg/timeline/renderers/llm_text_model.go`
- The `llm_text` renderer now only accepts typed `events.LLMInferenceData` and computes a tokens-per-second (tps) value from `Usage.OutputTokens` and `DurationMs`. Legacy map introspection has been removed.

```212:231:bobatea/pkg/timeline/renderers/llm_text_model.go
func formatMetadata(md any) string {
    if md == nil { return "" }
    switch t := md.(type) {
    case *geppetto_events.LLMInferenceData:
        return formatFromLLMInferenceData(t)
    case geppetto_events.LLMInferenceData:
        tt := t; return formatFromLLMInferenceData(&tt)
    }
    return ""
}
```

```236:267:bobatea/pkg/timeline/renderers/llm_text_model.go
func formatFromLLMInferenceData(m *geppetto_events.LLMInferenceData) string {
    parts := []string{}
    if m.Model != "" { parts = append(parts, m.Model) }
    if m.Temperature != nil { parts = append(parts, fmt.Sprintf("t: %.2f", *m.Temperature)) }
    if m.TopP != nil { parts = append(parts, fmt.Sprintf("top_p: %.2f", *m.TopP)) }
    if m.MaxTokens != nil { parts = append(parts, fmt.Sprintf("max: %d", *m.MaxTokens)) }
    if m.StopReason != nil && *m.StopReason != "" { parts = append(parts, "stop:"+*m.StopReason) }
    if m.Usage != nil && (m.Usage.InputTokens > 0 || m.Usage.OutputTokens > 0) {
        parts = append(parts, fmt.Sprintf("in: %d out: %d", m.Usage.InputTokens, m.Usage.OutputTokens))
    }
    if m.DurationMs != nil && *m.DurationMs > 0 {
        if m.Usage != nil && m.Usage.OutputTokens > 0 {
            sec := float64(*m.DurationMs) / 1000.0
            if sec > 0 {
                tps := float64(m.Usage.OutputTokens) / sec
                parts = append(parts, fmt.Sprintf("tps: %.2f", tps))
            }
        }
        parts = append(parts, fmt.Sprintf("%dms", *m.DurationMs))
    }
    return strings.Join(parts, " ")
}
```

- Note: The `Engine` field has been removed from `LLMInferenceData` as it duplicated `Model`.

### Summary of key locations

- Populate and update LLMInferenceData: engines (`engine_openai.go`, `engine_claude.go`); engines set `DurationMs`.
- Serialize/log EventMetadata fields: `events/chat-events.go` (no more `engine` logging).
- Dispatch to UI (events → Bubble Tea): `pinocchio/pkg/ui/backend.go` (typed `metadata` included in `Props/Patch/Result`).
- Direct chat model path: `bobatea/pkg/chat/model.go` (`toLLMInferenceData` maps directly, no local duration calc).
- Render and log received metadata: `bobatea/pkg/timeline/renderers/llm_text_model.go` (typed-only; displays `tps`).

### Appendix: Typed event casting and why dispatching on Type_ is required

All event structs embed `events.EventImpl` and add their own payload fields. When we receive JSON, we do a two-step decode:

1) Unmarshal to `*EventImpl` to read the envelope (notably `Type_` and `Metadata_`).
2) Switch on `Type_` and re-unmarshal the raw bytes into the correct typed struct via `ToTypedEvent[T]`.

Relevant code:

```351:431:geppetto/pkg/events/chat-events.go
func NewEventFromJson(b []byte) (Event, error) {
    var e *EventImpl
    err := json.Unmarshal(b, &e)
    if err != nil { return nil, err }
    e.payload = b
    switch e.Type_ {
    case EventTypeStart:
        ret, ok := ToTypedEvent[EventPartialCompletionStart](e)
        if !ok { return nil, fmt.Errorf("could not cast event to EventPartialCompletionStart") }
        return ret, nil
    case EventTypePartialCompletion:
        ret, ok := ToTypedEvent[EventPartialCompletion](e)
        if !ok { return nil, fmt.Errorf("could not cast event to EventPartialCompletion") }
        return ret, nil
    // ... other cases ...
    }
    return e, nil
}
```

```434:441:geppetto/pkg/events/chat-events.go
func ToTypedEvent[T any](e Event) (*T, bool) {
    var ret *T
    err := json.Unmarshal(e.Payload(), &ret)
    if err != nil { return nil, false }
    return ret, true
}
```

Why this is necessary:
- All event JSON envelopes share the same top-level fields (`type`, `meta`). If we only unmarshal to `EventImpl`, we lose typed payload fields (like `Delta`, `Completion`, `ToolCall`, etc.).
- Dispatching on the already-deserialized `Type_` ensures we unmarshal the exact struct that contains those payload fields.

Pitfalls and how to fix them if casting fails:
- Missing or wrong `type` value in the incoming JSON: ensure producers set `"type"` to one of the known `EventType*` constants.
- Not using `NewEventFromJson`: `ToTypedEvent` relies on `EventImpl.payload` being set. Always construct events from JSON via `NewEventFromJson` so `payload` is preserved for the second unmarshal.
- Divergent JSON shape vs typed struct: if provider payload field names differ, align the typed struct tags or add a custom adapter before `ToTypedEvent`.
- Unknown/unsupported type: add a default handler or log and return the plain `EventImpl` so callers can still access `Metadata_`.
- Add logs on failure: temporarily log the raw `payload` and `type` when `ToTypedEvent` returns `ok=false` to pinpoint schema mismatch.


### What we found so far (from /tmp/agent.log)

- Engine side:
  - LLMInferenceData is initialized and logged with `model`, `temperature`, `top_p`, `max_tokens`.
  - During streaming, `usage` and `stop_reason` are updated and logged; final/interrupt/error publish includes `duration_ms`.
- Event dispatch (backend → UI):
  - The dispatched JSON now includes `model`, `temperature`, `top_p`, `max_tokens`, `duration_ms` in `meta` for start/partial/final events (usage appears on final/stop when available).
- UI path used in the trace:
  - The run used the backend dispatcher path. Renderer-level debug lines go to the TUI logger (configure sinks to view in the same file if needed).

Conclusion: End-to-end, structured fields are present and transported. Renderer shows a compact metadata line with model, params, token usage, tps, and duration.

### Next steps

1) Ensure renderer logs are visible in the shared log sink (optional)
   - Configure the TUI logger to write to the same file sink as the engine/backend (e.g., `/tmp/agent.log`).

2) Display improvements (optional)
   - Consider compact formats and color cues in the metadata line for readability.

3) Robustness checks
   - Add a test constructing an `EventFinal` JSON with full `meta` and assert `NewEventFromJson` retains all fields.

4) Chat-model path parity
   - Already in parity: the chat model uses typed `LLMInferenceData` and no longer computes duration.

5) Docs
   - Keep this document updated as formats evolve.



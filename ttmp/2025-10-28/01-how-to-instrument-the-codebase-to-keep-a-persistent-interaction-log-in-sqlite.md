## How to instrument the codebase to keep a persistent LLM interaction log in SQLite (2025-10-28)

### TL;DR
- Implement a new `events.EventSink` that writes events to SQLite (e.g., `SQLiteSink`).
- Attach it in two places:
  - When creating the engine: `engine.WithSink(sqliteSink)`
  - On the run context: `ctx = events.WithEventSinks(ctx, sqliteSink)`
- No invasive edits to engines are required: engines already publish Start/Partial/Final/ToolCall/Error/Interrupt events; tool execution publishes dedicated ToolExecute/ToolResult events via the context.

---

### Why an EventSink is the right hook
The codebase already emits a rich stream of normalized events during LLM inference and tool execution. Engines publish via their internal `publishEvent` and also call `events.PublishEventToContext(ctx, ...)`. Tool execution uses `events.PublishEventToContext(...)` directly. A SQLite-backed `EventSink` will persist the complete interaction trace without modifying provider logic.

Key sources of events:
- Provider engines
  - OpenAI: `OpenAIEngine.RunInference` publishes `start`, `partial`, `tool-call`, `final`, `error`, `interrupt` and fills `Usage`/`StopReason`/`DurationMs` in `EventMetadata`.
  - Claude: `ClaudeEngine.RunInference` streams provider events through `ContentBlockMerger.Add`, which returns normalized events that the engine publishes.
  - Gemini: `GeminiEngine.RunInference` publishes `start`, `partial`, `tool-call`, `final`, `error`.
- Tool execution
  - `tools.BaseToolExecutor.PublishStart` → `tool-call-execute`
  - `tools.BaseToolExecutor.PublishResult` → `tool-call-execution-result`

Because tool execution events are published to the context, you must attach your sink to the context with `events.WithEventSinks` in addition to passing it with `engine.WithSink` for engine-level events.

---

### Concrete functions/locations to rely on (no code edits needed)
- OpenAI engine
  - `pkg/steps/ai/openai/engine_openai.go`: `OpenAIEngine.RunInference`, `OpenAIEngine.publishEvent`
- Claude engine
  - `pkg/steps/ai/claude/engine_claude.go`: `ClaudeEngine.RunInference`, `ClaudeEngine.publishEvent`
  - `pkg/steps/ai/claude/content-block-merger.go`: `ContentBlockMerger.Add` (produces normalized events)
- Gemini engine
  - `pkg/steps/ai/gemini/engine_gemini.go`: `GeminiEngine.RunInference`, `GeminiEngine.publishEvent`
- Tool execution lifecycle
  - `pkg/inference/tools/base_executor.go`: `PublishStart`, `PublishResult`, `ExecuteToolCall` (duration captured here)
- Event plumbing
  - `pkg/inference/engine/options.go`: `engine.WithSink(...)`
  - `pkg/events/context.go`: `events.WithEventSinks(ctx, ...)`, `events.PublishEventToContext(ctx, e)`
  - Example sink (pattern): `pkg/inference/middleware/sink_watermill.go`

Optional low-level capture (raw HTTP/SSE, request bodies):
- `pkg/inference/engine/debugtap.go`: attach via `engine.WithDebugTap(...)` on the context and implement `OnHTTP`, `OnSSE`, etc., if you also want raw provider I/O stored (out of scope for the minimal event log, but easy to add later).

---

### What to persist (normalized)
Persist one row per emitted event with useful denormalized columns; you can normalize further later if needed.

Recommended tables:
- `events`
  - `id` TEXT (UUID from `EventMetadata.ID`)
  - `type` TEXT (e.g., start, partial, final, tool-call, error, interrupt, tool-call-execute, tool-call-execution-result)
  - `created_at` INTEGER (Unix ms)
  - Correlation: `run_id` TEXT, `turn_id` TEXT
  - Model/config: `model` TEXT, `temperature` REAL, `top_p` REAL, `max_tokens` INTEGER
  - Outcome: `stop_reason` TEXT, `duration_ms` INTEGER
  - Tokens: `input_tokens` INTEGER, `output_tokens` INTEGER, `cached_tokens` INTEGER, `cache_creation_input_tokens` INTEGER, `cache_read_input_tokens` INTEGER
  - Text payloads: `delta` TEXT, `completion` TEXT, `text` TEXT
  - Tool payloads: `tool_id` TEXT, `tool_name` TEXT, `tool_input` TEXT, `tool_result` TEXT
  - Errors: `error_string` TEXT
  - `raw_payload` TEXT (store `json.Marshal(event)` for full fidelity)

Optional helper tables (populate from `events` if desired):
- `turns(run_id, turn_id, created_at)`
- `runs(run_id, created_at)`

Schema DDL (SQLite):
```sql
PRAGMA journal_mode=WAL;
PRAGMA synchronous=NORMAL;

CREATE TABLE IF NOT EXISTS events (
  id TEXT NOT NULL,
  type TEXT NOT NULL,
  created_at INTEGER NOT NULL,
  run_id TEXT,
  turn_id TEXT,
  model TEXT,
  temperature REAL,
  top_p REAL,
  max_tokens INTEGER,
  stop_reason TEXT,
  duration_ms INTEGER,
  input_tokens INTEGER,
  output_tokens INTEGER,
  cached_tokens INTEGER,
  cache_creation_input_tokens INTEGER,
  cache_read_input_tokens INTEGER,
  delta TEXT,
  completion TEXT,
  text TEXT,
  tool_id TEXT,
  tool_name TEXT,
  tool_input TEXT,
  tool_result TEXT,
  error_string TEXT,
  raw_payload TEXT
);

CREATE INDEX IF NOT EXISTS idx_events_created_at ON events(created_at);
CREATE INDEX IF NOT EXISTS idx_events_run ON events(run_id, turn_id);
CREATE INDEX IF NOT EXISTS idx_events_type ON events(type);
```

Notes:
- `id` is the event UUID; it is not necessarily unique across a full run when multiple engines or buses are involved, but is appropriate as the primary event identifier for a single process.
- Store `raw_payload` as-is for future-proofing; you can migrate to wider columns later without code changes.

---

### SQLite EventSink sketch
Implement a sink similar to `WatermillSink`, but writing to SQLite.

```go
type SQLiteSink struct {
    db *sql.DB
}

func NewSQLiteSink(path string) (*SQLiteSink, error) {
    db, err := sql.Open("sqlite3", path+"?_busy_timeout=5000&_fk=1")
    if err != nil { return nil, err }
    if _, err := db.Exec("PRAGMA journal_mode=WAL; PRAGMA synchronous=NORMAL;"); err != nil { return nil, err }
    if _, err := db.Exec(schemaDDL); err != nil { return nil, err }
    return &SQLiteSink{db: db}, nil
}

func (s *SQLiteSink) PublishEvent(e events.Event) error {
    meta := e.Metadata()
    nowMs := time.Now().UnixMilli()

    // Marshal full event for raw_payload
    raw, _ := json.Marshal(e)

    // Event-specific fields
    var delta, completion, text, toolID, toolName, toolInput, toolResult, errorStr string
    switch ev := e.(type) {
    case *events.EventPartialCompletion:
        delta = ev.Delta; completion = ev.Completion
    case *events.EventFinal:
        text = ev.Text
    case *events.EventInterrupt:
        text = ev.Text
    case *events.EventError:
        errorStr = ev.ErrorString
    case *events.EventToolCall:
        toolID = ev.ToolCall.ID; toolName = ev.ToolCall.Name; toolInput = ev.ToolCall.Input
    case *events.EventToolResult:
        toolID = ev.ToolResult.ID; toolResult = ev.ToolResult.Result
    case *events.EventToolCallExecute:
        toolID = ev.ToolCall.ID; toolName = ev.ToolCall.Name; toolInput = ev.ToolCall.Input
    case *events.EventToolCallExecutionResult:
        toolID = ev.ToolResult.ID; toolResult = ev.ToolResult.Result
    }

    // Usage fields (provider-specific usage may be nil)
    var inTok, outTok, cachedTok, cacheCreateTok, cacheReadTok *int
    if meta.Usage != nil {
        inTok = &meta.Usage.InputTokens
        outTok = &meta.Usage.OutputTokens
        if meta.Usage.CachedTokens != 0 { cachedTok = &meta.Usage.CachedTokens }
        if meta.Usage.CacheCreationInputTokens != 0 { cacheCreateTok = &meta.Usage.CacheCreationInputTokens }
        if meta.Usage.CacheReadInputTokens != 0 { cacheReadTok = &meta.Usage.CacheReadInputTokens }
    }

    // Insert row
    _, err := s.db.Exec(`INSERT INTO events (
        id, type, created_at, run_id, turn_id, model, temperature, top_p, max_tokens, stop_reason, duration_ms,
        input_tokens, output_tokens, cached_tokens, cache_creation_input_tokens, cache_read_input_tokens,
        delta, completion, text, tool_id, tool_name, tool_input, tool_result, error_string, raw_payload
    ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
        meta.ID.String(), string(e.Type()), nowMs, meta.RunID, meta.TurnID, meta.Model,
        meta.Temperature, meta.TopP, meta.MaxTokens, meta.StopReason, meta.DurationMs,
        inTok, outTok, cachedTok, cacheCreateTok, cacheReadTok,
        nullIfEmpty(delta), nullIfEmpty(completion), nullIfEmpty(text),
        nullIfEmpty(toolID), nullIfEmpty(toolName), nullIfEmpty(toolInput), nullIfEmpty(toolResult), nullIfEmpty(errorStr),
        string(raw),
    )
    return err
}

func nullIfEmpty(s string) any { if s == "" { return nil }; return s }
```

Notes:
- This is a sketch; wire imports and error handling as you see fit. Consider prepared statements for performance.
- If you need to redact secrets, do so before persisting (e.g., customize `tools.BaseToolExecutor.MaskArguments`).

---

### Wiring the sink
Attach the sink both to the engine and the context so you capture engine and tool execution events.

```go
sqliteSink, _ := NewSQLiteSink(filepath.Join(outDir, "events.sqlite3"))

// Engine-level (captures engine-published events)
eng, _ := factory.NewEngineFromStepSettings(stepSettings, engine.WithSink(sqliteSink))

// Context-level (captures tool executor events and also sees engine events)
runCtx := events.WithEventSinks(ctx, sqliteSink)
_, err := eng.RunInference(runCtx, turn)
```

You can see a similar pattern in `pkg/inference/fixtures/fixtures.go`, which wires a file-based sink; just replace with your SQLite sink.

---

### Correlation and reconstruction
- Use `(run_id, turn_id)` from `EventMetadata` to group events per interaction.
- Ordering: use `created_at` ascending per `(run_id, turn_id)` and then sequence by event `type` semantics:
  - Start → Partial* → ToolCall* → (ToolExecute/ToolResult)* → Final or Error/Interrupt
- Token usage and timings are in `EventMetadata` (`Usage`, `DurationMs`, `StopReason`).

---

### Optional: capture raw provider I/O
If you need raw request/response bodies or SSE frames, attach a `DebugTap` via `engine.WithDebugTap(...)` on the context and persist those callbacks to SQLite (separate tables) or to disk.

---

### Edge cases and guidance
- Make sure to store `Final` even when no `Partial` were emitted (non-streaming failure paths still produce `Error`).
- Tool arguments may be large; consider truncation or separate table if needed.
- Long-running sessions benefit from WAL mode and periodic `VACUUM`/`ANALYZE`.

---

### Minimal checklist
- [x] Implement `SQLiteSink` (schema + `PublishEvent`)
- [x] Attach sink with `engine.WithSink(sqliteSink)`
- [x] Also attach sink to context with `events.WithEventSinks(ctx, sqliteSink)`
- [x] Verify events persist for Start/Partial/Final/ToolCall/Error/Interrupt and ToolExecute/ToolResult



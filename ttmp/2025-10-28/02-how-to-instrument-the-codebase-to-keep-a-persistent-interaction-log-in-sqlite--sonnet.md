## How to Instrument the Codebase to Keep a Persistent LLM Interaction Log in SQLite (Sonnet Analysis - 2025-10-28)

### TL;DR
The codebase is already instrumented for comprehensive event logging through the **events.EventSink** interface. To persist LLM interactions to SQLite, you need to:
1. Implement a `SQLiteSink` that implements `events.EventSink`
2. Wire it at engine creation time via `engine.WithSink(sqliteSink)`
3. Also attach it to the context via `events.WithEventSinks(ctx, sqliteSink)`
4. The existing instrumentation will automatically capture all LLM interactions, tool calls, and execution results

**No engine code needs modification** - all instrumentation points already exist and publish events.

---

### Architecture Overview

The codebase follows an event-driven architecture where:
- **Engines** (OpenAI, Claude, Gemini) publish inference lifecycle events
- **Tool executors** publish tool execution events
- **EventSinks** receive and persist these events
- **Context propagation** ensures tool execution events are captured

```
┌─────────────────────────────────────────────────────────────┐
│                      User Application                        │
└──────────────────┬──────────────────────────────────────────┘
                   │
                   ▼
┌─────────────────────────────────────────────────────────────┐
│  Engine Factory (creates engine with EventSink attached)    │
└──────────────────┬──────────────────────────────────────────┘
                   │
                   ▼
┌─────────────────────────────────────────────────────────────┐
│        Engine.RunInference(ctx, turn)                       │
│  ┌───────────────────────────────────────────────────────┐  │
│  │  1. publishEvent(start)                               │  │
│  │  2. Stream LLM responses                              │  │
│  │  3. publishEvent(partial)  [multiple times]           │  │
│  │  4. publishEvent(tool-call) [for each tool]           │  │
│  │  5. publishEvent(final/error/interrupt)               │  │
│  └───────────────────────────────────────────────────────┘  │
└──────────────────┬──────────────────────────────────────────┘
                   │
                   ▼
┌─────────────────────────────────────────────────────────────┐
│  Tool Executor (via context EventSinks)                     │
│  ┌───────────────────────────────────────────────────────┐  │
│  │  1. PublishStart(tool-call-execute)                   │  │
│  │  2. Execute tool function                             │  │
│  │  3. PublishResult(tool-call-execution-result)         │  │
│  └───────────────────────────────────────────────────────┘  │
└──────────────────┬──────────────────────────────────────────┘
                   │
                   ▼
┌─────────────────────────────────────────────────────────────┐
│              EventSink.PublishEvent(event)                  │
│  ┌───────────────────────────────────────────────────────┐  │
│  │  - SQLiteSink → INSERT INTO events (...)              │  │
│  │  - WatermillSink → Publish to message bus             │  │
│  │  - FileSink → Write to NDJSON                         │  │
│  │  - Custom sinks...                                    │  │
│  └───────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

---

### Core Components and Instrumentation Points

#### 1. Event Types (pkg/events/chat-events.go)

All events implement the `Event` interface:
```go
type Event interface {
    Type() EventType
    Metadata() EventMetadata
    Payload() []byte
}
```

**Event Types Already Instrumented:**
- `EventTypeStart` - Inference begins
- `EventTypePartialCompletion` - Streaming text delta
- `EventTypeToolCall` - Model requests tool invocation
- `EventTypeFinal` - Inference completes successfully
- `EventTypeError` - Inference fails
- `EventTypeInterrupt` - Inference interrupted
- `EventTypeToolCallExecute` - Tool execution begins
- `EventTypeToolCallExecutionResult` - Tool execution completes

**Event Metadata (pkg/events/metadata.go):**
```go
type EventMetadata struct {
    ID          uuid.UUID  // Unique event ID
    RunID       string     // Correlation: which conversation run
    TurnID      string     // Correlation: which turn in the run
    Model       string
    Temperature *float64
    TopP        *float64
    MaxTokens   *int
    StopReason  *string
    Usage       *Usage     // Token counts
    DurationMs  *int64     // Timing
    Extra       map[string]interface{}
}

type Usage struct {
    InputTokens              int
    OutputTokens             int
    CachedTokens             int // OpenAI cache
    CacheCreationInputTokens int // Claude cache write
    CacheReadInputTokens     int // Claude cache read
}
```

---

#### 2. Engine Instrumentation (pkg/steps/ai/)

All three engines follow the same pattern:

##### **OpenAI Engine (pkg/steps/ai/openai/engine_openai.go)**

**Function: `OpenAIEngine.RunInference(ctx, turn)`** (lines 47-403)
- **Entry point** for OpenAI inference
- Already publishes events at key points:
  - Line 212: `e.publishEvent(ctx, events.NewStartEvent(metadata))`
  - Lines 240-350: Loop over streaming responses, publishes partial and tool-call events
  - Line 398: `e.publishEvent(ctx, events.NewFinalEvent(metadata, message))`
  - Line 223: `e.publishEvent(ctx, events.NewErrorEvent(metadata, err))` on errors

**Function: `OpenAIEngine.publishEvent(ctx, event)`** (lines 405-414)
```go
func (e *OpenAIEngine) publishEvent(ctx context.Context, event events.Event) {
    // Publish to engine-level sinks (attached via WithSink)
    for _, sink := range e.config.EventSinks {
        if err := sink.PublishEvent(event); err != nil {
            log.Warn().Err(err).Str("event_type", string(event.Type())).Msg("Failed to publish event to sink")
        }
    }
    // Publish to context sinks (attached via WithEventSinks)
    events.PublishEventToContext(ctx, event)
}
```

**Key Observation:** No modification needed - just attach your SQLiteSink.

##### **Claude Engine (pkg/steps/ai/claude/engine_claude.go)**

**Function: `ClaudeEngine.RunInference(ctx, turn)`** (lines 44-229)
- Uses `ContentBlockMerger` to normalize Claude's streaming events
- Already publishes events via the same pattern

**Function: `ClaudeEngine.publishEvent(ctx, event)`** (lines 231-240)
- Identical pattern to OpenAI

**ContentBlockMerger (pkg/steps/ai/claude/content-block-merger.go):**
- Normalizes Claude's complex streaming format into standard events
- Events are published by the engine after processing merger output

##### **Gemini Engine (pkg/steps/ai/gemini/engine_gemini.go)**

**Function: `GeminiEngine.RunInference(ctx, turn)`** (lines 91-337)
- Similar event publishing pattern
- Lines 241, 275, 290, 328: Event publishing

**Function: `GeminiEngine.publishEvent(ctx, event)`** (lines 439-447)
- Same pattern

---

#### 3. Tool Execution Instrumentation (pkg/inference/tools/)

##### **BaseToolExecutor (pkg/inference/tools/base_executor.go)**

**Function: `BaseToolExecutor.ExecuteToolCall(ctx, call, registry)`** (lines 122-175)
- **Entry point** for tool execution
- Line 142: `b.ToolExecutorExt.PublishStart(ctx, call, maskedArgs)`
- Line 173: `b.ToolExecutorExt.PublishResult(ctx, call, result)`
- Captures duration automatically (line 123: `start := time.Now()`, line 169: `Duration: time.Since(start)`)

**Function: `BaseToolExecutor.PublishStart(ctx, call, masked)`** (lines 73-78)
```go
func (b *BaseToolExecutor) PublishStart(ctx context.Context, call ToolCall, masked string) {
    events.PublishEventToContext(ctx, events.NewToolCallExecuteEvent(
        events.EventMetadata{},
        events.ToolCall{ID: call.ID, Name: call.Name, Input: masked},
    ))
}
```

**Function: `BaseToolExecutor.PublishResult(ctx, call, res)`** (lines 80-100)
```go
func (b *BaseToolExecutor) PublishResult(ctx context.Context, call ToolCall, res *ToolResult) {
    // Serializes result to JSON
    // Publishes EventToolCallExecutionResult with payload and duration
    events.PublishEventToContext(ctx, events.NewToolCallExecutionResultEvent(
        events.EventMetadata{},
        events.ToolResult{ID: call.ID, Result: payload},
    ))
}
```

**Key Observation:** These functions publish to **context sinks only** - this is why you must attach your sink via `events.WithEventSinks(ctx, sqliteSink)`.

##### **Tool Registry (pkg/inference/tools/registry.go)**

**Function: `InMemoryToolRegistry.RegisterTool(name, def)`** (lines 34-50)
- Not directly instrumented for events
- Consider logging if you want to track which tools were registered

---

#### 4. Event Context Propagation (pkg/events/context.go)

**Function: `WithEventSinks(ctx, sinks...)`** (lines 19-27)
- Attaches EventSinks to a context
- Used to ensure tool executor events are captured

**Function: `PublishEventToContext(ctx, event)`** (lines 39-52)
```go
func PublishEventToContext(ctx context.Context, event Event) {
    sinks := GetEventSinks(ctx)
    if len(sinks) == 0 {
        return
    }
    for _, sink := range sinks {
        _ = sink.PublishEvent(event)  // Best-effort
    }
}
```

**Key Function:** This is called by:
- Engine `publishEvent` methods (to also publish to context sinks)
- Tool executor `PublishStart` and `PublishResult` (context only)

---

#### 5. Engine Factory and Configuration (pkg/inference/engine/)

##### **Options (pkg/inference/engine/options.go)**

**Function: `WithSink(sink)`** (lines 24-28)
```go
func WithSink(sink events.EventSink) Option {
    return func(c *Config) error {
        c.EventSinks = append(c.EventSinks, sink)
        return nil
    }
}
```

**Usage Example from pkg/inference/fixtures/fixtures.go:**
```go
// Line 183
engOpts := []engine.Option{engine.WithSink(sink)}

// Line 188
eng, err := engineFactory(st, engOpts...)

// Line 203 (also attach to context for tool events)
runCtx := events.WithEventSinks(ctx, sqliteSink)
```

##### **Engine Factory (pkg/inference/engine/factory/factory.go)**

**Function: `StandardEngineFactory.CreateEngine(settings, options...)`** (lines 47-79)
- Delegates to provider-specific constructors
- Passes options through to engines

**Provider Constructors:**
- `openai.NewOpenAIEngine(settings, options...)` 
- `claude.NewClaudeEngine(settings, options...)`
- `gemini.NewGeminiEngine(settings, options...)`

Each constructor applies options to the engine's internal config.

---

#### 6. Tool Calling Loop (pkg/inference/toolhelpers/helpers.go)

**Function: `RunToolCallingLoop(ctx, turn, registry, config, eng)`** (lines 272-372)
- **High-level orchestration** for multi-turn tool calling
- Line 320: `updated, err := eng.RunInference(ctx, t)` - triggers engine events
- Line 342: `results := ExecuteToolCallsTurn(ctx, calls, registry)` - triggers tool events
- Optional snapshot hooks (lines 317-319, 325-327, 362-364) for debugging

**Key Observation:** No direct instrumentation needed here - just ensures the flow calls engines and tools correctly.

---

#### 7. Middleware (pkg/inference/middleware/)

##### **Tool Middleware (pkg/inference/middleware/tool_middleware.go)**

**Function: `NewToolMiddleware(toolbox, config)`** (lines 75-155)
- Wraps engine calls with tool execution loops
- Line 83: `updated, err := next(ctx, current)` - triggers engine events
- Line 116: `results, err := executeToolCallsTurn(...)` - triggers tool events

**Function: `executeToolCallsTurn(ctx, calls, toolbox, timeout)`** (not shown in search, but inferred)
- Uses `BaseToolExecutor` internally, which publishes events

---

#### 8. Correlation IDs (pkg/turns/types.go)

**Turn Structure (lines 91-100):**
```go
type Turn struct {
    ID      string   // TurnID - unique identifier for this turn
    RunID   string   // RunID - conversation/session identifier
    Blocks  []Block  // Ordered blocks (user input, assistant output, tool calls, etc.)
    Metadata map[string]interface{}
    Data     map[string]interface{}  // Carries tool registry, config
}
```

**Where IDs are assigned:**
- `Turn.ID` is typically set when creating a new turn (uuid.NewString())
- `Turn.RunID` is set at the conversation/run level
- These are propagated to `EventMetadata.TurnID` and `EventMetadata.RunID` in engine code (e.g., lines 199-202 in openai/engine_openai.go)

**How to ensure correlation:**
1. When creating a Turn, assign `turn.ID` and `turn.RunID`
2. Engines automatically copy these to event metadata
3. SQLite sink persists these for correlation

---

### Functions to Instrument (Summary Table)

| Component | File | Function | Lines | Already Instrumented? | What It Publishes |
|-----------|------|----------|-------|----------------------|-------------------|
| **OpenAI Engine** | `pkg/steps/ai/openai/engine_openai.go` | `RunInference` | 47-403 | ✅ Yes | start, partial, tool-call, final, error |
| | | `publishEvent` | 405-414 | ✅ Yes | (helper) |
| **Claude Engine** | `pkg/steps/ai/claude/engine_claude.go` | `RunInference` | 44-229 | ✅ Yes | start, partial, tool-call, final, error, interrupt |
| | | `publishEvent` | 231-240 | ✅ Yes | (helper) |
| **Gemini Engine** | `pkg/steps/ai/gemini/engine_gemini.go` | `RunInference` | 91-337 | ✅ Yes | start, partial, tool-call, final, error |
| | | `publishEvent` | 439-447 | ✅ Yes | (helper) |
| **Tool Executor** | `pkg/inference/tools/base_executor.go` | `ExecuteToolCall` | 122-175 | ✅ Yes | tool-call-execute, tool-call-execution-result |
| | | `PublishStart` | 73-78 | ✅ Yes | (called by ExecuteToolCall) |
| | | `PublishResult` | 80-100 | ✅ Yes | (called by ExecuteToolCall) |
| **Context Publishing** | `pkg/events/context.go` | `PublishEventToContext` | 39-52 | ✅ Yes | (dispatch to sinks) |
| | | `WithEventSinks` | 19-27 | ✅ Yes | (attach sinks to ctx) |
| **Engine Factory** | `pkg/inference/engine/factory/factory.go` | `CreateEngine` | 47-79 | N/A | (creates engines) |
| **Options** | `pkg/inference/engine/options.go` | `WithSink` | 24-28 | N/A | (config helper) |

**Legend:**
- ✅ **Already Instrumented** - No code changes needed, just attach your sink
- N/A - Configuration/factory code, not an instrumentation point

---

### Implementation: SQLiteSink

#### Interface to Implement

```go
// pkg/events/sink.go
type EventSink interface {
    PublishEvent(event Event) error
}
```

#### Example Implementation (Sketch)

```go
package middleware // or pkg/events

import (
    "database/sql"
    "encoding/json"
    "time"
    _ "github.com/mattn/go-sqlite3"
    "github.com/go-go-golems/geppetto/pkg/events"
)

type SQLiteSink struct {
    db *sql.DB
}

func NewSQLiteSink(path string) (*SQLiteSink, error) {
    db, err := sql.Open("sqlite3", path+"?_busy_timeout=5000&_fk=1")
    if err != nil {
        return nil, err
    }
    
    // Initialize schema
    _, err = db.Exec(`
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
    `)
    if err != nil {
        return nil, err
    }
    
    return &SQLiteSink{db: db}, nil
}

func (s *SQLiteSink) PublishEvent(e events.Event) error {
    meta := e.Metadata()
    nowMs := time.Now().UnixMilli()
    
    // Marshal full event for raw_payload
    raw, _ := json.Marshal(e)
    
    // Extract event-specific fields
    var delta, completion, text, toolID, toolName, toolInput, toolResult, errorStr string
    
    switch ev := e.(type) {
    case *events.EventPartialCompletion:
        delta = ev.Delta
        completion = ev.Completion
    case *events.EventFinal:
        text = ev.Text
    case *events.EventInterrupt:
        text = ev.Text
    case *events.EventError:
        errorStr = ev.ErrorString
    case *events.EventToolCall:
        toolID = ev.ToolCall.ID
        toolName = ev.ToolCall.Name
        toolInput = ev.ToolCall.Input
    case *events.EventToolCallExecute:
        toolID = ev.ToolCall.ID
        toolName = ev.ToolCall.Name
        toolInput = ev.ToolCall.Input
    case *events.EventToolCallExecutionResult:
        toolID = ev.ToolResult.ID
        toolResult = ev.ToolResult.Result
    }
    
    // Convert pointers to nullable types
    var inTok, outTok, cachedTok, cacheCreateTok, cacheReadTok interface{}
    if meta.Usage != nil {
        inTok = meta.Usage.InputTokens
        outTok = meta.Usage.OutputTokens
        if meta.Usage.CachedTokens != 0 {
            cachedTok = meta.Usage.CachedTokens
        }
        if meta.Usage.CacheCreationInputTokens != 0 {
            cacheCreateTok = meta.Usage.CacheCreationInputTokens
        }
        if meta.Usage.CacheReadInputTokens != 0 {
            cacheReadTok = meta.Usage.CacheReadInputTokens
        }
    }
    
    _, err := s.db.Exec(`
        INSERT INTO events (
            id, type, created_at, run_id, turn_id, 
            model, temperature, top_p, max_tokens, stop_reason, duration_ms,
            input_tokens, output_tokens, cached_tokens, 
            cache_creation_input_tokens, cache_read_input_tokens,
            delta, completion, text, 
            tool_id, tool_name, tool_input, tool_result, 
            error_string, raw_payload
        ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
    `,
        meta.ID.String(), string(e.Type()), nowMs, 
        nullStr(meta.RunID), nullStr(meta.TurnID),
        nullStr(meta.Model), meta.Temperature, meta.TopP, 
        meta.MaxTokens, nullStrPtr(meta.StopReason), meta.DurationMs,
        inTok, outTok, cachedTok, cacheCreateTok, cacheReadTok,
        nullStr(delta), nullStr(completion), nullStr(text),
        nullStr(toolID), nullStr(toolName), nullStr(toolInput), nullStr(toolResult),
        nullStr(errorStr), string(raw),
    )
    
    return err
}

func nullStr(s string) interface{} {
    if s == "" { return nil }
    return s
}

func nullStrPtr(s *string) interface{} {
    if s == nil || *s == "" { return nil }
    return *s
}
```

---

### Wiring the Sink (Usage Examples)

#### Example 1: Basic Engine Usage

```go
package main

import (
    "context"
    "github.com/go-go-golems/geppetto/pkg/inference/engine"
    "github.com/go-go-golems/geppetto/pkg/inference/engine/factory"
    "github.com/go-go-golems/geppetto/pkg/events"
    "github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
    "github.com/go-go-golems/geppetto/pkg/turns"
)

func main() {
    // 1. Create SQLite sink
    sqliteSink, err := NewSQLiteSink("./interactions.sqlite3")
    if err != nil {
        panic(err)
    }
    
    // 2. Create engine with sink attached
    stepSettings := &settings.StepSettings{ /* ... */ }
    engineFactory := factory.NewStandardEngineFactory()
    eng, err := engineFactory.CreateEngine(stepSettings, engine.WithSink(sqliteSink))
    if err != nil {
        panic(err)
    }
    
    // 3. Attach sink to context (for tool events)
    ctx := context.Background()
    ctx = events.WithEventSinks(ctx, sqliteSink)
    
    // 4. Create turn with correlation IDs
    turn := &turns.Turn{
        ID:    "turn-123",
        RunID: "run-abc",
        Blocks: []turns.Block{
            turns.NewUserTextBlock("What's the weather?"),
        },
    }
    
    // 5. Run inference - events automatically logged to SQLite
    finalTurn, err := eng.RunInference(ctx, turn)
    if err != nil {
        panic(err)
    }
}
```

#### Example 2: With Tool Calling Loop

```go
package main

import (
    "context"
    "github.com/go-go-golems/geppetto/pkg/inference/toolhelpers"
    "github.com/go-go-golems/geppetto/pkg/inference/tools"
    "github.com/go-go-golems/geppetto/pkg/turns"
)

func main() {
    // Setup (same as Example 1)
    sqliteSink, _ := NewSQLiteSink("./interactions.sqlite3")
    eng, _ := engineFactory.CreateEngine(stepSettings, engine.WithSink(sqliteSink))
    ctx := events.WithEventSinks(context.Background(), sqliteSink)
    
    // Create tool registry
    registry := tools.NewInMemoryToolRegistry()
    weatherTool, _ := tools.NewToolFromFunc(
        "get_weather",
        "Get current weather for a location",
        func(location string) (string, error) {
            return "Sunny, 72°F", nil
        },
    )
    registry.RegisterTool("get_weather", *weatherTool)
    
    // Create turn with tools
    turn := &turns.Turn{
        ID:    "turn-456",
        RunID: "run-xyz",
        Blocks: []turns.Block{
            turns.NewUserTextBlock("What's the weather in NYC?"),
        },
        Data: map[string]interface{}{
            turns.DataKeyToolRegistry: registry,
        },
    }
    
    // Run tool calling loop - all events (inference + tools) logged
    config := toolhelpers.DefaultToolCallingConfig()
    finalTurn, err := toolhelpers.RunToolCallingLoop(ctx, turn, registry, config, eng)
    if err != nil {
        panic(err)
    }
}
```

#### Example 3: Multiple Sinks

```go
// You can attach multiple sinks to capture events in different ways
sqliteSink, _ := NewSQLiteSink("./interactions.sqlite3")
fileSink := &FileSink{path: "./events.ndjson"}  // from fixtures
watermillSink := middleware.NewWatermillSink(publisher, "events")

eng, _ := engineFactory.CreateEngine(
    stepSettings,
    engine.WithSink(sqliteSink),
    engine.WithSink(fileSink),
    engine.WithSink(watermillSink),
)

ctx = events.WithEventSinks(ctx, sqliteSink, fileSink, watermillSink)
```

---

### Event Flow Examples

#### Example: Simple Inference (No Tools)

```
User: "Hello, who are you?"

Events published:
1. EventStart           (type=start)           - runID, turnID, model, temp, etc.
2. EventPartialCompletion (type=partial)      - delta="I", completion="I"
3. EventPartialCompletion (type=partial)      - delta=" am", completion="I am"
4. EventPartialCompletion (type=partial)      - delta=" an", completion="I am an"
5. EventPartialCompletion (type=partial)      - delta=" AI", completion="I am an AI"
6. ... (many more partials)
7. EventFinal            (type=final)         - text="I am an AI assistant", usage={...}, duration_ms=523
```

#### Example: Inference with Tool Call

```
User: "What's the weather in NYC?"

Events published:
1. EventStart                    (type=start)
2. EventPartialCompletion        (type=partial)      - delta="Let", completion="Let"
3. EventPartialCompletion        (type=partial)      - delta=" me", completion="Let me"
4. EventPartialCompletion        (type=partial)      - delta=" check", completion="Let me check"
5. EventToolCall                 (type=tool-call)    - tool_id="call_abc123", tool_name="get_weather", tool_input='{"location":"NYC"}'
6. EventFinal                    (type=final)        - text="", usage={...}
7. EventToolCallExecute          (type=tool-call-execute) - tool_id="call_abc123", tool_name="get_weather"
8. EventToolCallExecutionResult  (type=tool-call-execution-result) - tool_id="call_abc123", tool_result='{"temp":72,"conditions":"sunny"}'

[Second turn - model processes tool result]
9. EventStart                    (type=start)
10. EventPartialCompletion       (type=partial)      - delta="The"
11. EventPartialCompletion       (type=partial)      - delta=" weather"
12. ...
13. EventFinal                   (type=final)        - text="The weather in NYC is sunny and 72°F", usage={...}
```

---

### Correlation and Query Examples

Once events are in SQLite, you can reconstruct interactions:

#### Query 1: All Events for a Turn

```sql
SELECT 
    type, created_at, model, 
    delta, completion, text, 
    tool_name, tool_input, tool_result,
    input_tokens, output_tokens, duration_ms
FROM events
WHERE run_id = 'run-abc' AND turn_id = 'turn-123'
ORDER BY created_at ASC;
```

#### Query 2: Token Usage by Run

```sql
SELECT 
    run_id,
    SUM(input_tokens) as total_input,
    SUM(output_tokens) as total_output,
    SUM(input_tokens + output_tokens) as total_tokens,
    COUNT(DISTINCT turn_id) as num_turns
FROM events
WHERE type = 'final'
GROUP BY run_id;
```

#### Query 3: Tool Execution Performance

```sql
SELECT 
    tool_name,
    COUNT(*) as execution_count,
    AVG(duration_ms) as avg_duration_ms,
    MIN(duration_ms) as min_duration_ms,
    MAX(duration_ms) as max_duration_ms
FROM events
WHERE type = 'tool-call-execution-result'
GROUP BY tool_name
ORDER BY avg_duration_ms DESC;
```

#### Query 4: Reconstruct Full Conversation

```sql
-- Get all turns in a run, in order
WITH turn_summaries AS (
    SELECT 
        turn_id,
        MIN(created_at) as turn_start,
        MAX(CASE WHEN type = 'final' THEN text END) as assistant_response
    FROM events
    WHERE run_id = 'run-abc'
    GROUP BY turn_id
)
SELECT * FROM turn_summaries ORDER BY turn_start ASC;
```

---

### Additional Instrumentation Considerations

#### Optional: Conversation Manager (pkg/conversation/)

The `conversation.Manager` interface manages conversation trees:
- `Manager.AppendMessages(msgs...)` (pkg/conversation/manager-impl.go, lines 193-246)
- Already has extensive trace logging
- If you want to track conversation-level operations (not just LLM calls), consider:
  - Wrapping `ManagerImpl.AppendMessages` to log to SQLite
  - Adding conversation metadata to your SQLite schema

Not required for LLM interaction logging, but useful for higher-level conversation tracking.

#### Optional: Raw HTTP/SSE Capture (pkg/inference/engine/debugtap.go)

If you need to capture raw provider HTTP requests/responses:
- Use `engine.WithDebugTap(ctx, tap)`
- Implement `DebugTap` interface:
  ```go
  type DebugTap interface {
      OnHTTP(req *http.Request, resp *http.Response, body []byte)
      OnSSE(event string, data []byte)
  }
  ```
- Example: `pkg/inference/fixtures/disktap.go`

Useful for debugging provider issues, but not needed for standard logging.

#### Optional: Middleware Hooks

If you're using middleware (e.g., `NewToolMiddleware`), events are already published by underlying engines/tools. No additional instrumentation needed.

---

### Performance Considerations

1. **Use prepared statements** for the SQLite INSERT to reduce parsing overhead
2. **Enable WAL mode** (`PRAGMA journal_mode=WAL`) for better concurrent write performance
3. **Batch inserts** if you have many events in quick succession (accumulate in memory, flush periodically)
4. **Async writing** - consider buffering events and writing in a background goroutine
5. **Indexing** - ensure indexes on `(run_id, turn_id)` and `created_at` for fast queries

Example async sink:

```go
type AsyncSQLiteSink struct {
    inner   *SQLiteSink
    queue   chan events.Event
    wg      sync.WaitGroup
}

func (a *AsyncSQLiteSink) PublishEvent(e events.Event) error {
    select {
    case a.queue <- e:
        return nil
    default:
        return fmt.Errorf("event queue full")
    }
}

func (a *AsyncSQLiteSink) Start() {
    a.wg.Add(1)
    go func() {
        defer a.wg.Done()
        for e := range a.queue {
            _ = a.inner.PublishEvent(e)
        }
    }()
}

func (a *AsyncSQLiteSink) Stop() {
    close(a.queue)
    a.wg.Wait()
}
```

---

### Testing Your Instrumentation

Use the existing fixture framework to test:

```go
package main

import (
    "testing"
    "github.com/go-go-golems/geppetto/pkg/inference/fixtures"
    "github.com/go-go-golems/geppetto/pkg/turns"
)

func TestSQLiteSinkCapture(t *testing.T) {
    sqliteSink, err := NewSQLiteSink("./test_events.sqlite3")
    if err != nil {
        t.Fatal(err)
    }
    defer os.Remove("./test_events.sqlite3")
    
    turn := &turns.Turn{
        ID:    "test-turn",
        RunID: "test-run",
        Blocks: []turns.Block{
            turns.NewUserTextBlock("Hello"),
        },
    }
    
    opts := fixtures.ExecuteOptions{
        OutDir:       "./test_out",
        EchoEvents:   true,
        PrintTurns:   true,
    }
    
    // Custom engine factory that attaches our sink
    opts.EngineFactory = func(st *settings.StepSettings, opts ...engine.Option) (engine.Engine, error) {
        opts = append(opts, engine.WithSink(sqliteSink))
        return factory.NewStandardEngineFactory().CreateEngine(st, opts...)
    }
    
    _, err = fixtures.ExecuteFixture(context.Background(), turn, nil, stepSettings, opts)
    if err != nil {
        t.Fatal(err)
    }
    
    // Verify events in SQLite
    rows, err := sqliteSink.db.Query("SELECT type, run_id, turn_id FROM events ORDER BY created_at")
    if err != nil {
        t.Fatal(err)
    }
    defer rows.Close()
    
    eventCount := 0
    for rows.Next() {
        var typ, runID, turnID string
        rows.Scan(&typ, &runID, &turnID)
        t.Logf("Event: type=%s, run_id=%s, turn_id=%s", typ, runID, turnID)
        eventCount++
        
        if runID != "test-run" {
            t.Errorf("Expected run_id=test-run, got %s", runID)
        }
    }
    
    if eventCount == 0 {
        t.Error("No events captured")
    }
}
```

---

### Summary Checklist

To instrument your application for SQLite logging:

- [x] ✅ **Implement `SQLiteSink` type** that implements `events.EventSink`
- [x] ✅ **Create SQLite schema** with events table (see schema above)
- [x] ✅ **Attach sink to engine** via `engine.WithSink(sqliteSink)` when creating engine
- [x] ✅ **Attach sink to context** via `events.WithEventSinks(ctx, sqliteSink)` before calling `RunInference`
- [x] ✅ **Ensure Turn has RunID and TurnID set** for correlation
- [x] ✅ **Handle event type switches** in `PublishEvent` to extract event-specific fields
- [x] ✅ **Test with fixture framework** or real usage

**No engine or tool code needs modification** - all instrumentation is already in place!

---

### Reference: Complete File Paths

For easy navigation:

**Core Event System:**
- `pkg/events/sink.go` - EventSink interface
- `pkg/events/chat-events.go` - Event types and constructors
- `pkg/events/metadata.go` - EventMetadata and Usage
- `pkg/events/context.go` - Context sink management
- `pkg/events/publish.go` - PublisherManager (watermill-based, optional)

**Engines:**
- `pkg/steps/ai/openai/engine_openai.go` - OpenAI engine with event publishing
- `pkg/steps/ai/claude/engine_claude.go` - Claude engine with event publishing
- `pkg/steps/ai/gemini/engine_gemini.go` - Gemini engine with event publishing

**Tools:**
- `pkg/inference/tools/base_executor.go` - Tool execution with event publishing
- `pkg/inference/tools/registry.go` - Tool registration
- `pkg/inference/tools/definition.go` - Tool definitions

**Infrastructure:**
- `pkg/inference/engine/options.go` - WithSink and other options
- `pkg/inference/engine/factory/factory.go` - Engine factory
- `pkg/inference/toolhelpers/helpers.go` - Tool calling loop
- `pkg/inference/middleware/tool_middleware.go` - Tool middleware

**Examples:**
- `pkg/inference/fixtures/fixtures.go` - Fixture framework with sink wiring example
- `pkg/inference/middleware/sink_watermill.go` - Example EventSink implementation
- `cmd/examples/simple-inference/main.go` - Simple usage example

**Turns and Correlation:**
- `pkg/turns/types.go` - Turn, Block, Run types with IDs
- `pkg/turns/helpers_blocks.go` - Block constructors
- `pkg/turns/builders.go` - TurnBuilder


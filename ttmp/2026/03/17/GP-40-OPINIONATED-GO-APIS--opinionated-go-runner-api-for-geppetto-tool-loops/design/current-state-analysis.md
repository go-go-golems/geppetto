---
Title: "Current State: Geppetto/Pinocchio Usage Patterns and Pain Points"
DocType: analysis
Ticket: GP-40-OPINIONATED-GO-APIS
Topics:
  - geppetto
  - pinocchio
  - api-design
  - tools
  - middleware
Status: active
RelatedFiles:
  - Path: /home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/inference/toolloop/loop.go
    Note: Core tool loop orchestration — the Loop.RunLoop algorithm
  - Path: /home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/inference/toolloop/enginebuilder/builder.go
    Note: Engine builder that wires engine+middleware+tools+loop+events
  - Path: /home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/inference/session/session.go
    Note: Session lifecycle with async ExecutionHandle
  - Path: /home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/inference/tools/registry.go
    Note: Tool registry interface and InMemoryToolRegistry
  - Path: /home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/inference/engine/engine.go
    Note: Core Engine interface — RunInference(ctx, Turn) → Turn
  - Path: /home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/pkg/cmds/cmd.go
    Note: PinocchioCommand — existing opinionated wrapper
  - Path: /home/manuel/workspaces/2026-03-17/add-opinionated-apis/2026-03-14--cozodb-editor/backend/pkg/hints/engine.go
    Note: CozoDB editor — real-world inference setup with pain points
  - Path: /home/manuel/workspaces/2026-03-17/add-opinionated-apis/2026-03-16--gec-rag/internal/webchat/server.go
    Note: GEC-RAG — webchat with tool catalog and profile resolution
  - Path: /home/manuel/workspaces/2026-03-17/add-opinionated-apis/temporal-relationships/internal/extractor/gorunner/loop.go
    Note: Temporal relationships — batch extraction inference loop
---

# Current State: Geppetto/Pinocchio Usage Patterns and Pain Points

## 1. Architecture Overview

Geppetto provides a layered architecture for building LLM applications in Go:

### Layer Stack (bottom-up)

```
┌─────────────────────────────────────────────────────────┐
│  Consumer Applications (cozodb-editor, gec-rag, etc.)   │
├─────────────────────────────────────────────────────────┤
│  pinocchio/pkg/cmds — PinocchioCommand, RunContext      │
│  pinocchio/pkg/webchat — ConvManager, LLMLoopRunner     │
├─────────────────────────────────────────────────────────┤
│  geppetto/pkg/inference/toolloop/enginebuilder — Builder │
│  geppetto/pkg/inference/session — Session, ExecHandle    │
├─────────────────────────────────────────────────────────┤
│  geppetto/pkg/inference/toolloop — Loop orchestration    │
│  geppetto/pkg/inference/tools — Registry, Executor       │
├─────────────────────────────────────────────────────────┤
│  geppetto/pkg/inference/middleware — Middleware chain     │
│  geppetto/pkg/inference/engine — Engine interface         │
├─────────────────────────────────────────────────────────┤
│  geppetto/pkg/turns — Turn, Block, typed keys            │
│  geppetto/pkg/events — EventSink, Event types            │
└─────────────────────────────────────────────────────────┘
```

### Core Interfaces

**Engine** (`pkg/inference/engine/engine.go`):
```go
type Engine interface {
    RunInference(ctx context.Context, t *turns.Turn) (*turns.Turn, error)
}
```

**Middleware** (`pkg/inference/middleware/middleware.go`):
```go
type HandlerFunc func(ctx context.Context, t *turns.Turn) (*turns.Turn, error)
type Middleware func(HandlerFunc) HandlerFunc
```

**ToolRegistry** (`pkg/inference/tools/registry.go`):
```go
type ToolRegistry interface {
    RegisterTool(name string, def ToolDefinition) error
    GetTool(name string) (*ToolDefinition, bool)
    ListTools() map[string]ToolDefinition
    // ...
}
```

**Session** (`pkg/inference/session/session.go`):
```go
type Session struct {
    SessionID string
    Turns     []*turns.Turn
    Builder   EngineBuilder
    // ...
}
```

### Data Flow: User Input → LLM → Tools → Response

```
User prompt
  → Turn with SystemBlock + UserBlock
    → Session.StartInference(ctx)
      → EngineBuilder.Build() → runner
        → runner.RunInference(ctx, turn)
          → middleware.Chain(engine, mw1, mw2, ...)
            → Engine.RunInference (provider API call)
              → Turn with AssistantBlocks (possibly ToolCallBlocks)
                → Loop: ExtractPendingToolCalls
                  → ToolExecutor.Execute each call
                    → AppendToolResultBlocks
                      → Re-inference with results
                        → Final Turn with text response
```

## 2. Usage Patterns Across Consumer Projects

### 2.1 CozoDB Editor (`2026-03-14--cozodb-editor`)

**Use case:** Single-turn hint generation with structured extraction from streaming output.

**Pattern observed** (`backend/pkg/hints/engine.go:38-158`):
```go
// 1. Settings ceremony (~30 lines)
stepSettings := aisettings.NewStepSettings()
apiType := aitypes.ApiTypeClaude
stepSettings.Chat.ApiType = &apiType
streaming := true
stepSettings.Chat.Stream = &streaming
model := "claude-sonnet-4-20250514"
stepSettings.Chat.ModelName = &model
maxTokens := 8192
stepSettings.Chat.MaxResponseTokens = &maxTokens
stepSettings.Chat.APIKeys = map[string]string{"claude": os.Getenv("ANTHROPIC_API_KEY")}
stepSettings.Chat.BaseUrls = map[string]string{}

// 2. Engine creation
eng, err := factory.NewEngineFromStepSettings(stepSettings)

// 3. Turn construction
turn, _ := turns.NewTurnBuilder().
    AddSystemBlock(systemPrompt).
    AddUserBlock(userMessage).
    Build()

// 4. Event sink wiring (~10 lines)
streamingSink := newStreamingTextSink(onDelta)
filterSink := structuredsink.NewFilteringSinkWithContext(ctx, extractors...)

// 5. Session + builder + inference (~15 lines)
sess := session.NewSession()
sess.Builder = enginebuilder.New(
    enginebuilder.WithBase(eng),
    enginebuilder.WithEventSinks(filterSink),
    enginebuilder.WithStepController(stepCtrl),
)
sess.Append(turn)
handle, _ := sess.StartInference(ctx)
resultTurn, _ := handle.Wait()

// 6. Extract text from result
text := extractAssistantText(resultTurn)
```

**Notable:** No tool use — pure prompt-based with structured extraction via XML tags in output.

### 2.2 GEC-RAG / CoinVault (`2026-03-16--gec-rag`)

**Use case:** Interactive webchat with tool catalog (calculator, SQL, product search) and profile-based configuration.

**Pattern observed** (`internal/webchat/`):
```go
// 1. Settings from config (~20 lines in runtime.go)
stepSettings := aisettings.StepSettingsFromConfig(cfg)

// 2. Tool catalog definition (~70 lines in tool_catalog.go)
catalog := &ToolCatalog{entries: []ToolEntry{
    {Name: "calc", Registrar: toolRegistrar("calc", "calculator", calculatorTool)},
    {Name: "sql_query", Registrar: sqlToolRegistrar(db)},
    // ...
}}

// 3. Registry building per session (~25 lines in tool_catalog.go)
registry := geptools.NewInMemoryToolRegistry()
for _, entry := range catalog.entries {
    if allowed[entry.Name] { entry.Registrar(registry) }
}

// 4. Runtime composition (~40 lines in runtime.go)
eng, _ := infruntime.BuildEngineFromSettingsWithMiddlewares(ctx, stepSettings, sysPrompt, nil)
return infruntime.ComposedRuntime{Engine: eng, SystemPrompt: sysPrompt, AllowedTools: tools}

// 5. Session preparation (~60 lines in configurable_loop_runner_prepare.go)
sess := session.NewSession()
sess.Builder = &enginebuilder.Builder{
    Base:           eng,
    Registry:       registry,
    LoopConfig:     toolloop.NewLoopConfig().WithMaxIterations(20),
    ToolConfig:     geptools.DefaultToolConfig().WithExecutionTimeout(60*time.Second),
    EventSinks:     sinks,
    StepController: stepCtrl,
    SnapshotHook:   snapshotHook,
    Persister:      persister,
}
```

**Notable:** Dual registration path (server-level + per-session), profile resolution is 250+ lines.

### 2.3 Temporal Relationships (`temporal-relationships`)

**Use case:** Batch extraction with multi-iteration outer loop, tool-augmented inner loop.

**Pattern observed** (`internal/extractor/gorunner/loop.go:22-319`):
```go
// 1. Engine from profile or settings (~130 lines in config.go)
eng, _ := resolveEngine(cfg, profileRegistry)

// 2. Tool registry (~80 lines in tools_persistence.go)
registry := geptools.NewInMemoryToolRegistry()
entityhistory.RegisterQueryTool(registry, scopedDB, queryOpts)
transcripthistory.RegisterQueryTool(registry, scopedDB, queryOpts)

// 3. Session + builder (~20 lines in loop.go)
sess := session.NewSession()
sess.Builder = &enginebuilder.Builder{
    Base:       eng,
    EventSinks: sinks,
    Registry:   registry,
    LoopConfig: toolloop.NewLoopConfig().WithMaxIterations(cfg.ToolLoopMaxIterations),
}

// 4. Multi-iteration outer loop (~70 lines)
for i := 0; i < maxIterations; i++ {
    sess.AppendNewTurnFromUserPrompt(prompt)
    handle, _ := sess.StartInference(ctx)
    resultTurn, _ := handle.Wait()
    if shouldStop(resultTurn) { break }
}
```

**Notable:** Has its own outer loop on top of geppetto's inner tool loop, with custom stop conditions.

## 3. Pain Points Identified

### 3.1 Settings Ceremony (All projects, ~20-30 lines each)

Every project must manually construct `StepSettings` with pointer allocations:

```go
apiType := aitypes.ApiTypeClaude
stepSettings.Chat.ApiType = &apiType
streaming := true
stepSettings.Chat.Stream = &streaming
// ... repeat for every field
```

**Impact:** 20-30 lines of boilerplate before any inference can happen.

### 3.2 Session/Turn/Builder Assembly (All projects, ~15-20 lines each)

The same wiring pattern appears everywhere:
1. Build Turn with system + user blocks
2. Create Session
3. Create enginebuilder.Builder with 6-8 fields
4. Attach builder to session
5. Append turn to session
6. Start inference, wait for handle

**Impact:** 15-20 lines of identical boilerplate in every inference call site.

### 3.3 Tool Registration Duplication (gec-rag, temporal-relationships)

The tool registrar pattern requires:
```go
def, err := geptools.NewToolFromFunc(name, description, fn)
err = registry.RegisterTool(name, *def)
```

Repeated for every tool. In gec-rag, tools must be registered both at server level AND per-session with filtering.

**Impact:** ~10 lines per tool, duplicated registration paths.

### 3.4 Event Sink Orchestration (cozodb-editor, temporal-relationships)

Manual construction of sink chains:
```go
streamingSink := newStreamingTextSink(onDelta)
filterSink := structuredsink.NewFilteringSinkWithContext(ctx, extractors...)
sinks := []events.EventSink{filterSink, streamingSink}
```

**Impact:** 5-10 lines of plumbing that obscures the actual intent.

### 3.5 Engine Creation Cascade (temporal-relationships, gec-rag)

Complex fallback chains for resolving engine from profiles or settings:
- Try profile registry → try step settings → try config overrides → apply defaults
- Three separate functions in temporal-relationships just for this

**Impact:** 100-130 lines for "give me an engine for this model."

### 3.6 Result Text Extraction (All projects)

After inference, extracting the assistant's text from the Turn requires scanning blocks:
```go
for _, block := range turn.Blocks {
    if block.Kind == turns.BlockKindLLMText {
        text += block.Payload["text"].(string)
    }
}
```

**Impact:** Small but repetitive; should be a one-liner.

### 3.7 Cleanup Management (temporal-relationships, gec-rag)

Manual cleanup function slices:
```go
var closeFns []func()
closeFns = append(closeFns, func() { db.Close() })
// ... later
for i := len(closeFns) - 1; i >= 0; i-- { closeFns[i]() }
```

**Impact:** Error-prone, no guarantee of execution order.

## 4. The Boilerplate Tax

Counting lines of boilerplate to get a working tool-using inference:

| Step | Lines | Present in |
|------|-------|-----------|
| StepSettings construction | 20-30 | All 3 projects |
| Engine creation | 5-15 | All 3 projects |
| Tool registry + registration | 10-80 | gec-rag, temporal |
| Turn construction | 5-10 | All 3 projects |
| Session + builder wiring | 15-25 | All 3 projects |
| Event sink setup | 5-15 | All 3 projects |
| Start inference + wait | 5-10 | All 3 projects |
| Result extraction | 5-10 | All 3 projects |
| **Total** | **70-195** | |

A user who just wants to "call Claude with these tools and this prompt" must write 70-195 lines of infrastructure code before getting their first response.

## 5. What PinocchioCommand Already Solves (and Doesn't)

PinocchioCommand (`pinocchio/pkg/cmds/cmd.go`) already provides:
- YAML-based command definitions
- Template variable interpolation for prompts
- RunMode selection (blocking/interactive/chat)
- Profile selection and switching
- Glazed CLI integration

**But it doesn't solve:**
- The settings/engine/session ceremony (still required under the hood)
- Simple "just run inference" without the full Cobra command framework
- Lightweight tool registration for custom CLI tools
- The gap between "I have a Go function" and "it's a tool in a tool loop"

PinocchioCommand is optimized for the pinocchio CLI itself, not for third-party Go applications that want to embed LLM tool loops.

## 6. Key Insight: Two Distinct Use Cases

**Use Case A: Batch/CLI — "Run this prompt with these tools, give me the result"**
- Single-shot or few-turn
- No UI, no WebSocket, no persistence (usually)
- Examples: cozodb-editor hints, temporal-relationships extraction
- Need: Minimal setup, synchronous execution, structured results

**Use Case B: Interactive/Web — "Start a conversation with tools, stream results"**
- Long-lived sessions
- WebSocket streaming, persistence, profile switching
- Examples: gec-rag webchat, temporal-relationships run-chat
- Need: Conversation management, event routing, cleanup lifecycle

The opinionated runner should primarily target **Use Case A** since it's the "scaffold a powerful tool in a couple of lines" scenario. Use Case B can build on it but requires additional infrastructure.

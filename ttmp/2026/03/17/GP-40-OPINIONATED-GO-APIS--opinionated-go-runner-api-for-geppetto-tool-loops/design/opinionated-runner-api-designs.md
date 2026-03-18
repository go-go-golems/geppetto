---
Title: "Design Proposals: Opinionated Runner API for Geppetto Tool Loops"
DocType: design-doc
Ticket: GP-40-OPINIONATED-GO-APIS
Topics:
  - geppetto
  - pinocchio
  - api-design
  - tools
  - middleware
Status: active
RelatedFiles:
  - Path: /home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/inference/toolloop/enginebuilder/builder.go
    Note: Current builder — what we're wrapping
  - Path: /home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/inference/session/session.go
    Note: Session lifecycle — hidden by the runner
  - Path: /home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/inference/tools/definition.go
    Note: ToolDefinition — tool registration substrate
---

# Design Proposals: Opinionated Runner API for Geppetto Tool Loops

## Design Goals

1. **Zero-to-working in 5-10 lines** — The simplest case should be trivially easy
2. **Tools as first-class citizens** — Adding a Go function as a tool = one call
3. **Middleware composability preserved** — Don't lose geppetto's flexibility
4. **Escape hatches** — Drop to lower-level APIs at any point
5. **Profile-aware** — Profile registries integrate naturally
6. **Streaming-friendly** — Event sinks pluggable without ceremony
7. **Real Go idioms** — Functional options, not YAML config objects

## Design A: Minimal Functional API

### Rationale

The most radical simplification. Inspired by Go's `http.ListenAndServe` — a single function call that "just works" for the common case, with progressive disclosure of complexity through options.

**Philosophy:** If you have to read docs to use the basic case, the API is too complex.

### Core API

```go
package runner

// Run executes a single inference with an optional tool loop.
// It's the "http.ListenAndServe" of LLM inference.
func Run(ctx context.Context, prompt string, opts ...Option) (*Result, error)

// Result is the simplified output of an inference run.
type Result struct {
    Text       string            // Final assistant text (concatenated from all LLM blocks)
    ToolCalls  []ToolCallRecord  // Record of all tool calls made
    Turn       *turns.Turn       // Full turn for advanced inspection
    Usage      *turns.InferenceUsage
    StopReason string
}

type ToolCallRecord struct {
    Name   string
    Args   map[string]any
    Result string
    Error  error
}
```

### Options

```go
// Model selection (defaults to claude-sonnet if ANTHROPIC_API_KEY is set)
func WithModel(model string) Option
func WithProvider(provider string) Option  // "claude", "openai", "ollama"

// Prompts
func WithSystemPrompt(prompt string) Option
func WithMessages(messages ...Message) Option  // For multi-turn

// Tools — the key ergonomic improvement
func WithTool(name, description string, fn any) Option
func WithToolRegistry(reg tools.ToolRegistry) Option

// Middleware
func WithMiddleware(mw ...middleware.Middleware) Option

// Events/Streaming
func WithEventSink(sink events.EventSink) Option
func WithOnDelta(fn func(delta string)) Option  // Simplified streaming callback

// Configuration
func WithMaxTokens(n int) Option
func WithTemperature(t float64) Option
func WithMaxToolIterations(n int) Option
func WithToolTimeout(d time.Duration) Option

// Advanced — escape hatches
func WithEngine(eng engine.Engine) Option        // Bring your own engine
func WithStepSettings(s *settings.StepSettings) Option  // Full settings override
func WithProfile(name string, registries ...string) Option
```

### Real-World Examples

**Example 1: CozoDB hint generation (was ~80 lines, now ~10)**

```go
result, err := runner.Run(ctx, userQuestion,
    runner.WithModel("claude-sonnet-4-20250514"),
    runner.WithSystemPrompt(buildCozoSystemPrompt(schema)),
    runner.WithMaxTokens(8192),
    runner.WithOnDelta(func(delta string) {
        ws.SendDelta(delta)
    }),
)
hints := parseStructuredResponse(result.Text)
```

**Example 2: GEC-RAG with tools (was ~120 lines, now ~15)**

```go
result, err := runner.Run(ctx, userPrompt,
    runner.WithModel("claude-sonnet-4-20250514"),
    runner.WithSystemPrompt(systemPrompt),
    runner.WithTool("calc", "Evaluate math expressions", calculatorTool),
    runner.WithTool("sql_query", "Query the product database", sqlQueryTool(db)),
    runner.WithTool("search_products", "Search product catalog", searchFunc),
    runner.WithMaxToolIterations(20),
    runner.WithToolTimeout(60*time.Second),
    runner.WithOnDelta(sendStreamingDelta),
)
fmt.Println(result.Text)
```

**Example 3: Temporal relationships extraction (was ~100 lines, now ~20)**

```go
result, err := runner.Run(ctx, extractionPrompt,
    runner.WithProfile("extraction", profileRegistryPath),
    runner.WithSystemPrompt(extractionSystemPrompt),
    runner.WithTool("query_entity_history", "Query entity history DB",
        entityhistory.QueryFunc(scopedDB, queryOpts)),
    runner.WithTool("query_transcript_history", "Query transcript DB",
        transcripthistory.QueryFunc(scopedDB, queryOpts)),
    runner.WithMaxToolIterations(6),
    runner.WithEventSink(extractionSink),
)
artifacts := parseExtractionArtifacts(result.Text)
```

### Trade-offs

| Pro | Con |
|-----|-----|
| Extremely simple for common cases | Single function = kitchen-sink parameter list |
| No types to learn upfront | Hard to compose for multi-turn conversations |
| Discoverable via autocomplete on `With*` | Session management hidden = can't reuse sessions |
| Zero ceremony for basic usage | Profile resolution magic may surprise users |

### When this design breaks down

- Multi-turn conversations (need session persistence)
- Custom tool executors with retry policies
- WebSocket/long-lived streaming scenarios
- Fine-grained control over turn construction

---

## Design B: Builder Pattern with Method Chaining

### Rationale

A middle ground between the minimal function and full geppetto plumbing. The Builder pattern is idiomatic Go, gives clear separation between configuration and execution, and naturally supports both single-shot and multi-turn usage.

**Philosophy:** Configuration is explicit but concise. Execution is separate from setup.

### Core API

```go
package runner

type Runner struct { /* internal state */ }

func New(opts ...Option) *Runner

// Configuration (chainable)
func (r *Runner) WithModel(model string) *Runner
func (r *Runner) WithProvider(provider string) *Runner
func (r *Runner) WithSystemPrompt(prompt string) *Runner
func (r *Runner) WithTool(name, description string, fn any) *Runner
func (r *Runner) WithMiddleware(mw middleware.Middleware) *Runner
func (r *Runner) WithEventSink(sink events.EventSink) *Runner
func (r *Runner) WithMaxTokens(n int) *Runner
func (r *Runner) WithTemperature(t float64) *Runner
func (r *Runner) WithMaxToolIterations(n int) *Runner

// Execution
func (r *Runner) Run(ctx context.Context, prompt string) (*Result, error)
func (r *Runner) RunTurn(ctx context.Context, turn *turns.Turn) (*Result, error)
func (r *Runner) Chat(ctx context.Context, prompt string) (*Result, error)  // Appends to history

// Escape hatches
func (r *Runner) Engine() engine.Engine
func (r *Runner) Session() *session.Session
func (r *Runner) ToolRegistry() tools.ToolRegistry
```

### Real-World Examples

**Example 1: Reusable hint engine (CozoDB pattern)**

```go
hintRunner := runner.New(
    runner.WithModel("claude-sonnet-4-20250514"),
    runner.WithMaxTokens(8192),
).WithSystemPrompt(buildCozoSystemPrompt(schema))

// Use multiple times
result1, _ := hintRunner.Run(ctx, "How do I query all users?")
result2, _ := hintRunner.Run(ctx, "Show me joins with orders table")
```

**Example 2: Multi-turn tool conversation**

```go
r := runner.New(runner.WithModel("claude-sonnet-4-20250514")).
    WithSystemPrompt("You are a data analyst assistant.").
    WithTool("sql_query", "Query the database", sqlQueryTool(db)).
    WithTool("chart", "Generate a chart", chartTool).
    WithMaxToolIterations(10)

// Turn 1
result1, _ := r.Chat(ctx, "What tables are in the database?")
fmt.Println(result1.Text)

// Turn 2 (automatically includes previous context)
result2, _ := r.Chat(ctx, "Show me the top 10 customers by revenue")
fmt.Println(result2.Text)

// Turn 3
result3, _ := r.Chat(ctx, "Now chart that data")
fmt.Println(result3.Text)
```

**Example 3: CLI tool with profile support**

```go
r := runner.New(
    runner.WithProfile("fast-extraction", registryPath),
).
    WithTool("query_history", "Query entity history", historyQueryFunc).
    WithTool("query_transcripts", "Query transcripts", transcriptQueryFunc)

for _, doc := range documents {
    result, err := r.Run(ctx, buildExtractionPrompt(doc))
    if err != nil { log.Printf("extraction failed for %s: %v", doc.ID, err) }
    saveArtifacts(doc.ID, result.Text)
}
```

### Trade-offs

| Pro | Con |
|-----|-----|
| Reusable runner instances | Mutable builder = thread safety concerns |
| Natural multi-turn via Chat() | More types to learn than Design A |
| Clear separation: configure then execute | Method chaining can get long |
| Session accessible for advanced use | Runner holds state = lifecycle management |

### When this design breaks down

- Concurrent use of same Runner (needs Clone() or mutex)
- Very short-lived one-off calls (Design A simpler)
- Complex session management beyond linear chat

---

## Design C: Struct Configuration with Functional Options

### Rationale

The most "Go-standard-library" approach. A Config struct holds all settings, functional options modify it, and a single `Run` or `NewRunner` function consumes it. This is the pattern used by `http.Server`, `tls.Config`, etc.

**Philosophy:** Make the configuration inspectable and serializable. Support both programmatic and file-based configuration.

### Core API

```go
package runner

// Config holds all runner configuration. Zero value is usable with defaults.
type Config struct {
    // Model selection
    Model    string `yaml:"model"`     // e.g. "claude-sonnet-4-20250514"
    Provider string `yaml:"provider"`  // "claude", "openai", "ollama"

    // Prompts
    SystemPrompt string `yaml:"system_prompt"`

    // Tool loop
    MaxToolIterations int           `yaml:"max_tool_iterations"`  // default: 10
    ToolTimeout       time.Duration `yaml:"tool_timeout"`         // default: 30s

    // Inference parameters
    MaxTokens   int     `yaml:"max_tokens"`    // default: 4096
    Temperature float64 `yaml:"temperature"`   // default: 0
    Stream      bool    `yaml:"stream"`        // default: true

    // Profile (optional)
    Profile          string   `yaml:"profile"`
    ProfileRegistries []string `yaml:"profile_registries"`
}

// Tool wraps a Go function as an LLM tool.
type Tool struct {
    Name        string
    Description string
    Func        any  // func(Input) (Output, error) or func(ctx, Input) (Output, error)
}

// Run executes a single inference. Simplest entry point.
func Run(ctx context.Context, cfg Config, prompt string, opts ...Option) (*Result, error)

// NewRunner creates a reusable runner from config.
func NewRunner(cfg Config, opts ...Option) (*Runner, error)

// Options for things that don't serialize to YAML
func WithTools(tools ...Tool) Option
func WithMiddleware(mw ...middleware.Middleware) Option
func WithEventSink(sink events.EventSink) Option
func WithOnDelta(fn func(string)) Option
func WithEngine(eng engine.Engine) Option
func WithStepSettings(s *settings.StepSettings) Option
```

### Real-World Examples

**Example 1: Config from YAML + programmatic tools**

```yaml
# config.yaml
model: claude-sonnet-4-20250514
system_prompt: |
  You are a CozoDB expert. Given a database schema, help users write queries.
max_tokens: 8192
stream: true
```

```go
var cfg runner.Config
yaml.Unmarshal(configBytes, &cfg)

result, err := runner.Run(ctx, cfg, userQuestion,
    runner.WithOnDelta(streamToWebSocket),
)
```

**Example 2: Programmatic config with tools**

```go
cfg := runner.Config{
    Model:             "claude-sonnet-4-20250514",
    SystemPrompt:      extractionSystemPrompt,
    MaxToolIterations: 6,
    ToolTimeout:       2 * time.Minute,
    MaxTokens:         16384,
}

result, err := runner.Run(ctx, cfg, prompt,
    runner.WithTools(
        runner.Tool{Name: "query_history", Description: "Query entity history", Func: historyQuery},
        runner.Tool{Name: "query_transcripts", Description: "Query transcripts", Func: transcriptQuery},
    ),
)
```

**Example 3: Profile-based with environment defaults**

```go
cfg := runner.Config{
    Profile:           "production",
    ProfileRegistries: []string{"./profile-registry.yaml"},
}

r, _ := runner.NewRunner(cfg,
    runner.WithTools(
        runner.Tool{Name: "calc", Description: "Calculator", Func: calc},
        runner.Tool{Name: "sql", Description: "SQL queries", Func: sqlQuery(db)},
    ),
)

for _, question := range questions {
    result, _ := r.Run(ctx, question)
    fmt.Println(result.Text)
}
```

### Trade-offs

| Pro | Con |
|-----|-----|
| Config is inspectable and serializable | Two concepts to learn (Config + Options) |
| YAML/JSON config file support | Split between struct fields and options is arbitrary |
| Standard Go pattern (http.Server-like) | More verbose than Design A for one-offs |
| Config can be shared/templated | Tool registration still via Options (not serializable) |

### When this design breaks down

- When most configuration is programmatic (the struct adds noise)
- When tools need dynamic registration per-request
- When the config/option split feels arbitrary to users

---

## Design D: Convention-over-Configuration with Defaults

### Rationale

The most opinionated design. Automatically detect the provider from environment variables, apply sensible defaults for everything, and require users to specify only what's different from the defaults.

**Philosophy:** The 80% case should require zero configuration. Convention over configuration, like Rails for LLM tools.

### Core API

```go
package runner

// Run infers provider from env, applies defaults, executes prompt.
// ANTHROPIC_API_KEY → Claude, OPENAI_API_KEY → OpenAI, OLLAMA_HOST → Ollama
func Run(ctx context.Context, prompt string, opts ...Option) (*Result, error)

// Tool is a convenience for registering Go functions as LLM tools.
func Tool(name, description string, fn any) Option

// System sets the system prompt.
func System(prompt string) Option

// Model overrides the auto-detected model.
func Model(name string) Option

// Stream calls fn with each text delta as it arrives.
func Stream(fn func(delta string)) Option

// MaxTools sets max tool loop iterations (default: 10).
func MaxTools(n int) Option

// Profile loads settings from a named profile.
func Profile(name string, registries ...string) Option

// Middleware adds inference middleware.
func Middleware(mw ...middleware.Middleware) Option

// Sink adds an event sink.
func Sink(s events.EventSink) Option

// Temperature, MaxTokens, etc.
func Temperature(t float64) Option
func MaxTokens(n int) Option
func Timeout(d time.Duration) Option
```

### Real-World Examples

**Example 1: Absolute minimum — CozoDB hints in 4 lines**

```go
result, err := runner.Run(ctx, userQuestion,
    runner.System(buildCozoSystemPrompt(schema)),
    runner.MaxTokens(8192),
    runner.Stream(ws.SendDelta),
)
```

**Example 2: Tool-augmented data analysis in 8 lines**

```go
result, err := runner.Run(ctx, "Analyze our Q4 revenue trends",
    runner.System("You are a data analyst with access to our product database."),
    runner.Tool("sql_query", "Execute SQL against product DB", sqlQueryFunc(db)),
    runner.Tool("calc", "Evaluate mathematical expressions", calculatorFunc),
    runner.Tool("search", "Search product catalog", searchFunc),
    runner.MaxTools(20),
    runner.Stream(sendDelta),
)
```

**Example 3: Temporal extraction pipeline in 12 lines**

```go
for _, session := range sessions {
    result, err := runner.Run(ctx, buildExtractionPrompt(session),
        runner.Profile("extraction", registryPath),
        runner.System(extractionSystemPrompt),
        runner.Tool("query_history", "Query entity change history",
            entityhistory.QueryFunc(scopedDB, opts)),
        runner.Tool("query_transcripts", "Query session transcripts",
            transcripthistory.QueryFunc(scopedDB, opts)),
        runner.MaxTools(6),
        runner.Timeout(2*time.Minute),
    )
    if err != nil { continue }
    saveArtifacts(session.ID, result.Text)
}
```

**Example 4: A complete CLI tool (what it looks like end-to-end)**

```go
package main

import (
    "context"
    "fmt"
    "os"
    "github.com/go-go-golems/geppetto/pkg/runner"
)

func main() {
    ctx := context.Background()

    result, err := runner.Run(ctx, os.Args[1],
        runner.System("You are a helpful code review assistant."),
        runner.Tool("read_file", "Read a file from the filesystem", readFile),
        runner.Tool("list_files", "List files in a directory", listFiles),
        runner.Tool("search_code", "Search for patterns in code", searchCode),
        runner.MaxTools(15),
    )
    if err != nil {
        fmt.Fprintf(os.Stderr, "error: %v\n", err)
        os.Exit(1)
    }
    fmt.Println(result.Text)
}

func readFile(input struct{ Path string }) (string, error) {
    data, err := os.ReadFile(input.Path)
    return string(data), err
}

func listFiles(input struct{ Dir string; Pattern string }) ([]string, error) {
    entries, _ := filepath.Glob(filepath.Join(input.Dir, input.Pattern))
    return entries, nil
}

func searchCode(input struct{ Pattern string; Dir string }) (string, error) {
    out, err := exec.Command("rg", "--json", input.Pattern, input.Dir).Output()
    return string(out), err
}
```

This is a **complete, working CLI tool with 3 tools in ~35 lines**. Compare to the 70-195 lines currently required.

### Auto-Detection Rules

```
1. Check ANTHROPIC_API_KEY → provider=claude, model=claude-sonnet-4-20250514
2. Check OPENAI_API_KEY    → provider=openai, model=gpt-4o
3. Check OLLAMA_HOST       → provider=ollama, model=llama3
4. If multiple: prefer Claude > OpenAI > Ollama (configurable)
5. If none: return clear error with instructions
```

### Default Values

```
MaxTokens:         4096
Temperature:       0
MaxToolIterations: 10
ToolTimeout:       30s
Stream:            true (if sink provided)
```

### Trade-offs

| Pro | Con |
|-----|-----|
| Absolute minimum ceremony | Magic auto-detection may surprise |
| Complete CLI tools in <40 lines | Less explicit about what's happening |
| Option names are short & readable | No struct = can't serialize config |
| Discoverable via IDE autocomplete | Hard to switch providers explicitly |
| Functions are top-level = easy import | Namespace pollution if package grows |

### When this design breaks down

- When explicit control over provider selection matters
- When configuration needs to be serialized/templated
- When multiple different configurations coexist in one program
- When session reuse across calls is needed

---

## Comparison Matrix

| Aspect | Design A | Design B | Design C | Design D |
|--------|----------|----------|----------|----------|
| **Lines for basic call** | 5-8 | 8-12 | 10-15 | 3-6 |
| **Lines for tools** | 8-12 | 10-15 | 12-18 | 6-10 |
| **Multi-turn** | No | Yes (Chat) | Yes (NewRunner) | No (per-call) |
| **Serializable config** | No | No | Yes (YAML) | No |
| **Session reuse** | No | Yes | Yes | No |
| **Learning curve** | Very low | Medium | Medium | Very low |
| **Go idiom match** | http.Get-like | Builder | http.Server-like | fmt.Println-like |
| **Escape hatches** | Options | Methods | Options | Options |
| **Thread safety** | Stateless | Needs care | Config is value | Stateless |

## Recommendation: Hybrid of A + D with B's Multi-Turn

The sweet spot is a **layered API** that combines:

1. **Top-level functions from Design D** for the zero-config experience:
   ```go
   runner.Run(ctx, prompt, runner.Tool("x", "desc", fn))
   ```

2. **Runner struct from Design B** for multi-turn and reuse:
   ```go
   r := runner.New(runner.System("..."), runner.Tool("x", "desc", fn))
   r.Chat(ctx, "first message")
   r.Chat(ctx, "follow up")
   ```

3. **Config struct from Design C** for serialization when needed:
   ```go
   r := runner.NewFromConfig(cfg, runner.Tool("x", "desc", fn))
   ```

4. **Escape hatches at every level:**
   ```go
   // Override engine
   runner.Run(ctx, prompt, runner.WithEngine(myEngine))

   // Access internals
   r := runner.New(...)
   session := r.Session()  // Drop to geppetto level
   ```

### Package Location

Proposed: `geppetto/pkg/runner` — lives in geppetto itself since it wraps geppetto's own types. Not in pinocchio because pinocchio adds CLI/UI concerns that aren't needed here.

### Implementation Priority

1. **Phase 1:** `Run()` function + `Tool()` + `System()` + `Model()` + `Stream()` + `MaxTokens()` — covers 80% of use cases
2. **Phase 2:** `New()` → `Runner` with `Chat()` for multi-turn
3. **Phase 3:** `NewFromConfig()` with YAML support
4. **Phase 4:** Profile integration, advanced options

---

## Appendix: Middleware Integration Design

The runner should support middlewares both through the option API and through the underlying geppetto chain:

```go
// Option-level middleware
result, _ := runner.Run(ctx, prompt,
    runner.Middleware(
        middleware.NewSystemPromptMiddleware("extra instructions"),
        myCustomMiddleware,
    ),
)

// But also: auto-applied middlewares
// The runner automatically applies:
// 1. SystemPromptMiddleware (from System() option)
// 2. ToolResultReorderMiddleware (when tools are present)
// These are invisible to the user but configurable via options.
```

## Appendix: Error Handling Design

```go
result, err := runner.Run(ctx, prompt)
if err != nil {
    // Typed errors for common failure modes
    var apiErr *runner.APIError       // Provider returned an error
    var toolErr *runner.ToolError     // A tool failed during execution
    var configErr *runner.ConfigError // Missing API key, invalid model, etc.

    switch {
    case errors.As(err, &apiErr):
        log.Printf("API error (status %d): %s", apiErr.StatusCode, apiErr.Message)
    case errors.As(err, &toolErr):
        log.Printf("Tool %q failed: %v", toolErr.ToolName, toolErr.Cause)
    case errors.As(err, &configErr):
        log.Printf("Configuration error: %s", configErr.Message)
    }
}
```

## Appendix: Structured Output Design

For tools that need structured output (like CozoDB's YAML extraction), the runner could support typed results:

```go
type HintResponse struct {
    Text    string   `json:"text"`
    Code    string   `json:"code"`
    Chips   []string `json:"chips"`
}

// Option A: JSON mode with schema
result, err := runner.Run(ctx, prompt,
    runner.JSONOutput[HintResponse](),
)
var hint HintResponse
json.Unmarshal([]byte(result.Text), &hint)

// Option B: Structured extraction (geppetto's existing approach)
result, err := runner.Run(ctx, prompt,
    runner.WithExtractor("cozo", "hint", "v1", parsehelpers.NewDebouncedYAML[HintResponse]()),
)
```

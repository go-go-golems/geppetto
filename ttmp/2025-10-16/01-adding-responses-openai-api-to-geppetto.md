Below is a focused, code‑aware plan to add **OpenAI “thinking” models**—notably **o3** and **GPT‑5**—to your `geppetto` inference layer. I read through the OpenAI integration in your zip and cross‑checked current OpenAI API guidance (Responses API, model pages, and the official Go SDK). Citations are included where it matters.

---

## What you already have (from the zip)

**Package shape (Go):**

* `pkg/steps/ai/openai/engine_openai.go`

  * Defines `OpenAIEngine` and `RunInference(...)` (implements `engine.Engine`).
  * Uses **sashabaranov/go-openai** (`go_openai`) and **Chat Completions** only (builds a `go_openai.ChatCompletionRequest`, streams via `CreateChatCompletionStream`).
  * Converts internal tool definitions to `go_openai.Tool` (function tools).
* `pkg/steps/ai/openai/helpers.go`

  * `MakeCompletionRequestFromTurn(...)` builds a **ChatCompletionRequest** from a `turns.Turn`.
  * `MakeClient(...)` wires `BaseURL` and key onto `go_openai` client.
* `pkg/steps/ai/settings/openai/{chat.yaml, settings.go}`

  * Flags/fields are all named for **chat completions** (e.g. `openai-n`, presence/frequency penalty, `openai-base-url`, `openai-api-key`).
* `pkg/steps/ai/settings/settings-chat.go`

  * Holds `ChatSettings` with `Engine`, `ApiType`, `MaxResponseTokens`, `Temperature`, `TopP`, etc. (API‑agnostic).
* `pkg/steps/ai/types/types.go`

  * `ApiTypeOpenAI`, `ApiTypeClaude`, `ApiTypeGemini`, … (no notion of “Responses vs Chat”).

**Bottom line:** the OpenAI path is **Chat Completions only** right now; it cannot call the **Responses API** that o‑series and GPT‑5 reasoning flows expect. (The official docs present the **Responses API** as the home for reasoning, built‑in tools, and streaming events. ([OpenAI Platform][1]))

---

## What o3 / GPT‑5 need (in brief)

* **Use the Responses API**—not (just) Chat Completions—when targeting **o3** (and, for full reasoning features, **GPT‑5**). You send **`input`** items instead of `messages`, and you can set **`reasoning.effort`** and **`max_output_tokens`**. (Official docs pages: Responses API, model pages, reasoning guide.) ([OpenAI Platform][1])
* Typical request shape (conceptually):
  `model: "o3" | "gpt-5"`, `input: [...]`, `reasoning: { effort: "low" | "medium" | "high" }`, `max_output_tokens`, optional `tools` with function schemas. (API reference + reasoning guide.) ([OpenAI Platform][1])
* In Go, the **official OpenAI SDK** exposes **`client.Responses.New(...)`** and **`Responses.NewStreaming(...)`** with typed params (package `responses`). (See the official **openai‑go** repository and its API docs; also visible via Azure’s extension docs which show the same OpenAI types.) ([GitHub][2])

> Note: Some GPT‑5 “chat” variants (e.g., *gpt‑5‑chat‑latest*) are accessible via Chat Completions for compatibility, but **the reasoning controls live in Responses**, and o‑series models (e.g., **o3‑pro**) are documented for Responses usage. Keep Chat Completions for older models; route **o3 / GPT‑5 reasoning** to **Responses**. ([OpenAI Platform][3])

---

## Minimal‑risk design: dual stack (keep Chat Completions, add Responses)

### 1) Add the official SDK alongside `sashabaranov/go-openai`

* Keep `sashabaranov/go-openai` for existing Chat Completions paths.
* **Add** the official OpenAI Go SDK for Responses:

```go
// go.mod
require (
    github.com/sashabaranov/go-openai vX.Y.Z // existing
    github.com/openai/openai-go/v3 v3.x.x    // new
)
```

Core import aliases you’ll use:

```go
openai   "github.com/openai/openai-go/v3"
responses "github.com/openai/openai-go/v3/responses"
"github.com/openai/openai-go/v3/option"
"github.com/openai/openai-go/v3/shared"
```

(Official repo for SDK.) ([GitHub][2])

### 2) Route by model → choose API “mode”

Add a tiny router that decides **Responses** vs **Chat Completions**:

```go
// pkg/steps/ai/openai/engine_openai.go (or new file)
func requiresResponses(model string) bool {
    m := strings.ToLower(model)
    // Route o‑series and GPT‑5 to Responses by default
    return strings.HasPrefix(m, "o3") || strings.HasPrefix(m, "o4") || strings.HasPrefix(m, "gpt-5")
}
```

If `requiresResponses(model)` → call `runResponses(...)`; else keep the current Chat Completions path.

(Models + families referenced in OpenAI model docs.) ([OpenAI Platform][4])

### 3) Create a new Responses path (parallel to current path)

Add a new implementation file:

```
pkg/steps/ai/openai/engine_openai_responses.go
```

Skeleton:

```go
func (e *OpenAIEngine) runResponses(ctx context.Context, t *turns.Turn) (*turns.Turn, error) {
    // 1) Build OpenAI client (re-using your base URL + key settings)
    //    Keep support for api.openai.com and Azure if you already forward those in MakeClient-like logic.
    client := openai.NewClient(
        option.WithAPIKey(e.settings.API.ApiKeyFor("openai")), // adapt to your API key shape
        option.WithBaseURL(e.settings.OpenAIBaseURL()),        // keep compatibility with custom gateways, if any
    )

    // 2) Convert Turn -> responses params
    params, err := MakeResponsesParamsFromTurn(e.settings, t)
    if err != nil { return nil, err }

    // 3a) Streaming (preferred when e.config says stream)
    if shouldStream(e) {
        stream := client.Responses.NewStreaming(ctx, params)
        defer stream.Close()

        // Accumulate content + tool calls and publish deltas as events
        for stream.Next() {
            evt := stream.Current()

            switch data := evt.Data.(type) {
            case responses.ResponseOutputTextDeltaEvent:
                // publish incremental text delta to your event sink
            case responses.ResponseToolCallDeltaEvent:
                // accumulate arguments for tool call index, publish deltas
            case responses.ResponseCompletedEvent:
                // finalize content, usage, stop reason
            case responses.ResponseErrorEvent:
                // surface model-side error
            }
        }
        if err := stream.Err(); err != nil { return nil, err }
        // Return the finalized Turn (assistant message + tool calls executed, if any)
        return finalizeTurnFromStream(...)
    }

    // 3b) Non-streaming
    resp, err := client.Responses.New(ctx, params)
    if err != nil { return nil, err }

    // Convert Response -> Turn (assistant content, tool calls, usage)
    return turnFromResponse(resp)
}
```

(Responses API streaming & events are documented in the API reference; the official Go SDK exposes `Responses.New` and `Responses.NewStreaming`.) ([OpenAI Platform][1])

### 4) Build `MakeResponsesParamsFromTurn(...)`

Create a new helper mirroring `MakeCompletionRequestFromTurn(...)`:

```
pkg/steps/ai/openai/helpers_responses.go
```

Key conversions:

* **Messages → Input**
  Responses uses **`input`** (array of role‑tagged items with content parts). Convert your `turns.Turn.Blocks` into **`[]responses.InputItemParam`**, each with:

  * `role`: `"system" | "user" | "assistant"`.
  * `content`: array of parts:

    * text → `responses.InputTextParam{ Text: "..." }`
    * image (URL or base64) → `responses.InputImageParam{ ImageURL: ..., Detail: ... }`
    * (audio, if present) → `responses.InputAudioParam{ ... }`
* **Model**
  Set from `settings.Chat.Engine` (e.g., `"o3"`, `"gpt-5"`), use `shared.Model(settings.Chat.Engine)` with the official SDK type.
* **Reasoning**
  Map a new setting (see §5) to `Reasoning: &shared.ReasoningParam{ Effort: shared.ReasoningEffortMedium /* default */ }`.
* **Max output tokens**
  Map `settings.Chat.MaxResponseTokens` → `MaxOutputTokens: openai.Int(n)`.
* **Temperature / TopP / Stop**
  Forward when set (fields exist on Responses too).
* **Function tools**
  Convert your tool definitions to `[]responses.ToolUnionParam` with `OfFunction: &responses.FunctionToolParam{ Name, Description, Parameters(JSONSchema) }`.

The struct names above match the official SDK’s `responses` package (examples are shown in the official `openai-go` repo and in Azure’s extension docs that demonstrate the same OpenAI types for `client.Responses.New` / `NewStreaming`). ([GitHub][2])

### 5) Add two small settings for reasoning models

In `pkg/steps/ai/settings/openai/settings.go` and its YAML:

* **`openai-reasoning-effort`**: `"low" | "medium" | "high"` → maps to `shared.ReasoningEffortLow|Medium|High` on `Reasoning.Effort`.
* **(optional) `openai-parallel-tool-calls`**: bool to signal parallel tool execution preference; Responses supports setting tool‑choice/parallelization (you already have a notion of `MaxParallelTools` in config; this just threads through to the request when useful).

Reasoning controls are called out in model + reasoning docs. ([OpenAI Platform][5])

### 6) Tool conversions (second adapter, minimal)

Right now `PrepareToolsForRequest(...)` returns **`[]go_openai.Tool`** (Chat Completions shape). Add a sibling function for Responses:

```go
func (e *OpenAIEngine) PrepareToolsForResponses(toolDefs []engine.ToolDefinition, cfg engine.ToolConfig) ([]responses.ToolUnionParam, error) {
    if !cfg.Enabled { return nil, nil }
    out := make([]responses.ToolUnionParam, 0, len(toolDefs))
    for _, td := range toolDefs {
        out = append(out, responses.ToolUnionParam{
            OfFunction: &responses.FunctionToolParam{
                Name:        td.Name,
                Description: openai.String(td.Description),
                Parameters:  td.Parameters, // keep your JSON Schema as-is
            },
        })
    }
    return out, nil
}
```

(Responses uses **function tools** with JSON Schema, same conceptual model, just a different type.) ([OpenAI Platform][1])

### 7) Streaming event mapping (Responses → your event bus)

Responses streaming emits typed events (text deltas, tool‑call deltas, completed). You already publish deltas for Chat Completions. Mirror that:

* `ResponseOutputTextDeltaEvent` → emit your existing *assistant text delta* event.
* `ResponseToolCallDeltaEvent` → emit per‑tool‑call index+args deltas; when a call is closed, invoke your tool runner (you already do this pattern for Chat Completions).
* `ResponseCompletedEvent` → finalize the assistant message and copy **usage** (prompt/completion/reasoning tokens) back onto the `Turn`.

(Streaming + events covered in the Responses reference; official SDK has streaming primitives; there are also discussions/examples in the repo issues.) ([OpenAI Platform][1])

### 8) Back‑compat: keep the current codepath for older models

* Continue to build a `go_openai.ChatCompletionRequest` and stream with `CreateChatCompletionStream` when `requiresResponses(model)` is **false**.
* This preserves your Claude/Gemini/Ollama flows unchanged.

(Your existing code already does this; we’re only adding a new branch.)

---

## Concrete code sketch (drop‑in style)

**A. Router in `RunInference(...)`:**

```go
engine := ""
if e.settings.Chat.Engine != nil { engine = *e.settings.Chat.Engine }

if requiresResponses(engine) {
    return e.runResponses(ctx, t)   // new path
}
return e.runChatCompletions(ctx, t) // existing path, factored from your current RunInference
```

**B. Build the Responses request (helper):**

```go
func MakeResponsesParamsFromTurn(
    s *settings.StepSettings,
    t *turns.Turn,
) (responses.ResponseNewParams, error) {

    // 1) Input items
    in, err := buildInputFromTurn(t) // []responses.InputItemParam (system/user/assistant with text/images)
    if err != nil { return responses.ResponseNewParams{}, err }

    // 2) Model + generation params
    params := responses.ResponseNewParams{
        Model: shared.Model(*s.Chat.Engine),
        Input: responses.ResponseNewParamsInputUnion{ OfArray: in },
        MaxOutputTokens: openai.Int(pickInt(s.Chat.MaxResponseTokens, 1024)),
    }
    if s.Chat.Temperature != nil { params.Temperature = openai.Float32(float32(*s.Chat.Temperature)) }
    if s.Chat.TopP        != nil { params.TopP        = openai.Float32(float32(*s.Chat.TopP)) }
    if len(s.Chat.Stop) > 0      { params.StopSequences = s.Chat.Stop } // name as per SDK

    // 3) Reasoning
    if eff := s.OpenAI.ReasoningEffort; eff != nil {
        params.Reasoning = &shared.ReasoningParam{ Effort: mapEffort(*eff) }
    }

    // 4) Tools (Responses shape)
    if toolDefs := e.config.Tools; toolDefs.Enabled {
        tools, err := e.PrepareToolsForResponses(toolDefs.Definitions, toolDefs)
        if err != nil { return responses.ResponseNewParams{}, err }
        params.Tools = tools
    }

    return params, nil
}
```

*Field names align with the official SDK’s `responses` package; you will confirm the exact Go identifiers when you import `openai-go/v3`.* ([GitHub][2])

---

## Mapping table: Chat → Responses (what changes in code)

| Concept    | Chat Completions (`go_openai`)           | Responses API (`openai-go/v3`)                                                                  |
| ---------- | ---------------------------------------- | ----------------------------------------------------------------------------------------------- |
| Endpoint   | `/v1/chat/completions`                   | `/v1/responses`                                                                                 |
| Messages   | `[]Message` (role, content string/parts) | **`input`**: `[]InputItem` (role, `content: []{text/image/...}`)                                |
| Tokens cap | `MaxTokens`                              | **`MaxOutputTokens`**                                                                           |
| Reasoning  | *n/a*                                    | **`Reasoning{Effort: low/medium/high}`**                                                        |
| Tools      | `[]Tool{Function{Name, Parameters}}`     | **`[]ToolUnionParam{OfFunction{...}}`**                                                         |
| Streaming  | chunk `delta.content`                    | **events**: `response.output_text.delta`, `response.tool_call.delta`, `response.completed`, ... |
| SDK        | `github.com/sashabaranov/go-openai`      | **`github.com/openai/openai-go/v3`** (`responses` package)                                      |

(Responses API reference & model pages.) ([OpenAI Platform][1])

---

## Tests / quick validation

1. **Non‑streaming**: `model: "o3"`, `reasoning.effort: "medium"`, ask for a short answer; assert an assistant message arrives and `usage` contains reasoning token fields. (Responses API.) ([OpenAI Platform][1])
2. **Streaming**: same prompt with streaming; assert you receive `output_text.delta` chunks and a terminal completion event. (Responses streaming docs.) ([OpenAI Platform][6])
3. **Tool‑call**: register a single function tool and confirm `tool_call.delta` accumulation → your tool adapter is invoked once the call is closed; the final assistant message includes the post‑tool answer. (Function tools via Responses.) ([OpenAI Platform][1])
4. **Back‑compat**: run an existing `gpt-4o` flow and confirm it still uses Chat Completions.

---

## Why the official Go SDK here?

* It **natively supports Responses** (typed params, streaming, events), and it’s the **reference** for new OpenAI features. (Official SDK repo.) ([GitHub][2])
* Your current dependency (**sashabaranov/go-openai**) is great for Chat Completions; keeping it avoids churn to the older paths while you add Responses for o3/GPT‑5.

---

## Pointers to the relevant OpenAI docs / SDK surfaces

* **Responses API (reference + streaming events)** – request/response shape, event types, tokens fields. ([OpenAI Platform][1])
* **o3 / o3‑pro model pages** – “trained to think” and (Response‑only availability for full reasoning). ([OpenAI Platform][4])
* **GPT‑5 model page(s)** – recommended usage & tooling with the Responses API (chat variants exist). ([OpenAI Platform][7])
* **Official Go SDK (`openai-go`)** – package housing `client.Responses.New`, `NewStreaming`, streaming accumulators, etc. ([GitHub][2])

---

## Implementation checklist (copy/paste into an issue)

* [ ] Add dependency: `github.com/openai/openai-go/v3`. ([GitHub][2])
* [ ] Introduce `requiresResponses(model)`; call `runResponses(...)` when true.
* [ ] Implement `runResponses(...)` (non‑stream + streaming) and event translation.
* [ ] Add `helpers_responses.go` with `MakeResponsesParamsFromTurn(...)` and `buildInputFromTurn(...)`.
* [ ] Implement `PrepareToolsForResponses(...)` to convert internal tool defs to `responses.FunctionToolParam`.
* [ ] Extend OpenAI settings with `openai-reasoning-effort` (low/medium/high) → `Reasoning.Effort`. ([OpenAI Platform][5])
* [ ] Keep Chat Completions path untouched for non‑reasoning models.
* [ ] Add a small test matrix for o3 / gpt‑5 (streaming + non‑streaming + tool call).

If you want, I can sketch the exact `helpers_responses.go` signatures against your `turns.Turn` block types to make the mapping fully plug‑and‑play.

[1]: https://platform.openai.com/docs/api-reference/responses "OpenAI Platform"
[2]: https://github.com/openai/openai-go "GitHub - openai/openai-go: The official Go library for the OpenAI API"
[3]: https://platform.openai.com/docs/models/gpt-5-chat-latest?ref=legalopsai.org&utm_source=chatgpt.com "Model - OpenAI API"
[4]: https://platform.openai.com/docs/models/o3-pro?utm_source=chatgpt.com "Model - OpenAI API"
[5]: https://platform.openai.com/docs/guides/reasoning "OpenAI Platform"
[6]: https://platform.openai.com/docs/api-reference/responses-streaming?ref=blog.streamlit.io&utm_source=chatgpt.com "API Reference"
[7]: https://platform.openai.com/docs/models/gpt-5 "OpenAI Platform"


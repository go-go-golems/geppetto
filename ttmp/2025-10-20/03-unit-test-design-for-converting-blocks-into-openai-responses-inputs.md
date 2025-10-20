## Unit test design: converting `turns.Blocks` into OpenAI Responses input (helpers.go)

### Scope
- Validate `buildInputItemsFromTurn` and related request-shaping in `buildResponsesRequest` for OpenAI Responses API.
- Ensure correctness across plain chat, reasoning, tool calls, and mixed histories.
- Confirm ordering, encoding, and guardrails (no invalid reasoning without its required follower, no duplicates).

### Test strategy
- Table-driven tests in `geppetto/pkg/steps/ai/openai_responses/helpers_test.go` (package `openai_responses`).
- Use small builders for `turns.Block` to shape inputs. Assert on the returned `[]responsesInput` and selected fields from `responsesRequest`.
- Prefer structural assertions (type/role/content/type fields, IDs, arguments JSON) over full string comparisons. Validate order of items.

### Test scaffold (outline)
```go
func TestBuildInputItemsFromTurn(t *testing.T) {
  type wantItem struct {
    Type    string   // "reasoning" | "message" | "function_call" | "function_call_output" or empty for role-based
    Role    string   // when role-based or for message items
    PartTypes []string // e.g., ["input_text"], ["output_text"], empty for non-message items
    HasEncrypted bool
    IDPresent bool // for reasoning and message items that may carry provider IDs
    CallID    string // expected call_id for function_call/_output
    Name      string // function name
  }
  tests := []struct{
    name     string
    turn     turns.Turn
    want     []wantItem
  }{ /* cases below */ }
}

func TestBuildResponsesRequest_ModelAndReasoning(t *testing.T) { /* ... */ }
```

### Cases for `buildInputItemsFromTurn`

1) Plain chat: system + user only
- Input:
  - `System("You are a LLM.")`
  - `User("Hello")`
- Expect:
  - Two role-based items:
    - `{ role: "system", content: [{type:"input_text"}] }`
    - `{ role: "user", content: [{type:"input_text"}] }`
  - No `type:"reasoning"`, no function items.
- Asserts:
  - len==2; order preserved; part types are `input_text`.

2) Assistant text without reasoning
- Input:
  - `System("You are a LLM.")`, `User("Hi")`, `LLMText("Howdy")`
- Expect:
  - Role-based assistant message: `{ role:"assistant", content:[{type:"output_text"}] }`
  - No `type:"message"` item (item-mode not used when no reasoning/tools needed)
- Asserts:
  - Three items; last is role:"assistant" with one `output_text` part.

3) Reasoning + immediate assistant output (message follower)
- Input:
  - `...`, `Reasoning{ID: rs_1, encrypted_content: present}`, `LLMText("Answer")`
- Expect:
  - Item `type:"reasoning"` with `id==rs_1`, `encrypted_content` set, and `summary: []` present
  - Immediately followed by item `type:"message"`, `role:"assistant"`, `content:[{type:"output_text", text:"Answer"}]`
  - The same `LLMText` block must NOT appear later as a role-based message (no duplication)
- Asserts:
  - First is reasoning, second is message; no later role-based assistant with the same text.

4) Reasoning + contiguous tool calls and results
- Input:
  - `Reasoning{ID: rs_2, enc:?}`, `ToolCall(id: fc_1, name: "do", args: {x:1})`, `ToolUse(id: fc_1, result: {ok:true})`
- Expect:
  - `type:"reasoning"`
  - `type:"function_call"` with `call_id==fc_1`, `name:"do"`, `arguments` as compact JSON
  - `type:"function_call_output"` with `call_id==fc_1`, `output` as compact JSON
- Asserts:
  - Items are consecutive in that order; arguments/result serialized deterministically.

5) Reasoning without valid immediate follower (guard)
- Input:
  - `...`, `Reasoning{ID: rs_3}`, followed by `User("New")` (or end of turn)
- Expect:
  - Omit the `reasoning` item entirely to avoid 400s
  - Remaining blocks included normally (role-based messages)
- Asserts:
  - No item with `type:"reasoning"` in output.

6) Multiple reasoning blocks; only latest considered
- Input:
  - `Reasoning{old}`, `User("Q")`, `Reasoning{latest}`, `LLMText("A")`
- Expect:
  - Pre-context excludes older reasoning
  - Only the latest reasoning is emitted with its immediate message follower in item-mode
- Asserts:
  - Exactly one `type:"reasoning"` (for latest), followed by `type:"message"`.

7) Pre-context includes preceding tool calls and messages (but skips older reasoning)
- Input:
  - `System`, `User`, `ToolCall(fc_0)`, `ToolUse(fc_0)`, `Reasoning{latest}`, `LLMText("A")`
- Expect:
  - Pre-context contains `System`, `User` as role-based; `ToolCall(fc_0)` and `ToolUse(fc_0)` encoded as items before reasoning? (No — only blocks strictly before latest reasoning appear prior; tool items are added in place based on their position.)
  - Then `reasoning` + `message`
- Asserts:
  - Order: system(user), function_call(fc_0), function_call_output(fc_0), reasoning, message.

8) Assistant message consisting of whitespace only (skip)
- Input:
  - `Reasoning{}`, `LLMText("   \n  ")`
- Expect:
  - No message follower can be built; omit reasoning (guard)
- Asserts:
  - No `type:"reasoning"`; no `type:"message"` follower; nothing added for whitespace.

9) Function call with args pre-serialized string
- Input:
  - `ToolCall(id:"fc_2", name:"calc", args:"{\"a\":2}")`
- Expect:
  - `type:"function_call"` with `arguments` kept as-is (string)
- Asserts:
  - Arguments equals the provided string exactly.

10) Function call with args as structured object
- Input:
  - `ToolCall(id:"fc_3", name:"calc", args: map[string]any{"a":2})`
- Expect:
  - `type:"function_call"` with `arguments` as JSON string
- Asserts:
  - JSON parses to `{a:2}`; string not empty.

11) Function call output with result string vs object
- Input:
  - `ToolUse(id:"fc_4", result:"ok")` and `ToolUse(id:"fc_5", result: map[string]any{"ok":true})`
- Expect:
  - Two `type:"function_call_output"` items; `output` string preserved or JSON-encoded respectively
- Asserts:
  - Proper call_id mapping; outputs present and correct.

12) ItemID propagation on function_call
- Input:
  - `ToolCall(id:"fc_6", name:"n", args:{}, item_id:"it_123")`
- Expect:
  - `type:"function_call"` with `id=="it_123"`
- Asserts:
  - Item carries the expected `ID` field.

13) Mixed: system, user, assistant, tool call/use, reasoning, assistant
- Input:
  - `System`, `User`, `LLMText("Prev")`, `ToolCall(fc_x)`, `ToolUse(fc_x)`, `Reasoning{rs_x}`, `LLMText("Final")`
- Expect:
  - Pre-context includes system,user,assistant(role-based), function_call/_output
  - Then `reasoning` + item-message("Final")
- Asserts:
  - No duplication of the final assistant; previous assistant remains role-based.

14) Unknown/Other kind with text falls back to role:"assistant"
- Input:
  - `BlockKindOther` with `text:"misc"`
- Expect:
  - Role-based item with role:"assistant", part type `output_text`
- Asserts:
  - Single role-based assistant item.

15) Additional reasoning after the grouped segment should be skipped
- Input:
  - `Reasoning{latest}`, `LLMText("A")`, `Reasoning{extra}`
- Expect:
  - Only first reasoning+message emitted; later reasoning skipped in remainder
- Asserts:
  - Exactly one reasoning item.

### Cases for `buildResponsesRequest` (selected)

16) Include encrypted reasoning content always
- Input: any turn
- Expect: `Include` contains `"reasoning.encrypted_content"`

17) Temperature/TopP omitted for o3*/o4* models
- Settings: `Chat.Engine = "o4-mini"`, `Temperature=1`, `TopP=1`
- Expect: `Temperature==nil`, `TopP==nil` in request

18) Temperature/TopP preserved for non-reasoning models
- Settings: `Chat.Engine = "gpt-4o"`, `Temperature=0.7`, `TopP=0.9`
- Expect: those fields set

19) Stop sequences copied and Stream flag propagated
- Settings: `Chat.Stop=["END"]`, `Chat.Stream=true`
- Expect: same in request

20) Reasoning metadata mapping
- Settings: `OpenAI.ReasoningEffort = "high"`, `OpenAI.ReasoningSummary = "detailed"`
- Expect: `Reasoning = {effort:"high", summary:"detailed"}`

### Assertion details per item shape
- Role-based messages:
  - `Type==""` (zero value), `Role in {system,user,assistant}`, `Content` non-empty
  - `Content[i].Type == input_text|output_text` depending on role
- Item-based message:
  - `Type=="message"`, `Role=="assistant"`, `Content` with `output_text`
- Reasoning:
  - `Type=="reasoning"`, `ID` set (when present), `Summary` non-nil and length 0, `EncryptedContent` optional
- Function call:
  - `Type=="function_call"`, `CallID` non-empty, `Name` non-empty, `Arguments` JSON string
- Function call output:
  - `Type=="function_call_output"`, `CallID` non-empty, `Output` string (possibly JSON)

### Negative/guard validations
- If a `reasoning` item is added, it must be immediately followed by an item-based follower (message or function_call). Otherwise it must not be included.
- No duplicate assistant content: when an item-based message is emitted as follower, the same `LLMText` block must not reappear as a role-based message.
- Multiple reasoning blocks: only the latest one is considered; all others are skipped.

### Optional fuzz/robustness checks
- Args/result objects with nested maps/slices; ensure marshal succeeds and output is valid JSON string.
- Very long texts: ensure shape remains correct (we do not trim in helpers; only logs trim elsewhere).

### Example table-driven skeleton
```go
func TestBuildInputItemsFromTurn_ReasoningWithAssistantFollower(t *testing.T) {
  turn := turns.Turn{Blocks: []turns.Block{
    turns.NewSystemTextBlock("You are a LLM."),
    {Kind: turns.BlockKindReasoning, ID: "rs_1", Payload: map[string]any{turns.PayloadKeyEncryptedContent: "abc"}},
    turns.NewAssistantTextBlock("Answer"),
  }}
  got := buildInputItemsFromTurn(&turn)
  if len(got) != 3 { t.Fatalf("want 3 items, got %d", len(got)) }
  if got[1].Type != "reasoning" { t.Fatalf("want reasoning at #1") }
  if got[2].Type != "message" || got[2].Role != "assistant" { t.Fatalf("want item message follower") }
}
```

### Files and placement
- Tests: `geppetto/pkg/steps/ai/openai_responses/helpers_test.go`
- Subject under test: `geppetto/pkg/steps/ai/openai_responses/helpers.go`



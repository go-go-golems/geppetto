---
Title: Anthropic extended thinking
SourceURL: https://docs.anthropic.com/en/docs/build-with-claude/extended-thinking
SourceTool: defuddle
FetchedAt: 2026-06-05T08:08:39-04:00
Ticket: 2026-06-05-geppetto-provider-gap-audit
Topics:
  - geppetto
  - providers
  - api-docs
DocType: source
Summary: Official provider documentation captured for the Geppetto provider gap audit.
---

This feature is eligible for [Zero Data Retention 
(ZDR)](https://docs.anthropic.com/docs/en/build-with-claude/api-and-data-retention). When your 
organization has a ZDR arrangement, data sent through this feature is not stored after the API 
response is returned.

Extended thinking gives Claude enhanced reasoning capabilities for complex tasks, while providing 
varying levels of transparency into its step-by-step thought process before it delivers its final 
answer.

For Claude Opus 4.8 and Claude Opus 4.7, set `thinking: {type: "adaptive"}` to enable [adaptive 
thinking](https://docs.anthropic.com/docs/en/build-with-claude/adaptive-thinking) and use the 
[effort parameter](https://docs.anthropic.com/docs/en/build-with-claude/effort) to control thinking 
depth. On both models, manual extended thinking (`thinking: {type: "enabled", budget_tokens: N}`) 
is not supported and returns a 400 error. With adaptive thinking, the model decides when and how 
much to think based on each request, so it triggers thinking only as needed. For Claude Opus 4.6 
and Claude Sonnet 4.6, adaptive thinking is also recommended; the manual configuration is still 
functional on these models but is deprecated and will be removed in a future model release.

## Supported models

Manual extended thinking (`thinking: {type: "enabled", budget_tokens: N}`) is supported on all 
current Claude models **except Claude Opus 4.8 and Claude Opus 4.7**, where it is no longer 
accepted and returns a 400 error. A few models have mode-specific behavior:

- **Claude Opus 4.8 (claude-opus-4-8):** manual extended thinking is not supported and returns a 
400 error. Use [adaptive 
thinking](https://docs.anthropic.com/docs/en/build-with-claude/adaptive-thinking) (`thinking: 
{type: "adaptive"}`) with the [effort 
parameter](https://docs.anthropic.com/docs/en/build-with-claude/effort) instead. The model 
determines whether and how much to use extended thinking based on each request.
- **Claude Opus 4.7 (claude-opus-4-7):** manual extended thinking is no longer supported. Use 
[adaptive thinking](https://docs.anthropic.com/docs/en/build-with-claude/adaptive-thinking) 
(`thinking: {type: "adaptive"}`) with the [effort 
parameter](https://docs.anthropic.com/docs/en/build-with-claude/effort) instead.
- **[Claude Mythos Preview](https://anthropic.com/glasswing):** [adaptive 
thinking](https://docs.anthropic.com/docs/en/build-with-claude/adaptive-thinking) is the default; 
`thinking: {type: "enabled", budget_tokens: N}` is also accepted. `thinking: {type: "disabled"}` is 
not supported, and `display` defaults to `"omitted"` rather than returning thinking content. Pass 
`display: "summarized"` to receive summaries.
- **Claude Opus 4.6 (claude-opus-4-6):** [adaptive 
thinking](https://docs.anthropic.com/docs/en/build-with-claude/adaptive-thinking) recommended; 
manual mode (`type: "enabled"`) is deprecated but still functional.
- **Claude Sonnet 4.6 (claude-sonnet-4-6):** [adaptive 
thinking](https://docs.anthropic.com/docs/en/build-with-claude/adaptive-thinking) recommended; 
manual mode (`type: "enabled"`) with [interleaved mode](#interleaved-thinking) is deprecated but 
still functional.

Thinking behavior differs across Claude model versions. See [Differences in thinking across model 
versions](#differences-in-thinking-across-model-versions) for details.

## How extended thinking works

When extended thinking is turned on, Claude creates `thinking` content blocks where it outputs its 
internal reasoning. Claude incorporates insights from this reasoning before crafting a final 
response.

The API response includes `thinking` content blocks, followed by `text` content blocks.

Here's an example of the default response format:

```
{
  "content": [
    {
      "type": "thinking",
      "thinking": "Let me analyze this step by step...",
      "signature": 
"WaUjzkypQ2mUEVM36O2TxuC06KN8xyfbJwyem2dw3URve/op91XWHOEBLLqIOMfFG/UvLEczmEsUjavL...."
    },
    {
      "type": "text",
      "text": "Based on my analysis..."
    }
  ]
}
```

For more information about the response format of extended thinking, see the [Messages API 
Reference](https://docs.anthropic.com/docs/en/api/messages/create).

## How to use extended thinking

Here is an example of using extended thinking in the Messages API:

To turn on extended thinking, add a `thinking` object, with the `type` parameter set to `enabled` 
and the `budget_tokens` to a specified token budget for extended thinking. For Claude Opus 4.6 and 
Claude Sonnet 4.6, use `type: "adaptive"` instead. See [Adaptive 
thinking](https://docs.anthropic.com/docs/en/build-with-claude/adaptive-thinking) for details. 
While `type: "enabled"` with `budget_tokens` is still functional on these models, it is deprecated 
and will be removed in a future release.

The `budget_tokens` parameter determines the maximum number of tokens Claude is allowed to use for 
its internal reasoning process. This limit applies to full thinking tokens, not to [the summarized 
output](#summarized-thinking). Larger budgets can improve response quality by enabling more 
thorough analysis for complex problems, although Claude may not use the entire budget allocated, 
especially at ranges above 32k.

`budget_tokens` is 
[deprecated](https://docs.anthropic.com/docs/en/build-with-claude/overview#feature-availability) on 
Claude Opus 4.6 and Claude Sonnet 4.6 and will be removed in a future model release. Use [adaptive 
thinking](https://docs.anthropic.com/docs/en/build-with-claude/adaptive-thinking) with the [effort 
parameter](https://docs.anthropic.com/docs/en/build-with-claude/effort) to control thinking depth 
instead.

[Claude Mythos Preview](https://anthropic.com/glasswing), Claude Opus 4.8, Claude Opus 4.7, and 
Claude Opus 4.6 support up to 128k output tokens. Claude Sonnet 4.6 and Claude Haiku 4.5 support up 
to 64k. See the [models overview](https://docs.anthropic.com/docs/en/about-claude/models/overview) 
for limits on legacy models. On the [Message Batches 
API](https://docs.anthropic.com/docs/en/build-with-claude/batch-processing#extended-output-beta), 
the `output-300k-2026-03-24` [beta header](https://docs.anthropic.com/docs/en/api/beta-headers) 
raises the output limit to 300k for Claude Opus 4.8, Opus 4.7, Opus 4.6, and Sonnet 4.6.

`budget_tokens` must be set to a value less than `max_tokens`. However, when using [interleaved 
thinking with tools](#interleaved-thinking), you can exceed this limit as the token limit becomes 
your entire context window. Because `budget_tokens` must be less than `max_tokens`, extended 
thinking cannot be combined with `max_tokens: 0` ([cache 
pre-warming](https://docs.anthropic.com/docs/en/build-with-claude/prompt-caching#pre-warming-the-cac
he)).

### Summarized thinking

With extended thinking enabled, the Messages API for Claude 4 models returns a summary of Claude's 
full thinking process. Summarized thinking provides the full intelligence benefits of extended 
thinking, while preventing misuse. This is the default behavior on Claude 4 models when the 
`display` field on the thinking configuration is unset or set to `"summarized"`. On Claude Opus 
4.8, Claude Opus 4.7, and [Claude Mythos Preview](https://anthropic.com/glasswing), `display` 
defaults to `"omitted"` instead, so you must set `display: "summarized"` explicitly to receive 
summarized thinking.

Here are some important considerations for summarized thinking:

- You're charged for the full thinking tokens generated by the original request, not the summary 
tokens.
- The billed output token count will **not match** the count of tokens you see in the response.
- On Claude 4 models, the first few lines of thinking output are more verbose, providing detailed 
reasoning that's particularly helpful for prompt engineering purposes. [Claude Mythos 
Preview](https://anthropic.com/glasswing) summarizes from the first token, so its thinking blocks 
do not show this verbose preamble.
- As Anthropic seeks to improve the extended thinking feature, summarization behavior is subject to 
change.
- Summarization preserves the key ideas of Claude's thinking process with minimal added latency, 
enabling a streamable user experience.
- Summarization is processed by a different model than the one you target in your requests. The 
thinking model does not see the summarized output.

In rare cases where you need access to full thinking output for Claude 4 models, [contact Anthropic 
sales](https://docs.anthropic.com/cdn-cgi/l/email-protection#6615070a0315260708120e1409160f054805090
b).

### Controlling thinking display

The `display` field on the thinking configuration controls how thinking content is returned in API 
responses. It accepts two values:

- `"summarized"`: Thinking blocks contain summarized thinking text. See [Summarized 
thinking](#summarized-thinking) for details. This is the default on Claude Opus 4.6, Claude Sonnet 
4.6, and earlier Claude 4 models.
- `"omitted"`: Thinking blocks are returned with an empty `thinking` field. The `signature` field 
still carries the encrypted full thinking for multi-turn continuity (see [Thinking 
encryption](#thinking-encryption)). This is the default on Claude Opus 4.8, Claude Opus 4.7, and 
[Claude Mythos Preview](https://anthropic.com/glasswing).

Setting `display: "omitted"` is useful when your application doesn't surface thinking content to 
users. The primary benefit is **faster time-to-first-text-token when streaming:** The server skips 
streaming thinking tokens entirely and delivers only the signature, so the final text response 
begins streaming sooner.

Here are some important considerations for omitted thinking:

- You're still charged for the full thinking tokens. Omitting reduces latency, not cost.
- If you pass thinking blocks back in multi-turn conversations, pass them unchanged. The server 
decrypts the `signature` to reconstruct the original thinking for prompt construction (see 
[Preserving thinking 
blocks](https://docs.anthropic.com/docs/en/build-with-claude/extended-thinking#preserving-thinking-b
locks)). Any text you place in the `thinking` field of a round-tripped omitted block is ignored.
- `display` is invalid with `thinking.type: "disabled"` (there is nothing to display).
- When using `thinking.type: "adaptive"` and the model skips thinking for a simple request, no 
thinking block is produced regardless of `display`.

The `signature` field is identical whether `display` is `"summarized"` or `"omitted"`. Switching 
`display` values between turns in a conversation is supported.

On [Claude Mythos Preview](https://anthropic.com/glasswing), `display` defaults to `"omitted"`. The 
examples in this section pass `display` explicitly so they apply to all models, but on Mythos 
Preview you can leave it unset and receive the same behavior. To receive summarized thinking on 
Mythos Preview, set `display: "summarized"` explicitly.

Automated pipelines that never surface thinking content to end users can skip the overhead of 
receiving thinking tokens over the wire. Latency-sensitive applications get the same reasoning 
quality without waiting for thinking text to stream before the final response begins.

When `display: "omitted"` is set, the response contains `thinking` blocks with an empty `thinking` 
field:

```
{
  "content": [
    {
      "type": "thinking",
      "thinking": "",
      "signature": "EosnCkYICxIMMb3LzNrMu..."
    },
    {
      "type": "text",
      "text": "The answer is 12,231."
    }
  ]
}
```

When streaming with `display: "omitted"`, no `thinking_delta` events are emitted; see [Streaming 
thinking](#streaming-thinking) below for the event sequence.

### Streaming thinking

You can stream extended thinking responses using [server-sent events 
(SSE)](https://developer.mozilla.org/en-US/Web/API/Server-sent%5Fevents/Using%5Fserver-sent%5Fevents
).

When streaming is enabled for extended thinking, you receive thinking content via `thinking_delta` 
events.

When `display: "omitted"` is set, no `thinking_delta` events are emitted. See [Controlling thinking 
display](#controlling-thinking-display).

For more documentation on streaming via the Messages API, see [Streaming 
Messages](https://docs.anthropic.com/docs/en/build-with-claude/streaming).

Here's how to handle streaming with thinking:

Example streaming output:

```
event: message_start
data: {"type": "message_start", "message": {"id": "msg_01...", "type": "message", "role": 
"assistant", "content": [], "model": "claude-sonnet-4-6", "stop_reason": null, "stop_sequence": 
null}}

event: content_block_start
data: {"type": "content_block_start", "index": 0, "content_block": {"type": "thinking", "thinking": 
"", "signature": ""}}

event: content_block_delta
data: {"type": "content_block_delta", "index": 0, "delta": {"type": "thinking_delta", "thinking": 
"I need to find the GCD of 1071 and 462 using the Euclidean algorithm.\n\n1071 = 2 × 462 + 147"}}

event: content_block_delta
data: {"type": "content_block_delta", "index": 0, "delta": {"type": "thinking_delta", "thinking": 
"\n462 = 3 × 147 + 21\n147 = 7 × 21 + 0\n\nSo GCD(1071, 462) = 21"}}

// Additional thinking deltas...

event: content_block_delta
data: {"type": "content_block_delta", "index": 0, "delta": {"type": "signature_delta", "signature": 
"EqQBCgIYAhIM1gbcDa9GJwZA2b3hGgxBdjrkzLoky3dl1pkiMOYds..."}}

event: content_block_stop
data: {"type": "content_block_stop", "index": 0}

event: content_block_start
data: {"type": "content_block_start", "index": 1, "content_block": {"type": "text", "text": ""}}

event: content_block_delta
data: {"type": "content_block_delta", "index": 1, "delta": {"type": "text_delta", "text": "The 
greatest common divisor of 1071 and 462 is **21**."}}

// Additional text deltas...

event: content_block_stop
data: {"type": "content_block_stop", "index": 1}

event: message_delta
data: {"type": "message_delta", "delta": {"stop_reason": "end_turn", "stop_sequence": null}}

event: message_stop
data: {"type": "message_stop"}
```

When `display: "omitted"` is set, the thinking block opens, a single `signature_delta` arrives, and 
the block closes without any `thinking_delta` events. Text streaming begins immediately after:

```
event: content_block_start
data: 
{"type":"content_block_start","index":0,"content_block":{"type":"thinking","thinking":"","signature"
:""}}

event: content_block_delta
data: 
{"type":"content_block_delta","index":0,"delta":{"type":"signature_delta","signature":"EosnCkYICxIMM
b3LzNrMu..."}}

event: content_block_stop
data: {"type":"content_block_stop","index":0}

event: content_block_start
data: {"type":"content_block_start","index":1,"content_block":{"type":"text","text":""}}
```

When using streaming with thinking enabled, you might notice that text sometimes arrives in larger 
chunks alternating with smaller, token-by-token delivery. This is expected behavior, especially for 
thinking content.

The streaming system needs to process content in batches for optimal performance, which can result 
in this "chunky" delivery pattern, with possible delays between streaming events.

## Extended thinking with tool use

Extended thinking can be used alongside [tool 
use](https://docs.anthropic.com/docs/en/agents-and-tools/tool-use/overview), allowing Claude to 
reason through tool selection and results processing.

When using extended thinking with tool use, be aware of the following limitations:

1. **Tool choice limitation**: Tool use with thinking only supports `tool_choice: {"type": "auto"}` 
(the default) or `tool_choice: {"type": "none"}`. Using `tool_choice: {"type": "any"}` or 
`tool_choice: {"type": "tool", "name": "..."}` will result in an error because these options force 
tool use, which is incompatible with extended thinking.
2. **Preserving thinking blocks**: During tool use, you must pass `thinking` blocks back to the API 
for the last assistant message. Include the complete unmodified block back to the API to maintain 
reasoning continuity.

### Toggling thinking modes in conversations

You can't toggle thinking in the middle of an assistant turn, including during tool use loops. The 
entire assistant turn should operate in a single thinking mode:

- **If thinking is enabled**, the final assistant turn should start with a thinking block.
- **If thinking is disabled**, the final assistant turn shouldn't contain any thinking blocks

From the model's perspective, **tool use loops are part of the assistant turn**. An assistant turn 
doesn't complete until Claude finishes its full response, which may include multiple tool calls and 
results.

For example, this sequence is all part of a **single assistant turn**:

```
User: "What's the weather in Paris?"
Assistant: [thinking] + [tool_use: get_weather]
User: [tool_result: "20°C, sunny"]
Assistant: [text: "The weather in Paris is 20°C and sunny"]
```

Even though there are multiple API messages, the tool use loop is conceptually part of one 
continuous assistant response.

#### Graceful thinking degradation

When a mid-turn thinking conflict occurs (such as toggling thinking on or off during a tool use 
loop), the API automatically disables thinking for that request. To preserve model quality and 
remain on-distribution, the API may:

- Strip thinking blocks from the conversation when they would create an invalid turn structure
- Disable thinking for the current request when the conversation history is incompatible with 
thinking being enabled

This means that attempting to toggle thinking mid-turn won't cause an error, but thinking will be 
silently disabled for that request. To confirm whether thinking was active, check for the presence 
of `thinking` blocks in the response.

#### Practical guidance

**Best practice**: Plan your thinking strategy at the start of each turn rather than trying to 
toggle mid-turn.

**Example: Toggling thinking after completing a turn**

```
User: "What's the weather?"
Assistant: [tool_use] (thinking disabled)
User: [tool_result]
Assistant: [text: "It's sunny"]
User: "What about tomorrow?"
Assistant: [thinking] + [text: "..."] (thinking enabled - new turn)
```

By completing the assistant turn before toggling thinking, you ensure that thinking is actually 
enabled for the new request.

Toggling thinking modes also invalidates prompt caching for message history. For more details, see 
the [Extended thinking with prompt caching](#extended-thinking-with-prompt-caching) section.

### Preserving thinking blocks

During tool use, you must pass `thinking` blocks back to the API, and you must include the complete 
unmodified block back to the API. This is critical for maintaining the model's reasoning flow and 
conversation integrity.

While you can omit `thinking` blocks from prior `assistant` role turns, always pass back all 
thinking blocks to the API for any multi-turn conversation. The API:

- Automatically filters the provided thinking blocks
- Uses the relevant thinking blocks necessary to preserve the model's reasoning
- Only bills for the input tokens for the blocks shown to Claude

Which blocks are kept depends on the model. See [Thinking block preservation by 
model](#thinking-block-preservation-in-claude-opus-45-and-later) for the per-class defaults. To 
override the default, use the [`clear_thinking_20251015` context-editing 
strategy](https://docs.anthropic.com/docs/en/build-with-claude/context-editing#thinking-block-cleari
ng).

When toggling thinking modes during a conversation, remember that the entire assistant turn 
(including tool use loops) must operate in a single thinking mode. For more details, see [Toggling 
thinking modes in conversations](#toggling-thinking-modes-in-conversations).

When Claude invokes tools, it is pausing its construction of a response to await external 
information. When tool results are returned, Claude continues building that existing response. This 
necessitates preserving thinking blocks during tool use, for a couple of reasons:

1. **Reasoning continuity**: The thinking blocks capture Claude's step-by-step reasoning that led 
to tool requests. When you post tool results, including the original thinking ensures Claude can 
continue its reasoning from where it left off.
2. **Context maintenance**: While tool results appear as user messages in the API structure, 
they're part of a continuous reasoning flow. Preserving thinking blocks maintains this conceptual 
flow across multiple API calls. For more information on context management, see the [guide on 
context windows](https://docs.anthropic.com/docs/en/build-with-claude/context-windows).

**Important**: When providing `thinking` blocks, the entire sequence of consecutive `thinking` 
blocks must match the outputs generated by the model during the original request; you can't 
rearrange or modify the sequence of these blocks.

### Interleaved thinking

Extended thinking with tool use in Claude 4 models supports interleaved thinking, which enables 
Claude to think between tool calls and make more sophisticated reasoning after receiving tool 
results.

With interleaved thinking, Claude can:

- Reason about the results of a tool call before deciding what to do next
- Chain multiple tool calls with reasoning steps in between
- Make more nuanced decisions based on intermediate results

**Model support:**

- **Claude Opus 4.8**: Interleaved thinking is automatically enabled when using [adaptive 
thinking](https://docs.anthropic.com/docs/en/build-with-claude/adaptive-thinking) (the only 
supported thinking mode on Claude Opus 4.8). No beta header is needed.
- **[Claude Mythos Preview](https://anthropic.com/glasswing)**: Interleaved thinking happens 
automatically. Every inter-tool reasoning step moves into a thinking block instead of plain text, 
and thinking blocks are preserved across turns by default. No beta header is needed or supported.
- **Claude Opus 4.7**: Interleaved thinking is automatically enabled when using [adaptive 
thinking](https://docs.anthropic.com/docs/en/build-with-claude/adaptive-thinking) (the only 
supported thinking mode on Opus 4.7). No beta header is needed.
- **Claude Opus 4.6**: Interleaved thinking is automatically enabled when using [adaptive 
thinking](https://docs.anthropic.com/docs/en/build-with-claude/adaptive-thinking). No beta header 
is needed. The `interleaved-thinking-2025-05-14` beta header is **deprecated** on Opus 4.6 and is 
safely ignored if included.
- **Claude Sonnet 4.6**: Interleaved thinking is automatically enabled when using [adaptive 
thinking](https://docs.anthropic.com/docs/en/build-with-claude/adaptive-thinking) (recommended). 
The `interleaved-thinking-2025-05-14` beta header with manual extended thinking (`thinking: {type: 
"enabled"}`) is still functional but deprecated.

Here are some important considerations for interleaved thinking:

- With interleaved thinking, the `budget_tokens` can exceed the `max_tokens` parameter, as it 
represents the total budget across all thinking blocks within one assistant turn.
- Interleaved thinking is only supported for [tools used via the Messages 
API](https://docs.anthropic.com/docs/en/agents-and-tools/tool-use/overview).
- The Claude API and [Claude Platform on 
AWS](https://docs.anthropic.com/docs/en/build-with-claude/claude-platform-on-aws) accept 
`interleaved-thinking-2025-05-14` in requests to any model without returning an error. On models 
that don't support interleaved thinking, the header is ignored. On Claude Opus 4.8, Claude Opus 
4.7, and Claude Opus 4.6, it's deprecated and safely ignored. On Claude Mythos Preview, it's not 
needed and safely ignored.
- On partner-operated platforms (for example, [Amazon 
Bedrock](https://docs.anthropic.com/docs/en/build-with-claude/claude-in-amazon-bedrock) and [Vertex 
AI](https://docs.anthropic.com/docs/en/build-with-claude/claude-on-vertex-ai)), if you pass 
`interleaved-thinking-2025-05-14` to any model aside from Claude Opus 4.8, Claude Opus 4.7, Claude 
Opus 4.6, Claude Sonnet 4.6, Claude Opus 4.5, Claude Opus 4.1, Opus 4 (deprecated), Sonnet 4.5, or 
Sonnet 4 (deprecated), your request will fail.

## Extended thinking with prompt caching

[Prompt caching](https://docs.anthropic.com/docs/en/build-with-claude/prompt-caching) with thinking 
has several important considerations:

Extended thinking tasks often take longer than 5 minutes to complete. Consider using the [1-hour 
cache 
duration](https://docs.anthropic.com/docs/en/build-with-claude/prompt-caching#1-hour-cache-duration)
 to maintain cache hits across longer thinking sessions and multi-step workflows.

**Thinking block context removal**

- On earlier Opus/Sonnet models and all Haiku models, thinking blocks from previous turns are 
removed from context, which can affect cache breakpoints. On Opus 4.5+ and Sonnet 4.6+, they are 
kept by default.
- When continuing conversations with tool use, thinking blocks are cached and count as input tokens 
when read from cache
- This creates a tradeoff: while thinking blocks don't consume context window space visually, they 
still count toward your input token usage when cached
- If thinking becomes disabled and you pass thinking content in the current tool use turn, the 
thinking content will be stripped and thinking will remain disabled for that request

**Cache invalidation patterns**

- Changes to thinking parameters (enabled/disabled or budget allocation) invalidate message cache 
breakpoints
- [Interleaved thinking](#interleaved-thinking) amplifies cache invalidation, as thinking blocks 
can occur between multiple [tool calls](#extended-thinking-with-tool-use)
- System prompts and tools remain cached despite thinking parameter changes or block removal

On earlier Opus/Sonnet models and all Haiku models, thinking blocks are removed for caching and 
context calculations; on Opus 4.5+ and Sonnet 4.6+, they are kept by default. In either case, they 
must be preserved when continuing conversations with [tool use](#extended-thinking-with-tool-use), 
especially with [interleaved thinking](#interleaved-thinking).

### Understanding thinking block caching behavior

When using extended thinking with tool use, thinking blocks exhibit specific caching behavior that 
affects token counting:

**How it works:**

1. Caching only occurs when you make a subsequent request that includes tool results
2. When the subsequent request is made, the previous conversation history (including thinking 
blocks) can be cached
3. These cached thinking blocks count as input tokens in your usage metrics when read from the cache
4. When a non-tool-result user block is included: on Opus 4.5+ and Sonnet 4.6+, previous thinking 
blocks are kept; on earlier Opus/Sonnet models and all Haiku models, all previous thinking blocks 
are ignored and stripped from context

**Detailed example flow:**

**Request 1:**

```
User: "What's the weather in Paris?"
```

**Response 1:**

```
[thinking_block_1] + [tool_use block 1]
```

**Request 2:**

```
User: ["What's the weather in Paris?"],
Assistant: [thinking_block_1] + [tool_use block 1],
User: [tool_result_1, cache=True]
```

**Response 2:**

```
[thinking_block_2] + [text block 2]
```

Request 2 writes a cache of the request content (not the response). The cache includes the original 
user message, the first thinking block, tool use block, and the tool result.

**Request 3:**

```
User: ["What's the weather in Paris?"],
Assistant: [thinking_block_1] + [tool_use block 1],
User: [tool_result_1, cache=True],
Assistant: [thinking_block_2] + [text block 2],
User: [Text response, cache=True]
```

For Opus 4.5+ and Sonnet 4.6+, all previous thinking blocks are kept by default. For earlier 
Opus/Sonnet models and all Haiku models, because a non-tool-result user block was included, all 
previous thinking blocks are ignored and stripped from context. This request will be processed the 
same as:

```
User: ["What's the weather in Paris?"],
Assistant: [tool_use block 1],
User: [tool_result_1, cache=True],
Assistant: [text block 2],
User: [Text response, cache=True]
```

**Key points:**

- This caching behavior happens automatically, even without explicit `cache_control` markers
- This behavior is consistent whether using regular thinking or interleaved thinking

## Max tokens and context window size with extended thinking

`max_tokens` (which includes your thinking budget when thinking is enabled) is enforced as a strict 
limit. On Claude 4.5 models and newer, if input tokens plus `max_tokens` exceeds the context window 
size, the API accepts the request. If generation then reaches the context window limit, it stops 
with `stop_reason: "model_context_window_exceeded"`. On earlier models, the API returns a 
validation error instead. See [Handling stop 
reasons](https://docs.anthropic.com/docs/en/build-with-claude/handling-stop-reasons).

You can read through the [guide on context 
windows](https://docs.anthropic.com/docs/en/build-with-claude/context-windows) for a more thorough 
deep dive.

### The context window with extended thinking

When calculating context window usage with thinking enabled, there are some considerations to be 
aware of:

- On Opus 4.5+ and Sonnet 4.6+, thinking blocks from previous turns are kept and count towards your 
context window; on earlier Opus/Sonnet models and all Haiku models, they are stripped and not 
counted
- Current turn thinking counts towards your `max_tokens` limit for that turn

The diagram below demonstrates the specialized token management when extended thinking is enabled:

![Context window diagram with extended 
thinking](https://docs.anthropic.com/docs/images/context-window-thinking.svg)

The effective context window is calculated as:

```
context window =
  (current input tokens - previous thinking tokens) +
  (thinking tokens + encrypted thinking tokens + text output tokens)
```

Use the [token counting API](https://docs.anthropic.com/docs/en/build-with-claude/token-counting) 
to get accurate token counts for your specific use case, especially when working with multi-turn 
conversations that include thinking.

### The context window with extended thinking and tool use

When using extended thinking with tool use, thinking blocks must be explicitly preserved and 
returned with the tool results.

The effective context window calculation for extended thinking with tool use becomes:

```
context window =
  (current input tokens + previous thinking tokens + tool use tokens) +
  (thinking tokens + encrypted thinking tokens + text output tokens)
```

The diagram below illustrates token management for extended thinking with tool use:

![Context window diagram with extended thinking and tool 
use](https://docs.anthropic.com/docs/images/context-window-thinking-tools.svg)

### Managing tokens with extended thinking

Given the context window and `max_tokens` behavior with extended thinking, you may need to:

- More actively monitor and manage your token usage
- Adjust `max_tokens` values as your prompt length changes
- Potentially use the [token counting 
endpoints](https://docs.anthropic.com/docs/en/build-with-claude/token-counting) more frequently
- Be aware that previous thinking blocks don't accumulate in your context window

## Thinking encryption

Full thinking content is encrypted and returned in the `signature` field. This field is used to 
verify that thinking blocks were generated by Claude when passed back to the API.

It is only strictly necessary to send back thinking blocks when using [tools with extended 
thinking](https://docs.anthropic.com/docs/en/build-with-claude/extended-thinking#extended-thinking-w
ith-tool-use). Otherwise you can omit thinking blocks from previous turns. If you pass them back, 
whether the API keeps or strips them depends on the model: Opus 4.5+ and Sonnet 4.6+ keep them in 
context by default; earlier Opus/Sonnet models and all Haiku models strip them. See [context 
editing](https://docs.anthropic.com/docs/en/build-with-claude/context-editing) to configure this.

If sending back thinking blocks, pass everything back as you received it for consistency and to 
avoid potential issues.

Here are some important considerations on thinking encryption:

- When [streaming 
responses](https://docs.anthropic.com/docs/en/build-with-claude/extended-thinking#streaming-thinking
), the signature is added via a `signature_delta` inside a `content_block_delta` event just before 
the `content_block_stop` event.
- `signature` values are significantly longer in Claude 4 models than in previous models.
- The `signature` field is an opaque field and should not be interpreted or parsed.
- `signature` values are compatible across platforms (Claude APIs, [Amazon 
Bedrock](https://docs.anthropic.com/docs/en/build-with-claude/claude-in-amazon-bedrock), and 
[Vertex AI](https://docs.anthropic.com/docs/en/build-with-claude/claude-on-vertex-ai)). Values 
generated on one platform will be compatible with another.

## Redacted thinking blocks

In addition to regular `thinking` blocks, the API may return `redacted_thinking` blocks. A 
`redacted_thinking` block contains encrypted thinking content in a `data` field, with no readable 
summary:

```
{
  "type": "redacted_thinking",
  "data": "..."
}
```

The `data` field is opaque and encrypted. Like the `signature` field on regular thinking blocks, 
you should pass `redacted_thinking` blocks back to the API unchanged when continuing a multi-turn 
conversation with 
[tools](https://docs.anthropic.com/docs/en/build-with-claude/extended-thinking#extended-thinking-wit
h-tool-use).

If your code filters content blocks by type (for example, `block.type == "thinking"`) when 
round-tripping responses with tool use, also include `redacted_thinking` blocks. Filtering on 
`block.type == "thinking"` alone silently drops `redacted_thinking` blocks and breaks the 
multi-turn protocol described above.

`redacted_thinking` blocks are a distinct content block type returned by the API when portions of 
thinking are safety-redacted. This is separate from the [`display: 
"omitted"`](#controlling-thinking-display) option, which returns regular `thinking` blocks with an 
empty `thinking` field.

## Differences in thinking across model versions

The Messages API handles thinking differently across Claude model versions. The following table 
gives a condensed comparison:

| Feature | Claude 4 models (pre-Opus 4.5) | Claude Opus 4.5 | Claude Sonnet 4.6 | Claude Opus 4.6 
([adaptive thinking](https://docs.anthropic.com/docs/en/build-with-claude/adaptive-thinking)) | 
Claude Opus 4.7 ([adaptive 
thinking](https://docs.anthropic.com/docs/en/build-with-claude/adaptive-thinking)) | Claude Opus 
4.8 ([adaptive thinking](https://docs.anthropic.com/docs/en/build-with-claude/adaptive-thinking)) | 
[Claude Mythos Preview](https://anthropic.com/glasswing) ([adaptive 
thinking](https://docs.anthropic.com/docs/en/build-with-claude/adaptive-thinking)) |
| --- | --- | --- | --- | --- | --- | --- | --- |
| **Thinking output** | Returns summarized thinking | Returns summarized thinking | Returns 
summarized thinking | Returns summarized thinking | Omitted by default; set `display: "summarized"` 
to receive summarized thinking | Omitted by default; set `display: "summarized"` to receive 
summarized thinking | Omitted by default; set `display: "summarized"` to receive summarized 
thinking. Raw thinking tokens are never returned. |

### Thinking block preservation by model

Whether thinking blocks from previous assistant turns are preserved in context by default depends 
on the model class. **Opus**: Claude Opus 4.5 and later Opus models keep all prior thinking blocks; 
Claude Opus 4.1 and earlier Opus models keep only the last assistant turn's thinking. **Sonnet**: 
Claude Sonnet 4.6 and later Sonnet models keep all; Claude Sonnet 4.5 and earlier Sonnet models 
keep only the last turn. **Haiku**: all Haiku models through Claude Haiku 4.5 keep only the last 
turn. [Claude Mythos Preview](https://anthropic.com/glasswing) also keeps all prior thinking blocks.

**Benefits of thinking block preservation:**

- **Cache optimization**: When using tool use, preserved thinking blocks enable cache hits as they 
are passed back with tool results and cached incrementally across the assistant turn, resulting in 
token savings in multi-step workflows
- **No intelligence impact**: Preserving thinking blocks has no negative effect on model performance

**Important considerations:**

- **Context usage**: Long conversations will consume more context space since thinking blocks are 
retained in context
- **Automatic behavior**: This is the default for each model as listed above. No code changes or 
beta headers are required
- **Backward compatibility**: To leverage this feature, continue passing complete, unmodified 
thinking blocks back to the API as you would for tool use

For earlier models (Claude Sonnet 4.5, Opus 4.1, etc.), thinking blocks from previous turns 
continue to be removed from context. The existing behavior described in the [Extended thinking with 
prompt caching](#extended-thinking-with-prompt-caching) section applies to those models.

## Pricing

For complete pricing information including base rates, cache writes, cache hits, and output tokens, 
see the [pricing page](https://docs.anthropic.com/docs/en/about-claude/pricing).

The thinking process incurs charges for:

- Tokens used during thinking (output tokens)
- Thinking blocks from prior assistant turns kept in context: only the last turn on earlier 
Opus/Sonnet models and all Haiku models; all turns by default on Opus 4.5+ and Sonnet 4.6+ (input 
tokens)
- Standard text output tokens

When extended thinking is enabled, a specialized system prompt is automatically included to support 
this feature.

When using summarized thinking:

- **Input tokens:** Tokens in your original request (excludes thinking tokens from previous turns)
- **Output tokens (billed):** The original thinking tokens that Claude generated internally
- **Output tokens (visible):** The summarized thinking tokens you see in the response
- **No charge:** Tokens used to generate the summary

When using `display: "omitted"`:

- **Input tokens:** Tokens in your original request (same as summarized)
- **Output tokens (billed):** The original thinking tokens that Claude generated internally (same 
as summarized)
- **Output tokens (visible):** Zero thinking tokens (the `thinking` field is empty)

The billed output token count will **not** match the visible token count in the response. You are 
billed for the full thinking process, not the thinking content visible in the response.

To see how many billed output tokens were spent on internal reasoning, read 
`usage.output_tokens_details.thinking_tokens` in the response. This value reflects the raw 
reasoning the model generated (not the summarized text returned in the body) and is always less 
than or equal to `output_tokens`. Subtract it from `output_tokens` to approximate the non-reasoning 
portion of the output.

```
{
  "usage": {
    "input_tokens": 25,
    "output_tokens": 348,
    "output_tokens_details": {
      "thinking_tokens": 312
    }
  }
}
```

`output_tokens` remains the inclusive, authoritative total used for billing. 
`output_tokens_details` is a read-only breakdown for observability.

## Best practices and considerations for extended thinking

### Working with thinking budgets

- **Budget optimization:** The minimum budget is 1,024 tokens. Start at the minimum and increase 
the thinking budget incrementally to find the optimal range for your use case. Higher token counts 
enable more comprehensive reasoning but with diminishing returns depending on the task. Increasing 
the budget can improve response quality at the tradeoff of increased latency. For critical tasks, 
test different settings to find the optimal balance. Note that the thinking budget is a target 
rather than a strict limit. Actual token usage may vary based on the task.
- **Starting points:** Start with larger thinking budgets (16k+ tokens) for complex tasks and 
adjust based on your needs.
- **Large budgets:** For thinking budgets above 32k, use [batch 
processing](https://docs.anthropic.com/docs/en/build-with-claude/batch-processing) to avoid 
networking issues. Requests pushing the model to think above 32k tokens causes long running 
requests that might run up against system timeouts and open connection limits.
- **Token usage tracking:** Monitor thinking token usage to optimize costs and performance. The 
`usage.output_tokens_details.thinking_tokens` field in the response reports how many of the billed 
output tokens were internal reasoning. When streaming, this breakdown appears only on the final 
`message_delta` event.

### Performance considerations

- **Response times:** Be prepared for longer response times due to additional processing. 
Generating thinking blocks increases overall response time.
- **Streaming requirements:** The SDKs require streaming when `max_tokens` is greater than 21,333 
to avoid HTTP timeouts on long-running requests. This is a client-side validation, not an API 
restriction. If you don't need to process events incrementally, use `.stream()` with 
`.get_final_message()` (Python) or `.finalMessage()` (TypeScript) to get the complete `Message` 
object without handling individual events. See [Streaming 
Messages](https://docs.anthropic.com/docs/en/build-with-claude/streaming#get-the-final-message-witho
ut-handling-events) for details. When streaming, be prepared to handle both thinking and text 
content blocks as they arrive.
- **Omitting thinking for latency:** If your application doesn't display thinking content, set 
`display: "omitted"` on the thinking configuration to reduce time-to-first-text-token. See 
[Controlling thinking display](#controlling-thinking-display).

### Feature compatibility

- Thinking isn't compatible with `temperature` or `top_k` modifications as well as [forced tool 
use](https://docs.anthropic.com/docs/en/agents-and-tools/tool-use/define-tools#forcing-tool-use).
- When thinking is enabled, you can set `top_p` to values between 1 and 0.95.
- You can't pre-fill responses when thinking is enabled.
- Changes to the thinking budget invalidate cached prompt prefixes that include messages. However, 
cached system prompts and tool definitions will continue to work when thinking parameters change.

### Usage guidelines

- **Task selection:** Use extended thinking for particularly complex tasks that benefit from 
step-by-step reasoning, like math, coding, and analysis.
- **Context handling:** You don't need to remove previous thinking blocks yourself. On Opus 4.5+ 
and Sonnet 4.6+, the Claude API keeps thinking blocks from previous turns by default; on earlier 
Opus/Sonnet models and all Haiku models, it automatically ignores them and they aren't included 
when calculating context usage.
- **Prompt engineering:** Review the [extended thinking prompting 
tips](https://docs.anthropic.com/docs/en/build-with-claude/prompt-engineering/claude-prompting-best-
practices#leverage-thinking-and-interleaved-thinking-capabilities) if you want to maximize Claude's 
thinking capabilities.

---
Title: Anthropic Messages streaming
SourceURL: https://docs.anthropic.com/en/api/messages-streaming
SourceTool: defuddle
FetchedAt: 2026-06-05T08:08:42-04:00
Ticket: 2026-06-05-geppetto-provider-gap-audit
Topics:
  - geppetto
  - providers
  - api-docs
DocType: source
Summary: Official provider documentation captured for the Geppetto provider gap audit.
---

When creating a Message, you can set `"stream": true` to incrementally stream the response using 
[server-sent 
events](https://developer.mozilla.org/en-US/Web/API/Server-sent%5Fevents/Using%5Fserver-sent%5Fevent
s) (SSE).

## Streaming with SDKs

The [Python](https://github.com/anthropics/anthropic-sdk-python) and 
[TypeScript](https://github.com/anthropics/anthropic-sdk-typescript) SDKs offer multiple ways of 
streaming. The [PHP](https://github.com/anthropics/anthropic-sdk-php) SDK provides streaming via 
`createStream()`. The Python SDK allows both sync and async streams. See the documentation in each 
SDK for details.

```
client = anthropic.Anthropic()

with client.messages.stream(
    max_tokens=1024,
    messages=[{"role": "user", "content": "Hello"}],
    model="claude-opus-4-8",
) as stream:
    for text in stream.text_stream:
        print(text, end="", flush=True)
```

## Get the final message without handling events

If you don't need to process text as it arrives, the SDKs provide a way to use streaming under the 
hood while returning the complete `Message` object, identical to what `.create()` returns. This is 
especially useful for requests with large `max_tokens` values, where the SDKs require streaming to 
avoid HTTP timeouts.

The `.stream()` call keeps the HTTP connection alive with server-sent events, then 
`.get_final_message()` (Python) or `.finalMessage()` (TypeScript) accumulates all events and 
returns the complete `Message` object. In Go, you call `message.Accumulate(event)` inside the 
stream loop to build the same complete `Message`. In Java, use `MessageAccumulator.create()` and 
call `accumulator.accumulate(event)` on each event. In Ruby, call `.accumulated_message` on the 
stream. In the PHP SDK, you iterate over stream events manually to accumulate the response.

## Event types

Each server-sent event includes a named event type and associated JSON data. Each event uses an SSE 
event name (e.g. `event: message_stop`), and includes the matching event `type` in its data.

Each stream uses the following event flow:

1. `message_start`: contains a `Message` object with empty `content`.
2. A series of content blocks, each of which have a `content_block_start`, one or more 
`content_block_delta` events, and a `content_block_stop` event. Each content block has an `index` 
that corresponds to its index in the final Message `content` array.
3. One or more `message_delta` events, indicating top-level changes to the final `Message` object.
4. A final `message_stop` event.

The token counts shown in the `usage` field of the `message_delta` event are *cumulative*.

### Ping events

Event streams may also include any number of `ping` events.

The API may occasionally send [errors](https://docs.anthropic.com/docs/en/api/errors) in the event 
stream. For example, during periods of high usage, you may receive an `overloaded_error`, which 
would normally correspond to an HTTP 529 in a non-streaming context:

```
event: error
data: {"type": "error", "error": {"type": "overloaded_error", "message": "Overloaded"}}
```

### Other events

In accordance with the [versioning policy](https://docs.anthropic.com/docs/en/api/versioning), new 
event types may be added, and your code should handle unknown event types gracefully.

## Content block delta types

Each `content_block_delta` event contains a `delta` of a type that updates the `content` block at a 
given `index`.

### Text delta

A `text` content block delta looks like:

```
event: content_block_delta
data: {"type": "content_block_delta","index": 0,"delta": {"type": "text_delta", "text": "ello 
frien"}}
```

### Input JSON delta

The deltas for `tool_use` content blocks correspond to updates for the `input` field of the block. 
To support maximum granularity, the deltas are *partial JSON strings*, whereas the final 
`tool_use.input` is always an *object*.

You can accumulate the string deltas and parse the JSON once you receive a `content_block_stop` 
event, by using a library like 
[Pydantic](https://docs.pydantic.dev/latest/concepts/json/#partial-json-parsing) to do partial JSON 
parsing, or by using the [SDKs](https://docs.anthropic.com/docs/en/api/client-sdks), which provide 
helpers to access parsed incremental values.

A `tool_use` content block delta looks like:

```
event: content_block_delta
data: {"type": "content_block_delta","index": 1,"delta": {"type": 
"input_json_delta","partial_json": "{\"location\": \"San Fra"}}}
```

Note: Current models only support emitting one complete key and value property from `input` at a 
time. As such, when using tools, there may be delays between streaming events while the model is 
working. Once an `input` key and value are accumulated, they are emitted as multiple 
`content_block_delta` events with chunked partial json so that the format can automatically support 
finer granularity in future models.

### Thinking delta

When using [extended 
thinking](https://docs.anthropic.com/docs/en/build-with-claude/extended-thinking#streaming-thinking)
 with streaming enabled, you'll receive thinking content via `thinking_delta` events. These deltas 
correspond to the `thinking` field of the `thinking` content blocks.

For thinking content, a special `signature_delta` event is sent just before the 
`content_block_stop` event. This signature is used to verify the integrity of the thinking block.

When `display: "omitted"` is set on the thinking configuration, no `thinking_delta` events are 
sent. The thinking block opens, receives a single `signature_delta`, and closes. See [Controlling 
thinking 
display](https://docs.anthropic.com/docs/en/build-with-claude/extended-thinking#controlling-thinking
-display).

A typical thinking delta looks like:

```
event: content_block_delta
data: {"type": "content_block_delta", "index": 0, "delta": {"type": "thinking_delta", "thinking": 
"I need to find the GCD of 1071 and 462 using the Euclidean algorithm.\n\n1071 = 2 × 462 + 147"}}
```

The signature delta looks like:

```
event: content_block_delta
data: {"type": "content_block_delta", "index": 0, "delta": {"type": "signature_delta", "signature": 
"EqQBCgIYAhIM1gbcDa9GJwZA2b3hGgxBdjrkzLoky3dl1pkiMOYds..."}}
```

## Full HTTP stream response

Use the [client SDKs](https://docs.anthropic.com/docs/en/api/client-sdks) when using streaming 
mode. However, if you are building a direct API integration, you need to handle these events 
yourself.

A stream response consists of:

1. A `message_start` event
2. Potentially multiple content blocks, each of which contains:
	- A `content_block_start` event
		- Potentially multiple `content_block_delta` events
		- A `content_block_stop` event
3. One or more `message_delta` events
4. A `message_stop` event

There may be `ping` events dispersed throughout the response as well. See [Event 
types](#event-types) for more details on the format.

### Basic streaming request

```
event: message_start
data: {"type": "message_start", "message": {"id": "msg_1nZdL29xx5MUA1yADyHTEsnR8uuvGzszyY", "type": 
"message", "role": "assistant", "content": [], "model": "claude-opus-4-8", "stop_reason": null, 
"stop_sequence": null, "usage": {"input_tokens": 25, "output_tokens": 1}}}

event: content_block_start
data: {"type": "content_block_start", "index": 0, "content_block": {"type": "text", "text": ""}}

event: ping
data: {"type": "ping"}

event: content_block_delta
data: {"type": "content_block_delta", "index": 0, "delta": {"type": "text_delta", "text": "Hello"}}

event: content_block_delta
data: {"type": "content_block_delta", "index": 0, "delta": {"type": "text_delta", "text": "!"}}

event: content_block_stop
data: {"type": "content_block_stop", "index": 0}

event: message_delta
data: {"type": "message_delta", "delta": {"stop_reason": "end_turn", "stop_sequence":null}, 
"usage": {"output_tokens": 15}}

event: message_stop
data: {"type": "message_stop"}
```

### Streaming request with tool use

Tool use supports [fine-grained 
streaming](https://docs.anthropic.com/docs/en/agents-and-tools/tool-use/fine-grained-tool-streaming)
 for parameter values. Enable it per tool with `eager_input_streaming`.

This request asks Claude to use a tool to report the weather.

```
event: message_start
data: 
{"type":"message_start","message":{"id":"msg_014p7gG3wDgGV9EUtLvnow3U","type":"message","role":"assi
stant","model":"claude-opus-4-8","stop_sequence":null,"usage":{"input_tokens":472,"output_tokens":2}
,"content":[],"stop_reason":null}}

event: content_block_start
data: {"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}

event: ping
data: {"type": "ping"}

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Okay"}}

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":","}}

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":" let"}}

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"'s"}}

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":" check"}}

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":" the"}}

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":" weather"}}

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":" for"}}

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":" San"}}

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":" Francisco"}}

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":","}}

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":" CA"}}

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":":"}}

event: content_block_stop
data: {"type":"content_block_stop","index":0}

event: content_block_start
data: 
{"type":"content_block_start","index":1,"content_block":{"type":"tool_use","id":"toolu_01T1x1fJ34qAm
k2tNTrN7Up6","name":"get_weather","input":{}}}

event: content_block_delta
data: {"type":"content_block_delta","index":1,"delta":{"type":"input_json_delta","partial_json":""}}

event: content_block_delta
data: 
{"type":"content_block_delta","index":1,"delta":{"type":"input_json_delta","partial_json":"{\"locati
on\":"}}

event: content_block_delta
data: {"type":"content_block_delta","index":1,"delta":{"type":"input_json_delta","partial_json":" 
\"San"}}

event: content_block_delta
data: {"type":"content_block_delta","index":1,"delta":{"type":"input_json_delta","partial_json":" 
Francisc"}}

event: content_block_delta
data: 
{"type":"content_block_delta","index":1,"delta":{"type":"input_json_delta","partial_json":"o,"}}

event: content_block_delta
data: {"type":"content_block_delta","index":1,"delta":{"type":"input_json_delta","partial_json":" 
CA\"}"}}

event: content_block_stop
data: {"type":"content_block_stop","index":1}

event: message_delta
data: 
{"type":"message_delta","delta":{"stop_reason":"tool_use","stop_sequence":null},"usage":{"output_tok
ens":89}}

event: message_stop
data: {"type":"message_stop"}
```

### Streaming request with extended thinking

This request enables extended thinking with streaming. The `display: "summarized"` setting streams 
a condensed summary of Claude's reasoning rather than the full chain of thought.

```
event: message_start
data: {"type": "message_start", "message": {"id": "msg_01...", "type": "message", "role": 
"assistant", "content": [], "model": "claude-opus-4-8", "stop_reason": null, "stop_sequence": null}}

event: content_block_start
data: {"type": "content_block_start", "index": 0, "content_block": {"type": "thinking", "thinking": 
"", "signature": ""}}

event: content_block_delta
data: {"type": "content_block_delta", "index": 0, "delta": {"type": "thinking_delta", "thinking": 
"I need to find the GCD of 1071 and 462 using the Euclidean algorithm.\n\n1071 = 2 × 462 + 147"}}

event: content_block_delta
data: {"type": "content_block_delta", "index": 0, "delta": {"type": "thinking_delta", "thinking": 
"\n462 = 3 × 147 + 21"}}

event: content_block_delta
data: {"type": "content_block_delta", "index": 0, "delta": {"type": "thinking_delta", "thinking": 
"\n147 = 7 × 21 + 0"}}

event: content_block_delta
data: {"type": "content_block_delta", "index": 0, "delta": {"type": "thinking_delta", "thinking": 
"\nThe remainder is 0, so GCD(1071, 462) = 21."}}

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

event: content_block_stop
data: {"type": "content_block_stop", "index": 1}

event: message_delta
data: {"type": "message_delta", "delta": {"stop_reason": "end_turn", "stop_sequence": null}}

event: message_stop
data: {"type": "message_stop"}
```

This request asks Claude to search the web for current weather information.

```
event: message_start
data: 
{"type":"message_start","message":{"id":"msg_01G...","type":"message","role":"assistant","model":"cl
aude-opus-4-8","content":[],"stop_reason":null,"stop_sequence":null,"usage":{"input_tokens":2679,"ca
che_creation_input_tokens":0,"cache_read_input_tokens":0,"output_tokens":3}}}

event: content_block_start
data: {"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"I'll check"}}

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":" the current 
weather in New York City for you"}}

event: ping
data: {"type": "ping"}

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"."}}

event: content_block_stop
data: {"type":"content_block_stop","index":0}

event: content_block_start
data: 
{"type":"content_block_start","index":1,"content_block":{"type":"server_tool_use","id":"srvtoolu_014
hJH82Qum7Td6UV8gDXThB","name":"web_search","input":{}}}

event: content_block_delta
data: {"type":"content_block_delta","index":1,"delta":{"type":"input_json_delta","partial_json":""}}

event: content_block_delta
data: 
{"type":"content_block_delta","index":1,"delta":{"type":"input_json_delta","partial_json":"{\"query"
}}

event: content_block_delta
data: 
{"type":"content_block_delta","index":1,"delta":{"type":"input_json_delta","partial_json":"\":"}}

event: content_block_delta
data: {"type":"content_block_delta","index":1,"delta":{"type":"input_json_delta","partial_json":" 
\"weather"}}

event: content_block_delta
data: {"type":"content_block_delta","index":1,"delta":{"type":"input_json_delta","partial_json":" 
NY"}}

event: content_block_delta
data: {"type":"content_block_delta","index":1,"delta":{"type":"input_json_delta","partial_json":"C 
to"}}

event: content_block_delta
data: 
{"type":"content_block_delta","index":1,"delta":{"type":"input_json_delta","partial_json":"day\"}"}}

event: content_block_stop
data: {"type":"content_block_stop","index":1 }

event: content_block_start
data: 
{"type":"content_block_start","index":2,"content_block":{"type":"web_search_tool_result","tool_use_i
d":"srvtoolu_014hJH82Qum7Td6UV8gDXThB","content":[{"type":"web_search_result","title":"Weather in 
New York City in May 2025 (New York) - detailed Weather Forecast for a 
month","url":"https://world-weather.info/forecast/usa/new_york/may-2025/","encrypted_content":"Ev0DC
ioIAxgCIiQ3NmU4ZmI4OC1k...","page_age":null},...]}}

event: content_block_stop
data: {"type":"content_block_stop","index":2}

event: content_block_start
data: {"type":"content_block_start","index":3,"content_block":{"type":"text","text":""}}

event: content_block_delta
data: {"type":"content_block_delta","index":3,"delta":{"type":"text_delta","text":"Here's the 
current weather information for New York"}}

event: content_block_delta
data: {"type":"content_block_delta","index":3,"delta":{"type":"text_delta","text":" City:\n\n# 
Weather"}}

event: content_block_delta
data: {"type":"content_block_delta","index":3,"delta":{"type":"text_delta","text":" in New York 
City"}}

event: content_block_delta
data: {"type":"content_block_delta","index":3,"delta":{"type":"text_delta","text":"\n\n"}}

...

event: content_block_stop
data: {"type":"content_block_stop","index":17}

event: message_delta
data: 
{"type":"message_delta","delta":{"stop_reason":"end_turn","stop_sequence":null},"usage":{"input_toke
ns":10682,"cache_creation_input_tokens":0,"cache_read_input_tokens":0,"output_tokens":510,"server_to
ol_use":{"web_search_requests":1}}}

event: message_stop
data: {"type":"message_stop"}
```

### Claude 4.5 and earlier

For Claude 4.5 models and earlier, you can recover a streaming request that was interrupted due to 
network issues, timeouts, or other errors by resuming from where the stream was interrupted. This 
approach saves you from re-processing the entire response.

The basic recovery strategy involves:

1. **Capture the partial response:** Save all content that was successfully received before the 
error occurred
2. **Construct a continuation request:** Create a new API request that includes the partial 
assistant response as the beginning of a new assistant message
3. **Resume streaming:** Continue receiving the rest of the response from where it was interrupted

### Claude 4.6 and later

For Claude 4.6 and later models, the same capture-and-resume strategy applies, but step 2 changes: 
instead of placing the partial response in an assistant message, add a user message that instructs 
the model to continue from where it left off.

1. **Capture the partial response:** Save all content that was successfully received before the 
error occurred
2. **Construct a continuation request:** Create a new API request with a user message containing 
the partial response and an instruction to continue, for example:
	```
	Your previous response was interrupted and ended with [previous_response]. Continue from 
where you left off.
	```
3. **Resume streaming:** Continue receiving the rest of the response from where it was interrupted
1. **Use SDK features:** Leverage the SDK's built-in message accumulation and error handling 
capabilities
2. **Handle content types:** Be aware that messages can contain multiple content blocks (`text`, 
`tool_use`, `thinking`). Tool use and extended thinking blocks cannot be partially recovered. You 
can resume streaming from the most recent text block.

Was this page helpful?

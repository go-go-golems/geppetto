## Unified Streaming Events for Advanced Models (OpenAI Responses, Anthropic)

Purpose: Provide a stable, provider-agnostic event surface that captures text, reasoning, citations, and tool (client/built-in/server) activity with minimal event kinds, expressed via enums and IDs. Based on 08-unified-events-for-advanced-model-functionality.md and mapped to 09-openai-responses-events.md and 10-anthropic-response-events.md.

### Core Event Families

- Message lifecycle: `stream.message_start`, `stream.message_delta{phase,usage,stop_reason}`, `stream.message_stop`, `stream.ping`
- Content blocks: `stream.content_block_start`, `stream.content_block_delta{kind,channel,...}`, `stream.content_block_stop`
- Tool calls (client/server/built-in/MCP):
  - `stream.tool_call_start{tool{kind,builtin|name}}`
  - `stream.tool_call_input.delta` (append-only JSON args)
  - `stream.tool_call_status{phase,action}` for long-running built-ins
  - `stream.tool_call_result{result_raw,search_results,text,image_*}` (normalized + raw)
  - `stream.tool_call_stop{phase,error}`

### Enums (selection)

- BlockType: `text | image | thinking | redacted_thinking | tool_use | tool_result | server_tool_use | server_tool_result | web_search_result | web_search_tool_result | web_search_tool_result_error | web_search_result_location | mcp_tool_use | mcp_tool_result`
- DeltaKind: `text_delta | input_json_delta | citations_delta | annotation_added | refusal_delta | reasoning_delta | reasoning_summary_delta | reasoning_summary_text_delta | audio_delta | audio_transcript_delta`
- ContentChannel: `output_text | refusal | reasoning | reasoning_summary | reasoning_summary_text | audio_transcript`
- ToolKind: `client | server | builtin | mcp | custom`
- BuiltInTool: `web_search | file_search | code_interpreter | computer_use | image_generation`
- StatusPhase: `started | queued | searching | in_progress | interpreting | generating | result | completed | incomplete | canceled | failed`

### IDs

`StreamIDs{ response_id, message_id, item_id, block_id, tool_call_id, output_index, content_index }`

### Provider Mappings (quick reference)

- OpenAI Responses → unified
  - Text: `response.output_text.delta|done` → `stream.content_block_delta{DeltaText}`; `stream.content_block_stop`
  - Citations: `response.output_text.annotation.added` → `stream.content_block_delta{DeltaAnnotation}`
  - Reasoning: `response.reasoning_summary_*`, `response.reasoning_text.*` → `stream.content_block_delta{Reasoning*}`
  - Tools (function/custom/MCP): `response.function_call_arguments.*`, `response.mcp_*` → `stream.tool_call_input.delta`, `stream.tool_call_status|stop`
  - Built-ins: `response.web_search_call.*`, `response.file_search_call.*`, `response.code_interpreter_call.*`, `response.image_generation_call.*` → `stream.tool_call_status/result/stop`

- Anthropic → unified
  - Text/thinking/citations/signature: `content_block_*` deltas → `stream.content_block_*`
  - Client/server tools: `tool_use` start + `input_json_delta` + results blocks → tool events

### Web Search Specialization (Built-in/Server)

- Start: `stream.tool_call_start{Tool: {kind: builtin|server, builtin:web_search|name:"web_search"}}`
- Progress: `stream.tool_call_status{phase: searching|in_progress, action: {query|open_page:url}}`
- Result: `stream.tool_call_result{search_results:[{url,title,snippet,ext}...]}` and/or normalized content blocks
- Stop: `stream.tool_call_stop{phase: completed|failed, error?}`

### Citations

Surfaced as `stream.content_block_delta{kind: annotation_added, annotation:{url,title,start_index,end_index,...}}` for OpenAI, and `citations_delta` for Anthropic. Optionally aggregated into `Final` metadata (`citations:[]`).

### Printer Guidance

- Boundaries: show thinking/output start/end via info or mapped message events
- Text stream: print partial text deltas
- Search progress: print `🔎 Searching: <query>`, `🌐 Open: <url>`, then `✅ Search done`
- Citations: print `📎 <title> - <url>` on annotation/citation deltas and summarize at end

This spec is minimal yet expressive; adding new built-ins typically only requires extending enums and adapter mappings.



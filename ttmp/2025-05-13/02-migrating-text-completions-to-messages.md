# Migrating from Text Completions to Messages in Anthropic's API

## Overview

This guide covers the key differences and changes required when migrating from Anthropic's Text Completions API to the new Messages API. The Messages API provides a more structured approach to conversations with Claude and includes several important changes in how you interact with the model.

## Key Changes

### 1. Input Format

#### Text Completions (Old)
```python
# Raw string input with explicit markers
prompt = "\n\nHuman: Hello there\n\nAssistant: Hi, I'm Claude. How can I help?\n\nHuman: Can you explain Glycolysis to me?\n\nAssistant:"
```

#### Messages (New)
```python
# Structured message list
messages = [
    {"role": "user", "content": "Hello there."},
    {"role": "assistant", "content": "Hi, I'm Claude. How can I help?"},
    {"role": "user", "content": "Can you explain Glycolysis to me?"}
]
```

### 2. Role Names

- Text Completions: Uses `\n\nHuman:` and `\n\nAssistant:` markers
- Messages: Uses `"role": "user"` and `"role": "assistant"` fields
- Note: "human" and "user" roles are equivalent, but "user" is the standard going forward

### 3. Output Format

#### Text Completions (Old)
```python
response = anthropic.completions.create(...)
response.completion  # Returns raw string
# " Hi, I'm Claude"
```

#### Messages (New)
```python
response = anthropic.messages.create(...)
response.content  # Returns list of content blocks
# [{"type": "text", "text": "Hi, I'm Claude"}]
```

### 4. System Prompts

#### Text Completions (Old)
```python
# System prompt as prefix text
prompt = "Today is January 1, 2024.\n\nHuman: Hello, Claude\n\nAssistant:"
```

#### Messages (New)
```python
# Dedicated system parameter
anthropic.Anthropic().messages.create(
    model="claude-3-opus-20240229",
    max_tokens=1024,
    system="Today is January 1, 2024.",
    messages=[
        {"role": "user", "content": "Hello, Claude"}
    ]
)
```

### 5. Continuing Assistant's Response

#### Text Completions (Old)
```python
prompt = "\n\nHuman: Hello\n\nAssistant: Hello, my name is"
```

#### Messages (New)
```python
messages = [
    {"role": "user", "content": "Hello"},
    {"role": "assistant", "content": "Hello, my name is"}
]

# Response continues from last message
# {
#   "role": "assistant",
#   "content": [{"type": "text", "text": " Claude. How can I assist you today?"}],
#   ...
# }
```

### 6. Model Versioning

- Text Completions: Supported major version numbers (e.g., `claude-2`)
- Messages: Requires full model version (e.g., `claude-3-opus-20240229`)
- Auto-upgrades to minor versions are no longer supported

### 7. Stop Reasons

#### Text Completions
- `"stop_sequence"`: Natural end or custom stop sequence
- `"max_tokens"`: Token limit reached

#### Messages
- `"end_turn"`: Natural conversation end
- `"stop_sequence"`: Custom stop sequence generated
- `"max_tokens"`: Token limit reached (unchanged)
- `"tool_use"`: Model invoked one or more tools
- `"pause_turn"`: Long-running turn was paused
- `"refusal"`: Model refused to respond

### 8. Token Limits

#### Text Completions
- Parameter: `max_tokens_to_sample`
- No validation
- Capped per model

#### Messages
- Parameter: `max_tokens`
- Validates against model limits
- Returns error if limit exceeded

### 9. Streaming

#### Text Completions (Old)
```python
# Simple event types
# - completion
# - ping
# - error
```

#### Messages (New)
The Messages API provides a more sophisticated streaming system with multiple event types and content blocks.

##### Event Flow
1. `message_start`: Initial message object with empty content
2. Series of content blocks, each with:
   - `content_block_start`
   - One or more `content_block_delta` events
   - `content_block_stop`
3. One or more `message_delta` events for top-level changes
4. Final `message_stop` event

##### Example Streaming Request
```bash
curl https://api.anthropic.com/v1/messages \
     --header "anthropic-version: 2023-06-01" \
     --header "content-type: application/json" \
     --header "x-api-key: $ANTHROPIC_API_KEY" \
     --data \
'{
  "model": "claude-3-7-sonnet-20250219",
  "messages": [{"role": "user", "content": "Hello"}],
  "max_tokens": 256,
  "stream": true
}'
```

##### Example Response Stream
```json
event: message_start
data: {"type": "message_start", "message": {"id": "msg_1nZdL29xx5MUA1yADyHTEsnR8uuvGzszyY", "type": "message", "role": "assistant", "content": [], "model": "claude-3-7-sonnet-20250219", "stop_reason": null, "stop_sequence": null, "usage": {"input_tokens": 25, "output_tokens": 1}}}

event: content_block_start
data: {"type": "content_block_start", "index": 0, "content_block": {"type": "text", "text": ""}}

event: content_block_delta
data: {"type": "content_block_delta", "index": 0, "delta": {"type": "text_delta", "text": "Hello"}}

event: content_block_delta
data: {"type": "content_block_delta", "index": 0, "delta": {"type": "text_delta", "text": "!"}}

event: content_block_stop
data: {"type": "content_block_stop", "index": 0}

event: message_delta
data: {"type": "message_delta", "delta": {"stop_reason": "end_turn", "stop_sequence":null}, "usage": {"output_tokens": 15}}

event: message_stop
data: {"type": "message_stop"}
```

##### Content Block Delta Types

1. Text Delta
```json
event: content_block_delta
data: {"type": "content_block_delta", "index": 0, "delta": {"type": "text_delta", "text": "ello frien"}}
```

2. Input JSON Delta (for tool use)
```json
event: content_block_delta
data: {"type": "content_block_delta", "index": 1, "delta": {"type": "input_json_delta", "partial_json": "{\"location\": \"San Fra"}}
```

3. Thinking Delta (for extended thinking)
```json
event: content_block_delta
data: {"type": "content_block_delta", "index": 0, "delta": {"type": "thinking_delta", "thinking": "Let me solve this step by step:\n\n1. First..."}}
```

##### Special Events

1. Ping Events
```json
event: ping
data: {"type": "ping"}
```

2. Error Events
```json
event: error
data: {"type": "error", "error": {"type": "overloaded_error", "message": "Overloaded"}}
```

##### Server Tool Use Streaming

The Messages API introduces server-side tools, like web search, with specialized streaming events:

```json
# Initial message
event: message_start
data: {"type":"message_start","message":{"id":"msg_01Gxyz...","type":"message","role":"assistant","model":"claude-3-7-sonnet-20250219","content":[],"stop_reason":null,"stop_sequence":null,"usage":{"input_tokens":120,"output_tokens":3}}}

# Initial text response
event: content_block_start
data: {"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"I'll search for information about that."}}

event: content_block_stop
data: {"type":"content_block_stop","index":0}

# Web search tool use begins
event: content_block_start
data: {"type":"content_block_start","index":1,"content_block":{"type":"server_tool_use","id":"srvtoolu_abc123","name":"web_search","input":{}}}

# Search query is streamed incrementally
event: content_block_delta
data: {"type":"content_block_delta","index":1,"delta":{"type":"input_json_delta","partial_json":"{\"query"}}

event: content_block_delta
data: {"type":"content_block_delta","index":1,"delta":{"type":"input_json_delta","partial_json":"\":\"latest AI research"}}

event: content_block_delta
data: {"type":"content_block_delta","index":1,"delta":{"type":"input_json_delta","partial_json":" 2025\"}"}}

event: content_block_stop
data: {"type":"content_block_stop","index":1}

# Search results are returned
event: content_block_start
data: {"type":"content_block_start","index":2,"content_block":{"type":"web_search_tool_result","tool_use_id":"srvtoolu_abc123","content":[{"type":"web_search_result","title":"Latest AI Research Breakthroughs in 2025","url":"https://example.com/ai-research","encrypted_content":"Ev0DCioIAxgCIiQ3NmU...","page_age":"2025-05-10"}]}}}

event: content_block_stop
data: {"type":"content_block_stop","index":2}

# Claude's response with search results
event: content_block_start
data: {"type":"content_block_start","index":3,"content_block":{"type":"text","text":""}}

event: content_block_delta
data: {"type":"content_block_delta","index":3,"delta":{"type":"text_delta","text":"Based on the latest research in 2025, AI has made significant progress in..."}}

event: content_block_stop
data: {"type":"content_block_stop","index":3}

# Final message with usage statistics
event: message_delta
data: {"type":"message_delta","delta":{"stop_reason":"end_turn","stop_sequence":null},"usage":{"input_tokens":8750,"output_tokens":245,"server_tool_use":{"web_search_requests":1}}}

event: message_stop
data: {"type":"message_stop"}
```

##### Handling Server Tool Streaming

When working with server tool streaming:

1. Tool Input Accumulation
```python
# Accumulate partial JSON
json_parts = []
for chunk in stream:
    if (chunk.type == "content_block_delta" and 
        hasattr(chunk.delta, "type") and 
        chunk.delta.type == "input_json_delta"):
        json_parts.append(chunk.delta.partial_json)
        
        # Check if the JSON is complete
        if chunk.delta.partial_json.endswith("}"):
            full_json = "".join(json_parts)
            tool_input = json.loads(full_json)
            # Process the complete tool input...
            json_parts = []  # Reset for potential next tool
```

2. Tool Result Processing
```python
def handle_web_search_results(block):
    if block.type == "web_search_tool_result":
        # Store encrypted content for multi-turn conversations
        for result in block.content:
            # Process each search result
            print(f"Result: {result.title} - {result.url}")
            
            # Store encrypted content for future messages
            save_encrypted_content(result.encrypted_content)
```

3. Tool Usage Tracking
```python
def monitor_usage(message_delta):
    if hasattr(message_delta, "usage") and hasattr(message_delta.usage, "server_tool_use"):
        tool_usage = message_delta.usage.server_tool_use
        if hasattr(tool_usage, "web_search_requests"):
            count = tool_usage.web_search_requests
            print(f"Used {count} web search requests")
```

##### SDK Support
Both Python and TypeScript SDKs provide built-in support for streaming with sync and async options. Example with Python SDK:

```python
import anthropic

client = anthropic.Anthropic()

# Synchronous streaming
for chunk in client.messages.create(
    model="claude-3-7-sonnet-20250219",
    max_tokens=1024,
    messages=[{"role": "user", "content": "Hello"}],
    stream=True
):
    if chunk.type == "content_block_delta":
        print(chunk.delta.text, end="")

# Async streaming
async for chunk in client.messages.create(
    model="claude-3-7-sonnet-20250219",
    max_tokens=1024,
    messages=[{"role": "user", "content": "Hello"}],
    stream=True
):
    if chunk.type == "content_block_delta":
        print(chunk.delta.text, end="")
```

##### Best Practices for Streaming

1. Use SDK helpers when possible for proper event handling
2. Implement proper error handling for all event types
3. Handle unknown event types gracefully
4. Accumulate partial JSON for tool use events
5. Track token usage through message_delta events
6. Consider implementing retry logic for error events
7. Handle ping events appropriately to maintain connection
8. Properly process and store encrypted content for multi-turn conversations with server tools

### 10. Content Types

#### Text Completions (Old)
- Only raw text was returned

#### Messages (New)
Multiple content types are supported and must be handled appropriately:

- `text` - Basic text content (most common)
- `image` - Image content from the model
- `tool_use` - Tool usage by the model
- `tool_result` - Results from tool executions
- `server_tool_use` - Server-side tool usage from Claude
- `web_search_tool_result` - Results from web search
- `thinking` - Extended thinking content when enabled

Example of handling different content types:
```python
# Parse response content blocks by type
for content_block in response.content:
    if content_block["type"] == "text":
        # Handle text content
        print(content_block["text"])
    elif content_block["type"] == "tool_use":
        # Handle tool calls
        tool_name = content_block["name"]
        tool_input = content_block["input"]
        # Process tool call...
    elif content_block["type"] == "image":
        # Handle image content
        pass
    elif content_block["type"] == "web_search_tool_result":
        # Handle web search results
        for result in content_block["content"]:
            print(f"Search result: {result['title']} - {result['url']}")
    else:
        # Handle other or unknown content types gracefully
        print(f"Received content of type: {content_block['type']}")
```

## Best Practices for Migration

1. Update all input formatting to use structured messages
2. Replace system prompt prefixes with the `system` parameter
3. Update model version strings to include full versions
4. Modify response handling to work with content blocks
5. Update streaming handlers for new format
6. Review and update token limit handling
7. Test conversation flows with new stop reasons
8. Implement handling for all content block types
9. For server tools like web search, properly store and pass encrypted content
10. Update error handling to match new error responses

## Requirements

- Latest Anthropic Python SDK
- Updated model version strings
- Modified conversation handling logic
- Updated streaming handlers if using streaming responses
- Tool handling logic for any tool-based workflows

## Common Migration Pitfalls

1. Forgetting to update model version strings
2. Mixing old and new role names
3. Not handling content blocks in responses
4. Incorrect system prompt placement
5. Not updating token limit parameters
6. Incomplete handling of streaming events
7. Not properly handling server tool results
8. Missing encrypted content in multi-turn conversations 
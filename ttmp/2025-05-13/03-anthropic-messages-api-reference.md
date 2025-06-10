# Anthropic Messages API Reference

The Messages API is Anthropic's recommended interface for communicating with Claude models. It allows you to send structured conversations and receive structured responses with various content types.

## API Endpoint

```
POST /v1/messages
```

## Required Headers

- `anthropic-version`: The version of the Anthropic API (e.g., "2023-06-01")
- `x-api-key`: Your API key
- `content-type`: "application/json"

## Optional Headers

- `anthropic-beta`: String array for beta features (comma-separated)

## Request Body Parameters

### Required Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `model` | `string` | Claude model identifier (e.g., "claude-3-7-sonnet-20250219") |
| `max_tokens` | `integer` | Maximum tokens to generate (model-specific limits apply) |
| `messages` | `array` | Conversation history as an array of message objects |

### Optional Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `system` | `string` | System prompt for context and instructions |
| `temperature` | `number` | Controls randomness (0.0-1.0, default: 1.0) |
| `top_p` | `number` | Nucleus sampling parameter (0.0-1.0) |
| `top_k` | `integer` | Only sample from top K options for each token |
| `stop_sequences` | `string[]` | Custom sequences that will cause the model to stop |
| `stream` | `boolean` | Whether to stream the response incrementally |
| `metadata` | `object` | Additional metadata about the request |
| `thinking` | `object` | Configuration for extended thinking |
| `tools` | `object[]` | Tools that the model may use |
| `tool_choice` | `object` | How the model should use provided tools |

## Message Format

Each message in the `messages` array must have:

- `role`: Either "user" or "assistant"
- `content`: Either a string or an array of content blocks

### Content Blocks

Content can be a string (shorthand for a text block) or an array of blocks:

#### Text Content
```json
{
  "role": "user", 
  "content": "Hello, Claude"
}
```

#### Equivalent with Content Blocks
```json
{
  "role": "user",
  "content": [
    {
      "type": "text",
      "text": "Hello, Claude"
    }
  ]
}
```

#### Image Content (Vision)
```json
{
  "role": "user",
  "content": [
    {
      "type": "image",
      "source": {
        "type": "base64",
        "media_type": "image/jpeg",
        "data": "/9j/4AAQSkZJRg..."
      }
    },
    {
      "type": "text",
      "text": "What is in this image?"
    }
  ]
}
```

### Supported Image Formats
- `image/jpeg`
- `image/png`
- `image/gif`
- `image/webp`

## Tool Use

The Messages API supports both client-side and server-side tools.

### Tool Definition Example
```json
{
  "tools": [
    {
      "name": "get_stock_price",
      "description": "Get the current stock price for a given ticker symbol.",
      "input_schema": {
        "type": "object",
        "properties": {
          "ticker": {
            "type": "string",
            "description": "The stock ticker symbol, e.g. AAPL for Apple Inc."
          }
        },
        "required": ["ticker"]
      }
    }
  ]
}
```

### Web Search Tool
```json
{
  "tools": [
    {
      "type": "web_search_20250305",
      "name": "web_search",
      "max_uses": 5,
      "allowed_domains": ["example.com", "trusteddomain.org"],
      "user_location": {
        "type": "approximate",
        "city": "San Francisco",
        "region": "California",
        "country": "US",
        "timezone": "America/Los_Angeles"
      }
    }
  ]
}
```

## Response Structure

```json
{
  "id": "msg_01AbCdEfGhIjKlMnOpQrStUv",
  "type": "message",
  "role": "assistant",
  "content": [
    {
      "type": "text",
      "text": "Hello! How can I help you today?"
    }
  ],
  "model": "claude-3-7-sonnet-20250219",
  "stop_reason": "end_turn",
  "stop_sequence": null,
  "usage": {
    "input_tokens": 25,
    "output_tokens": 10,
    "cache_creation_input_tokens": null,
    "cache_read_input_tokens": null,
    "server_tool_use": null
  }
}
```

### Content Block Types in Responses

- `text`: Basic text response
- `tool_use`: Model's use of a defined tool
- `server_tool_use`: Server-side tool use (like web search)
- `web_search_tool_result`: Results from web search
- `thinking`: Extended thinking content (when enabled)

### Stop Reason Values

- `end_turn`: Natural stopping point
- `max_tokens`: Token limit reached
- `stop_sequence`: Custom stop sequence triggered
- `tool_use`: Model invoked one or more tools
- `pause_turn`: Long-running turn was paused
- `refusal`: Model refused to respond

## Usage Tracking

The `usage` field provides token utilization details:

```json
"usage": {
  "input_tokens": 105,
  "output_tokens": 204,
  "cache_creation_input_tokens": 150,
  "cache_read_input_tokens": 0,
  "server_tool_use": {
    "web_search_requests": 1
  }
}
```

## Streaming

When `stream: true` is set, the response is delivered as server-sent events (SSE) in this order:

1. `message_start`: Initial message with empty content
2. For each content block:
   - `content_block_start`: Start of a content block
   - One or more `content_block_delta`: Content updates
   - `content_block_stop`: End of content block
3. `message_delta`: Top-level message changes
4. `message_stop`: End of the message

### Example Stream Response

```
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

### Delta Types

- `text_delta`: For text content updates
- `input_json_delta`: For tool use input JSON fragments
- `thinking_delta`: For extended thinking content
- `signature_delta`: For thinking content verification

## Error Handling

Error response examples:

```json
{
  "type": "error",
  "error": {
    "type": "overloaded_error",
    "message": "Overloaded"
  }
}
```

Common error types:
- `overloaded_error`: Service is currently overloaded
- `invalid_request_error`: Request is invalid
- `authentication_error`: Invalid API key
- `permission_error`: API key doesn't have access to the requested resource
- `not_found_error`: Requested resource doesn't exist

## Best Practices

1. Use the most recent model version for optimal performance
2. Keep system prompts concise and focused
3. Structure conversations with alternating user/assistant turns
4. Set temperature based on task type (lower for factual, higher for creative)
5. Implement proper handling for all content block types
6. Add proper error handling for all API responses
7. For streaming, handle all event types and unexpected events gracefully
8. Monitor token usage to manage costs effectively
9. Use tool definitions with detailed descriptions
10. For multi-turn conversations, maintain conversation history appropriately 
# Anthropic Web Search Tool API Documentation

## Overview

The web search tool is a powerful feature that gives Claude direct access to real-time web content, enabling it to provide up-to-date information beyond its knowledge cutoff. The tool automatically handles citations and source attribution in responses.

## Supported Models

Currently available on:
- Claude 3.7 Sonnet (`claude-3-7-sonnet-20250219`)
- Claude 3.5 Sonnet (new) (`claude-3-5-sonnet-latest`)
- Claude 3.5 Haiku (`claude-3-5-haiku-latest`)

## How It Works

1. Claude evaluates the prompt and decides when to perform searches
2. The API executes searches and provides results to Claude
3. Multiple searches may occur during a single request
4. Claude provides final responses with cited sources

## Implementation Guide

### Basic API Request

```json
{
  "model": "claude-3-7-sonnet-latest",
  "max_tokens": 1024,
  "messages": [
    {
      "role": "user",
      "content": "How do I update a web app to TypeScript 5.5?"
    }
  ],
  "tools": [{
    "type": "web_search_20250305",
    "name": "web_search",
    "max_uses": 5
  }]
}
```

### Tool Configuration Parameters

```json
{
  "type": "web_search_20250305",
  "name": "web_search",
  
  // Optional parameters
  "max_uses": 5,
  "allowed_domains": ["example.com", "trusteddomain.org"],
  "blocked_domains": ["untrustedsource.com"],
  "user_location": {
    "type": "approximate",
    "city": "San Francisco",
    "region": "California", 
    "country": "US",
    "timezone": "America/Los_Angeles"
  }
}
```

### Domain Filtering Rules

- Omit HTTP/HTTPS scheme (use `example.com` not `https://example.com`)
- Subdomains are automatically included (`example.com` covers `docs.example.com`)
- Subpaths are supported (`example.com/blog`)
- Cannot use both `allowed_domains` and `blocked_domains` in same request

### Response Structure

```json
{
  "role": "assistant",
  "content": [
    {
      "type": "text",
      "text": "I'll search for the information."
    },
    {
      "type": "server_tool_use",
      "id": "srvtoolu_xyz789",
      "name": "web_search",
      "input": {
        "query": "search query here"
      }
    },
    {
      "type": "web_search_tool_result",
      "tool_use_id": "srvtoolu_xyz789",
      "content": [
        {
          "type": "web_search_result",
          "url": "https://example.com",
          "title": "Example Title",
          "encrypted_content": "encrypted_content_string",
          "page_age": "2025-05-13"
        }
      ]
    },
    {
      "text": "Response with citations",
      "type": "text",
      "citations": [
        {
          "type": "web_search_result_location",
          "url": "https://example.com",
          "title": "Example Title",
          "encrypted_index": "encrypted_index_string",
          "cited_text": "Cited content up to 150 characters..."
        }
      ]
    }
  ]
}
```

### Error Handling

Error response format:
```json
{
  "type": "web_search_tool_result",
  "tool_use_id": "servertoolu_id",
  "content": {
    "type": "web_search_tool_result_error",
    "error_code": "error_code_here"
  }
}
```

Error codes:
- `too_many_requests`: Rate limit exceeded
- `invalid_input`: Invalid search query parameter
- `max_uses_exceeded`: Maximum web search tool uses exceeded
- `query_too_long`: Query exceeds maximum length
- `unavailable`: Internal error occurred

## Advanced Features

### Prompt Caching

- Works with web search tool
- Add `cache_control` breakpoint in request
- System caches up to last `web_search_tool_result` block
- For multi-turn conversations, set breakpoint after last search result

Example with caching:
```python
import anthropic

client = anthropic.Anthropic()

# First request
response1 = client.messages.create(
    model="claude-3-7-sonnet-latest",
    max_tokens=1024,
    messages=[{
        "role": "user",
        "content": "What's the weather in San Francisco?"
    }],
    tools=[{
        "type": "web_search_20250305",
        "name": "web_search",
        "user_location": {
            "type": "approximate",
            "city": "San Francisco",
            "region": "California",
            "country": "US",
            "timezone": "America/Los_Angeles"
        }
    }]
)

# Second request with cache
messages = [
    # Previous messages...
    {
        "role": "user",
        "content": "Will it rain this week?",
        "cache_control": {"type": "ephemeral"}
    }
]

response2 = client.messages.create(
    model="claude-3-7-sonnet-latest",
    max_tokens=1024,
    messages=messages,
    tools=[{
        "type": "web_search_20250305",
        "name": "web_search"
    }]
)
```

### Streaming Support

The API supports streaming with search events:
```
event: message_start
data: {"type": "message_start", "message": {"id": "msg_abc123", "type": "message"}}

event: content_block_start
data: {"type": "content_block_start", "index": 0, "content_block": {"type": "text", "text": ""}}

// Search decision and query
event: content_block_start
data: {"type": "content_block_start", "index": 1, "content_block": {"type": "server_tool_use", "id": "srvtoolu_xyz789", "name": "web_search"}}

event: content_block_delta
data: {"type": "content_block_delta", "index": 1, "delta": {"type": "input_json_delta", "value": "{\"query\":\"search query here\"}"}}

// Search results
event: content_block_start
data: {"type": "content_block_start", "index": 2, "content_block": {"type": "web_search_tool_result", "tool_use_id": "srvtoolu_xyz789", "content": [{"type": "web_search_result", "title": "Result Title", "url": "https://example.com"}]}}
```

## Usage and Pricing

- $10 per 1,000 searches
- Standard token costs apply for search-generated content
- Search results count as input tokens
- Each search counts as one use regardless of results
- Failed searches are not billed
- Token usage structure:
```json
"usage": {
  "input_tokens": 105,
  "output_tokens": 6039,
  "cache_read_input_tokens": 7123,
  "cache_creation_input_tokens": 7345,
  "server_tool_use": {
    "web_search_requests": 1
  }
}
```

## Best Practices

1. Use appropriate model versions that support web search
2. Implement proper error handling for all error codes
3. Consider using domain filtering for targeted searches
4. Implement caching for multi-turn conversations
5. Display citations properly in user interface
6. Monitor usage and implement rate limiting as needed
7. Use streaming for better user experience with long-running searches

## Requirements

- Organization administrator must enable web search in Console
- Valid API key with web search permissions
- Proper citation display in user interface
- Handling of encrypted content and indices for multi-turn conversations 
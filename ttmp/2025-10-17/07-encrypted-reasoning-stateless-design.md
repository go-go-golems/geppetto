# Encrypted Reasoning Blocks: Stateless Design for Responses API

## Overview

OpenAI's Responses API supports **encrypted reasoning content** as a way to maintain reasoning continuity in fully stateless API calls. Reasoning items are actual output items (like messages and function calls) that can be passed back in subsequent requests.

This document focuses on implementing encrypted reasoning as a new **Block type** in Geppetto's Turn-based architecture. This provides a simple, stateless approach that doesn't require conversation_id or previous_response_id tracking.

**Key benefit**: Fully stateless operation while maintaining reasoning continuity - no server-side state, no response boundary tracking needed.

**Key References**:
- [OpenAI Cookbook: Reasoning Items](https://cookbook.openai.com/examples/responses_api/reasoning_items)
- [Azure OpenAI Responses API](https://learn.microsoft.com/en-us/azure/ai-foundry/openai/how-to/responses)
- [OpenAI Community Discussion](https://community.openai.com/t/using-reasoning-encrypted-content-with-background-mode/1297811)

---

## Table of Contents

1. [What is Encrypted Reasoning Content?](#what-is-encrypted-reasoning-content)
2. [Why Reasoning Blocks (Not Metadata)?](#why-reasoning-blocks-not-metadata)
3. [Architecture: Reasoning as Blocks](#architecture-reasoning-as-blocks)
4. [Schema Changes](#schema-changes)
5. [Response Parsing](#response-parsing)
6. [Request Building](#request-building)
7. [Complete Flow Example](#complete-flow-example)
8. [Lifecycle Management](#lifecycle-management)
9. [Security and Privacy Considerations](#security-and-privacy-considerations)
10. [Testing Strategy](#testing-strategy)
11. [Implementation Phases](#implementation-phases)
12. [Future: Combining with State Management](#future-combining-with-state-management)

---

## What is Encrypted Reasoning Content?

### Concept

The Responses API can emit an **opaque encrypted blob** representing the model's internal reasoning trace. This blob:
- Cannot be read, modified, or interpreted by the client
- Can be passed back in subsequent requests to restore reasoning continuity
- Acts as a "state capsule" for the model's thought process
- Enables stateless API calls while maintaining reasoning context

### API Interaction

**Request** (initial):
```json
{
  "model": "o4-mini",
  "input": [...],
  "reasoning": {
    "effort": "high"
  },
  "include": ["reasoning.encrypted_content"]
}
```

**Response** (includes blob):
```json
{
  "id": "resp_abc123",
  "output": [...],
  "reasoning": {
    "encrypted_content": "BASE64_ENCODED_BLOB_HERE..."
  },
  "usage": {...}
}
```

**Request** (continuation):
```json
{
  "model": "o4-mini",
  "input": [{"role": "user", "content": [...]}],
  "reasoning": {
    "effort": "high",
    "encrypted_content": "BASE64_ENCODED_BLOB_HERE..."
  }
}
```

### Properties

- **Opaque**: Client cannot decrypt or inspect
- **Immutable**: Client cannot modify (will invalidate signature)
- **Provider-specific**: Format tied to OpenAI, not portable
- **Ephemeral**: May have expiration (TTL unknown, needs testing)
- **Optional**: Model works without it, but loses reasoning continuity

---

## Why Reasoning Blocks (Not Metadata)?

According to the [OpenAI Cookbook](https://cookbook.openai.com/examples/responses_api/reasoning_items?utm_source=chatgpt.com#encrypted-reasoning-items), reasoning items are **output items** in the response, just like messages and function calls. They are NOT metadata or configuration.

From the API response structure:
```json
{
  "output": [
    {
      "id": "rs_6821...",
      "type": "reasoning",
      "summary": [],
      "encrypted_content": "gAAAAABoISQ24OyVRYbkYfuk..."
    },
    {
      "id": "msg_6820...",
      "type": "message",
      "content": [{"type": "output_text", "text": "..."}]
    }
  ]
}
```

**Key insight**: Reasoning items are peers to messages and function calls, so they should be represented as Blocks in Geppetto's Turn model.

---

## Architecture: Reasoning as Blocks

### High-Level Flow (Stateless)

```
┌─────────────────────────────────────────────────────────────┐
│ Turn Before Request                                         │
│  [user, reasoning(encrypted), tool_call, tool_use, user]    │
│                                                             │
│  1. Convert ALL blocks to input items                       │
│     - User blocks → input_text                              │
│     - Reasoning blocks → reasoning items (with encrypted)   │
│     - Tool blocks → function_call / tool_result             │
│                                                             │
└─────────────────┬───────────────────────────────────────────┘
                  │
                  ▼
┌─────────────────────────────────────────────────────────────┐
│ Responses API Request (Stateless)                           │
│  {                                                          │
│    "model": "o4-mini",                                      │
│    "input": [                                               │
│      {"role": "user", "content": [...]},                    │
│      {"type": "reasoning", "id": "rs_...", "encrypted_..."},│
│      {"type": "function_call", ...},                        │
│      {"type": "function_call_output", ...},                 │
│      {"role": "user", "content": [...]}                     │
│    ],                                                       │
│    "include": ["reasoning.encrypted_content"]               │
│  }                                                          │
└─────────────────┬───────────────────────────────────────────┘
                  │
                  ▼
┌─────────────────────────────────────────────────────────────┐
│ Responses API Response                                      │
│  {                                                          │
│    "output": [                                              │
│      {"type": "reasoning", "encrypted_content": "..."},     │
│      {"type": "message", "content": [...]}                  │
│    ]                                                        │
│  }                                                          │
└─────────────────┬───────────────────────────────────────────┘
                  │
                  ▼
┌─────────────────────────────────────────────────────────────┐
│ Turn After Response                                         │
│  [user, OLD_reasoning, tool_call, tool_use, user,           │
│   NEW_reasoning, assistant]                                 │
│                                                             │
│  - NEW reasoning block appended with new encrypted content  │
│  - Next request will include ALL blocks (including NEW)     │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

**Stateless property**: No server-side state. Every request includes full Turn. Reasoning continuity maintained via encrypted blocks.

---

## Schema Changes

### New BlockKind

```go
// In turns/types.go - add new BlockKind
const (
    BlockKindUser BlockKind = iota
    BlockKindLLMText
    BlockKindToolCall
    BlockKindToolUse
    BlockKindSystem
    BlockKindReasoning  // NEW: for reasoning items
    BlockKindOther
)

// In turns/keys.go - add payload key for encrypted content
const (
    PayloadKeyText   = "text"
    PayloadKeyID     = "id"
    PayloadKeyName   = "name"
    PayloadKeyArgs   = "args"
    PayloadKeyResult = "result"
    PayloadKeyEncryptedContent = "encrypted_content"  // NEW
)
```

### Example Response Structure

From the API (as shown in cookbook):
```json
{
  "output": [
    {
      "id": "rs_6821243503d481919e1b385c2a154d5103d2cbc5a14f3696",
      "type": "reasoning",
      "summary": [],
      "encrypted_content": "gAAAAABoISQ24OyVRYbkYfuk..."
    },
    {
      "id": "fc_68210c78357c8191977197499d5de6ca00c77cc15fd2f785",
      "type": "function_call",
      "name": "get_weather",
      "arguments": "{...}"
    }
  ]
}
```

### Turn Representation

```go
// After receiving response with encrypted reasoning
t.Blocks = [
    Block{
        Kind: BlockKindUser,
        Payload: map[string]any{
            PayloadKeyText: "What's the weather in Paris?",
        },
    },
    Block{
        Kind: BlockKindReasoning,  // NEW
        ID: "rs_6821...",
        Payload: map[string]any{
            PayloadKeyEncryptedContent: "gAAAAABoISQ24OyVRYbkYfuk...",
        },
    },
    Block{
        Kind: BlockKindToolCall,
        ID: "fc_68210...",
        Payload: map[string]any{
            PayloadKeyName: "get_weather",
            PayloadKeyArgs: {...},
        },
    },
]
```

### Engine Integration

```go
// In openai_responses/engine.go
func (e *Engine) RunInference(ctx context.Context, t *turns.Turn) (*turns.Turn, error) {
    // Build request - include ALL blocks (reasoning blocks are part of input)
    req := buildResponsesRequest(s, t)
    
    // Send request
    resp := sendRequest(req)
    
    // Parse response output items into blocks
    for _, item := range resp.Output {
        switch item.Type {
        case "reasoning":
            // Append reasoning block
            block := turns.Block{
                Kind: turns.BlockKindReasoning,
                ID: item.ID,
                Payload: map[string]any{},
            }
            if item.EncryptedContent != "" {
                block.Payload[turns.PayloadKeyEncryptedContent] = item.EncryptedContent
            }
            turns.AppendBlock(t, block)
            
        case "message":
            // Append message block (existing logic)
            
        case "function_call":
            // Append tool call block (existing logic)
        }
    }
    
    return t, nil
}
```

### Request Building

```go
// In openai_responses/helpers.go
func buildInputItemsFromTurn(t *turns.Turn) []responsesInput {
    var items []responsesInput
    
    for _, b := range t.Blocks {
        switch b.Kind {
        case turns.BlockKindUser:
            // User text input
            items = append(items, responsesInput{
                Role: "user",
                Content: []responsesContentPart{
                    {Type: "input_text", Text: b.Payload[turns.PayloadKeyText].(string)},
                },
            })
            
        case turns.BlockKindReasoning:
            // Reasoning item - pass back as-is
            reasoningItem := responsesInput{
                Type: "reasoning",
                ID: b.ID,
            }
            if encContent, ok := b.Payload[turns.PayloadKeyEncryptedContent].(string); ok {
                reasoningItem.EncryptedContent = encContent
            }
            items = append(items, reasoningItem)
            
        case turns.BlockKindToolCall:
            // Function call (existing logic)
            
        case turns.BlockKindToolUse:
            // Tool result (existing logic)
        }
    }
    
    return items
}
```

### Key Insights from Cookbook

According to the [OpenAI Cookbook](https://cookbook.openai.com/examples/responses_api/reasoning_items?utm_source=chatgpt.com#encrypted-reasoning-items):

1. **Reasoning items are output items**: They appear in `response.output[]` alongside messages and function calls
2. **They have IDs**: Each reasoning item has a unique ID (e.g., `rs_6821...`)
3. **They can be included in input**: You pass reasoning items back in subsequent requests by including them in the `input` array
4. **Encrypted content is a field**: `encrypted_content` is a field on the reasoning item, not a separate parameter

Example from cookbook:
```python
context += response.output  # Add entire output (including reasoning items)
response_2 = client.responses.create(
    model="o3",
    input=context,  # Reasoning items are now in the input
    tools=tools,
)
```

### Pros

- **Correct model**: Matches API structure exactly
- **Natural flow**: Reasoning items flow through Turn like any other block
- **Middleware compatible**: Middleware can inspect/log reasoning blocks
- **Automatic inclusion**: Reasoning blocks are automatically included when building requests from Turn
- **Type-safe**: BlockKind enum ensures type safety
- **Response boundary tracking works**: Reasoning blocks get tagged with response_id like other blocks

### Cons

- Requires new BlockKind (schema change to turns package)
- Reasoning blocks increase Turn size (but so do tool calls/results)
- Need to handle reasoning blocks in input conversion logic

###Benefits of This Approach

1. **Matches API structure exactly**: Reasoning items are output items per [OpenAI Cookbook](https://cookbook.openai.com/examples/responses_api/reasoning_items?utm_source=chatgpt.com#encrypted-reasoning-items)
2. **Natural model**: Blocks represent conversation items; reasoning is a conversation item
3. **Automatic handling**: Reasoning blocks are automatically included when converting Turn to API input
4. **Middleware compatible**: Middleware can see, log, or filter reasoning blocks like any other block
5. **Simple**: No separate state management needed
6. **Consistent**: Tool calls and tool results are blocks; reasoning items follow same pattern

---

## Response Parsing

### How to Parse Reasoning Items from Response

```go
// In openai_responses/engine.go
func (e *Engine) RunInference(ctx context.Context, t *turns.Turn) (*turns.Turn, error) {
    // Build request from all blocks (including reasoning blocks)
    req := buildResponsesRequest(s, t)
    
    // Send request
    resp := sendRequest(req)
    
    // Parse response output items into blocks
    for _, item := range resp.Output {
        switch item.Type {
        case "reasoning":
            // Create reasoning block
            block := turns.Block{
                Kind: turns.BlockKindReasoning,
                ID: item.ID,
                Payload: map[string]any{},
            }
            
            // Store encrypted content if present
            if item.EncryptedContent != "" {
                block.Payload[turns.PayloadKeyEncryptedContent] = item.EncryptedContent
            }
            
            // Store summary if present (optional)
            if len(item.Summary) > 0 {
                block.Payload[turns.PayloadKeySummary] = item.Summary
            }
            
            turns.AppendBlock(t, block)
            
        case "message":
            // Parse message content (existing logic)
            for _, content := range item.Content {
                if content.Type == "output_text" {
                    turns.AppendBlock(t, turns.NewAssistantTextBlock(content.Text))
                }
            }
            
        case "function_call":
            // Parse tool call (existing logic)
            args := parseJSON(item.Arguments)
            turns.AppendBlock(t, turns.NewToolCallBlock(item.CallID, item.Name, args))
        }
    }
    
    return t, nil
}
```

**Key points**:
- Each reasoning item becomes one block
- `encrypted_content` stored in block payload
- Summary (if requested) also stored in payload
- Blocks flow naturally through Turn

---

## Lifecycle Management

### Blob Creation

**When**: First request with reasoning-capable model

**How**:
1. Include `"include": ["reasoning.encrypted_content"]` in request
2. Receive blob in response
3. Store in ConversationState

**Initial value**: Empty string (no blob yet)

---

### Blob Propagation

**When**: Every subsequent request in the same conversation

**How**:
1. Read blob from ConversationState
2. Include in `reasoning.encrypted_content` field
3. Receive NEW blob in response
4. Replace old blob with new one

**Important**: Always use the LATEST blob. Old blobs are stale.

---

### Blob Expiration

**Unknown**: API docs don't specify TTL

**Possibilities**:
1. Blobs expire after N hours/days
2. Blobs expire after N turns
3. Blobs never expire (tied to model version)

**Mitigation**:
- Track blob age (timestamp when received)
- If request fails with blob error, discard blob and retry without it
- Log warnings for old blobs (>24 hours?)

---

### Blob Invalidation

**Triggers**:
1. User starts new conversation (explicit reset)
2. Model version changes
3. Blob corruption detected by server
4. TTL expiration (if exists)

**Handling**:
```go
// Invalidate blob
state.EncryptedReasoning = ""
SetState(t, state)

// Next request will create new blob
```

---

### Blob Serialization

**For persistence** (saving Turn to disk/DB):

```go
// ConversationState is already in Turn.Data
// Serializing Turn automatically includes blob

// Example: JSON
turnJSON := json.Marshal(t)
// t.Data["openai_responses_state"].EncryptedReasoning is preserved

// Example: Resume from disk
t := loadTurnFromDisk()
state := GetState(t)
// state.EncryptedReasoning is restored
```

**Considerations**:
- Blob might be large (10KB-100KB?)
- Encrypted, so safe to store (no sensitive data exposure)
- But ephemeral: may expire, so don't rely on long-term storage

---

## Security and Privacy Considerations

### Client-Side Security

**What client can do**:
- Store blob (encrypted, no plaintext access)
- Pass blob back to API
- Discard blob

**What client CANNOT do**:
- Decrypt or read blob contents
- Modify blob (will invalidate signature)
- Extract reasoning traces

**Implication**: Safe to store client-side (localStorage, cookies, DB)

---

### Server-Side Security

**What server protects**:
- Reasoning trace content (encrypted in blob)
- Model internal state (not exposed)

**What server allows**:
- Client to resume reasoning continuity
- Client to drop blob (loses continuity but no security risk)

**Implication**: Server trusts client to manage blob lifecycle, but doesn't trust blob contents (verifies signature)

---

### Privacy Implications

**For zero-data-retention deployments**:
- Server doesn't store conversation history
- Blob is only reasoning state mechanism
- Client must manage blob or lose continuity

**For audit/compliance**:
- Blob contents are opaque (can't audit reasoning)
- If compliance requires reasoning visibility, use `reasoning.summary` instead
- Encrypted content is not suitable for explainability

---

### Tampering Resistance

**Attack**: Client modifies blob to inject malicious reasoning

**Defense**: Server validates blob signature
- Invalid signature → blob rejected
- Request continues without reasoning continuity
- No security vulnerability, just degraded experience

**Recommendation**: Don't attempt to modify blob (will fail)

---

## Testing Strategy

### Unit Tests

#### Test: Blob Extraction from Response

**Scenario**: Response includes encrypted reasoning

**Input**: Mock API response with `reasoning.encrypted_content`

**Expected**:
- Blob extracted and stored in ConversationState
- Blob is non-empty string

---

#### Test: Blob Inclusion in Request

**Scenario**: ConversationState has blob

**Input**: ConversationState with `EncryptedReasoning = "test_blob"`

**Expected**:
- Request includes `reasoning.encrypted_content: "test_blob"`
- Request structure is valid

---

#### Test: Missing Blob Handling

**Scenario**: No blob in ConversationState

**Input**: ConversationState with empty `EncryptedReasoning`

**Expected**:
- Request omits `reasoning.encrypted_content` field
- Request still succeeds

---

#### Test: Blob Replacement

**Scenario**: Multiple requests with different blobs

**Input**: 
- Request 1: no blob → response has blob_A
- Request 2: blob_A → response has blob_B

**Expected**:
- After request 1: state.EncryptedReasoning = blob_A
- After request 2: state.EncryptedReasoning = blob_B (replaced)

---

### Integration Tests

#### Test: Reasoning Continuity

**Scenario**: Multi-turn conversation with encrypted reasoning

**Flow**:
1. Turn 1: "Explain quantum entanglement" → blob_A
2. Turn 2: "Continue with examples" + blob_A → blob_B
3. Turn 3: "Summarize" + blob_B → final output

**Assertions**:
- Model's response in Turn 2 builds on Turn 1 reasoning
- Model's response in Turn 3 references previous context
- Blobs are different at each turn

**Comparison**:
- Run same conversation WITHOUT blobs
- Verify reasoning continuity is BETTER with blobs

---

#### Test: Blob Expiration/Invalidation

**Scenario**: Use old blob after expiration

**Flow**:
1. Get blob from response
2. Wait N hours (or simulate timestamp change)
3. Use blob in new request

**Expected**:
- API rejects blob (400 error?) OR accepts but ignores
- Need to test with real API to determine behavior

---

### Real API Tests

#### Test: Request with `include` Parameter

**Purpose**: Verify API returns encrypted content

**Request**:
```json
{
  "model": "o4-mini",
  "input": [{"role": "user", "content": [{"type": "input_text", "text": "Test"}]}],
  "reasoning": {"effort": "medium"},
  "include": ["reasoning.encrypted_content"]
}
```

**Expected**:
- Response includes `reasoning.encrypted_content` field
- Field is non-empty BASE64 string

---

#### Test: Blob Reuse

**Purpose**: Verify blob can be passed back

**Request**:
```json
{
  "model": "o4-mini",
  "input": [{"role": "user", "content": [{"type": "input_text", "text": "Continue"}]}],
  "reasoning": {
    "effort": "medium",
    "encrypted_content": "BLOB_FROM_PREVIOUS_RESPONSE"
  }
}
```

**Expected**:
- Request succeeds (200 OK)
- Response shows reasoning continuity

---

## Complete Flow Example

### Scenario: Weather Query with Tool Calling

```
Turn 1: User asks "What's the weather in Paris?"

Blocks before request:
  [user: "What's the weather in Paris?"]

Request:
  input: [
    {role: "user", content: [{type: "input_text", text: "What's..."}]}
  ]
  include: ["reasoning.encrypted_content"]
  tools: [{name: "get_weather", ...}]

Response:
  output: [
    {type: "reasoning", id: "rs_123", encrypted_content: "BLOB_A"},
    {type: "function_call", call_id: "call_456", name: "get_weather", arguments: "{...}"}
  ]

Blocks after response:
  [user: "What's...",
   reasoning(id=rs_123, encrypted=BLOB_A),
   tool_call(id=call_456, name=get_weather)]

---

Turn 2: Middleware executes tool, appends result

Blocks before request (middleware added tool_use):
  [user: "What's...",
   reasoning(encrypted=BLOB_A),
   tool_call(name=get_weather),
   tool_use(result="22°C")]

Request (includes ALL blocks):
  input: [
    {role: "user", content: [{type: "input_text", text: "What's..."}]},
    {type: "reasoning", id: "rs_123", encrypted_content: "BLOB_A"},
    {type: "function_call", call_id: "call_456", name: "get_weather", arguments: "{...}"},
    {type: "function_call_output", call_id: "call_456", output: "22°C"}
  ]
  include: ["reasoning.encrypted_content"]

Response:
  output: [
    {type: "reasoning", id: "rs_789", encrypted_content: "BLOB_B"},
    {type: "message", content: [{type: "output_text", text: "It's 22°C in Paris."}]}
  ]

Blocks after response:
  [user: "What's...",
   reasoning(encrypted=BLOB_A),
   tool_call(...),
   tool_use(result="22°C"),
   reasoning(encrypted=BLOB_B),
   assistant: "It's 22°C in Paris."]
```

**Key observations**:
- Every request includes full Turn history
- Reasoning blocks (with encrypted content) are passed through automatically
- New reasoning block added with each response
- Middleware can modify Turn freely - no coherence issues

---

---
## Related Documents

- [OpenAI Cookbook: Reasoning Items](https://cookbook.openai.com/examples/responses_api/reasoning_items)
- [Azure OpenAI Responses API](https://learn.microsoft.com/en-us/azure/ai-foundry/openai/how-to/responses)


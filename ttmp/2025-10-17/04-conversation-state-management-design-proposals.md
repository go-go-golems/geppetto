# OpenAI Responses API: Conversation State Management Design Proposals

## Overview

The OpenAI Responses API (https://platform.openai.com/docs/guides/conversation-state?api-mode=responses) introduces advanced conversation state management capabilities beyond the traditional "send all history" approach:

1. **Server-side conversations**: Create a conversation object on OpenAI's side, get a `conversation_id`, and subsequent requests only need to reference that ID
2. **Response chaining**: Use `previous_response_id` to chain responses without re-sending full history
3. **Traditional mode**: Continue sending full `input` array (current implementation)

This document explores design approaches to integrate these capabilities into Geppetto's Turn-based architecture while maintaining:
- Genericity of `RunInference(ctx, *Turn) (*Turn, error)`
- Provider agnosticism where possible
- Flexibility to use different modes per use case

---

## Current State (Baseline)

```go
// Current signature (all providers)
type Engine interface {
    RunInference(ctx context.Context, t *turns.Turn) (*turns.Turn, error)
}

// Turn carries all conversation blocks
type Turn struct {
    ID     uuid.UUID
    RunID  uuid.UUID
    Blocks []Block  // system, user, assistant, tool_call, tool_use
    Data   map[string]any  // registry, config, etc.
}
```

**Current Responses implementation**: Converts all `Turn.Blocks` to `input` array on every call. No conversation/response chaining.

---

## Design Constraints

1. **Keep `RunInference` signature unchanged**: Core contract must remain provider-agnostic
2. **Per-Turn configuration**: State management strategy should be configurable per Turn (via `Turn.Data`)
3. **Backward compatibility**: Existing engines (Chat Completions, Claude, Gemini) unaffected
4. **Explicit over implicit**: Caller controls conversation lifecycle, engine doesn't manage global state
5. **Streaming-aware**: Must work with Watermill event sinks and SSE

---

## Proposal 1: Turn.Data Keys (Minimal, Explicit)

### Concept
Store conversation/response IDs in `Turn.Data` as opaque metadata. Engine reads them if present, writes them back after inference.

### API Sketch

```go
// New keys in turns package
const (
    DataKeyConversationID  = "openai_conversation_id"  // string
    DataKeyPreviousResponseID = "openai_previous_response_id"  // string
    DataKeyConversationMode = "openai_conversation_mode"  // string: "stateless" | "conversation" | "chained"
)

// Usage (caller side)
t := &turns.Turn{
    Data: map[string]any{
        turns.DataKeyConversationMode: "chained",
        turns.DataKeyPreviousResponseID: "resp_abc123",
    },
}
t, err := engine.RunInference(ctx, t)
// After call: t.Data[DataKeyPreviousResponseID] updated to new response_id

// Engine side (openai_responses/engine.go)
func (e *Engine) RunInference(ctx context.Context, t *turns.Turn) (*turns.Turn, error) {
    mode := getMode(t)  // reads DataKeyConversationMode, defaults to "stateless"
    
    switch mode {
    case "conversation":
        convID := getConversationID(t)
        if convID == "" {
            // Create new conversation via POST /conversations
            convID = createConversation(ctx, e.settings)
            t.Data[DataKeyConversationID] = convID
        }
        // POST /conversations/{convID}/responses with minimal input
        resp := sendToConversation(ctx, convID, ...)
        
    case "chained":
        prevRespID := getPreviousResponseID(t)
        // POST /responses with previous_response_id field
        resp := sendChainedResponse(ctx, prevRespID, ...)
        t.Data[DataKeyPreviousResponseID] = resp.ID
        
    default: // "stateless"
        // Current behavior: full input array
        resp := sendFullInput(ctx, buildInputItemsFromTurn(t))
    }
    
    // Common: append assistant blocks, publish events
    return t, nil
}
```

### Pros
- Minimal API surface: no new interfaces, just data keys
- Caller has full control: pick mode per Turn
- Easy to test: just populate `Turn.Data`
- No global state in engine

### Cons
- String-typed keys: typos possible, no compile-time safety
- Mode logic specific to OpenAI Responses (leaks provider details into Turn.Data)
- Caller must understand OpenAI-specific IDs

### Variants
- Add typed accessors: `turns.SetConversationID(t, id)`, `turns.GetConversationMode(t)`
- Namespace keys: `"provider.openai.conversation_id"` for multi-provider collision avoidance

---

## Proposal 2: Provider-Specific Options (Type-Safe)

### Concept
Introduce a typed settings struct for OpenAI Responses state, store in `Turn.Data` with a known key.

### API Sketch

```go
// In openai_responses package
type ConversationState struct {
    Mode              string  // "stateless" | "conversation" | "chained"
    ConversationID    string
    PreviousResponseID string
    // Future: encrypted reasoning tokens, conversation metadata
}

const DataKeyResponsesState = "openai_responses_state"

// Usage (caller side)
state := &openai_responses.ConversationState{
    Mode: "chained",
    PreviousResponseID: "resp_xyz",
}
t := &turns.Turn{
    Data: map[string]any{
        openai_responses.DataKeyResponsesState: state,
    },
}
t, err := engine.RunInference(ctx, t)
// After call: state.PreviousResponseID updated

// Engine side
func (e *Engine) RunInference(ctx context.Context, t *turns.Turn) (*turns.Turn, error) {
    state := getOrCreateState(t)  // returns *ConversationState
    
    switch state.Mode {
    case "conversation":
        if state.ConversationID == "" {
            state.ConversationID = e.createConversation(ctx)
        }
        resp := e.sendToConversation(ctx, state, t)
        
    case "chained":
        resp := e.sendChainedResponse(ctx, state, t)
        state.PreviousResponseID = resp.ID
        
    default:
        resp := e.sendFullInput(ctx, t)
    }
    
    t.Data[DataKeyResponsesState] = state  // write back
    return t, nil
}
```

### Pros
- Type-safe: `ConversationState` is a proper struct
- Self-documenting: fields and modes explicit
- Extensible: add fields (encrypted reasoning, conversation metadata) without new keys
- Provider-scoped: lives in `openai_responses` package

### Cons
- Still mixes provider-specific logic into Turn.Data
- Caller must import `openai_responses` package to construct state
- Not fully provider-agnostic (but that's okay for advanced features)

### Variants
- Make `ConversationState` an interface; other providers can have their own state types
- Add a `StateManager` helper to abstract creation/retrieval from Turn.Data

---

## Proposal 3: Context-Carried State (Implicit)

### Concept
Use `context.Context` to carry conversation state, similar to how we carry event sinks.

### API Sketch

```go
// In openai_responses package
type contextKey string

const conversationStateKey contextKey = "openai_responses_conversation_state"

func WithConversationState(ctx context.Context, state *ConversationState) context.Context {
    return context.WithValue(ctx, conversationStateKey, state)
}

func GetConversationState(ctx context.Context) *ConversationState {
    if v := ctx.Value(conversationStateKey); v != nil {
        return v.(*ConversationState)
    }
    return nil
}

// Usage (caller side)
state := &openai_responses.ConversationState{Mode: "chained"}
ctx = openai_responses.WithConversationState(ctx, state)
t, err := engine.RunInference(ctx, t)
// After call: read updated state from context (requires passing back via context somehow)

// Engine side
func (e *Engine) RunInference(ctx context.Context, t *turns.Turn) (*turns.Turn, error) {
    state := GetConversationState(ctx)
    if state == nil {
        state = &ConversationState{Mode: "stateless"}
    }
    
    // Use state as in Proposal 2
    switch state.Mode { ... }
    
    // Problem: how to write back updated state? Context is immutable.
    // Option A: Mutate state in place (caller holds pointer)
    // Option B: Return state somehow (breaks signature)
    return t, nil
}
```

### Pros
- Follows Geppetto's pattern (event sinks use context)
- Clean separation: state isn't mixed with Turn business logic
- Per-request scoping via context propagation

### Cons
- **Context mutation issue**: Can't return updated state without breaking signature or mutating (anti-pattern)
- Less discoverable: state is "invisible" in function signatures
- Harder to test: must construct context correctly
- Not idiomatic for bidirectional data (context is for request-scoped read-only data)

### Verdict
Not recommended for mutable conversation state. Better for read-only config.

---

## Proposal 4: Middleware Layer (Transparent)

### Concept
Introduce a conversation state management middleware that wraps the base Responses engine, handling ID tracking transparently.

### API Sketch

```go
// In middleware package (or openai_responses package)
type ConversationMiddleware struct {
    base   engine.Engine
    mode   string  // "stateless" | "conversation" | "chained"
    convID string
    lastResponseID string
    mu     sync.Mutex  // if shared across goroutines
}

func NewConversationMiddleware(base engine.Engine, mode string) *ConversationMiddleware {
    return &ConversationMiddleware{base: base, mode: mode}
}

func (m *ConversationMiddleware) RunInference(ctx context.Context, t *turns.Turn) (*turns.Turn, error) {
    m.mu.Lock()
    // Inject state into Turn.Data before calling base engine
    switch m.mode {
    case "conversation":
        t.Data[DataKeyConversationID] = m.convID
    case "chained":
        t.Data[DataKeyPreviousResponseID] = m.lastResponseID
    }
    m.mu.Unlock()
    
    // Call base engine (which reads from Turn.Data)
    t, err := m.base.RunInference(ctx, t)
    if err != nil { return nil, err }
    
    m.mu.Lock()
    // Extract updated state from Turn.Data
    if id, ok := t.Data[DataKeyConversationID].(string); ok {
        m.convID = id
    }
    if id, ok := t.Data[DataKeyPreviousResponseID].(string); ok {
        m.lastResponseID = id
    }
    m.mu.Unlock()
    
    return t, nil
}

// Usage (caller side)
baseEngine := openai_responses.NewEngine(settings)
// Wrap for conversation mode
engine := middleware.NewConversationMiddleware(baseEngine, "chained")

// Use normally
t1, _ := engine.RunInference(ctx, turn1)  // creates chain
t2, _ := engine.RunInference(ctx, turn2)  // continues chain automatically
```

### Pros
- Transparent to caller: no Turn.Data manipulation needed
- Stateful when needed: middleware holds conversation ID across calls
- Composable: can wrap other middleware (logging, retries, etc.)
- Clean separation: base engine stays stateless, middleware adds state

### Cons
- Middleware instance is stateful: not safe for concurrent use without locking
- Less explicit: caller doesn't see IDs in Turn.Data
- Harder to serialize state: middleware holds it in memory
- Lifecycle management: who owns the middleware instance?

### Variants
- Make middleware thread-safe with `sync.Mutex`
- Add `GetState()` method to expose IDs for serialization
- Combine with Proposal 2: middleware reads/writes typed state struct

---

## Proposal 5: Conversation Manager (High-Level Abstraction)

### Concept
Introduce a `ConversationManager` that owns the engine and manages state lifecycle, similar to `conversation.Manager` for messages.

### API Sketch

```go
// In openai_responses package (or new package)
type ConversationManager struct {
    engine       *Engine
    mode         string
    conversationID string
    lastResponseID string
    history      []*turns.Turn  // optional: keep Turn history
}

func NewConversationManager(engine *Engine, mode string) *ConversationManager {
    return &ConversationManager{engine: engine, mode: mode}
}

func (m *ConversationManager) RunTurn(ctx context.Context, t *turns.Turn) (*turns.Turn, error) {
    // Inject state based on mode
    if m.mode == "conversation" && m.conversationID != "" {
        t.Data[DataKeyConversationID] = m.conversationID
    }
    if m.mode == "chained" && m.lastResponseID != "" {
        t.Data[DataKeyPreviousResponseID] = m.lastResponseID
    }
    
    result, err := m.engine.RunInference(ctx, t)
    if err != nil { return nil, err }
    
    // Update state
    if id, ok := result.Data[DataKeyConversationID].(string); ok {
        m.conversationID = id
    }
    if id, ok := result.Data[DataKeyPreviousResponseID].(string); ok {
        m.lastResponseID = id
    }
    
    m.history = append(m.history, result)
    return result, nil
}

func (m *ConversationManager) GetConversationID() string { return m.conversationID }
func (m *ConversationManager) GetHistory() []*turns.Turn { return m.history }

// Usage (caller side)
engine := openai_responses.NewEngine(settings)
mgr := openai_responses.NewConversationManager(engine, "chained")

t1, _ := mgr.RunTurn(ctx, turn1)
t2, _ := mgr.RunTurn(ctx, turn2)  // automatically chains

// Access state
conversationID := mgr.GetConversationID()
```

### Pros
- High-level: abstracts all state management
- Natural for multi-turn workflows: one manager per conversation session
- Testable: manager is a concrete type with methods
- History tracking: can optionally keep full Turn history
- Serializable: export/import manager state for persistence

### Cons
- Another layer: caller must choose between `Engine` and `Manager`
- Not as generic: specific to OpenAI Responses (but could be provider-agnostic interface)
- Lifecycle ownership: who creates/destroys managers?
- Doesn't fit single-shot inference patterns

### Variants
- Make `ConversationManager` an interface; different providers implement it
- Add `Save()/Load()` methods for state persistence
- Integrate with existing `conversation.Manager` (message manager) for unified API

---

## Proposal 6: Hybrid (Turn.Data + Accessors)

### Concept
Combine Proposal 1 and 2: use `Turn.Data` for storage, provide typed helper functions for access.

### API Sketch

```go
// In turns package (provider-agnostic location)
const (
    DataKeyProviderState = "provider_state"  // stores provider-specific state
)

// In openai_responses package
type ConversationState struct {
    Mode              string
    ConversationID    string
    PreviousResponseID string
}

// Typed accessors (in openai_responses)
func GetState(t *turns.Turn) *ConversationState {
    if v, ok := t.Data[turns.DataKeyProviderState]; ok {
        if state, ok := v.(*ConversationState); ok {
            return state
        }
    }
    return &ConversationState{Mode: "stateless"}
}

func SetState(t *turns.Turn, state *ConversationState) {
    t.Data[turns.DataKeyProviderState] = state
}

// Usage (caller side)
state := &openai_responses.ConversationState{Mode: "chained", PreviousResponseID: "resp_123"}
openai_responses.SetState(t, state)
t, err := engine.RunInference(ctx, t)
updatedState := openai_responses.GetState(t)

// Engine side
func (e *Engine) RunInference(ctx context.Context, t *turns.Turn) (*turns.Turn, error) {
    state := GetState(t)
    // use state...
    SetState(t, state)  // write back
    return t, nil
}
```

### Pros
- Type-safe accessors hide map manipulation
- Generic key in `turns` package (`DataKeyProviderState`) for any provider
- Provider-specific state lives in provider package
- Explicit: caller controls state, visible in Turn.Data
- Easy to serialize: Turn.Data is already serializable

### Cons
- Generic key means providers could collide (both trying to use `DataKeyProviderState`)
- Slightly more boilerplate (accessors)

### Variants
- Use namespaced keys: `"openai.conversation_state"`, `"claude.conversation_state"`
- Add validation in accessors (e.g., check mode enum values)

---

## Comparison Matrix

| Proposal | Explicitness | Type Safety | Genericity | Ease of Use | State Ownership | Serialization |
|----------|-------------|-------------|-----------|-------------|----------------|---------------|
| 1. Turn.Data Keys | High | Low (strings) | High | Medium | Caller | Easy (in Turn) |
| 2. Typed Options | High | High | Medium | Medium | Caller | Easy (in Turn) |
| 3. Context State | Low | High | High | Low | Context | Hard |
| 4. Middleware | Low | High | Medium | High | Middleware | Medium |
| 5. Manager | Medium | High | Low | High | Manager | Easy (Manager API) |
| 6. Hybrid | High | High | High | High | Caller | Easy (in Turn) |

---

## Recommendations

### Short-term (MVP for conversation state)
**Proposal 6: Hybrid (Turn.Data + Accessors)**

Reasoning:
- Balances explicitness and type safety
- Minimal changes: engine reads/writes Turn.Data, caller controls state
- Easy to test and serialize
- Doesn't break existing API
- Provider-specific but doesn't pollute core `turns` package

Implementation sketch:
```go
// 1. Add DataKeyProviderState to turns/keys.go
// 2. Define ConversationState in openai_responses/state.go
// 3. Add GetState/SetState accessors in openai_responses/state.go
// 4. Update engine.go to read/write state
// 5. Caller opts in by calling SetState before RunInference
```

### Mid-term (for complex workflows)
**Add Proposal 5: ConversationManager**

Reasoning:
- High-level abstraction for multi-turn conversations
- Complements low-level Turn.Data approach
- Users can choose: direct engine for one-shot, manager for sessions
- Can be built on top of Proposal 6 (manager internally uses GetState/SetState)

### Long-term (provider-agnostic)
**Standardize state interface across providers**

```go
// In turns or engine package
type ConversationState interface {
    Mode() string
    IsStateful() bool
    // Provider-specific methods accessed via type assertion
}

// Providers implement their own state types
type OpenAIResponsesState struct { ... }  // implements ConversationState
type ClaudeState struct { ... }           // implements ConversationState (if Claude adds similar features)
```

---

## Open Questions

1. **Conversation lifecycle**: Who creates/destroys conversations? Should there be explicit `CreateConversation()` / `DeleteConversation()` calls?

2. **Error recovery**: If a chained request fails, should we fall back to full input? How to signal this?

3. **Concurrency**: If the same Turn is used across goroutines, how to handle state races? (Answer: don't share Turns; clone if needed)

4. **Provider detection**: Should the engine auto-detect if conversation mode is available, or require explicit opt-in?

5. **Testing**: How to mock server-side conversations? Provide a `MockConversationEngine` that simulates ID tracking?

6. **Observability**: Should we emit events when conversation IDs are created/updated? (e.g., `EventTypeInfo` with `conversation_created`)

7. **Backward compat**: What's the migration path for existing code? (Answer: default mode is "stateless", no change needed)

---

## Next Steps (Post-Design)

1. Review proposals with team/users
2. Pick initial implementation (recommend Proposal 6)
3. Update `openai_responses` engine to support modes
4. Add integration tests for each mode
5. Document in `geppetto/pkg/doc/topics/06-inference-engines.md`
6. Consider Proposal 5 (Manager) for v2 if demand exists

---

## References

- OpenAI Responses API Conversation State: https://platform.openai.com/docs/guides/conversation-state?api-mode=responses
- Geppetto Turn-based Architecture: `geppetto/pkg/doc/topics/06-inference-engines.md`
- Event Context Pattern: `geppetto/pkg/events/context.go` (how we carry sinks)
- Conversation Manager: `geppetto/pkg/conversation/manager.go` (message-level manager example)


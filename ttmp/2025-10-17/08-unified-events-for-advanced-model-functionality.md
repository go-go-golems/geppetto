([Home][1])([Microsoft Learn][2])([Microsoft Learn][2])([Microsoft Learn][2])([Microsoft Learn][2])([Microsoft Learn][2])([MDN Web Docs][3])([Microsoft Learn][2])([Microsoft Learn][2])([Claude Docs][4])([Home][1])([Claude Docs][5])([AWS Documentation][6])([Claude Docs][7])([Claude Docs][4])([Microsoft Learn][2])([Microsoft Learn][8])([Microsoft Learn][2])([Microsoft Learn][2])([Microsoft Learn][2])([Microsoft Learn][2])([Microsoft Learn][2])([Microsoft Learn][2])([Microsoft Learn][2])([Microsoft Learn][2])([Microsoft Learn][2])([Microsoft Learn][2])([Claude Docs][4])([Claude Docs][4])([Claude Docs][9])([Microsoft Learn][2])

[1]: https://docs.spring.io/spring-ai/docs/current/api/org/springframework/ai/anthropic/api/AnthropicApi.ContentBlock.html?utm_source=chatgpt.com "Record Class AnthropicApi.ContentBlock"
[2]: https://learn.microsoft.com/en-us/azure/ai-foundry/openai/reference-preview-latest "Azure OpenAI in Azure AI Foundry Models REST API v1 preview reference - Azure OpenAI | Microsoft Learn"
[3]: https://developer.mozilla.org/en-US/docs/Web/API/Server-sent_events/Using_server-sent_events?utm_source=chatgpt.com "Using server-sent events - Web APIs | MDN - Mozilla"
[4]: https://docs.claude.com/en/docs/build-with-claude/citations?utm_source=chatgpt.com "Citations - Claude Docs"
[5]: https://docs.claude.com/en/docs/build-with-claude/extended-thinking?utm_source=chatgpt.com "Building with extended thinking"
[6]: https://docs.aws.amazon.com/bedrock/latest/userguide/model-parameters-anthropic-claude-messages-request-response.html?utm_source=chatgpt.com "Request and Response - Amazon Bedrock"
[7]: https://docs.claude.com/de/docs/agents-and-tools/tool-use/web-search-tool?utm_source=chatgpt.com "Web-Suche-Tool"
[8]: https://learn.microsoft.com/en-us/azure/ai-foundry/openai/how-to/responses "Azure OpenAI Responses API - Azure OpenAI | Microsoft Learn"
[9]: https://docs.claude.com/en/docs/build-with-claude/tool-use/web-search-tool?utm_source=chatgpt.com "Web search tool - Claude Docs"


---

// Package events defines a provider‑agnostic event model for
// streaming output and tool calls across multiple LLM providers,
// with first‑class coverage for Anthropic Messages streaming
// (incl. server tools, thinking & citations) and OpenAI Responses
// streaming (incl. built‑in tools like web/file search, interpreter,
// image generation, and MCP).
//
// Minimal surface, lossless via Extensions:
//  • Keep the number of event kinds small and stable
//  • Use enums (BlockType, DeltaKind, StatusPhase, BuiltInTool) to
//    model newly added tool/kind variants without changing routers
//  • Preserve provider details in Extensions / Raw
//
// Notation:
//  • “Block” == a content block (Anthropic content[i], OpenAI content part/item)
//  • “Tool call” == client/server/built‑in tool invocation
//
// NOTE: This file references package types defined elsewhere:
//  - EventType, EventImpl, EventMetadata, Usage (e.g., in chat-events.go, metadata.go)
//
package events

import (
    "encoding/json"
    "time"
)

// Provider identifies the upstream LLM provider that originated an event.
type Provider string

const (
    ProviderAnthropic Provider = "anthropic"
    ProviderOpenAI    Provider = "openai"
    // Add other providers here as needed.
)

// BlockType enumerates content block kinds seen across providers.
// (Anthropic content array; OpenAI content parts/items)
type BlockType string

const (
    // Core
    BlockText    BlockType = "text"
    BlockImage   BlockType = "image"

    // Thinking (Anthropic extended thinking)
    BlockThinking         BlockType = "thinking"
    BlockRedactedThinking BlockType = "redacted_thinking"

    // Client tools (Anthropic)
    BlockToolUse    BlockType = "tool_use"
    BlockToolResult BlockType = "tool_result"

    // Server tools (Anthropic)
    BlockServerToolUse    BlockType = "server_tool_use"
    BlockServerToolResult BlockType = "server_tool_result"
    // Web search specific (Anthropic server tool result payloads)
    BlockWebSearchToolResult      BlockType = "web_search_tool_result"
    BlockWebSearchResult          BlockType = "web_search_result"
    BlockWebSearchToolResultError BlockType = "web_search_tool_result_error"
    BlockWebSearchResultLocation  BlockType = "web_search_result_location" // also used inside text.citations[]

    // MCP tool blocks (if surfaced as content)
    BlockMCPToolUse    BlockType = "mcp_tool_use"
    BlockMCPToolResult BlockType = "mcp_tool_result"
)

// ToolKind classifies the category of tool call.
type ToolKind string

const (
    ToolKindClient  ToolKind = "client"   // user‑hosted function tools (Anthropic tool_use, OpenAI function_call)
    ToolKindServer  ToolKind = "server"   // Anthropic server tools (web_search, web_fetch, code_execution, ...)
    ToolKindBuiltIn ToolKind = "builtin"  // OpenAI built‑ins (web_search, file_search, code_interpreter, computer_use, image_generation, ...)
    ToolKindMCP     ToolKind = "mcp"      // Model Context Protocol tools
    ToolKindCustom  ToolKind = "custom"   // Custom/other
)

// BuiltInTool identifies well‑known built‑in tools (OpenAI Responses).
type BuiltInTool string

const (
    BuiltInWebSearch       BuiltInTool = "web_search"
    BuiltInFileSearch      BuiltInTool = "file_search"
    BuiltInCodeInterpreter BuiltInTool = "code_interpreter"
    BuiltInComputerUse     BuiltInTool = "computer_use"
    BuiltInImageGeneration BuiltInTool = "image_generation"
    BuiltInUnknown         BuiltInTool = ""
)

// ContentChannel tags which 'lane' within a content stream a delta belongs to.
// (Used mostly for OpenAI Responses: refusal, reasoning, reasoning summaries, transcripts, etc.)
type ContentChannel string

const (
    ChannelDefault              ContentChannel = ""
    ChannelOutputText           ContentChannel = "output_text"
    ChannelRefusal              ContentChannel = "refusal"
    ChannelReasoning            ContentChannel = "reasoning"
    ChannelReasoningSummary     ContentChannel = "reasoning_summary"
    ChannelReasoningSummaryText ContentChannel = "reasoning_summary_text"
    ChannelAudioTranscript      ContentChannel = "audio_transcript"
)

// DeltaKind captures the kind of streaming delta for a content block.
type DeltaKind string

const (
    // Shared
    DeltaText       DeltaKind = "text_delta"
    DeltaInputJSON  DeltaKind = "input_json_delta"  // partial JSON arguments (Anthropic tool/server tool; OpenAI function/MCP/custom input)
    DeltaCitations  DeltaKind = "citations_delta"   // Anthropic text citations
    DeltaAnnotation DeltaKind = "annotation_added"  // OpenAI output_text.annotation.added

    // Anthropic thinking/signature
    DeltaThinking  DeltaKind = "thinking_delta"     // thinking stream
    DeltaSignature DeltaKind = "signature_delta"    // signed/structured deltas

    // OpenAI specific channels
    DeltaRefusalText           DeltaKind = "refusal_delta"
    DeltaReasoningText         DeltaKind = "reasoning_delta"
    DeltaReasoningSummary      DeltaKind = "reasoning_summary_delta"
    DeltaReasoningSummaryText  DeltaKind = "reasoning_summary_text_delta"

    // Audio
    DeltaAudio           DeltaKind = "audio_delta"            // base64 audio chunk
    DeltaAudioTranscript DeltaKind = "audio_transcript_delta" // text transcript chunk

    DeltaUnknown DeltaKind = ""
)

// StatusPhase describes stages for long‑running response/tool lifecycles.
type StatusPhase string

const (
    // Generic
    PhaseStarted    StatusPhase = "started"
    PhaseQueued     StatusPhase = "queued"
    PhaseSearching  StatusPhase = "searching"
    PhaseInProgress StatusPhase = "in_progress"
    PhaseInterpreting StatusPhase = "interpreting"
    PhaseGenerating StatusPhase = "generating"
    PhaseResult     StatusPhase = "result"
    PhaseCompleted  StatusPhase = "completed"
    PhaseIncomplete StatusPhase = "incomplete"
    PhaseCanceled   StatusPhase = "canceled"
    PhaseFailed     StatusPhase = "failed"
)

// EventType values for the unified stream model.
const (
    // Message lifecycle
    EventTypeStreamMessageStart EventType = "stream.message_start"
    EventTypeStreamMessageDelta EventType = "stream.message_delta"
    EventTypeStreamMessageStop  EventType = "stream.message_stop"
    EventTypeStreamPing         EventType = "stream.ping" // provider keep‑alive

    // Content blocks
    EventTypeStreamContentStart EventType = "stream.content_block_start"
    EventTypeStreamContentDelta EventType = "stream.content_block_delta"
    EventTypeStreamContentStop  EventType = "stream.content_block_stop"

    // Tool lifecycle (client/server/built‑in/MCP)
    EventTypeStreamToolCallStart      EventType = "stream.tool_call_start"
    EventTypeStreamToolCallInputDelta EventType = "stream.tool_call_input.delta"
    EventTypeStreamToolCallResult     EventType = "stream.tool_call_result"
    EventTypeStreamToolCallStop       EventType = "stream.tool_call_stop"
    EventTypeStreamToolCallStatus     EventType = "stream.tool_call_status" // status/progress for long‑running built‑ins
)

// Commonly used IDs across providers (present when applicable).
type StreamIDs struct {
    ResponseID   string `json:"response_id,omitempty"`  // OpenAI response.id
    MessageID    string `json:"message_id,omitempty"`   // Anthropic msg_*, OpenAI message id if present
    ItemID       string `json:"item_id,omitempty"`      // OpenAI output item id (e.g., tool call item)
    BlockID      string `json:"block_id,omitempty"`     // Anthropic content block id (e.g., tool_use id)
    ToolCallID   string `json:"tool_call_id,omitempty"` // Normalized tool call identifier (maps from BlockID or ItemID)
    StepID       string `json:"step_id,omitempty"`      // OpenAI step id if available
    OutputIndex  *int   `json:"output_index,omitempty"` // OpenAI output index
    ContentIndex *int   `json:"content_index,omitempty"`// Anthropic content index in message.content[]
}

// StreamMessageStart signals the beginning of a streamed message.
type StreamMessageStart struct {
    EventImpl
    Provider   Provider       `json:"provider"`
    IDs        StreamIDs      `json:"ids,omitempty"`
    CreatedAt  time.Time      `json:"created_at"`
    Extensions map[string]any `json:"ext,omitempty"`
}

// StreamMessageDelta conveys high‑level deltas (status/usage/stop_reason/errors).
type StreamMessageDelta struct {
    EventImpl
    Provider   Provider       `json:"provider"`
    IDs        StreamIDs      `json:"ids,omitempty"`
    Phase      StatusPhase    `json:"phase,omitempty"`       // queued|in_progress|completed|failed|incomplete
    StopReason *string        `json:"stop_reason,omitempty"` // Anthropic stop_reason; OpenAI incomplete.reason (mapped when final)
    Usage      *Usage         `json:"usage,omitempty"`       // cumulative usage if available
    Error      string         `json:"error,omitempty"`       // top‑level failure details when surfaced
    Extensions map[string]any `json:"ext,omitempty"`
}

// StreamMessageStop closes a streamed message.
type StreamMessageStop struct {
    EventImpl
    Provider   Provider       `json:"provider"`
    IDs        StreamIDs      `json:"ids,omitempty"`
    Error      string         `json:"error,omitempty"`
    Extensions map[string]any `json:"ext,omitempty"`
}

// StreamPing represents a provider keep‑alive ping.
type StreamPing struct {
    EventImpl
    Provider  Provider  `json:"provider"`
    CreatedAt time.Time `json:"created_at"`
}

// ContentBlockHeader describes the block that is starting.
type ContentBlockHeader struct {
    Type      BlockType `json:"type"`
    ID        string    `json:"id,omitempty"`          // tool_use/server_tool_use id
    Name      string    `json:"name,omitempty"`        // tool name when applicable
    ToolUseID string    `json:"tool_use_id,omitempty"` // result blocks reference originating id
    MIMEType  string    `json:"mime_type,omitempty"`   // for image/audio parts when provided
}

// StreamContentBlockStart announces a new content block in the stream.
type StreamContentBlockStart struct {
    EventImpl
    Provider   Provider           `json:"provider"`
    IDs        StreamIDs          `json:"ids,omitempty"` // ContentIndex typically set
    Block      ContentBlockHeader `json:"block"`
    Extensions map[string]any     `json:"ext,omitempty"`
}

// ContentDelta captures deltas within a content block.
type ContentDelta struct {
    Kind        DeltaKind       `json:"kind"`
    Channel     ContentChannel  `json:"channel,omitempty"` // reasoning/refusal/etc.
    Text        string          `json:"text,omitempty"`         // text‑like deltas
    PartialJSON string          `json:"partial_json,omitempty"` // JSON arguments / input
    // Citations arrive incrementally for Anthropic text blocks.
    Citation    map[string]any  `json:"citation,omitempty"`
    // An annotation payload attached to text (OpenAI output_text.annotation.added).
    Annotation  map[string]any  `json:"annotation,omitempty"`
    // Audio chunk (base64) for audio.delta‑style events.
    AudioBase64 string          `json:"audio_base64,omitempty"`
    // Signature fragment for Anthropic signature_delta.
    Signature   string          `json:"signature,omitempty"`
}

// StreamContentBlockDelta carries an incremental update for the current block.
type StreamContentBlockDelta struct {
    EventImpl
    Provider   Provider       `json:"provider"`
    IDs        StreamIDs      `json:"ids,omitempty"` // ContentIndex set
    Delta      ContentDelta   `json:"delta"`
    Extensions map[string]any `json:"ext,omitempty"`
}

// StreamContentBlockStop marks the end of a content block.
type StreamContentBlockStop struct {
    EventImpl
    Provider   Provider       `json:"provider"`
    IDs        StreamIDs      `json:"ids,omitempty"`
    Extensions map[string]any `json:"ext,omitempty"`
}

// ToolDescriptor defines the tool being invoked.
type ToolDescriptor struct {
    Kind    ToolKind       `json:"kind"`
    // For OpenAI built‑ins
    BuiltIn BuiltInTool    `json:"builtin,omitempty"`
    // For client/server/MCP tools
    Name    string         `json:"name,omitempty"`
    // Provider payload (e.g., OpenAI action, Anthropic input schema, etc.).
    Raw     json.RawMessage `json:"raw,omitempty"`
}

// StreamToolCallStart indicates a tool call has started.
type StreamToolCallStart struct {
    EventImpl
    Provider     Provider       `json:"provider"`
    IDs          StreamIDs      `json:"ids,omitempty"` // ToolCallID set from item/block id
    Tool         ToolDescriptor `json:"tool"`
    InitialInput json.RawMessage `json:"initial_input,omitempty"` // optional initial args
    Extensions   map[string]any  `json:"ext,omitempty"`
}

// StreamToolCallInputDelta represents argument/input streaming.
// Examples:
//  • Anthropic: content_block_delta { type: input_json_delta, partial_json: "..." }
//  • OpenAI:   response.function_call_arguments.delta / .custom_tool_call_input.delta / .mcp_call.arguments_delta
type StreamToolCallInputDelta struct {
    EventImpl
    Provider   Provider       `json:"provider"`
    IDs        StreamIDs      `json:"ids,omitempty"` // ToolCallID + indices when present
    DeltaJSON  string         `json:"delta_json"`    // raw JSON delta string (append‑only)
    Extensions map[string]any `json:"ext,omitempty"`
}

// StreamToolCallResult carries the result payload for a tool call.
// Examples:
//  • Anthropic: web_search_tool_result/web_fetch_tool_result/code_execution_tool_result
//  • OpenAI:   file_search_call results, code_interpreter outputs, image_generation results
type StreamToolCallResult struct {
    EventImpl
    Provider   Provider       `json:"provider"`
    IDs        StreamIDs      `json:"ids,omitempty"`
    ResultRaw  json.RawMessage `json:"result_raw,omitempty"` // lossless payload

    // Common normalizations (best‑effort by adapters)
    Text                 string            `json:"text,omitempty"`
    SearchResults        []SearchResult    `json:"search_results,omitempty"`
    InterpreterOutputs   []map[string]any  `json:"interpreter_outputs,omitempty"`
    // For image_generation partial/final artifacts (URIs or base64)
    ImageURI             string            `json:"image_uri,omitempty"`
    ImageBase64          string            `json:"image_base64,omitempty"`

    Extensions           map[string]any    `json:"ext,omitempty"`
}

// StreamToolCallStop marks the end of a tool call (success/cancel/fail).
type StreamToolCallStop struct {
    EventImpl
    Provider   Provider       `json:"provider"`
    IDs        StreamIDs      `json:"ids,omitempty"`
    Phase      StatusPhase    `json:"phase"`
    Error      string         `json:"error,omitempty"`
    Extensions map[string]any `json:"ext,omitempty"`
}

// StreamToolCallStatus covers progress updates emitted by built‑ins,
// e.g. web_search_call.searching|in_progress|completed,
//      code_interpreter_call.in_progress|interpreting|completed,
//      image_generation_call.generating|partial_image|completed.
type StreamToolCallStatus struct {
    EventImpl
    Provider   Provider       `json:"provider"`
    IDs        StreamIDs      `json:"ids,omitempty"`
    Tool       ToolDescriptor `json:"tool"`
    Phase      StatusPhase    `json:"phase"`
    // Optional extra metadata/payload for the status step.
    Action     map[string]any `json:"action,omitempty"`
    // Optional media chunk for partial image frames, etc.
    ImageBase64 string        `json:"image_base64,omitempty"`
    ImageURI    string        `json:"image_uri,omitempty"`
    Extensions map[string]any `json:"ext,omitempty"`
}

// SearchResult is a normalized record for search‑like tools.
type SearchResult struct {
    URL        string         `json:"url,omitempty"`
    Title      string         `json:"title,omitempty"`
    Snippet    string         `json:"snippet,omitempty"`
    // Provider‑specific fields (scores, ranks, locations, etc.).
    Extensions map[string]any `json:"ext,omitempty"`
}

// --- Helper constructors (adapters may set fields directly as well).

func NewStreamMessageStart(meta EventMetadata, p Provider, ids StreamIDs) *StreamMessageStart {
    return &StreamMessageStart{
        EventImpl: EventImpl{Type_: EventTypeStreamMessageStart, Metadata_: meta},
        Provider:  p,
        IDs:       ids,
        CreatedAt: time.Now(),
    }
}

func NewStreamMessageDelta(meta EventMetadata, p Provider, ids StreamIDs, phase StatusPhase, stop *string, usage *Usage, err string) *StreamMessageDelta {
    return &StreamMessageDelta{
        EventImpl:  EventImpl{Type_: EventTypeStreamMessageDelta, Metadata_: meta},
        Provider:   p,
        IDs:        ids,
        Phase:      phase,
        StopReason: stop,
        Usage:      usage,
        Error:      err,
    }
}

func NewStreamMessageStop(meta EventMetadata, p Provider, ids StreamIDs, err string) *StreamMessageStop {
    return &StreamMessageStop{
        EventImpl: EventImpl{Type_: EventTypeStreamMessageStop, Metadata_: meta},
        Provider:  p,
        IDs:       ids,
        Error:     err,
    }
}

func NewStreamPing(meta EventMetadata, p Provider) *StreamPing {
    return &StreamPing{
        EventImpl: EventImpl{Type_: EventTypeStreamPing, Metadata_: meta},
        Provider:  p,
        CreatedAt: time.Now(),
    }
}

func NewStreamContentBlockStart(meta EventMetadata, p Provider, ids StreamIDs, h ContentBlockHeader) *StreamContentBlockStart {
    return &StreamContentBlockStart{
        EventImpl: EventImpl{Type_: EventTypeStreamContentStart, Metadata_: meta},
        Provider:  p,
        IDs:       ids,
        Block:     h,
    }
}

func NewStreamContentBlockDelta(meta EventMetadata, p Provider, ids StreamIDs, d ContentDelta) *StreamContentBlockDelta {
    return &StreamContentBlockDelta{
        EventImpl: EventImpl{Type_: EventTypeStreamContentDelta, Metadata_: meta},
        Provider:  p,
        IDs:       ids,
        Delta:     d,
    }
}

func NewStreamContentBlockStop(meta EventMetadata, p Provider, ids StreamIDs) *StreamContentBlockStop {
    return &StreamContentBlockStop{
        EventImpl: EventImpl{Type_: EventTypeStreamContentStop, Metadata_: meta},
        Provider:  p,
        IDs:       ids,
    }
}

func NewStreamToolCallStart(meta EventMetadata, p Provider, ids StreamIDs, t ToolDescriptor) *StreamToolCallStart {
    return &StreamToolCallStart{
        EventImpl: EventImpl{Type_: EventTypeStreamToolCallStart, Metadata_: meta},
        Provider:  p,
        IDs:       ids,
        Tool:      t,
    }
}

func NewStreamToolCallInputDelta(meta EventMetadata, p Provider, ids StreamIDs, deltaJSON string) *StreamToolCallInputDelta {
    return &StreamToolCallInputDelta{
        EventImpl: EventImpl{Type_: EventTypeStreamToolCallInputDelta, Metadata_: meta},
        Provider:  p,
        IDs:       ids,
        DeltaJSON: deltaJSON,
    }
}

func NewStreamToolCallResult(meta EventMetadata, p Provider, ids StreamIDs) *StreamToolCallResult {
    return &StreamToolCallResult{
        EventImpl: EventImpl{Type_: EventTypeStreamToolCallResult, Metadata_: meta},
        Provider:  p,
        IDs:       ids,
    }
}

func NewStreamToolCallStop(meta EventMetadata, p Provider, ids StreamIDs, phase StatusPhase, err string) *StreamToolCallStop {
    return &StreamToolCallStop{
        EventImpl: EventImpl{Type_: EventTypeStreamToolCallStop, Metadata_: meta},
        Provider:  p,
        IDs:       ids,
        Phase:     phase,
        Error:     err,
    }
}

func NewStreamToolCallStatus(meta EventMetadata, p Provider, ids StreamIDs, t ToolDescriptor, phase StatusPhase) *StreamToolCallStatus {
    return &StreamToolCallStatus{
        EventImpl: EventImpl{Type_: EventTypeStreamToolCallStatus, Metadata_: meta},
        Provider:  p,
        IDs:       ids,
        Tool:      t,
        Phase:     phase,
    }
}

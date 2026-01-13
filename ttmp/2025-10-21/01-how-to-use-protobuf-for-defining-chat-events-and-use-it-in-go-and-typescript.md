### Goal

Define protobuf schemas for the chat events in `geppetto/pkg/events/chat-events.go`, and show how to generate and consume strongly-typed code in Go and TypeScript (Node) to emit and process progress events.

### Why protobuf for chat events

- **Interoperability**: One schema, multi-language generation (Go + TS/Node now, others later).
- **Streaming-friendly**: Compact binary encoding for high-frequency progress events; supports JSON via protojson when needed.
- **Schema evolution**: Backward-compatible changes using field numbers and presence semantics.

### High-level design

- **Single wrapper message with a oneof payload**: A top-level `Event` message carries `metadata` and exactly one event-specific payload (e.g. `start`, `partial`, `final`, ...). This replaces the current string `EventType` discriminator at the wire level while remaining easy to map to/from existing Go types.
- **Metadata extracted**: `EventMetadata`, `LLMInferenceData`, and `Usage` become dedicated protobuf messages.
- **Free-form extras**: `EventMetadata.extra` modeled as `google.protobuf.Struct` (maps well to `map[string]any` in Go and objects in TS).
- **IDs and correlation**: `message_id` as canonical UUID string; `run_id`/`turn_id` strings. We can switch to `bytes` UUID if desired, but string keeps interop easy.
- **Versioning and package layout**: `geppetto/events/v1` as the protobuf package. Future breaking changes land in `v2`.

### Proposed repository layout

- Protos live under `geppetto/proto/geppetto/events/v1/`
  - `metadata.proto`: `Usage`, `LLMInferenceData`, `EventMetadata`, shared types like `SearchResult`
  - `events.proto`: `Event` wrapper and all event payload messages (Start, Partial, Final, Error, Tool*, WebSearch*, Citation, FileSearch*, CodeInterpreter*, Reasoning*, MCP*, ImageGen*, ToolSearchResults, ...)
- Codegen output
  - Go: `geppetto/pkg/proto/geppetto/events/v1/`
  - TypeScript: `geppetto/node/gen/geppetto/events/v1/` (or publishable package path of your choice)

### Mapping from current Go types to protobuf

- Wrapper: `EventImpl` + concrete types → `Event` with `oneof payload`
- `EventMetadata` fields map 1:1
  - `message_id: string` (UUID as string)
  - `run_id`, `turn_id`: `string`
  - `extra`: `google.protobuf.Struct`
  - Embedded `LLMInferenceData` in Go becomes a nested field in protobuf for clarity and schema stability: `llm: LLMInferenceData`.
- `LLMInferenceData`
  - `model: string`
  - Optional scalars use `optional` (presence matters for UI/analytics): `optional double temperature`, `optional double top_p`, `optional int32 max_tokens`, `optional string stop_reason`, `optional int64 duration_ms`
  - `usage: Usage` (itself optional by field presence)
- Event payloads mirror the current event structs and fields. A few names are normalized for protobuf style (snake_case field names, lowerCamelCase in JSON).

### Draft protobuf schema (representative and complete)

Below are draft `.proto` files covering all event kinds present today. Field numbers are assigned densely per message; keep them stable.

metadata.proto

```proto
syntax = "proto3";

package geppetto.events.v1;

import "google/protobuf/struct.proto";

option go_package = "github.com/go-go-golems/geppetto/pkg/proto/geppetto/events/v1;eventsv1";

message Usage {
  int32 input_tokens = 1;
  int32 output_tokens = 2;
  int32 cached_tokens = 3;                      // optional by presence in JSON
  int32 cache_creation_input_tokens = 4;        // provider-specific
  int32 cache_read_input_tokens = 5;
}

message LLMInferenceData {
  string model = 1;
  optional double temperature = 2;
  optional double top_p = 3;
  optional int32 max_tokens = 4;
  optional string stop_reason = 5;
  Usage usage = 6;                              // presence indicates set
  optional int64 duration_ms = 7;
}

message EventMetadata {
  string message_id = 1;                        // UUID string
  string run_id = 2;
  string turn_id = 3;
  LLMInferenceData llm = 4;                     // embedded in Go; explicit here
  google.protobuf.Struct extra = 5;             // free-form provider/ctx data
}

message SearchResult {
  string url = 1;
  string title = 2;
  string snippet = 3;
  map<string, google.protobuf.Value> ext = 4;   // normalized tool extras
}
```

events.proto

```proto
syntax = "proto3";

package geppetto.events.v1;

import "geppetto/events/v1/metadata.proto";

option go_package = "github.com/go-go-golems/geppetto/pkg/proto/geppetto/events/v1;eventsv1";

// Wrapper for all chat events; exactly one payload must be set.
message Event {
  EventMetadata metadata = 1;

  oneof payload {
    Start start = 10;
    PartialCompletion partial = 11;
    ThinkingPartial thinking_partial = 12;
    Final final = 13;
    Interrupt interrupt = 14;
    Error error = 15;

    ToolCall tool_call = 20;
    ToolResult tool_result = 21;
    ToolCallExecute tool_call_execute = 22;
    ToolCallExecutionResult tool_call_execution_result = 23;

    Log log = 30;
    Info info = 31;
    AgentModeSwitch agent_mode_switch = 32;

    WebSearchStarted web_search_started = 40;
    WebSearchSearching web_search_searching = 41;
    WebSearchOpenPage web_search_open_page = 42;
    WebSearchDone web_search_done = 43;

    Citation citation = 50;

    FileSearchStarted file_search_started = 60;
    FileSearchSearching file_search_searching = 61;
    FileSearchDone file_search_done = 62;

    CodeInterpreterStarted code_interpreter_started = 70;
    CodeInterpreterInterpreting code_interpreter_interpreting = 71;
    CodeInterpreterDone code_interpreter_done = 72;
    CodeInterpreterCodeDelta code_interpreter_code_delta = 73;
    CodeInterpreterCodeDone code_interpreter_code_done = 74;

    ReasoningTextDelta reasoning_text_delta = 80;
    ReasoningTextDone reasoning_text_done = 81;

    MCPArgsDelta mcp_args_delta = 90;
    MCPArgsDone mcp_args_done = 91;
    MCPInProgress mcp_in_progress = 92;
    MCPCompleted mcp_completed = 93;
    MCPFailed mcp_failed = 94;
    MCPListInProgress mcp_list_in_progress = 95;
    MCPListCompleted mcp_list_completed = 96;
    MCPListFailed mcp_list_failed = 97;

    ImageGenInProgress image_gen_in_progress = 100;
    ImageGenGenerating image_gen_generating = 101;
    ImageGenPartialImage image_gen_partial_image = 102;
    ImageGenCompleted image_gen_completed = 103;

    ToolSearchResults tool_search_results = 110;
  }
}

// Core text events
message Start {}

message PartialCompletion {
  string delta = 1;
  string completion = 2;
}

message ThinkingPartial {
  string delta = 1;
  string completion = 2;
}

message Final { string text = 1; }
message Interrupt { string text = 1; }
message Error { string error_string = 1; }

// Tool calls
message ToolCall {
  string id = 1;
  string name = 2;
  string input = 3; // raw JSON string of args
}

message ToolResult {
  string id = 1;
  string result = 2; // raw JSON string of result
}

message ToolCallExecute { ToolCall tool_call = 1; }
message ToolCallExecutionResult { ToolResult tool_result = 1; }

// Logs & info
message Log {
  string level = 1;                          // e.g. debug/info/warn/error
  string message = 2;
  map<string, string> fields = 3;            // stringified for simplicity
}

message Info {
  string message = 1;
  map<string, string> data = 2;              // stringified for simplicity
}

message AgentModeSwitch {
  string from = 1;
  string to = 2;
  string analysis = 3;                       // optional narrative
}

// Web search
message WebSearchStarted { string item_id = 1; string query = 2; }
message WebSearchSearching { string item_id = 1; }
message WebSearchOpenPage { string item_id = 1; string url = 2; }
message WebSearchDone { string item_id = 1; }

// Citation annotations
message Citation {
  string title = 1;
  string url = 2;
  optional int32 start_index = 3;
  optional int32 end_index = 4;
  optional int32 output_index = 5;
  optional int32 content_index = 6;
  optional int32 annotation_index = 7;
}

// File search
message FileSearchStarted { string item_id = 1; }
message FileSearchSearching { string item_id = 1; }
message FileSearchDone { string item_id = 1; }

// Code interpreter
message CodeInterpreterStarted { string item_id = 1; }
message CodeInterpreterInterpreting { string item_id = 1; }
message CodeInterpreterDone { string item_id = 1; }
message CodeInterpreterCodeDelta { string item_id = 1; string delta = 2; }
message CodeInterpreterCodeDone { string item_id = 1; string code = 2; }

// Reasoning text
message ReasoningTextDelta { string delta = 1; }
message ReasoningTextDone { string text = 1; }

// MCP
message MCPArgsDelta { string item_id = 1; string delta = 2; }
message MCPArgsDone { string item_id = 1; string arguments = 2; }
message MCPInProgress { string item_id = 1; }
message MCPCompleted { string item_id = 1; }
message MCPFailed { string item_id = 1; }
message MCPListInProgress { string item_id = 1; }
message MCPListCompleted { string item_id = 1; }
message MCPListFailed { string item_id = 1; }

// Image generation
message ImageGenInProgress { string item_id = 1; }
message ImageGenGenerating { string item_id = 1; }
message ImageGenPartialImage { string item_id = 1; string partial_image_base64 = 2; }
message ImageGenCompleted { string item_id = 1; }

// Normalized search results from tools
message ToolSearchResults {
  string tool = 1;
  string item_id = 2;
  repeated SearchResult results = 3;
}
```

Notes:
- The `oneof payload` removes the need for a `type` string at the wire level. If you still want a string type for logging or debugging, add an optional `legacy_type` field on `Event`, computed by SDKs.
- For `Log.fields` and `Info.data`, we use `map<string,string>` to keep generated TS ergonomics simple. If you need arbitrary JSON, switch to `google.protobuf.Struct`.

### Buf setup and code generation

Use Buf for consistent generation and linting.

`geppetto/proto/buf.yaml`

```yaml
version: v1
name: buf.build/go-go-golems/geppetto
deps:
  - buf.build/googleapis/googleapis
lint:
  use:
    - STANDARD
breaking:
  use:
    - FILE
```

`geppetto/proto/buf.gen.yaml` (Go + TypeScript via `@bufbuild/protobuf`)

```yaml
version: v1
plugins:
  - plugin: buf.build/protocolbuffers/go
    out: ../pkg/proto
    opt: paths=source_relative
  - plugin: buf.build/bufbuild/es
    out: ../node/gen
    opt:
      - target=ts
```

Generate code from the module root `geppetto/proto`:

```bash
cd geppetto/proto && buf dep update && buf generate
```

This will produce:
- Go: `geppetto/pkg/proto/geppetto/events/v1/*.pb.go`
- TS: `geppetto/node/gen/geppetto/events/v1/*.ts` (imports `@bufbuild/protobuf`)

If you prefer `ts-proto`:

```yaml
plugins:
  - plugin: buf.build/community/stephenh/ts-proto
    out: ../node/gen
    opt:
      - esModuleInterop=true
      - env=node
      - outputJsonMethods=true
      - useExactTypes=false
```

### Consuming in Go

Construct and serialize an event (binary protobuf) to publish on your bus:

```go
import (
  eventsv1 "github.com/go-go-golems/geppetto/pkg/proto/geppetto/events/v1"
  "google.golang.org/protobuf/proto"
)

ev := &eventsv1.Event{
  Metadata: &eventsv1.EventMetadata{
    MessageId: uuid.NewString(),
    RunId: runID,
    TurnId: turnID,
    Llm: &eventsv1.LLMInferenceData{Model: model},
  },
  Payload: &eventsv1.Event_Partial{Partial: &eventsv1.PartialCompletion{Delta: delta, Completion: cum}},
}
bytes, _ := proto.Marshal(ev)
// publish bytes with content-type: application/x-protobuf
```

If you need JSON for debugging/UI endpoints:

```go
import "google.golang.org/protobuf/encoding/protojson"

b, _ := protojson.MarshalOptions{EmitUnpopulated: false, UseProtoNames: true}.Marshal(ev)
// publish b with content-type: application/json
```

### Consuming in TypeScript (Node)

Using `@bufbuild/protobuf` generated code:

```ts
import { Event, PartialCompletion } from "../../node/gen/geppetto/events/v1/events_pb";

// Encode
const ev = new Event({
  metadata: { messageId: crypto.randomUUID(), runId: "r", turnId: "t", llm: { model: "gpt-4o" } },
  partial: new PartialCompletion({ delta: "he", completion: "hello" })
});
const wire = ev.toBinary();

// Decode
const decoded = Event.fromBinary(wire);
if (decoded.partial) {
  console.log(decoded.partial.delta);
}
```

If you chose `ts-proto`, the shape will be slightly different (namespace-less functions, plain objects), but the flow is analogous (`Event.encode(ev).finish()`, `Event.decode(bytes)`).

### Migration plan

- Phase 1: Introduce protobuf alongside existing JSON events.
  - Add a serializer: when producing events, emit protobuf on the message bus; keep JSON for UI surfaces if needed.
  - Add deserializer: accept both protobuf and JSON until all producers/consumers switch.
- Phase 2: Flip defaults to protobuf everywhere; retain JSON only at HTTP boundaries.
- Phase 3: Remove legacy JSON struct emitters once all code paths are converted.

### Edge cases and notes

- **Optional scalar presence**: `optional` scalars are used where UI needs to know if a value was set vs default.
- **`extra` map**: If you need lossless round-tripping of arbitrary JSON, `google.protobuf.Struct` is the correct choice. Keep values small; large blobs should be referenced by URL/ID.
- **Unknown/new events**: Protobuf unknown fields are ignored by older consumers. Adding new payload messages is a compatible change as long as field numbers don’t collide.
- **Tool arguments/results**: Modeled as JSON strings. If you want stronger typing, define per-tool `oneof` arguments/results in a separate package and reference them.

### Concrete next steps

1) Add `geppetto/proto/` with `buf.yaml`, `buf.gen.yaml` and the two `.proto` files above.
2) `buf dep update && buf generate` to produce Go and TS code.
3) Introduce adapters to map between current Go `events.*` structs and the generated protobuf `Event` during the migration.
4) Start emitting protobuf on the message bus; update Node producer/consumer to use generated TS.
5) Remove JSON-only event structs when all downstreams have migrated.

### Appendix: Mapping table (Go → Protobuf message)

- `EventPartialCompletionStart` → `Start`
- `EventPartialCompletion` → `PartialCompletion`
- `EventThinkingPartial` → `ThinkingPartial`
- `EventFinal` → `Final`
- `EventInterrupt` → `Interrupt`
- `EventError` → `Error`
- `EventToolCall` → `ToolCall`
- `EventToolResult` → `ToolResult`
- `EventToolCallExecute` → `ToolCallExecute`
- `EventToolCallExecutionResult` → `ToolCallExecutionResult`
- `EventLog` → `Log`
- `EventInfo` → `Info`
- `EventAgentModeSwitch` → `AgentModeSwitch`
- `EventWebSearchStarted` → `WebSearchStarted`
- `EventWebSearchSearching` → `WebSearchSearching`
- `EventWebSearchOpenPage` → `WebSearchOpenPage`
- `EventWebSearchDone` → `WebSearchDone`
- `EventCitation` → `Citation`
- `EventFileSearchStarted` → `FileSearchStarted`
- `EventFileSearchSearching` → `FileSearchSearching`
- `EventFileSearchDone` → `FileSearchDone`
- `EventCodeInterpreterStarted` → `CodeInterpreterStarted`
- `EventCodeInterpreterInterpreting` → `CodeInterpreterInterpreting`
- `EventCodeInterpreterDone` → `CodeInterpreterDone`
- `EventCodeInterpreterCodeDelta` → `CodeInterpreterCodeDelta`
- `EventCodeInterpreterCodeDone` → `CodeInterpreterCodeDone`
- `EventReasoningTextDelta` → `ReasoningTextDelta`
- `EventReasoningTextDone` → `ReasoningTextDone`
- `EventMCPArgsDelta` → `MCPArgsDelta`
- `EventMCPArgsDone` → `MCPArgsDone`
- `EventMCPInProgress` → `MCPInProgress`
- `EventMCPCompleted` → `MCPCompleted`
- `EventMCPFailed` → `MCPFailed`
- `EventMCPListInProgress` → `MCPListInProgress`
- `EventMCPListCompleted` → `MCPListCompleted`
- `EventMCPListFailed` → `MCPListFailed`
- `EventImageGenInProgress` → `ImageGenInProgress`
- `EventImageGenGenerating` → `ImageGenGenerating`
- `EventImageGenPartialImage` → `ImageGenPartialImage`
- `EventImageGenCompleted` → `ImageGenCompleted`
- `EventToolSearchResults` → `ToolSearchResults`



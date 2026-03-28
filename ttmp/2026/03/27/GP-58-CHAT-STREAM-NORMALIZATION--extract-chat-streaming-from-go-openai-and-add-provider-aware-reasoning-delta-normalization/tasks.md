# Tasks

## Documentation Deliverables

- [x] Create ticket workspace and primary documents.
- [x] Write an intern-facing design and implementation guide for the chat-streaming extraction.
- [x] Record the investigation diary with commands, evidence, and decisions.
- [x] Relate the key source files and prior investigation artifacts to the ticket.
- [x] Validate the ticket with `docmgr doctor`.
- [x] Upload the design bundle to reMarkable and verify the remote listing.

## Implementation Plan

- [x] Step 1: Add chat streaming endpoint/config helpers that resolve API key, base URL, HTTP client, and `/chat/completions` target URL without using `go-openai` stream APIs.
- [x] Step 2: Add a raw SSE frame reader for chat-completions streaming responses.
- [x] Step 3: Define internal normalized streaming types for text deltas, reasoning deltas, tool-call fragments, usage, and finish reasons.
- [x] Step 4: Add delta normalization for `content`, `reasoning`, `reasoning_content`, and tool-call fragments.
- [x] Step 5: Refactor `pkg/steps/ai/openai/engine_openai.go` to use the new streaming client instead of `CreateChatCompletionStream`.
- [x] Step 6: Publish `partial-thinking` and `reasoning-text-*` events from the chat-completions path when reasoning is present.
- [x] Step 7: Persist reasoning blocks to the output turn for chat-completions providers that expose raw reasoning text.
- [x] Step 8: Preserve existing tool-call merge semantics, usage metadata, stop reason handling, and final assistant/tool block ordering.
- [x] Step 9: Keep `go-openai` limited to request-building, embeddings, and transcription for this ticket.

## Testing Plan

- [x] Add unit tests for SSE frame parsing.
- [x] Add unit tests for provider delta normalization.
- [x] Add fixture-driven tests for Together `delta.reasoning`.
- [x] Add fixture-driven tests for DeepSeek-style `delta.reasoning_content`.
- [x] Add engine tests proving reasoning events and reasoning turn blocks are emitted for chat-completions streams.
- [x] Add regression tests proving text-only providers still emit unchanged `partial`/`final` events.
- [x] Add regression tests for fragmented tool calls and final usage chunks.
- [x] Confirm embeddings and transcription packages remain untouched and continue compiling.

## Deferred Work

- [ ] Evaluate whether a later migration to `openai-go/v3` is worthwhile for non-streaming OpenAI-native paths.
- [ ] Decide whether to fully remove `go-openai` request structs from `pkg/steps/ai/openai/helpers.go` after the streaming refactor lands.

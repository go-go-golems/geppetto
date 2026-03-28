# Changelog

## 2026-03-27

- Initial workspace created
- Added a design ticket for extracting chat streaming from `go-openai` while keeping embeddings and transcription on the existing client.
- Captured current-state evidence from the OpenAI chat engine, Open Responses engine, Together raw SSE experiments, and the `go-openai` stream delta type.
- Wrote the intern-facing architecture and implementation guide plus the diary.
- Validated the ticket and uploaded the bundle to reMarkable.

## 2026-03-28

- Implemented a Geppetto-owned `/chat/completions` streaming transport with direct HTTP, SSE parsing, and provider delta normalization.
- Refactored the OpenAI chat engine to consume normalized stream events instead of `CreateChatCompletionStream`.
- Added reasoning event publication and reasoning block persistence for chat-completions providers that expose raw reasoning deltas.
- Added fixture-driven regression tests for Together-style reasoning, DeepSeek-style reasoning-content, text-only streams, and fragmented tool calls.
- Verified the full repository test suite with `go test ./... -count=1`.

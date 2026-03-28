# Tasks

## TODO

- [x] Preserve the original Together experiment scripts under `scripts/`.
- [x] Add a Pinocchio JS surface probe under `scripts/` and document its limitations.
- [x] Run a raw Together `/chat/completions` SSE control experiment with the `together-qwen-3.5-9b` profile.
- [x] Run the `go-openai` probe with the same profile and compare behavior against the raw SSE control.
- [x] Run the Geppetto OpenAI chat-stream path with the same profile and compare behavior against the raw SSE control.
- [x] Identify and fix the Geppetto request-shape regression that prevented streaming (`stream=true` missing on the outgoing chat-completions request).
- [x] Re-run the experiment matrix and store bounded outputs under `sources/experiments/`.
- [x] Capture the exact request bodies used by raw SSE, `go-openai`, and Geppetto under `sources/experiments/`.
- [x] Add a detailed postmortem / intern guide that ties the runtime fix, experiment evidence, and remaining `go-openai` questions together.
- [x] Upload the refreshed GP-57 ticket bundle to reMarkable.
- [x] Compare those request bodies in more detail, especially around Together-specific extras, enough to justify taking ownership of the chat boundary instead of waiting on the SDK.
- [x] Investigate why `go-openai` surfaces repeated `role="assistant"` chunks but no `reasoning_content` or `content` for Together Qwen, to the point where the remaining work can be archived as non-blocking after the chat-layer cutover.
- [x] Update the design doc with the new request-construction findings.

## Chat Extraction

- [x] Add Geppetto-local chat request/message/tool/response-format structs so the chat layer no longer depends on `go-openai` types.
- [x] Migrate `MakeCompletionRequestFromTurn` and related helper logic to the new local chat structs.
- [x] Migrate the custom chat stream decoder and tool-call merger to the new local chat structs.
- [x] Remove remaining `go-openai` usage from `pkg/steps/ai/openai` chat runtime code while keeping embeddings and transcription on the SDK.
- [x] Update and extend chat-layer tests so they validate the local structs instead of SDK types.
- [x] Refresh the GP-57 postmortem, task list, changelog, and diary with the chat-layer cutover details.

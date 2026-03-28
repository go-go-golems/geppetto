# Changelog

## 2026-03-27

- Initial workspace created

## 2026-03-28

- Preserved the Pinocchio JS probe under `scripts/`, expanded the GP-57 tasks, and started a detailed investigation diary.
- Reproduced the Together issue with real profile-backed runs: raw SSE streamed `delta.reasoning`, `go-openai` produced only role chunks, and Geppetto initially produced zero chunks.
- Fixed the Geppetto request-shape bug by forcing `stream=true` at the custom chat-stream boundary and added a regression test.
- Re-ran the Together experiments and stored artifact outputs under `sources/experiments/`.
- Added exact request-body capture to the probe artifacts so GP-57 can compare raw SSE, `go-openai`, and Geppetto payloads directly.
- Added a detailed postmortem/intern guide that explains the system architecture, experiment matrix, root cause split, implemented fix, and remaining `go-openai` follow-up work.
- Refreshed the ticket bundle and uploaded `GP-57 Together Thinking Postmortem Package` to reMarkable under `/ai/2026/03/28/GP-57-TOGETHER-THINKING`, including a recorded Pandoc formatting failure and recovery.
- Replaced `go-openai` request/message/tool structs in the OpenAI chat layer with Geppetto-local chat structs, leaving embeddings and transcription on the SDK.

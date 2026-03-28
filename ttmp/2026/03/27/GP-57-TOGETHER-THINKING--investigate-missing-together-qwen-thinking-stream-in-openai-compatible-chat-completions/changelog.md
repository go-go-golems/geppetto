# Changelog

## 2026-03-27

- Initial workspace created

## 2026-03-28

- Preserved the Pinocchio JS probe under `scripts/`, expanded the GP-57 tasks, and started a detailed investigation diary.
- Reproduced the Together issue with real profile-backed runs: raw SSE streamed `delta.reasoning`, `go-openai` produced only role chunks, and Geppetto initially produced zero chunks.
- Fixed the Geppetto request-shape bug by forcing `stream=true` at the custom chat-stream boundary and added a regression test.
- Re-ran the Together experiments and stored artifact outputs under `sources/experiments/`.

# Changelog

## 2026-02-21

- Initial workspace created
- Implemented cached-token usage propagation for OpenAI Responses streaming and non-streaming paths
  - Code commit: `970b936cec31e07c2928af9c491638ed59974991`
  - Added shared usage parsing helpers and wired cached usage into final metadata
  - Added/updated tests for streaming and non-streaming cached usage assertions
- Closed ticket documentation loop
  - Updated tasks checklist, design implementation notes, and diary with validation details

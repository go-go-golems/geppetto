# Changelog

## 2026-02-21

- Initial workspace created
- Implemented cached-token usage propagation for OpenAI Responses streaming and non-streaming paths
  - Code commit: `970b936cec31e07c2928af9c491638ed59974991`
  - Added shared usage parsing helpers and wired cached usage into final metadata
  - Added/updated tests for streaming and non-streaming cached usage assertions
- Closed ticket documentation loop
  - Updated tasks checklist, design implementation notes, and diary with validation details
- Follow-up hardening: fixed `gosec` G115 integer-overflow findings in usage parsing conversion helper
  - Code commit: `bcbe17d5c5d0f4ee4534270785b4e87384a72975`
  - Added bounds checks before converting float/int/uint values to `int` in `toInt(...)`
  - Re-ran targeted tests and `gosec` for `pkg/steps/ai/openai_responses` (0 issues)
- Review-driven optimization: non-stream usage parsing no longer performs a second full envelope unmarshal
  - Code commit: `e489ab2c883a601bf6e1c10f0af882530f0e3564`
  - Kept one full decode into `responsesResponse` and parsed `usage` via `json.RawMessage`
  - Added regression test for nested `response.usage` parsing

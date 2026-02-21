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

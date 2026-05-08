# Changelog

## 2026-05-07

- Initial workspace created


## 2026-05-07

Added Geppetto normalized provider correlation fields for Chat Completions and Responses observability.

### Related Files

- pkg/observability/observer.go — Adds scalar correlation fields to Geppetto records
- pkg/steps/ai/openai/engine_openai.go — Propagates correlation metadata into reasoning/content/tool events
- pkg/steps/ai/openai/observability.go — Builds Chat Completions correlation keys from response id
- pkg/steps/ai/openai_responses/observability.go — Adds Responses correlation keys while preserving native item ids


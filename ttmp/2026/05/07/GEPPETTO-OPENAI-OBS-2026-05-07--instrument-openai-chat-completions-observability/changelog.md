# Changelog

## 2026-05-07

- Initial workspace created


## 2026-05-07

Created ticket, captured source evidence, wrote intern-oriented OpenAI Chat Completions observability implementation guide, and started implementation diary.


## 2026-05-07

Validated ticket metadata with docmgr doctor after replacing ad-hoc topics with vocabulary topics chat and intern-onboarding.


## 2026-05-07

Uploaded guide and diary bundle to reMarkable at /ai/2026/05/07/GEPPETTO-OPENAI-OBS-2026-05-07.


## 2026-05-07

Uploaded the standalone OpenAI Chat Completions observability design doc PDF to reMarkable.


## 2026-05-07

Implemented OpenAI Chat Completions observability and committed source changes as 1c2c9dfdada18163afde41a7024d7468982a0662.

### Related Files

- pkg/inference/engine/factory/factory.go — Adds WithOpenAIOptions and passes them into OpenAI engines
- pkg/inference/engine/factory/factory_observability_test.go — Factory option plumbing test
- pkg/steps/ai/openai/engine_openai.go — Wires publish/provider/normalization observation into RunInference
- pkg/steps/ai/openai/observability.go — New OpenAI observer/config options and record helpers
- pkg/steps/ai/openai/observability_test.go — Focused OpenAI observability tests


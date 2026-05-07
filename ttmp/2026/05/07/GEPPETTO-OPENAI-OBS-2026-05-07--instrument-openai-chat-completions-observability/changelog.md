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


## 2026-05-07

Removed OpenAI Chat Completions publish-boundary observability records and kept provider/normalization records only.

### Related Files

- pkg/inference/engine/factory/factory_observability_test.go — Factory test now verifies provider records instead of publish records
- pkg/steps/ai/openai/engine_openai.go — publishEvent no longer emits observability started/done records
- pkg/steps/ai/openai/observability.go — Removed OpenAI observePublish helper
- pkg/steps/ai/openai/observability_test.go — Updated expectations for provider-only OpenAI observability


## 2026-05-07

Aligned OpenAI Chat Completions and Responses publish observability to compact started-only records; no publish-done JSON payload records.

### Related Files

- pkg/steps/ai/openai/engine_openai.go — Publishes compact observability record before event fan-out
- pkg/steps/ai/openai/observability.go — Adds compact observePublishStarted helper
- pkg/steps/ai/openai_responses/engine.go — Removes post-publish done record
- pkg/steps/ai/openai_responses/observability.go — Keeps publish observation compact without event/metadata JSON


## 2026-05-07

Updated the design guide for the publish-started-only policy and re-uploaded the standalone PDF to reMarkable.

### Related Files

- ttmp/2026/05/07/GEPPETTO-OPENAI-OBS-2026-05-07--instrument-openai-chat-completions-observability/design-doc/01-openai-chat-completions-observability-analysis-and-implementation-guide.md — Updated publish observability policy and implementation/test guidance


## 2026-05-07

Ran Playwright web-chat runthrough with gpt-5-nano-low OpenAI Responses profile, captured debug artifacts, and documented event-size analysis.

### Related Files

- ttmp/2026/05/07/GEPPETTO-OPENAI-OBS-2026-05-07--instrument-openai-chat-completions-observability/sources/06-openai-responses-webchat-runthrough.png — Browser runthrough screenshot
- ttmp/2026/05/07/GEPPETTO-OPENAI-OBS-2026-05-07--instrument-openai-chat-completions-observability/sources/10-openai-responses-event-size-analysis.md — Runthrough event-size breakdown


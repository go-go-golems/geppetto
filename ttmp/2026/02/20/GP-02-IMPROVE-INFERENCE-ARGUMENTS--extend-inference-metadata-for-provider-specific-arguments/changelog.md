# Changelog

## 2026-02-20

- Initial workspace created


## 2026-02-20

Step 3: Implementation — types, keys, and engine wiring across all 4 providers (commit 36d93f1)

### Related Files

- /home/manuel/workspaces/2026-02-20/improve-inference-metadata/geppetto/pkg/inference/engine/inference_config.go — Core types and resolution helpers
- /home/manuel/workspaces/2026-02-20/improve-inference-metadata/geppetto/pkg/steps/ai/claude/helpers.go — Claude engine wiring
- /home/manuel/workspaces/2026-02-20/improve-inference-metadata/geppetto/pkg/steps/ai/gemini/engine_gemini.go — Gemini engine wiring
- /home/manuel/workspaces/2026-02-20/improve-inference-metadata/geppetto/pkg/steps/ai/openai/helpers.go — OpenAI Chat engine wiring
- /home/manuel/workspaces/2026-02-20/improve-inference-metadata/geppetto/pkg/steps/ai/openai_responses/helpers.go — OpenAI Responses engine wiring


## 2026-02-20

Step 7: Fixed explicit-empty stop override precedence leak in OpenAI/Claude/Responses builders and added regression tests (commit 2e0b55e).

### Related Files

- /home/manuel/workspaces/2026-02-20/improve-inference-metadata/geppetto/pkg/steps/ai/claude/helpers.go — Stop override now applies on non-nil slice (explicit empty clears)
- /home/manuel/workspaces/2026-02-20/improve-inference-metadata/geppetto/pkg/steps/ai/openai/helpers.go — Stop override now applies on non-nil slice (explicit empty clears)
- /home/manuel/workspaces/2026-02-20/improve-inference-metadata/geppetto/pkg/steps/ai/openai_responses/helpers.go — Stop override now applies on non-nil slice (explicit empty clears)


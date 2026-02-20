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


## 2026-02-20

Step 8: Synchronized ticket docs with current precedence semantics after Step 7 (merge behavior + explicit-empty stop clear) so analysis/index guidance matches implementation.

### Related Files

- /home/manuel/workspaces/2026-02-20/improve-inference-metadata/geppetto/ttmp/2026/02/20/GP-02-IMPROVE-INFERENCE-ARGUMENTS--extend-inference-metadata-for-provider-specific-arguments/index.md — Added current-state overview and links
- /home/manuel/workspaces/2026-02-20/improve-inference-metadata/geppetto/ttmp/2026/02/20/GP-02-IMPROVE-INFERENCE-ARGUMENTS--extend-inference-metadata-for-provider-specific-arguments/design/01-analysis-inference-arguments.md — Corrected precedence text to field-level merge and nil/empty/non-empty stop semantics
- /home/manuel/workspaces/2026-02-20/improve-inference-metadata/geppetto/ttmp/2026/02/20/GP-02-IMPROVE-INFERENCE-ARGUMENTS--extend-inference-metadata-for-provider-specific-arguments/analysis/03-rigorous-merge-and-validation-for-inferenceconfig.md — Added status update and removed stale “open bug” language
- /home/manuel/workspaces/2026-02-20/improve-inference-metadata/geppetto/ttmp/2026/02/20/GP-02-IMPROVE-INFERENCE-ARGUMENTS--extend-inference-metadata-for-provider-specific-arguments/reference/01-diary.md — Marked Step 6 stop-clear note as superseded by Step 7

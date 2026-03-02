# Tasks

## TODO

- [x] Add tasks here

- [x] Map end-to-end stop reason flow from provider stream to gorunner stop policy
- [x] Reproduce low-token max_tokens behavior and capture evidence
- [x] Document root-cause analysis with line-anchored evidence
- [x] Produce intern-oriented architecture and implementation guide
- [x] Maintain chronological investigation diary
- [x] Run docmgr doctor validation
- [x] Upload final document bundle to reMarkable
- [x] Analyze current inference-result signaling channels (Turn, Block, Events, Session Handle, persistence)
- [x] Design and compare alternative inference-result communication models
- [x] Write intern-oriented research design doc for inference-result signaling
- [x] Update diary with investigation and delivery steps for inference-result signaling study
- [x] Upload updated MEN-TR-005 bundle to reMarkable
- [x] Write detailed InferenceResult implementation plan with wrapper strategy and migration phases
- [x] Implement `InferenceResult` type and canonical `turn.metadata.inference_result` key via codegen
- [x] Add `RunInferenceWithResult` wrapper helper around existing `RunInference`
- [x] Make Claude/OpenAI/OpenAI-Responses/Gemini engines populate canonical inference result metadata
- [x] Migrate gorunner stop-policy reads to canonical-result-first with legacy fallback
- [x] Add provider parity + wrapper + gorunner regression tests for canonical inference result contract
- [x] Upload implementation-plan update bundle to reMarkable
- [x] Remove runtime provider API-key environment fallbacks in JS engine bindings and gorunner
- [x] Validate profile-backed key path by running longest anonymized transcript repro script end to end
- [x] Write deep postmortem documenting env key fallback origin, failure modes, and removal rationale
- [x] Produce credential/provider wiring playbook for JS bindings and Go runner

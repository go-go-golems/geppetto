# Changelog

## 2026-03-02

- Initial workspace created


## 2026-03-02

Created MEN-TR-005 architecture guide with line-anchored root-cause analysis and phased remediation plan for stop_reason propagation across geppetto engines.

### Related Files

- /home/manuel/workspaces/2026-03-02/deliver-mento-1/geppetto/pkg/steps/ai/claude/engine_claude.go — Root-cause evidence
- /home/manuel/workspaces/2026-03-02/deliver-mento-1/temporal-relationships/internal/extractor/gorunner/run.go — Consumer-side stop logic
- /home/manuel/workspaces/2026-03-02/deliver-mento-1/temporal-relationships/ttmp/2026/03/02/MEN-TR-005--go-runner-max-tokens-stop-reason-propagation-and-loop-continuation-hardening/design-doc/01-max-tokens-stop-reason-propagation-architecture-and-intern-implementation-guide.md — Primary deliverable


## 2026-03-02

Added investigation diary and reproducible low-token experiment script for intern onboarding and verification.

### Related Files

- /home/manuel/workspaces/2026-03-02/deliver-mento-1/temporal-relationships/ttmp/2026/03/02/MEN-TR-005--go-runner-max-tokens-stop-reason-propagation-and-loop-continuation-hardening/reference/01-investigation-diary.md — Chronological execution log
- /home/manuel/workspaces/2026-03-02/deliver-mento-1/temporal-relationships/ttmp/2026/03/02/MEN-TR-005--go-runner-max-tokens-stop-reason-propagation-and-loop-continuation-hardening/scripts/01-repro-max-tokens-stop-reason.sh — Reproduction automation


## 2026-03-02

Validated ticket docs with docmgr doctor (after adding vocabulary slugs: claude, extraction, stop-policy).

### Related Files

- /home/manuel/workspaces/2026-03-02/deliver-mento-1/temporal-relationships/ttmp/2026/03/02/MEN-TR-005--go-runner-max-tokens-stop-reason-propagation-and-loop-continuation-hardening/index.md — Doctor target
- /home/manuel/workspaces/2026-03-02/deliver-mento-1/temporal-relationships/ttmp/vocabulary.yaml — Added missing topic vocabulary entries


## 2026-03-02

Uploaded MEN-TR-005 bundle to reMarkable at /ai/2026/03/02/MEN-TR-005 and verified remote listing.

### Related Files

- /home/manuel/workspaces/2026-03-02/deliver-mento-1/temporal-relationships/ttmp/2026/03/02/MEN-TR-005--go-runner-max-tokens-stop-reason-propagation-and-loop-continuation-hardening/design-doc/01-max-tokens-stop-reason-propagation-architecture-and-intern-implementation-guide.md — Included in uploaded bundle
- /home/manuel/workspaces/2026-03-02/deliver-mento-1/temporal-relationships/ttmp/2026/03/02/MEN-TR-005--go-runner-max-tokens-stop-reason-propagation-and-loop-continuation-hardening/reference/01-investigation-diary.md — Included in uploaded bundle


## 2026-03-02

Added a second deep research design study comparing inference-result communication models (Turn section, events, blocks, handle, and interface alternatives) with a recommended hybrid architecture.

### Related Files

- /home/manuel/workspaces/2026-03-02/deliver-mento-1/geppetto/pkg/inference/engine/engine.go — Current engine contract reference
- /home/manuel/workspaces/2026-03-02/deliver-mento-1/geppetto/pkg/turns/keys_gen.go — Current metadata key contract reference
- /home/manuel/workspaces/2026-03-02/deliver-mento-1/temporal-relationships/ttmp/2026/03/02/MEN-TR-005--go-runner-max-tokens-stop-reason-propagation-and-loop-continuation-hardening/design-doc/02-inference-result-signaling-architecture-study-turn-metadata-sections-events-and-alternative-contracts.md — Primary new design study


## 2026-03-02

Added executable inventory script for inference-result signaling channels and recorded the new investigation step in the diary.

### Related Files

- /home/manuel/workspaces/2026-03-02/deliver-mento-1/temporal-relationships/ttmp/2026/03/02/MEN-TR-005--go-runner-max-tokens-stop-reason-propagation-and-loop-continuation-hardening/reference/01-investigation-diary.md — Chronological update with prompt context and findings
- /home/manuel/workspaces/2026-03-02/deliver-mento-1/temporal-relationships/ttmp/2026/03/02/MEN-TR-005--go-runner-max-tokens-stop-reason-propagation-and-loop-continuation-hardening/scripts/02-inventory-inference-result-signals.sh — Evidence inventory automation


## 2026-03-02

Uploaded updated bundle with inference-result signaling study to reMarkable as MEN-TR-005 stop-reason propagation guide v2 and verified cloud listing.

### Related Files

- /home/manuel/workspaces/2026-03-02/deliver-mento-1/temporal-relationships/ttmp/2026/03/02/MEN-TR-005--go-runner-max-tokens-stop-reason-propagation-and-loop-continuation-hardening/design-doc/02-inference-result-signaling-architecture-study-turn-metadata-sections-events-and-alternative-contracts.md — Included in v2 upload
- /home/manuel/workspaces/2026-03-02/deliver-mento-1/temporal-relationships/ttmp/2026/03/02/MEN-TR-005--go-runner-max-tokens-stop-reason-propagation-and-loop-continuation-hardening/reference/01-investigation-diary.md — Updated with new study step


## 2026-03-02

Added a third deep design document with an implementation-first plan to introduce canonical `InferenceResult` metadata, add `RunInferenceWithResult` wrapper semantics, enforce provider parity, and stage legacy key deprecation safely.

### Related Files

- /home/manuel/workspaces/2026-03-02/deliver-mento-1/geppetto/pkg/inference/engine/engine.go — Existing interface baseline for wrapper plan
- /home/manuel/workspaces/2026-03-02/deliver-mento-1/geppetto/pkg/steps/ai/openai_responses/engine.go — Provider parity planning target
- /home/manuel/workspaces/2026-03-02/deliver-mento-1/geppetto/pkg/turns/keys_gen.go — Key migration target
- /home/manuel/workspaces/2026-03-02/deliver-mento-1/temporal-relationships/internal/extractor/gorunner/run.go — Consumer migration target
- /home/manuel/workspaces/2026-03-02/deliver-mento-1/temporal-relationships/ttmp/2026/03/02/MEN-TR-005--go-runner-max-tokens-stop-reason-propagation-and-loop-continuation-hardening/design-doc/03-inference-result-implementation-plan-runinferencewithresult-wrapper-and-metadata-contract-migration.md — New implementation plan deliverable

## 2026-03-02

Added InferenceResult implementation plan (doc 03) with wrapper API, provider parity, and legacy key migration strategy; updated ticket tasks and diary accordingly.

### Related Files

- /home/manuel/workspaces/2026-03-02/deliver-mento-1/temporal-relationships/ttmp/2026/03/02/MEN-TR-005--go-runner-max-tokens-stop-reason-propagation-and-loop-continuation-hardening/design-doc/03-inference-result-implementation-plan-runinferencewithresult-wrapper-and-metadata-contract-migration.md — Primary implementation-plan deliverable
- /home/manuel/workspaces/2026-03-02/deliver-mento-1/temporal-relationships/ttmp/2026/03/02/MEN-TR-005--go-runner-max-tokens-stop-reason-propagation-and-loop-continuation-hardening/reference/01-investigation-diary.md — Diary step added for this planning and delivery pass
- /home/manuel/workspaces/2026-03-02/deliver-mento-1/temporal-relationships/ttmp/2026/03/02/MEN-TR-005--go-runner-max-tokens-stop-reason-propagation-and-loop-continuation-hardening/tasks.md — Task plan updated for implementation phases


## 2026-03-02

Validated MEN-TR-005 after implementation-plan additions and uploaded v3 bundle to reMarkable including design-doc 03 and updated diary.

### Related Files

- /home/manuel/workspaces/2026-03-02/deliver-mento-1/temporal-relationships/ttmp/2026/03/02/MEN-TR-005--go-runner-max-tokens-stop-reason-propagation-and-loop-continuation-hardening/design-doc/03-inference-result-implementation-plan-runinferencewithresult-wrapper-and-metadata-contract-migration.md — Included in v3 upload
- /home/manuel/workspaces/2026-03-02/deliver-mento-1/temporal-relationships/ttmp/2026/03/02/MEN-TR-005--go-runner-max-tokens-stop-reason-propagation-and-loop-continuation-hardening/reference/01-investigation-diary.md — Updated with validation and upload step


## 2026-03-02

Implemented canonical InferenceResult contract in geppetto: added typed turn metadata key, RunInferenceWithResult wrapper, provider persistence updates across Claude/OpenAI/OpenAI Responses/Gemini, tests, and topic-doc updates.

### Related Files

- /home/manuel/workspaces/2026-03-02/deliver-mento-1/geppetto/pkg/doc/topics/06-inference-engines.md — Updated docs for RunInferenceWithResult and canonical metadata
- /home/manuel/workspaces/2026-03-02/deliver-mento-1/geppetto/pkg/inference/engine/run_with_result.go — Compatibility wrapper returning normalized inference result
- /home/manuel/workspaces/2026-03-02/deliver-mento-1/geppetto/pkg/steps/ai/openai_responses/engine.go — Provider persistence of canonical inference metadata
- /home/manuel/workspaces/2026-03-02/deliver-mento-1/geppetto/pkg/turns/inference_result.go — Canonical durable inference result schema


## 2026-03-02

Migrated temporal-relationships gorunner to canonical inference-result-first stop reason reads with legacy fallback and added regression tests.

### Related Files

- /home/manuel/workspaces/2026-03-02/deliver-mento-1/temporal-relationships/internal/extractor/gorunner/run.go — Canonical-first stop reason extraction
- /home/manuel/workspaces/2026-03-02/deliver-mento-1/temporal-relationships/internal/extractor/gorunner/run_test.go — Regression tests for canonical precedence and fallback
- /home/manuel/workspaces/2026-03-02/deliver-mento-1/temporal-relationships/ttmp/2026/03/02/MEN-TR-005--go-runner-max-tokens-stop-reason-propagation-and-loop-continuation-hardening/tasks.md — Task checklist completion


## 2026-03-02

Validated ticket after implementation completion and uploaded v4 bundle to reMarkable with implemented InferenceResult contract artifacts.

### Related Files

- /home/manuel/workspaces/2026-03-02/deliver-mento-1/temporal-relationships/ttmp/2026/03/02/MEN-TR-005--go-runner-max-tokens-stop-reason-propagation-and-loop-continuation-hardening/design-doc/03-inference-result-implementation-plan-runinferencewithresult-wrapper-and-metadata-contract-migration.md — Included in v4 upload bundle
- /home/manuel/workspaces/2026-03-02/deliver-mento-1/temporal-relationships/ttmp/2026/03/02/MEN-TR-005--go-runner-max-tokens-stop-reason-propagation-and-loop-continuation-hardening/reference/01-investigation-diary.md — Diary records final validation and upload verification


## 2026-03-02

Removed provider API-key environment fallback from runtime extraction code paths and aligned JS tests/docs with profile/StepSettings key resolution.

### Related Files

- /home/manuel/workspaces/2026-03-02/deliver-mento-1/geppetto/pkg/js/modules/geppetto/api_engines.go — Removed `os.Getenv` key fallback and profile env selector fallback in JS engine bindings
- /home/manuel/workspaces/2026-03-02/deliver-mento-1/geppetto/pkg/js/modules/geppetto/module_test.go — Updated profile fixtures to schema-backed key sections (`openai-chat`, `claude-chat`)
- /home/manuel/workspaces/2026-03-02/deliver-mento-1/geppetto/pkg/doc/topics/13-js-api-reference.md — Added explicit no-env-fallback key guidance for `fromConfig`
- /home/manuel/workspaces/2026-03-02/deliver-mento-1/geppetto/pkg/doc/topics/14-js-api-user-guide.md — Clarified profile patch key fields for `fromProfile`
- /home/manuel/workspaces/2026-03-02/deliver-mento-1/temporal-relationships/internal/extractor/gorunner/run.go — Removed gorunner step-settings key inference from process environment
- /home/manuel/workspaces/2026-03-02/deliver-mento-1/temporal-relationships/ttmp/2026/03/02/MEN-TR-005--go-runner-max-tokens-stop-reason-propagation-and-loop-continuation-hardening/scripts/01-repro-max-tokens-stop-reason.sh — Dropped `ANTHROPIC_API_KEY` preflight check, now profile-registry driven


## 2026-03-02

Ran real extraction repro against the longest anonymized transcript with profile-backed credentials and captured repeatable evidence for first-iteration continuation under low-token constraints.

### Related Files

- /home/manuel/workspaces/2026-03-02/deliver-mento-1/temporal-relationships/ttmp/2026/03/02/MEN-TR-005--go-runner-max-tokens-stop-reason-propagation-and-loop-continuation-hardening/scripts/01-repro-max-tokens-stop-reason.sh — Executed end-to-end on `anonymized/a2be5ded.txt`
- /home/manuel/workspaces/2026-03-02/deliver-mento-1/temporal-relationships/ttmp/2026/03/02/MEN-TR-005--go-runner-max-tokens-stop-reason-propagation-and-loop-continuation-hardening/reference/01-investigation-diary.md — Recorded outputs and diagnostics (`iterations=2`, `run_inference_starts=2`, `message_delta_max_tokens=2`)


## 2026-03-02

Validated MEN-TR-005 documentation integrity after path cleanup and uploaded refreshed v5 bundle to reMarkable.

### Related Files

- /home/manuel/workspaces/2026-03-02/deliver-mento-1/temporal-relationships/ttmp/2026/03/02/MEN-TR-005--go-runner-max-tokens-stop-reason-propagation-and-loop-continuation-hardening/index.md — Normalized related-file entries to pass doctor in this workspace layout
- /home/manuel/workspaces/2026-03-02/deliver-mento-1/temporal-relationships/ttmp/2026/03/02/MEN-TR-005--go-runner-max-tokens-stop-reason-propagation-and-loop-continuation-hardening/reference/01-investigation-diary.md — Added Steps 13-14 with implementation + validation + upload chronology
- /home/manuel/workspaces/2026-03-02/deliver-mento-1/temporal-relationships/ttmp/2026/03/02/MEN-TR-005--go-runner-max-tokens-stop-reason-propagation-and-loop-continuation-hardening/changelog.md — Added v5 delivery record


## 2026-03-02

Added a detailed postmortem explaining why env key fallback was introduced, why it became harmful after profile-first migration, and how removal changed runtime contracts.

### Related Files

- /home/manuel/workspaces/2026-03-02/deliver-mento-1/temporal-relationships/ttmp/2026/03/02/MEN-TR-005--go-runner-max-tokens-stop-reason-propagation-and-loop-continuation-hardening/design-doc/04-env-api-key-fallback-removal-postmortem.md — New deep postmortem with timeline, root cause, alternatives, and guardrails
- /home/manuel/workspaces/2026-03-02/deliver-mento-1/geppetto/pkg/js/modules/geppetto/api_engines.go — Primary code-path evidence for JS binding fallback removal
- /home/manuel/workspaces/2026-03-02/deliver-mento-1/temporal-relationships/internal/extractor/gorunner/run.go — Primary code-path evidence for go runner fallback removal


## 2026-03-02

Added operational playbooks for deterministic credential and provider wiring in both ticket docs and upstream Geppetto docs.

### Related Files

- /home/manuel/workspaces/2026-03-02/deliver-mento-1/temporal-relationships/ttmp/2026/03/02/MEN-TR-005--go-runner-max-tokens-stop-reason-propagation-and-loop-continuation-hardening/playbook/01-credential-and-provider-wiring-playbook-for-js-and-go-runner.md — Ticket-local execution playbook for JS + go runner wiring and validation
- /home/manuel/workspaces/2026-03-02/deliver-mento-1/geppetto/pkg/doc/playbooks/07-wire-provider-credentials-for-js-and-go-runner.md — Upstream Geppetto playbook reference for same contract
- /home/manuel/workspaces/2026-03-02/deliver-mento-1/geppetto/pkg/doc/topics/13-js-api-reference.md — Contract alignment reference for `fromProfile` and `fromConfig`

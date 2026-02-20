# Tasks

## Done

- [x] Implement InferenceConfig types and resolution helpers (commit 36d93f1)
- [x] Wire all four provider engines to read Turn.Data overrides (commit 36d93f1)
- [x] Add StepSettings.Inference field for engine-level defaults (commit 36d93f1)
- [x] Complete existing KeyStructuredOutputConfig wiring gap (commit 36d93f1)
- [x] Analysis: glazed section design (analysis/02-glazed-section-for-inferenceconfig.md)
- [x] Experiment: verify glazed pointer nil behavior (scripts/test_glazed_pointer_nil/)
- [x] Add `yaml:` + `glazed:` tags to InferenceConfig (commit 71e8154)
- [x] Create `flags/inference.yaml` with no defaults (commit 71e8154)
- [x] Create `settings-inference.go` with InferenceValueSection (commit 71e8154)
- [x] Wire into `sections.CreateGeppettoSections()` and env var whitelist (commit 71e8154)
- [x] Wire `DecodeSectionInto` in `settings.UpdateFromParsedValues()` (commit 71e8154)
- [x] Add `glazed:"ai-inference"` tag to `StepSettings.Inference` (commit 71e8154)
- [x] Fix Claude: re-validate temperature/top_p exclusivity after overrides (commit 0c06789)
- [x] Fix Claude: reject temperature != 1.0 when thinking enabled (commit 0c06789)
- [x] Fix OpenAI Responses: guard temperature/top_p overrides for reasoning models (commit 0c06789)
- [x] Fix OpenAI Chat: guard temperature/top_p overrides for reasoning models (commit 0c06789)

- [x] Add MergeInferenceConfig with field-level merge and deep-copy safety (commit 3bb7a62)
- [x] Add SanitizeForReasoningModel / SanitizeOpenAIForReasoningModel helpers (commit 3bb7a62)
- [x] Convert Claude MakeMessageRequestFromTurn to method on ClaudeEngine (commit 3bb7a62)
- [x] Convert OpenAI MakeCompletionRequestFromTurn to method on OpenAIEngine (commit 3bb7a62)
- [x] Convert OpenAI Responses buildResponsesRequest to method on Engine (commit 3bb7a62)
- [x] Fix Bug #2: OpenAI Chat penalty bypass for reasoning models (commit 3bb7a62)
- [x] Replace per-field reasoning guards with upfront sanitize pattern (commit 3bb7a62)
- [x] Add tests for MergeInferenceConfig, sanitize helpers, reasoning model behavior (commit 3bb7a62)
- [x] Update analysis doc 03 with Option B recommendation (commit 3bb7a62)
- [x] Fix empty Stop override leak in OpenAI/Claude/Responses builders and add regression tests (commit 2e0b55e)
- [x] Sync ticket docs to reflect merged precedence and explicit-empty stop clear semantics

## TODO (follow-up)

- [ ] Integration test: create command with ai-inference section, parse flags, verify InferenceConfig nil/non-nil
- [ ] Test choice flags without defaults in real Cobra command (--help display, omission behavior)
- [ ] Consider exposing ClaudeInferenceConfig / OpenAIInferenceConfig as glazed sections
- [ ] Add example profiles with inference settings to misc/profiles.yaml
- [ ] Integration tests for full Turn.Data → merge → sanitize → request pipeline
- [ ] Consider consolidating isReasoningModel helpers across OpenAI packages

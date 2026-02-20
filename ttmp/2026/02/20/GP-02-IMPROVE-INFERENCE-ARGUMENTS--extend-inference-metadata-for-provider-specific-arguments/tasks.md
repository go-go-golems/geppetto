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

## TODO (follow-up)

- [ ] Integration test: create command with ai-inference section, parse flags, verify InferenceConfig nil/non-nil
- [ ] Test choice flags without defaults in real Cobra command (--help display, omission behavior)
- [ ] Consider exposing ClaudeInferenceConfig / OpenAIInferenceConfig as glazed sections
- [ ] Add example profiles with inference settings to misc/profiles.yaml
- [ ] Unit tests for InferenceConfig overrides on reasoning models (regression prevention)
- [ ] Consider consolidating isReasoningModel helpers across OpenAI packages

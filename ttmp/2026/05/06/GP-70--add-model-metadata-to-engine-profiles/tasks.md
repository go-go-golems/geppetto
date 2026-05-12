# Tasks

## TODO

- [ ] Add tasks here

- [ ] Create ModelInfo, InputModality, ModelCost types in geppetto/pkg/steps/ai/settings/model_info.go
- [ ] Add ModelInfo field to InferenceSettings and update Clone/GetMetadata/GetSummary
- [ ] Implement MergeModelInfo and integrate into inference settings merge
- [ ] Write unit tests for ModelInfo types, merge, cost computation, context limits
- [ ] Add model_info to profile YAML fixtures and verify stack merge
- [ ] Replace isReasoningModel() with ModelInfo.Reasoning (with fallback)
- [ ] Expose ModelInfo through JS module surface
- [ ] Add Cost field to InferenceResult and compute post-inference
- [ ] Integrate ModelInfo into Pinocchio context budgeting and UI

# Tasks

## TODO

- [x] Decide whether GP-09 is design-only or implementation ticket
- [x] Phase 1: Extract request policy into `BuildEngineFromReq` (single profile/override resolver used by HTTP + WS)
- [x] Phase 2: Make tool registry driven by `EngineConfig.Tools` (honor `overrides["tools"]` or remove the feature)
- [x] Phase 2: Enforce `Profile.AllowOverrides` (explicit reject vs ignore)
- [x] Phase 3: Sanitize `EngineConfig.Signature()` to avoid secrets in `StepSettings`
- [x] Phase 4: Replace `SetConversationManager` with a minimal `ConversationLookup` dependency for sink wrappers

# Tasks

## Phase 0: Prep and Baseline

- [ ] GP02-000 Capture compile/test baseline for pinocchio webchat (`go test ./cmd/web-chat`, `go test ./pkg/webchat/...`, `go test ./...`).
- [ ] GP02-001 Add rename-map section in planning doc and freeze scope for this ticket.
- [ ] GP02-002 Identify exported symbols requiring compatibility aliases.
- [ ] GP02-003 Define deprecation window policy for old names (internal vs exported).

## Phase 1: Contract and DTO Renames

- [ ] GP02-100 Rename `ConversationRequestPlan` -> `ResolvedConversationRequest` in `pkg/webchat/http/api.go`.
- [ ] GP02-101 Rename `AppConversationRequest` -> `ConversationRuntimeRequest` in `pkg/webchat/conversation_service.go`.
- [ ] GP02-102 Rename `RuntimeComposeRequest.RuntimeKey` -> `ProfileKey`.
- [ ] GP02-103 Rename `RuntimeComposeRequest.ResolvedRuntime` -> `ResolvedProfileRuntime`.
- [ ] GP02-104 Rename `RuntimeComposeRequest.Overrides` -> `RuntimeOverrides`.
- [ ] GP02-105 Add temporary aliases/wrappers for old exported DTO names where needed.
- [ ] GP02-106 Update DTO call-sites in `cmd/web-chat`, `pkg/webchat`, and `pkg/inference/runtime`.
- [ ] GP02-107 Update tests impacted by DTO/field renames.

## Phase 2: Resolver Naming Cleanup

- [ ] GP02-200 Rename `webChatProfileResolver` -> `ProfileRequestResolver`.
- [ ] GP02-201 Rename constructor helpers accordingly (for example `newProfileRequestResolver`).
- [ ] GP02-202 Rename `resolveProfile` -> `resolveProfileSelection`.
- [ ] GP02-203 Rename `runtimeKeyFromPath` -> `profileSlugFromPath`.
- [ ] GP02-204 Rename `baseOverridesForProfile` -> `runtimeDefaultsFromProfile`.
- [ ] GP02-205 Rename `mergeOverrides` -> `mergeRuntimeOverrides`.
- [ ] GP02-206 Rename `newInMemoryProfileRegistry` -> `newInMemoryProfileService`.
- [ ] GP02-207 Rename `registerProfileHandlers` -> `registerProfileAPIHandlers`.
- [ ] GP02-208 Add compatibility wrappers for old resolver symbol names when required.

## Phase 3: Runtime Composer Naming Cleanup

- [ ] GP02-300 Rename `webChatRuntimeComposer` -> `ProfileRuntimeComposer`.
- [ ] GP02-301 Rename `runtimeFingerprint` -> `buildRuntimeFingerprint`.
- [ ] GP02-302 Rename `runtimeFingerprintPayload` -> `RuntimeFingerprintInput`.
- [ ] GP02-303 Rename `validateOverrides` -> `validateRuntimeOverrides`.
- [ ] GP02-304 Rename `parseMiddlewareOverrides` -> `parseRuntimeMiddlewareOverrides`.
- [ ] GP02-305 Rename `parseToolOverrides` -> `parseRuntimeToolOverrides`.
- [ ] GP02-306 Update composer tests for renamed helpers and symbols.

## Phase 4: Constants and Literals

- [ ] GP02-400 Replace `chat_profile` string literals with `currentProfileCookieName` constant.
- [ ] GP02-401 Rename `defaultWebChatRegistrySlug` -> `defaultRegistrySlug`.
- [ ] GP02-402 Audit remaining magic literals in resolver/composer path and replace with constants where useful.

## Phase 5: Cross-Repo Feature Flag Name Cleanup

- [ ] GP02-500 Introduce `GEPPETTO_ENABLE_PROFILE_REGISTRY_MIDDLEWARE` in Geppetto sections feature-flag reader.
- [ ] GP02-501 Keep `PINOCCHIO_ENABLE_PROFILE_REGISTRY_MIDDLEWARE` as fallback during transition.
- [ ] GP02-502 Add tests covering dual-read behavior and precedence.
- [ ] GP02-503 Add deprecation note for old env var in docs/help text.

## Phase 6: Documentation and Migration Notes

- [ ] GP02-600 Update postmortem/reference docs with old->new rename mapping.
- [ ] GP02-601 Add short developer migration note for downstream importers.
- [ ] GP02-602 Update code comments to use new terminology consistently.

## Phase 7: Validation and Cleanup

- [ ] GP02-700 Run full pinocchio test suite and confirm no behavior regressions.
- [ ] GP02-701 Run grep sweep for old symbol names and verify only intentional shims remain.
- [ ] GP02-702 Decide whether to remove selected shims now or defer to follow-up ticket.
- [ ] GP02-703 Update changelog with rename migration summary and compatibility notes.

## Optional Follow-up (if not completed in this ticket)

- [ ] GP02-900 Remove deprecated aliases after downstream repos complete migration.
- [ ] GP02-901 Remove legacy env var fallback after deprecation window closes.

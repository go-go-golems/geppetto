# Tasks

## Phase 1: Core contract

- [x] Add `RuntimeComposer` contract to `pkg/webchat`.
- [x] Add router option `WithRuntimeComposer(...)` and require non-nil composer.
- [x] Introduce `RuntimeComposeRequest` and `RuntimeArtifacts` types.

## Phase 2: ConvManager refactor

- [x] Replace buildConfig/buildFromConfig callbacks with composer call.
- [x] Rename conversation rebuild key field to `RuntimeFingerprint`.
- [x] Keep runtime metadata (`RuntimeKey`) separate from rebuild fingerprint.

## Phase 3: Core cleanup

- [x] Remove `EngineConfig` from `pkg/webchat` public surface.
- [x] Remove `EngineBuilder`, `BuildConfig`, `BuildFromConfig`, and parser helpers.
- [x] Update/replace unit tests to use stub composers.

## Phase 4: App migrations

- [x] Implement composer in `pinocchio/cmd/web-chat`.
- [x] Implement composer in `web-agent-example`.
- [x] Validate behavior parity for defaults + overrides in both apps.

## Phase 5: Docs

- [x] Update framework/user/tutorial docs for resolver + composer architecture.
- [x] Remove stale references to core build config APIs.

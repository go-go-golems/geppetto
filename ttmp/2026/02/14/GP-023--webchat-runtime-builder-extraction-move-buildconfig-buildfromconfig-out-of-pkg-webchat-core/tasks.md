# Tasks

## Phase 1: Core contract

- [ ] Add `RuntimeComposer` contract to `pkg/webchat`.
- [ ] Add router option `WithRuntimeComposer(...)` and require non-nil composer.
- [ ] Introduce `RuntimeComposeRequest` and `RuntimeArtifacts` types.

## Phase 2: ConvManager refactor

- [ ] Replace buildConfig/buildFromConfig callbacks with composer call.
- [ ] Rename conversation rebuild key field to `RuntimeFingerprint`.
- [ ] Keep runtime metadata (`RuntimeKey`) separate from rebuild fingerprint.

## Phase 3: Core cleanup

- [ ] Remove `EngineConfig` from `pkg/webchat` public surface.
- [ ] Remove `EngineBuilder`, `BuildConfig`, `BuildFromConfig`, and parser helpers.
- [ ] Update/replace unit tests to use stub composers.

## Phase 4: App migrations

- [ ] Implement composer in `pinocchio/cmd/web-chat`.
- [ ] Implement composer in `web-agent-example`.
- [ ] Validate behavior parity for defaults + overrides in both apps.

## Phase 5: Docs

- [ ] Update framework/user/tutorial docs for resolver + composer architecture.
- [ ] Remove stale references to core build config APIs.

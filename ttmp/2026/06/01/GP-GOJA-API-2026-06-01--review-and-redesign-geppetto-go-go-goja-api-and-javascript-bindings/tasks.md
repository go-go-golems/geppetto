# Tasks

## Research / design deliverables

- [x] Map current Geppetto, Pinocchio, and go-go-goja runtime/binding architecture
- [x] Review current API strengths, weaknesses, missing pieces, and confusing parts
- [x] Design opinionated Go-backed fluid builder API for JS bindings
- [x] Write intern-facing implementation guide with file references, pseudocode, and diagrams
- [x] Validate docmgr ticket and upload final bundle to reMarkable

## Implementation task list — hard-cut Geppetto registry-backed JS API

### Phase 0: Contract lock and baseline inventory

- [x] Add build-tagged API contract test for final top-level `require("geppetto")` keys: `agent`, `inferenceProfiles`, `turn`, `engine`, `tool`, `toolRegistry`, `embeddings`, `schema`, `events`, `unsafe`
- [x] Add negative API inventory checks that old map-first names are absent: `inferenceSettings`, `turns.newTurn`, `engines.fromConfig`, `createBuilder`, `createSession`, `runInference`, ordinary `runner.run`
- [x] Document accepted `gp.inferenceProfiles.load(...)` source forms: YAML path, `yaml:PATH`, `yaml://PATH`, SQLite path, `sqlite:PATH`, `sqlite-dsn:DSN`
- [x] Run baseline focused tests: `go test ./pkg/js/modules/geppetto ./pkg/js/runtime -count=1` — passed
- [x] Run build-tagged contract test and record expected current failure until the hard-cut exports are implemented: `go test -tags geppetto_js_hardcut_contract ./pkg/js/modules/geppetto -run TestHardCutPublicSurfaceContract -count=1` — expected failure: `missing hard-cut export: agent`

### Phase 1: Go-owned `InferenceSettings` result wrapper

- [x] Add `api_inference_settings.go` in `geppetto/pkg/js/modules/geppetto`
- [x] Implement immutable/copy-on-write `InferenceSettingsJS` wrapper around cloned `*settings.InferenceSettings`
- [x] Store provenance metadata on the wrapper: registry slug, profile slug, stack lineage, and source metadata
- [x] Implement read-only helper methods: `toJSON`, `clone`, and redacted `debug`/snapshot helpers
- [x] Skip read-only `model()` / `provider()` getters for now so model-parameter method names remain absent
- [x] Explicitly do not implement mutating model-parameter methods: no `provider`, no `model`, no `temperature`, no `topP`, no `maxTokens`, no `timeoutMs`, no `modelInfo`, no `credentialRef`
- [x] Forbid public raw credential APIs: no `apiKey`, no `apiKeyEnv`, no `fromEnv`
- [x] Add tests proving snapshots are detached and mutating JS snapshots does not mutate Go-owned settings
- [x] Add negative tests proving forbidden mutating/credential methods are absent

### Phase 2: Geppetto registry loader wrapper

- [x] Add or rewrite `api_inference_profiles.go` for the new `gp.inferenceProfiles` namespace
- [x] Implement `gp.inferenceProfiles.load(source)` for string and string-array source inputs
- [x] Use `engineprofiles.ParseRegistrySourceSpecs` and `engineprofiles.NewChainedRegistryFromSourceSpecs` internally
- [x] Implement `InferenceRegistryJS` wrapper with registry reader, optional closer, and source metadata
- [x] Implement `registry.resolve(input)` for string profile names and typed `{ registry, profile }` input snapshots
- [x] Wrap `ResolvedEngineProfile.InferenceSettings` as `InferenceSettingsJS` with provenance metadata
- [x] Implement `registry.listRegistries`, `registry.listProfiles`, and `registry.close`
- [x] Implement host-default `gp.inferenceProfiles.resolve(...)` and clear error when no default registry is configured
- [x] Add temporary registry YAML tests for `slug`, supported default resolution behavior, multiple source precedence, invalid source errors, and current `default_profile_slug` rejection behavior
- [x] Add explicit rejection/documentation test for Pinocchio unified config docs with `app:` as unsupported by `gp.inferenceProfiles.load(...)`

### Phase 3: Engine builder integration

- [x] Add/update `EngineBuilderJS`
- [x] Implement `gp.engine().inference(settings).build()` accepting only `InferenceSettingsJS` or trusted Go settings wrappers
- [ ] Remove/withhold public `fromConfig(map)` and map-first engine constructors (deferred to Phase 8 cleanup; new `gp.engine()` path is implemented)
- [ ] Resolve symbolic credentials through host `CredentialResolver` only inside Go-side engine build (not implemented yet; current real tests build provider engines from registry API keys and redact JS snapshots)
- [x] Ensure raw credentials are never visible through JS snapshots/debug output
- [x] Add tests for building engines from registry-resolved settings
- [x] Add tests rejecting plain JS objects passed as inference settings
- [x] Add tests for redacted debug/snapshot output; missing credential resolver remains part of host credential work

### Phase 4: Agent API integration — explicit turns only

- [x] Add/update `api_agent.go`
- [x] Implement `gp.agent()` builder methods: `name`, `inference`, `engine`, `middleware`, `goMiddleware`, `tool`, `goTool`, `toolLoop`, `events`, `runDefaults`, `build`
- [x] Explicitly do not implement `agent.ask(prompt)`, `agent.system(prompt)`, `agent.profile(name)`, or first-pass `agent.inferenceProfile(name)`
- [x] Ensure `.inference(...)` accepts `InferenceSettingsJS`, not profile names or JS maps
- [x] Implement `agent.run(turn, options?)` requiring a Go-owned `Turn` wrapper
- [x] Ensure `agent.run` clones input turn, applies runtime/tool/middleware setup to an effective turn, and never mutates caller input
- [x] Implement `agent.stream(turn, options?)` with explicit turn requirement and runtime-owner-safe cancel/promise behavior (event callback shape is present; detailed event forwarding remains limited)
- [x] Implement `RunResultJS` helpers: `inputTurn`, `effectiveTurn`, `outputTurn`, `text`, `usage`, `stopReason`, `events`, `toJSON`
- [x] Add tests for echo-engine `agent.run(turn)`, explicit system block in turn, no `agent.ask`, no `agent.system`, non-mutating input turns, result turn traceability, stream promise, and registry-settings agent build; JS/Go tool and middleware-ordering coverage remains for Phase 5/6 hardening

### Phase 5: Tool, schema, turn, and multimodal message wrappers

- [x] Implement `gp.schema` builders for object, string, integer, number, boolean, array, enum, required helpers (default/min/max deferred)
- [x] Implement `gp.tool(name)` builder with `description`, `input`, `handler`, `build`
- [x] Implement `gp.toolRegistry()` wrapper with `add`, `addGo`, `list`, `call`
- [x] Implement `gp.turn()` builder with `system`, `user` string shorthand, `user(messageBuilderFn)`, `assistant`, `metadata`, `build` (`toolCall`/`toolResult` deferred)
- [x] Implement `MessageBuilder` with `text`, `imageFile`, `imageURL`, and `imageBytes`
- [x] Ensure built schema/tool/turn/message objects are Go-owned wrappers with explicit snapshots where applicable
- [x] Add construction/integration tests for schema/tool/turn/message wrappers (more invalid edge-case tests deferred)
- [x] Add multimodal image tests with codec-level assertions

### Phase 6: xgoja and host integration

- [x] Update Geppetto xgoja provider config schema for registry source configuration: `profileRegistries`, `defaultProfile`, and `allowRegistryLoad`
- [x] Add provider-side registry loading wiring and preserve host service hook; credential resolver remains deferred
- [x] Ensure `allowRegistryLoad` defaults to safe deny policy
- [x] Add provider tests for requiring Geppetto, resolving default registry profiles, and explicit registry load allow/deny policy; host-only credential resolution remains deferred

### Phase 7: Documentation, examples, and declaration generation

- [x] Update TypeScript declarations for `InferenceSettings`, `InferenceProfileNamespace`, `InferenceRegistry`, `EngineBuilder`, `AgentBuilder`, `ToolBuilder`, `SchemaNamespace`, `TurnBuilder`, `MessageBuilder`, and `RunResult`
- [x] Keep `dts_parity_test.go` passing for the current transitional export surface; final absence of removed names remains Phase 8
- [x] Add examples under `examples/js/geppetto/hardcut/`: `01_load_registry_resolve_profile.js`, `02_engine_from_registry_profile.js`, `03_agent_from_registry_profile.js`, `04_tools_and_schema.js`, `05_multimodal_turn.js`, `06_embeddings_with_registry_profile.js` (embeddings self-skips until implemented)
- [x] Document Geppetto registry YAML fields and current `default_profile_slug` runtime-loader caveat
- [x] Document that Pinocchio unified config docs are application-side and not loaded by `gp.inferenceProfiles.load(...)`

### Phase 8: Cleanup and hard-cut removal

- [x] Remove public exports for old map-first namespaces/functions from the default surface (no `gp.unsafe` shim added)
- [x] Remove docs/examples that teach old APIs as normal usage
- [x] Keep internal codecs/helpers only where needed by implementation internals; default legacy tests removed
- [x] Updated hard-cut surface tests to assert old public names stay absent (no additional legacy regression suite added)
- [x] Run focused tests: `go test ./pkg/js/modules/geppetto ./pkg/js/runtime -count=1`
- [x] Run provider/runner focused tests: `go test ./pkg/js/modules/geppetto ./pkg/js/modules/geppetto/provider ./pkg/js/runtime ./cmd/examples/geppetto-js-run -count=1`

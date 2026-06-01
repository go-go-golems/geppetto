# Tasks

## Research / design deliverables

- [x] Map current Geppetto, Pinocchio, and go-go-goja runtime/binding architecture
- [x] Review current API strengths, weaknesses, missing pieces, and confusing parts
- [x] Design opinionated Go-backed fluid builder API for JS bindings
- [x] Write intern-facing implementation guide with file references, pseudocode, and diagrams
- [x] Validate docmgr ticket and upload final bundle to reMarkable

## Implementation task list — hard-cut Geppetto registry-backed JS API

### Phase 0: Contract lock and baseline inventory

- [ ] Add API contract test for final top-level `require("geppetto")` keys: `agent`, `inferenceSettings`, `inferenceProfiles`, `turn`, `engine`, `tool`, `toolRegistry`, `embeddings`, `schema`, `events`, `unsafe`
- [ ] Add negative API inventory checks that old map-first names are absent: `turns.newTurn`, `engines.fromConfig`, `createBuilder`, `createSession`, `runInference`, ordinary `runner.run`
- [ ] Document accepted `gp.inferenceProfiles.load(...)` source forms: YAML path, `yaml:PATH`, `yaml://PATH`, SQLite path, `sqlite:PATH`, `sqlite-dsn:DSN`
- [ ] Run baseline focused tests: `go test ./pkg/js/modules/geppetto ./pkg/js/runtime -count=1`

### Phase 1: Go-owned `InferenceSettings` wrapper

- [ ] Add `api_inference_settings.go` in `geppetto/pkg/js/modules/geppetto`
- [ ] Implement `InferenceSettingsBuilderJS` with Go-owned builder state and symbolic credential reference storage
- [ ] Implement immutable/copy-on-write `InferenceSettingsJS` wrapper around cloned `*settings.InferenceSettings`
- [ ] Implement builder methods: `provider`, `model`, `credentialRef`, `baseURL`, `temperature`, `topP`, `maxTokens`, `timeoutMs`, `modelInfo`, `build`
- [ ] Validate provider/model requirements, numeric ranges, positive timeout, and non-empty symbolic credential refs
- [ ] Forbid public raw credential APIs: no `apiKey`, no `apiKeyEnv`, no `fromEnv`
- [ ] Implement `toJSON`, `clone`, and redacted `debug`/snapshot helpers
- [ ] Add tests proving snapshots are detached and mutating JS snapshots does not mutate Go-owned settings

### Phase 2: Geppetto registry loader wrapper

- [ ] Add or rewrite `api_inference_profiles.go` for the new `gp.inferenceProfiles` namespace
- [ ] Implement `gp.inferenceProfiles.load(source)` for string and string-array source inputs
- [ ] Use `engineprofiles.ParseRegistrySourceSpecs` and `engineprofiles.NewChainedRegistryFromSourceSpecs` internally
- [ ] Implement `InferenceRegistryJS` wrapper with registry reader, optional closer, and source metadata
- [ ] Implement `registry.resolve(input)` for string profile names and typed `{ registry, profile }` input snapshots
- [ ] Wrap `ResolvedEngineProfile.InferenceSettings` as `InferenceSettingsJS` with provenance metadata
- [ ] Implement `registry.listRegistries`, `registry.listProfiles`, and `registry.close`
- [ ] Implement host-default `gp.inferenceProfiles.resolve(...)` and clear error when no default registry is configured
- [ ] Add temporary registry YAML tests for `slug`, `default_profile_slug`, `profiles.default`, stacks, multiple source precedence, and invalid source errors
- [ ] Add explicit rejection/documentation test for Pinocchio unified config docs with `app:` as unsupported by `gp.inferenceProfiles.load(...)`

### Phase 3: Engine builder integration

- [ ] Add/update `EngineBuilderJS`
- [ ] Implement `gp.engine().inference(settings).build()` accepting only `InferenceSettingsJS` or trusted Go settings wrappers
- [ ] Remove/withhold public `fromConfig(map)` and map-first engine constructors
- [ ] Resolve symbolic credentials through host `CredentialResolver` only inside Go-side engine build
- [ ] Ensure raw credentials are never visible through JS snapshots/debug output
- [ ] Add tests for building engines from builder-created settings and registry-resolved settings
- [ ] Add tests rejecting plain JS objects passed as inference settings
- [ ] Add tests for missing credential resolver and redacted debug output

### Phase 4: Agent API integration — explicit turns only

- [ ] Add/update `api_agent.go`
- [ ] Implement `gp.agent()` builder methods: `name`, `inference`, `engine`, `middleware`, `goMiddleware`, `tool`, `goTool`, `toolLoop`, `events`, `runDefaults`, `build`
- [ ] Explicitly do not implement `agent.ask(prompt)`, `agent.system(prompt)`, `agent.profile(name)`, or first-pass `agent.inferenceProfile(name)`
- [ ] Implement `agent.run(turn, options?)` requiring a Go-owned `Turn` wrapper
- [ ] Ensure `agent.run` clones input turn, applies runtime/tool/middleware setup to an effective turn, and never mutates caller input
- [ ] Implement `agent.stream(turn, options?)` with explicit turn requirement and runtime-owner-safe event/cancel/promise behavior
- [ ] Implement `RunResultJS` helpers: `inputTurn`, `effectiveTurn`, `outputTurn`, `text`, `usage`, `stopReason`, `events`, `toJSON`
- [ ] Add tests for fake/echo engine `agent.run(turn)`, explicit system block in turn, no `agent.ask`, no `agent.system`, JS tools, Go tools, middleware ordering, non-mutating input turns, and result turn traceability

### Phase 5: Tool, schema, turn, and multimodal message wrappers

- [ ] Implement `gp.schema` builders for object, string, integer, number, boolean, array, enum, required/default/min/max helpers
- [ ] Implement `gp.tool(name)` builder with `description`, `input`, `handler`, `build`
- [ ] Implement `gp.toolRegistry()` wrapper with `add`, `addGo`, `list`, `call`
- [ ] Implement `gp.turn()` builder with `system`, `user` string shorthand, `user(messageBuilderFn)`, `assistant`, `toolCall`, `toolResult`, `metadata`, `build`
- [ ] Implement `MessageBuilder` with `text`, `imageFile`, `imageURL`, and `imageBytes`
- [ ] Ensure all built schema/tool/turn/message objects are Go-owned wrappers with explicit snapshots
- [ ] Add invalid construction tests for schema/tool/turn/message wrappers
- [ ] Add multimodal image tests with deterministic fake provider or codec-level assertions

### Phase 6: xgoja and host integration

- [ ] Update Geppetto xgoja provider config schema for registry source configuration: `profileRegistries`, `defaultProfile`, and optional `allowRegistryLoad`
- [ ] Add host service wiring for default registry reader, credential resolver, and approved Go tool registry
- [ ] Ensure `allowRegistryLoad` defaults to safe host policy
- [ ] Add xgoja tests for requiring Geppetto, resolving default registry profiles, explicit registry load allow/deny policy, and host-only credential resolution

### Phase 7: Documentation, examples, and declaration generation

- [ ] Update TypeScript declarations for `InferenceSettingsBuilder`, `InferenceSettings`, `InferenceProfileNamespace`, `InferenceRegistry`, `EngineBuilder`, `AgentBuilder`, `ToolBuilder`, `SchemaNamespace`, `TurnBuilder`, `MessageBuilder`, and `RunResult`
- [ ] Update `dts_parity_test.go` for final top-level exports and absence of removed names
- [ ] Add examples: `01_inference_settings_builder.js`, `02_load_registry_resolve_profile.js`, `03_engine_from_registry_profile.js`, `04_agent_from_registry_profile.js`, `05_tools_and_schema.js`, `06_multimodal_turn.js`, `07_embeddings_with_credential_ref.js`
- [ ] Document Geppetto registry YAML fields: `slug`, `default_profile_slug`, `profiles.<slug>`, `stack`, `inference_settings`
- [ ] Document that Pinocchio unified config docs are application-side and not loaded by `gp.inferenceProfiles.load(...)`

### Phase 8: Cleanup and hard-cut removal

- [ ] Remove public exports for old map-first namespaces or move intentionally to `gp.unsafe`
- [ ] Remove docs/examples that teach old APIs as normal usage
- [ ] Keep internal codecs only where needed for snapshots/import tests
- [ ] Add regression tests that old public names stay absent
- [ ] Run focused tests: `go test ./pkg/js/modules/geppetto ./pkg/js/runtime -count=1`
- [ ] Run broader Geppetto and xgoja tests if registry/core/provider wiring changed

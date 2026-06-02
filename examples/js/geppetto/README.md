# Geppetto JS Example Scripts

This directory contains runnable scripts for the hard-cut wrapper-first API exposed by:

```js
const gp = require("geppetto");
```

Legacy map/session/runner examples and public turn-run execution were removed during the clean cutover. New scripts should use:

- `gp.inferenceProfiles`
- `gp.engine()`
- `gp.agent().session()`
- `session.next().run()` / `session.next().runAsync()` for execution
- `session.fork()` and `resumeLatest()` for explicit lifecycle flows
- `gp.turnStores` for host-configured durable turn storage
- `gp.schema`
- `gp.tool()`
- `gp.toolRegistry()`

## Hard-Cut Examples

- `25_inference_profiles_load_resolve_settings.js`
- `26_engine_builder_from_registry_profile.js`
- `28_agent_from_registry_profile.js`
- `29_tools_schema_multimodal_turn.js`
- `30_real_provider_multiturn.js`
- `31_event_emitter_run_async.js`
- `32_event_emitter_progress_summary.js`
- `33_event_emitter_multiturn_run_async.js`
- `34_turn_store_persistence.js` (requires host-configured turn storage)

## Numbered Tutorial Examples

`hardcut/` contains short numbered examples matching the hard-cut documentation:

- `hardcut/01_load_registry_resolve_profile.js`
- `hardcut/02_engine_from_registry_profile.js`
- `hardcut/03_agent_from_registry_profile.js`
- `hardcut/04_tools_and_schema.js`
- `hardcut/05_multimodal_turn.js`
- `hardcut/06_embeddings_with_registry_profile.js` (self-skips until the hard-cut embeddings wrapper exists)

## Profile Registry Fixtures

`profiles/` provides runtime YAML registry fixtures (one file = one registry):

- `50-hardcut-phase123.yaml`

Older fixture files may still exist for lower-level Go tests, but the JavaScript examples above use the hard-cut fixture.

## Run the Real Provider Examples

```bash
./examples/js/geppetto/run_real_provider_multiturn.sh
```

Override the default Pinocchio registry/profile with:

```bash
GEPPETTO_PROFILE_REGISTRIES="$HOME/.config/pinocchio/profiles.yaml" \
GEPPETTO_PROFILE=default \
GEPPETTO_TIMEOUT_MS=120000 \
./examples/js/geppetto/run_real_provider_multiturn.sh
```

The wrapper calls:

```bash
go run ./cmd/examples/geppetto-js-run run \
  --script examples/js/geppetto/30_real_provider_multiturn.js \
  --profile-registries "$HOME/.config/pinocchio/profiles.yaml" \
  --profile default \
  --timeout-ms 120000
```

Run all EventEmitter + session `runAsync` examples with the smoke wrapper:

```bash
GEPPETTO_PROFILE_REGISTRIES="$HOME/.config/pinocchio/profiles.yaml" \
GEPPETTO_PROFILE=default \
./examples/js/geppetto/run_event_emitter_examples.sh
```

The wrapper checks for final JSON output rather than requiring `text-delta`, because provider streaming behavior varies.

`34_turn_store_persistence.js` demonstrates the JS API for `gp.turnStores.default()`, `agent.store(store)`, and `agent.session().store(store)`. It requires a host that enables storage and provides a TurnStore implementation; the plain `geppetto-js-run` helper does not open a store itself.

Or run individual EventEmitter + session `runAsync` examples with:

```bash
go run ./cmd/examples/geppetto-js-run run \
  --script examples/js/geppetto/31_event_emitter_run_async.js \
  --profile-registries "$HOME/.config/pinocchio/profiles.yaml" \
  --profile default \
  --timeout-ms 120000

go run ./cmd/examples/geppetto-js-run run \
  --script examples/js/geppetto/32_event_emitter_progress_summary.js \
  --profile-registries "$HOME/.config/pinocchio/profiles.yaml" \
  --profile default \
  --timeout-ms 120000

go run ./cmd/examples/geppetto-js-run run \
  --script examples/js/geppetto/33_event_emitter_multiturn_run_async.js \
  --profile-registries "$HOME/.config/pinocchio/profiles.yaml" \
  --profile default \
  --timeout-ms 120000
```

## Run a Local Example

```bash
go test ./pkg/js/modules/geppetto -run TestHardCutExamples -count=1 -v
```

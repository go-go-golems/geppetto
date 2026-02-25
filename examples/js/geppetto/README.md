# Geppetto JS Example Scripts

This directory contains runnable scripts for the JS API surface exposed by:

```js
const gp = require("geppetto");
```

## Existing Core Surface

- `01_turns_and_blocks.js`
- `02_session_echo.js`
- `03_middleware_composition.js`
- `04_tools_and_toolloop.js`
- `05_go_tools_from_js.js`
- `06_live_profile_inference.js`
- `07_context_and_constants.js`

## Profile/Schema Surface (New)

- `08_profiles_registry_inventory.js`
- `09_profiles_resolve_stack_precedence.js`
- `10_engines_from_profile_metadata.js`
- `11_profiles_resolve_explicit_registry.js`
- `12_profiles_request_overrides_policy.js`
- `13_schemas_middlewares_catalog.js`
- `14_schemas_extensions_catalog.js`
- `15_profiles_crud_sqlite.js`
- `16_mixed_registry_precedence.js`
- `17_from_profile_legacy_registry_option_error.js`
- `18_missing_profile_registry_errors.js`

## Profile Registry Fixtures

`profiles/` provides runtime YAML registry fixtures (one file = one registry):

- `10-provider-openai.yaml`
- `20-team-agent.yaml`
- `30-user-overrides.yaml`

## Run One Script

```bash
go run ./cmd/examples/geppetto-js-lab \
  --script examples/js/geppetto/08_profiles_registry_inventory.js \
  --profile-registries examples/js/geppetto/profiles/10-provider-openai.yaml,examples/js/geppetto/profiles/20-team-agent.yaml,examples/js/geppetto/profiles/30-user-overrides.yaml
```

## Run Full Suite

```bash
./examples/js/geppetto/run_profile_registry_examples.sh
```

The suite seeds a temporary sqlite profile registry and runs scripts against:

- stacked YAML registries,
- sqlite registry,
- mixed YAML + sqlite stack,
- no-profile-registry mode for expected error scripts.

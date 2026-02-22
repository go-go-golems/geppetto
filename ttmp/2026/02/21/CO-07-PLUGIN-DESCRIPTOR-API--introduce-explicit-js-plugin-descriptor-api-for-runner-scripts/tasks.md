# Tasks

## Completed

- [x] Phase 0: Design and contract definition
- [x] Defined descriptor contract (`apiVersion`, `kind`, `id`, `name`, `create`).
- [x] Defined runtime instance contract (`run(input, options)`).
- [x] Defined host context contract passed to `create(...)`.
- [x] Confirmed migration policy: descriptor-only, no legacy global fallback.

- [x] Phase 1: JS plugin SDK module (script-side ergonomics)
- [x] Added `scripts/lib/plugin_api.js` with `defineExtractorPlugin(descriptor)`.
- [x] Added validation helpers and actionable descriptor error messages.
- [x] Migrated script examples to SDK-based plugin exports.

- [x] Phase 2: Host loader and execution lifecycle (runner-side)
- [x] Added plugin loader (`cozo-relationship-js-runner/plugin_loader.go`) that:
  - loads entry module through `require(...)`
  - resolves module export as plugin descriptor
  - validates descriptor contract
  - instantiates plugin with `create(hostContext)`
  - invokes `instance.run(input, options)`
- [x] Preserved turn/timeline recording and event sink behavior around lifecycle.
- [x] Added metadata output fields for descriptor mode and plugin identity.

- [x] Phase 3: Descriptor-only migration (no compatibility bridge)
- [x] Removed CLI `--function` dispatch path.
- [x] Removed runtime fallback lookup for legacy globals (`extractRelations`, `extract`, `run`, `main`).
- [x] Updated runner docs to describe descriptor-only execution.

- [x] Phase 4: Script migration and experiments
- [x] Migrated base extractor script to descriptor export.
- [x] Migrated reflective extractor script to descriptor export.
- [x] Added ticket-local experiment scripts under `ttmp/.../scripts/` for:
  - descriptor run command template
  - malformed descriptor contract failure case

- [x] Phase 5: Documentation, diary, and task closure
- [x] Update diary with implementation steps, commands, and commit hashes.
- [x] Update changelog with commit references and behavior notes.
- [x] Relate new code/docs files in docmgr index and diary.

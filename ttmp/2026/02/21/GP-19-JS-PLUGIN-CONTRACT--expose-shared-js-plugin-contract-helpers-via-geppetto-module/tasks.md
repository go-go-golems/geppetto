# Tasks

## TODO

- [x] Phase 0: Ticket scaffolding and architecture baseline
- [x] Write implementation plan for shared plugin-contract module.
- [x] Replace placeholder tasks with phased execution checklist.
- [x] Initialize detailed diary with prompt context and assumptions.
- [x] Relate geppetto and runner files to ticket docs.

- [x] Phase 1: Geppetto module export surface
- [x] Register native module alias `geppetto/plugins` in geppetto module registration.
- [x] Expose shared constant `EXTRACTOR_PLUGIN_API_VERSION`.
- [x] Implement `defineExtractorPlugin(descriptor)` in geppetto module.
- [x] Implement `wrapExtractorRun(runImpl)` in geppetto module.

- [x] Phase 2: Conformance semantics in shared helpers
- [x] Validate descriptor fields (`apiVersion`, `kind`, `id`, `name`, `create`) with stable error messages.
- [x] Canonicalize run input in wrapper (`transcript`, `prompt`, `profile`, `timeoutMs`, `engineOptions`).
- [x] Freeze descriptor and canonical input objects.
- [x] Preserve current plugin API shape expected by runner scripts.

- [x] Phase 3: Geppetto module tests
- [x] Add module test that requires `geppetto/plugins`.
- [x] Verify descriptor helper usage in JS test snippet.
- [x] Verify wrapped run enforces transcript and timeout defaults.

- [x] Phase 4: Runner script migration
- [x] Update extractor scripts to import helpers from `require("geppetto/plugins")`.
- [x] Remove runner-local duplicate helper file `scripts/lib/plugin_api.js`.
- [x] Keep extractor logic unchanged apart from import boundary.

- [x] Phase 5: Documentation and ticket placement
- [ ] Update GP-19 changelog with commit references.
- [ ] Extend GP-19 diary with worked/failed commands and review notes.
- [ ] Move GP-19 ticket to `geppetto/ttmp/...` layout.
- [ ] Commit move + docs closure updates.

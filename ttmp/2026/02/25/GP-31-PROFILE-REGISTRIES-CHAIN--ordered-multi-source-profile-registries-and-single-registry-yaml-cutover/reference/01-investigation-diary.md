---
Title: Investigation diary
Ticket: GP-31-PROFILE-REGISTRIES-CHAIN
Status: active
Topics:
    - profile-registry
    - pinocchio
    - geppetto
    - migration
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/geppetto/ttmp/2026/02/25/GP-31-PROFILE-REGISTRIES-CHAIN--ordered-multi-source-profile-registries-and-single-registry-yaml-cutover/design-doc/01-implementation-guide-ordered-profile-registries-chain-and-single-registry-yaml-cutover.md
      Note: Main design deliverable for GP-31
    - Path: /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/geppetto/pkg/profiles/codec_yaml.go
      Note: Existing YAML format behavior analyzed for single-registry hard cut
    - Path: /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/geppetto/pkg/profiles/service.go
      Note: Existing resolution/fingerprint/provenance behavior analyzed for reuse
    - Path: /home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/pinocchio/pkg/webchat/http/profile_api.go
      Note: Existing CRUD registry selection behavior analyzed for GP-31 scope
ExternalSources: []
Summary: Diary for scoping and documenting GP-31 ordered source-chain registry loading design.
LastUpdated: 2026-02-25T17:15:00-05:00
WhatFor: Capture rationale and decisions for GP-31 planning deliverable.
WhenToUse: Use when reviewing why GP-31 is structured the way it is and what tradeoffs were accepted.
---

# Investigation diary

## Goal

Track how GP-31 design scope was converted from user proposal into a concrete implementation guide and task breakdown.

## Step 1: Define GP-31 architecture and hard-cut behavior

This step translated the requested source-chain model (`--profile-registries file1,file2,file3`) into a concrete implementation guide that is compatible with the existing stack-profile resolution engine. The focus was to preserve deterministic behavior and avoid reintroducing overlay complexity.

The design intentionally keeps source chaining as a routing/ownership concern and reuses existing stack merge + provenance + fingerprint logic unchanged.

### Prompt Context

**User prompt (verbatim):** "so i would propose something like: --profile-registries file1,file2,file3 where we automatically determine if it's yaml or sqlite, and i tink the yaml would contain the slug and the db can contain multiple registries, and then you just chose profile names and the resolution is done in the order of the registries loaded. And for CRUD, allow all registries for now. Can you write up a detailed implementation guide and make a new ticket for this? This would also mean updating our yaml config format now to only contain a single registry."

**Assistant interpretation:** create a new planning ticket and produce a detailed implementation guide for ordered multi-source registry loading and single-registry YAML runtime format.

**Inferred user intent:** move from conceptual direction to executable engineering plan that can be implemented phase-by-phase without ambiguity.

### What I did

1. Created ticket `GP-31-PROFILE-REGISTRIES-CHAIN`.
2. Added a design doc with:
   - source autodetection,
   - ordered profile resolution semantics,
   - registry ownership/write routing,
   - single-registry YAML hard cut,
   - CRUD exposure scope and risks.
3. Added granular tasks by phase (settings, loader, chain router, resolution semantics, YAML cutover, tests, docs).
4. Updated ticket index and changelog for review readiness.

### Why

1. The requested behavior affects both runtime selection and storage semantics; it required a full contract definition before coding.
2. Existing code already has mature stack merge/provenance behavior; plan should reuse this rather than create parallel composition layers.

### What worked

1. Existing service abstractions already support most of the required mechanics (registry slugs, cross-registry stack refs, fingerprint/provenance metadata).
2. A router-style source chain model cleanly supports ordered resolution without overlay merge semantics.

### What didn't work

N/A

### What I learned

1. The key complexity is not stack merge itself; it is source ownership and deterministic profile lookup rules across multiple stores.
2. Exposing all registries in CRUD while using YAML for private credentials creates clear data-exposure risk and should be called out explicitly.

### What was tricky to build

The tricky part was reconciling three constraints without introducing new complexity:

1. ordered profile search across registries,
2. single source ownership for writes,
3. no overlay abstraction.

This required explicitly separating “registry chain routing” from “stack field merge semantics.”

### What warrants a second pair of eyes

1. Startup policy for duplicate registry slugs across sources (hard fail vs tolerated-first-wins).
2. Error mapping for writes against read-only (YAML) sources in CRUD handlers.
3. Whether profile trace payloads need immediate redaction if YAML registries include secrets and CRUD read exposure stays open.

### What should be done in the future

1. Implement Phase 1-8 tasks from `tasks.md`.
2. Add a follow-up ticket for registry visibility controls/redaction if private YAML registries remain exposed in CRUD.

### Code review instructions

1. Review implementation contract in:
   - `design-doc/01-implementation-guide-ordered-profile-registries-chain-and-single-registry-yaml-cutover.md`.
2. Validate scope completeness in:
   - `tasks.md`.
3. Cross-check assumptions with current code in:
   - `/home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/geppetto/pkg/profiles/service.go`,
   - `/home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/geppetto/pkg/profiles/codec_yaml.go`,
   - `/home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/pinocchio/pkg/webchat/http/profile_api.go`.

### Technical details

1. Ticket created via `docmgr ticket create-ticket --ticket GP-31-PROFILE-REGISTRIES-CHAIN ...`.
2. Documents created via `docmgr doc add` for `design-doc` and `reference`.
3. No code changes in this step; deliverable is planning and implementation guidance.

## Step 2: Scope correction to strict stack-only runtime model

This step applied a design correction to match the requested hard-cut model more tightly. The prior draft still assumed optional registry switching and YAML `default_profile_slug`; both were removed from runtime scope.

The updated model treats loaded registry sources as a pure precedence stack and resolves profiles from stack top to bottom without runtime registry switching inputs.

### Prompt Context

**User prompt (verbatim):** "we don't need a default_profile_slug in yaml files either, that's weird. we just have the stack of profile registries and the one on top is the one used. I don't think we need to switch registries, it's just a stack."

**Assistant interpretation:** remove default profile slug semantics from runtime YAML and remove runtime registry switching from the GP-31 design.

**Inferred user intent:** simplify GP-31 to a strict hard-cut stack model and avoid extra decision surface.

### What I did

1. Updated design doc semantics to:
   - stack-top-first profile precedence,
   - no runtime registry selector in this flow,
   - runtime YAML single-registry format with `default_profile_slug` rejected.
2. Updated tasks to reflect:
   - no runtime registry switching implementation,
   - explicit validation for rejecting `default_profile_slug`,
   - tests for stack-top-first behavior.
3. Added changelog entry documenting the scope correction.

### Why

1. The user requested a stricter model with fewer knobs.
2. Removing runtime registry switching keeps behavior deterministic and simpler to reason about.
3. Removing `default_profile_slug` from runtime YAML avoids carrying an unnecessary selector mechanism in this stack model.

### What worked

1. Scope correction is now consistent across design and tasks.
2. The revised contract remains compatible with existing stack merge/provenance internals.

### What didn't work

N/A

### What I learned

1. `registry` as a runtime selector and `default_profile_slug` as a runtime default are separate complexity multipliers; dropping both simplifies the contract significantly.

### What was tricky to build

The tricky part was ensuring all planning artifacts remained internally consistent after the scope correction:

1. design semantics,
2. task phases,
3. change history/diary rationale.

### What warrants a second pair of eyes

1. Whether any existing runtime API surfaces should still carry `registry_slug` for non-GP-31 flows, or be removed/ignored consistently.
2. Whether stack-top-first should be defined as “last entry wins” vs “first entry wins” in CLI docs/examples; current design uses last entry as top.

### What should be done in the future

1. Implement GP-31 with explicit tests proving:
   - no runtime registry switching,
   - `default_profile_slug` rejection for runtime YAML sources,
   - stack-top-first resolution determinism.

## Step 3: Final consistency pass for stack-only wording

This step tightened remaining wording so GP-31 does not imply any registry switching path and consistently frames the model as a single ordered registry stack.

### Prompt Context

**User prompt (verbatim):** "we don't need a default_profile_slug in yaml files either, that's weird. we just have the stack of profile registries and the one on top is the one used. I don't think we need to switch registries, it's just a stack."

**Assistant interpretation:** keep the existing hard-cut decisions and remove any ambiguous wording that might suggest optional registry selection.

**Inferred user intent:** finalize the contract language before implementation so there is no confusion during GP-31 coding.

### What I did

1. Re-checked all GP-31 ticket docs for mentions of runtime registry switching and `default_profile_slug`.
2. Updated `index.md` summary and overview text to explicitly state stack-top-first lookup and avoid selector-style phrasing.
3. Verified that remaining `--registry` mention is only in rejected alternatives, not in proposed behavior.

### Why

1. Small wording ambiguity at planning stage can become accidental behavior during implementation.
2. This clarification aligns all ticket artifacts with the intended hard cut contract.

### What worked

1. Design doc, tasks, changelog, and index are now aligned on stack-only semantics.

### What didn't work

N/A

### What I learned

1. Eliminating selector vocabulary from user-facing docs is as important as removing selector code paths.

### What should be done in the future

1. Keep implementation PR review checklist explicit: no runtime registry selector inputs, no runtime `default_profile_slug`, stack-top-first profile lookup only.

## Step 4: Implement source-chain core in geppetto profiles

This step implemented the GP-31 runtime core inside `geppetto/pkg/profiles`: source parsing/autodetection, strict runtime YAML loading, and chained registry routing. The objective was to make stack-based runtime behavior executable before touching command wiring.

The implementation introduced a dedicated chained registry service that composes existing stack merge/provenance logic with owner-routed writes and source-level read-only policy.

### Prompt Context

**User prompt (verbatim):** "alright, now implement it, task by task, committing as you go, and keeping a diary"

**Assistant interpretation:** execute GP-31 implementation in phased commits and keep a detailed ticket diary.

**Inferred user intent:** move from design docs to production code without batching everything into one opaque change.

**Commit (code):** `c88a1e3` — `profiles: add source-chain registry service and strict runtime YAML loader`

### What I did

1. Added `pkg/profiles/source_chain.go`:
   - `ParseProfileRegistrySourceEntries`,
   - `ParseRegistrySourceSpecs`,
   - autodetection for `yaml`/`sqlite`/`sqlite-dsn`,
   - `ChainedRegistry` implementing `profiles.Registry`,
   - top-of-stack profile lookup when registry is not specified,
   - owner write routing + read-only enforcement.
2. Added `pkg/profiles/codec_yaml_runtime.go`:
   - strict runtime YAML decode for single-registry files,
   - explicit rejection of:
     - `registries:` bundle format,
     - legacy profile-map format,
     - `default_profile_slug`.
3. Added `pkg/profiles/source_chain_test.go` with parser, strict YAML, mixed-source load, top-precedence, and write-routing assertions.

### Why

1. GP-31 requires deterministic behavior across mixed sources before CLI wiring can be safely migrated.
2. Implementing routing in one place avoids duplicating merge logic in pinocchio/web-chat code.

### What worked

1. Existing `StoreRegistry` stack resolution was reusable as the chain read engine.
2. Mixed YAML+SQLite chain behavior was validated end-to-end in one unit/integration-style test.

### What didn't work

1. First commit attempt failed pre-commit lint:
   - command: `git commit -m "profiles: add source-chain registry service and strict runtime YAML loader"`,
   - error: `S1011: should replace loop with precedenceTopFirst = append(precedenceTopFirst, owner.registrySlugs...)`.
2. Fixed by replacing the loop with append-slice form and recommitting.

### What I learned

1. The chain abstraction stays simple if read composition and write ownership are treated as separate concerns.

### What was tricky to build

The hardest part was preserving stack resolution semantics while changing root profile lookup rules:

1. profile stack merge still relies on resolved root registry/profile,
2. runtime selection needed a new stack-top-first search path when registry is unspecified,
3. explicit registry operations still needed to work for CRUD/write routing.

### What warrants a second pair of eyes

1. Duplicate registry slug startup error text quality and operator diagnostics.
2. Edge cases where a source loads zero registries (empty sqlite/yaml files).

### What should be done in the future

1. Add explicit duplicate-slug unit test coverage (task still open).

### Code review instructions

1. Start in:
   - `/home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/geppetto/pkg/profiles/source_chain.go`
   - `/home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/geppetto/pkg/profiles/codec_yaml_runtime.go`
2. Validate with:
   - `go test ./pkg/profiles -count=1`

### Technical details

1. Source detection uses extension-first (`.db/.sqlite/.sqlite3`), then SQLite header probe (`SQLite format 3`), then YAML fallback.
2. Write failures for read-only sources return `ErrReadOnlyStore`.

## Step 5: Cut geppetto middleware over to profile-registry stacks

This step replaced runtime `profile-file` behavior in geppetto section middleware with strict `profile-settings.profile-registries`. The middleware now opens a chain service for profile patch resolution and fails fast when no registry sources are configured.

This completed the geppetto-side hard cut for profile selection input wiring.

### Prompt Context

**User prompt (verbatim):** (same as Step 4)

**Assistant interpretation:** continue phased implementation and commit each completed phase.

**Inferred user intent:** enforce the new runtime contract through actual middleware behavior.

**Commit (code):** `683fc10` — `sections: require profile-registries and load profile stack middleware`

### What I did

1. Updated `pkg/sections/sections.go`:
   - bootstrap profile settings now parse `profile` + `profile-registries`,
   - runtime fails when `profile-registries` is empty,
   - removed runtime `profile-file` fallback path.
2. Updated `pkg/sections/profile_registry_source.go`:
   - loader now resolves profile patches from `ChainedRegistry` sources.
3. Reworked `pkg/sections/profile_registry_source_test.go`:
   - stack-top precedence tests,
   - hard-cut missing-source failure test,
   - middleware precedence test (`config < profile < env < flags`) with `--profile-registries`.

### Why

1. GP-31 changes are not real until command middleware consumes the new setting surface.
2. Hard-cut behavior should fail early at command parse time instead of silently falling back.

### What worked

1. Bootstrap parse pattern remained valid after changing profile source inputs.
2. Section tests captured both precedence and hard-cut validation behavior.

### What didn't work

N/A

### What I learned

1. The fastest migration path was adding a dedicated bootstrap section for `profile-registries` while preserving existing command-profile parsing for `profile`.

### What was tricky to build

The tricky part was preserving Glazed middleware precedence while replacing a core input contract:

1. bootstrap parse order had to stay consistent with runtime middleware order,
2. profile loading must remain between config and defaults in effective precedence.

### What warrants a second pair of eyes

1. Whether any CLI entrypoints outside pinocchio/web-chat still rely on `profile-file` assumptions.

### What should be done in the future

1. Add `--print-parsed-fields` smoke coverage for `--profile-registries` (task still open).

### Code review instructions

1. Start in:
   - `/home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/geppetto/pkg/sections/sections.go`
   - `/home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/geppetto/pkg/sections/profile_registry_source.go`
2. Validate with:
   - `go test ./pkg/sections -count=1`

### Technical details

1. `profile-settings.profile-registries` is parsed as comma-separated source entries and normalized before chain creation.

## Step 6: Cut pinocchio/web-chat over to chained registries and no runtime registry switching

This step migrated pinocchio runtime surfaces to GP-31 behavior:

1. CLI root now accepts `--profile-registries`,
2. web-chat bootstraps profile service from chain sources,
3. request-time registry selection (`registry_slug`) is removed from runtime resolver behavior.

The result aligns web-chat request resolution with stack-only profile selection by slug.

### Prompt Context

**User prompt (verbatim):** (same as Step 4)

**Assistant interpretation:** continue implementation across dependent consumers, not only geppetto core.

**Inferred user intent:** finish end-to-end runtime behavior and not leave pinocchio behind.

**Commit (code):** `0108628` — `web-chat: load profile registry chains and remove runtime registry switching`

### What I did

1. Updated pinocchio CLI root:
   - added persistent `--profile-registries` flag in `/home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/pinocchio/cmd/pinocchio/main.go`.
2. Updated web-chat startup:
   - replaced `profile-registry-dsn/db` bootstrap with `profile-registries` chain loading.
3. Updated web-chat resolver:
   - removed runtime registry selector path from request resolution,
   - resolver now selects by profile slug only and delegates stack search to chain service.
4. Updated profile API error mapping:
   - `ErrReadOnlyStore` now returns `403`.
5. Updated resolver tests:
   - removed assumptions of runtime registry switching,
   - added/adjusted tests for stack lookup behavior and ignored registry selector inputs.

### Why

1. GP-31 runtime contract includes no request-time registry switching.
2. Consumers needed to adopt source-chain loading to avoid split behavior between CLI and web-chat.

### What worked

1. Existing web-chat tests adapted cleanly once fixtures switched to chained registry sources.
2. Pre-commit full pinocchio test/lint suite passed with the new behavior.

### What didn't work

N/A

### What I learned

1. Runtime selector removal is easiest to enforce by removing the resolution branch entirely, not by keeping parsing and ignoring at call sites.

### What was tricky to build

The tricky part was test fixture migration:

1. old tests used multi-registry in-memory services with explicit registry selection,
2. new behavior required chain-backed fixtures and revised expectations for `registry_slug` inputs.

### What warrants a second pair of eyes

1. web-chat CRUD/list semantics still default to one registry unless a `registry` selector is provided; verify whether GP-31 should expose aggregate listing in this phase.

### What should be done in the future

1. Decide whether profile API should include an explicit “list registries” endpoint or aggregate profile listing for multi-registry visibility.

### Code review instructions

1. Start in:
   - `/home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/pinocchio/cmd/web-chat/main.go`
   - `/home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/pinocchio/cmd/web-chat/profile_policy.go`
   - `/home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/pinocchio/pkg/webchat/http/profile_api.go`
2. Validate with:
   - `go test ./cmd/pinocchio ./cmd/web-chat ./pkg/webchat/http -count=1`

### Technical details

1. Read-only source writes now fail as `403` through profile API mapping.
2. Runtime resolver still accepts payloads containing `registry_slug`, but it no longer uses them for profile selection.

## Step 7: Add explicit duplicate-slug chain startup test

This step closed a remaining validation gap by adding focused coverage for duplicate registry slug rejection across source chains.

### Prompt Context

**User prompt (verbatim):** (same as Step 4)

**Assistant interpretation:** keep implementing remaining open checklist items, not only the large architecture pieces.

**Inferred user intent:** ensure the hard-cut runtime model is fully validated by tests.

**Commit (code):** `bc338dd` — `profiles: add duplicate registry slug chain test`

### What I did

1. Added `TestChainedRegistry_RejectsDuplicateRegistrySlugsAcrossSources` in:
   - `/home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/geppetto/pkg/profiles/source_chain_test.go`.
2. Marked the corresponding GP-31 task as completed.

### Why

1. Duplicate slug rejection is a critical startup invariant for deterministic routing.

### What worked

1. Coverage now explicitly guards the startup error path that was previously implicit in implementation code.

### What didn't work

N/A

### What I learned

1. Explicit invariant tests prevent accidental “first source wins” regressions in future refactors.

### What was tricky to build

Very low complexity; the only care point was generating two valid runtime YAML source files with identical slugs.

### What warrants a second pair of eyes

1. Error message wording consistency for operator-facing diagnostics across all startup failure paths.

### What should be done in the future

N/A

### Code review instructions

1. Review:
   - `/home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/geppetto/pkg/profiles/source_chain_test.go`
2. Validate:
   - `go test ./pkg/profiles -count=1`

### Technical details

1. Test asserts `NewChainedRegistryFromSourceSpecs` error contains `duplicate registry slug`.

## Step 8: Expand web-chat profile API list/get across loaded registries

This step addressed GP-31 CRUD-read scope by making profile list/get work across all loaded registries when callers do not specify a `registry` selector.

The implementation preserved list response shape to avoid contract regressions in existing app-owned integration tests.

### Prompt Context

**User prompt (verbatim):** (same as Step 4)

**Assistant interpretation:** keep closing remaining GP-31 runtime gaps in consumer surfaces.

**Inferred user intent:** make multi-registry runs operational in web-chat without requiring registry selection for read paths.

**Commit (code):** `10815ea` — `web-chat: list and get profiles across loaded registries`

### What I did

1. Updated `/home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/pinocchio/pkg/webchat/http/profile_api.go`:
   - `/api/chat/profiles` GET now aggregates from all loaded registries when `registry` is absent,
   - `/api/chat/profiles/{slug}` GET now searches across loaded registries when `registry` is absent.
2. Added/updated tests in:
   - `/home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/pinocchio/cmd/web-chat/profile_policy_test.go`
   - `TestProfileAPI_ListAndGetAcrossLoadedRegistriesWhenRegistryUnset`.

### Why

1. Operators should be able to inspect loaded profiles without knowing registry ownership upfront.

### What worked

1. Cross-registry read behavior now works with chain-backed resolver fixtures.
2. Existing contract-shape guard tests remained green after preserving list item schema.

### What didn't work

1. Initial implementation added a new `registry` field to list items and broke contract tests:
   - failing test: `TestAppOwnedProfileAPI_CRUDLifecycle_ContractShape`,
   - error: `unexpected profile API contract key: registry`.
2. Fixed by keeping list item schema unchanged and returning only slug-scoped entries.

### What I learned

1. GP-31 behavior changes still need to respect established HTTP contract guardrails unless explicitly versioned.

### What was tricky to build

The tricky part was balancing multi-registry visibility with compatibility:

1. needed aggregate behavior for list/get,
2. could not expand list item schema without failing downstream contract tests.

### What warrants a second pair of eyes

1. Ambiguity for duplicate slugs across registries in list responses (shape intentionally unchanged for now).

### What should be done in the future

1. Consider adding an explicit “list registries” endpoint or versioned list schema with registry identifiers.

### Code review instructions

1. Review:
   - `/home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/pinocchio/pkg/webchat/http/profile_api.go`
   - `/home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/pinocchio/cmd/web-chat/profile_policy_test.go`
2. Validate:
   - `go test ./cmd/web-chat ./pkg/webchat/http -count=1`

## Step 9: Add parsed-fields coverage and validate multi-source cutover smoke workflow

This step closed the remaining GP-31 runtime validation gap around `pinocchio --print-parsed-fields` with `--profile-registries`, and updated the operator smoke script to exercise stacked YAML+SQLite sources end-to-end.

I also ran the smoke script repeatedly and fixed two real defects discovered during execution, instead of only updating docs/tests.

### Prompt Context

**User prompt (verbatim):** (same as Step 4)

**Assistant interpretation:** complete remaining GP-31 implementation tasks with real validation and frequent diary updates.

**Inferred user intent:** finish hard-cut rollout details with executable operator workflows, not only architecture code.

**Commit (code):** `c8fcdef` — `profiles: add profile-registries parsed-fields coverage and cutover smoke script`

### What I did

1. Added integration coverage in:
   - `/home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/pinocchio/cmd/pinocchio/main_profile_registries_test.go`
   - test shells out to `go run ./cmd/pinocchio ... --print-parsed-fields --profile-registries ...` and asserts:
     - profile section present,
     - `mode: profile-registry-stack`,
     - `profileRegistries` metadata includes source path,
     - profile-derived `ai-engine` value is applied.
2. Reworked smoke script:
   - `/home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/pinocchio/scripts/profile_registry_cutover_smoke.sh`
   - now performs:
     - legacy YAML backup,
     - legacy -> bundle migration,
     - bundle import into SQLite DB,
     - generation of top runtime single-registry YAML,
     - web-chat startup with `--profile-registries <db>,<top-yaml>`,
     - `/chat` and invalid runtime smoke checks,
     - pinocchio `--print-parsed-fields` source metadata assertions.
3. Updated pinocchio runtime docs:
   - `/home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/pinocchio/pkg/doc/topics/webchat-profile-registry.md`
   - `/home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/pinocchio/pkg/doc/topics/webchat-http-chat-setup.md`
   - `/home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/pinocchio/cmd/web-chat/README.md`
   - removed request-time `registry_slug` runtime selector guidance and switched examples to `--profile-registries`.

### Why

1. GP-31 task matrix still had an open parsed-fields coverage item and smoke script/doc closeout items.
2. The user explicitly asked for runnable, practical workflows and verification.

### What worked

1. Targeted tests passed:
   - `go test ./cmd/pinocchio ./cmd/web-chat ./pkg/webchat/http -count=1`
2. End-to-end smoke run passed:
   - `scripts/profile_registry_cutover_smoke.sh --port 18125`

### What didn't work

1. First smoke run failed:
   - error: `ERROR: default /chat response metadata does not match top stack profile`
   - cause: default profile resolution searches for slug `default`; generated top registry only had `stack-top`, so default came from base DB registry.
   - fix: changed assertion to expect base registry/profile on default `/chat`.
2. Second smoke run failed:
   - error: `rg: unrecognized flag -`
   - cause: grep pattern started with `-` and was parsed as an option.
   - fix: used `rg --` to terminate options before the literal pattern.
3. First test approach using in-process `rootCmd.SetOut/SetErr` was unreliable:
   - command output still bypassed buffers.
   - fix: switched to subprocess execution (`go run ./cmd/pinocchio ...`) and captured combined output.

### What I learned

1. Default profile behavior in stack mode is slug-driven (`default`), not “always top source.”
2. A subprocess-based assertion is more robust for this CLI path than embedding root command execution.

### What was tricky to build

The tricky part was validating parsed-fields output without contaminating the test with user-level config:

1. command output includes merged config/env/profile logs,
2. default config discovery can inject local machine values,
3. test needed deterministic source paths and metadata assertions.

Approach:

1. set `XDG_CONFIG_HOME` to a test temp directory,
2. pass explicit temporary config and registry files,
3. assert only stable metadata markers and expected resolved value.

### What warrants a second pair of eyes

1. Migration command output remains bundle-shaped (`registries:`), while runtime YAML loader is strict single-registry; verify operator docs keep this distinction clear.

### What should be done in the future

1. Add a dedicated converter/export command from bundle YAML to single-registry runtime YAML for zero-manual split workflows.

### Code review instructions

1. Review:
   - `/home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/pinocchio/cmd/pinocchio/main_profile_registries_test.go`
   - `/home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/pinocchio/scripts/profile_registry_cutover_smoke.sh`
   - `/home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/pinocchio/pkg/doc/topics/webchat-profile-registry.md`
   - `/home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/pinocchio/pkg/doc/topics/webchat-http-chat-setup.md`
   - `/home/manuel/workspaces/2026-02-24/geppetto-profile-registry-js/pinocchio/cmd/web-chat/README.md`
2. Validate:
   - `go test ./cmd/pinocchio ./cmd/web-chat ./pkg/webchat/http -count=1`
   - `scripts/profile_registry_cutover_smoke.sh --port 18125`

## Related

- `GP-28-STACK-PROFILES`
- `GP-29-PINOCCHIO-STACK-PROFILE-CUTOVER`

---
Title: Investigation diary
Ticket: GP-45-REMOVE-LEGACY-UNUSED-FUNCTIONALITY
Status: active
Topics:
    - geppetto
    - architecture
    - cleanup
    - migration
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/pkg/inference/engine/run_with_result.go
      Note: |-
        Investigated for migration-era scalar metadata mirroring.
        Diary evidence for migration-era scalar metadata mirroring
    - Path: geppetto/pkg/profiles/extensions.go
      Note: |-
        Investigated for lightly integrated extension normalization.
        Diary evidence for lightly integrated extension code
    - Path: geppetto/pkg/profiles/middleware_extensions.go
      Note: |-
        Investigated for test-only middleware-extension support.
        Diary evidence for test-oriented middleware extension helpers
    - Path: geppetto/pkg/steps/ai/runtimeattrib/runtimeattrib.go
      Note: |-
        Investigated for legacy-shape support.
        Diary evidence for legacy runtime metadata support
    - Path: /home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/pkg/cmds/helpers/parse-helpers.go
      Note: |-
        Downstream fallout fix after removing the dead Geppetto profile flag bridge.
        Diary evidence for Pinocchio build repair tied to GP-45 cleanup
ExternalSources: []
Summary: Chronological record of the broader geppetto legacy and unused functionality audit.
LastUpdated: 2026-03-18T02:34:00-04:00
WhatFor: Use this diary to understand how the broader cleanup candidates were identified and classified.
WhenToUse: Use when continuing GP-45 or reviewing whether a cleanup candidate is truly legacy, unused, or just under-documented.
---


# Diary

## Goal

Capture how the broader legacy and unused-functionality inventory was assembled after the RuntimeKeyFallback review.

## Step 1: Classify the rest of the suspicious code instead of deleting by instinct

After identifying `RuntimeKeyFallback` as removable, I widened the review to adjacent compatibility layers and complexity hotspots. The goal was to avoid the common failure mode of cleanup work: deleting everything that "looks old" without distinguishing between active compatibility support, underused infrastructure, and merely confusing documentation.

I grouped the findings into four buckets: explicit backward compatibility, stale documentation, likely unused helpers, and over-complex but still possibly intentional machinery. That classification became the structure of the design doc and the phased task list.

### Prompt Context

**User prompt (verbatim):** "ok, make a docmgr ticket to remove RuntimeKeyFallback, with an implementation plan and a detailed list of tasks, and then a second one for removing legacy / unused functionality, and do a detailed write up of the other things you found. 

Create a detailed analysis / design / implementation guide that is very detailed for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file
  references.
  It should be very clear and detailed. Store in the ticket and the nupload to remarkable."

**Assistant interpretation:** Create a second documentation ticket that turns the non-RuntimeKey findings into a detailed, execution-ready cleanup inventory and implementation guide.

**Inferred user intent:** Preserve the broader technical-debt review as a durable artifact so future cleanup work can be staged deliberately instead of rediscovered.

**Commit (code):** N/A

### What I did

- Re-read the runtime metadata path in `pkg/steps/ai/runtimeattrib/runtimeattrib.go`.
- Re-read migration-era inference metadata mirroring in `pkg/inference/engine/run_with_result.go`.
- Re-read profile bootstrapping shims in `pkg/sections/profile_registry_source.go` and `pkg/sections/sections.go`.
- Checked `pkg/profiles/adapters.go` usage.
- Checked extension normalization and middleware-extension projection usage.
- Checked stack-trace generation and chained-registry fields for likely unused bookkeeping.
- Cross-checked code findings against docs that still describe removed features.

### Why

- I wanted the second ticket to distinguish "delete now" candidates from "grep downstream first" candidates.
- I wanted the implementation plan to be safe for a new intern to follow.

### What worked

- Grouping by category made the ticket much clearer than a flat list of complaints.
- The file-level usage searches were enough to identify several high-confidence stale or weakly integrated areas.
- The docs drift was especially easy to prove because the code now rejects behaviors that the docs still promote.

### What didn't work

- Some candidates, especially extension-related machinery, were not conclusively removable from in-repo evidence alone.
- That means the ticket needed to be explicit about uncertainty rather than pretending every finding was ready for deletion.

### What I learned

- Documentation drift is one of the largest remaining sources of legacy complexity in this repository.
- Some of the suspicious code is not "old product behavior" so much as "future-facing infrastructure that never fully integrated."
- The best cleanup plan is phased, not maximalist.

### What was tricky to build

- The tricky part was tone and categorization. It is easy to over-rotate into "this looks unused, delete it." I instead labeled confidence and risk in the design doc so the work can be sequenced rationally.

### What warrants a second pair of eyes

- Any deletion in `extensions.go` or `middleware_extensions.go`.
- Any deletion of legacy metadata compatibility before checking downstream readers and writers.
- Whether `profile.stack.trace` is relied on as stable API output.

### What should be done in the future

- Split GP-45 into smaller follow-up changes if implementation starts to mix unrelated risk classes.

### Code review instructions

- Read the design doc category by category.
- For each candidate, verify the grep-based evidence and confirm whether the proposed phase matches the risk level.
- If implementation begins, require a before/after search for downstream consumers.

### Technical details

- Representative commands:

```bash
rg -n "requestOverrides|allow_overrides|allowed_override_keys|denied_override_keys" geppetto/pkg/doc geppetto/pkg/js/modules/geppetto geppetto/pkg/profiles
rg -n "RegistrySlugFromString|ProfileSlugFromString|RuntimeKeyFromString|RegistrySlugToString|ProfileSlugToString|RuntimeKeyToString" geppetto
rg -n "NormalizeProfileExtensions|ProjectRuntimeMiddlewareConfigsToExtensions|MiddlewareConfigFromExtensions" geppetto
rg -n "aggregateStore|registryOwners" geppetto/pkg/profiles/source_chain.go geppetto/pkg/profiles/source_chain_test.go
```

## Step 2: Turn the audit into a concrete removal queue and land the obvious dead code

The next step was to stop treating GP-45 as a broad “maybe later” audit and turn it into a reviewable execution queue. I split the work into low-risk hard cuts first, then medium-risk compatibility cleanup only after confirming live usage. That made it possible to actually delete code instead of leaving the ticket as a parking lot.

The first landed slice removed things that had effectively become dead symbols: the no-op profile flag bridge, the unused profile adapter wrappers, and the removed-but-still-exported JS `engines.fromProfile` API surface.

### Prompt Context

**User prompt (verbatim):** "Ok, make detailed tasks in the ticket to tackle all of them. then implement"

**Assistant interpretation:** Turn GP-45 into an actionable task board and immediately start landing the safe cleanup slices instead of leaving them at the design stage.

**Inferred user intent:** Move from analysis to execution, but keep the work staged and documented so each deletion is defensible.

**Commit (code):** `e10d584` — `remove js engines fromprofile api`

### What I did

- Rewrote `tasks.md` around a concrete removal queue.
- Removed the dead no-op bridge:
  - `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/sections/profile_registry_source.go`
  - `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/sections/profile_registry_source_test.go`
- Removed the dead profile adapter wrappers:
  - `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/adapters.go`
  - `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/adapters_test.go`
- Removed `gp.engines.fromProfile` entirely from:
  - `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/js/modules/geppetto/module.go`
  - `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/js/modules/geppetto/api_engines.go`
  - `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/js/modules/geppetto/module_test.go`
  - `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/examples/js/geppetto/README.md`
  - `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/examples/js/geppetto/run_profile_registry_examples.sh`
  - `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/examples/js/geppetto/18_missing_profile_registry_errors.js`
  - `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/doc/topics/13-js-api-reference.md`
  - `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/doc/topics/14-js-api-user-guide.md`
- Deleted the dead JS removal-example file:
  - `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/examples/js/geppetto/17_from_profile_legacy_registry_option_error.js`
- Validated with:

```bash
cd geppetto && go test ./pkg/sections ./pkg/profiles ./pkg/js/modules/geppetto ./cmd/examples/geppetto-js-lab -count=1
cd geppetto && ./.bin/golangci-lint run ./pkg/js/modules/geppetto ./pkg/sections ./pkg/profiles ./cmd/examples/geppetto-js-lab
```

### Why

- `GatherFlagsFromProfileRegistry` was already a no-op and had no remaining callers.
- `pkg/profiles/adapters.go` had no in-repo uses outside its own test.
- `engines.fromProfile` had already been hard-cut semantically; keeping it around only preserved dead API surface and dead docs/examples.

### What worked

- The search-based evidence was clean: once I removed those files and references, the corresponding greps went to zero.
- The focused package tests and lint passed before commit, and the full pre-commit hook also passed.

### What didn't work

- I accidentally started two Geppetto commits in parallel while trying to split the changes more finely. One commit won and the other failed with a ref-lock race:

```text
fatal: cannot lock ref 'HEAD': is at e10d584... but expected 65e8f94...
```

- No code was lost; it just meant the low-risk removals landed as one combined slice instead of two smaller commits.

### What I learned

- A removed API should not linger as a hard-error stub for long. Once the migration note is documented and callers are gone, the symbol itself becomes noise.
- GP-45 is more useful when phrased as “prove deadness, then delete” rather than “clean up vaguely legacy things.”

### What was tricky to build

The trickiest part was separating “removed but still useful to mention in docs” from “removed and now actively harmful to keep exported.” `engines.fromProfile` had crossed that line already: the symbol itself made the JS surface look larger than it really is, and the test/example burden was no longer buying compatibility.

### What warrants a second pair of eyes

- The JS docs should be reviewed once more to ensure they no longer imply a profile-to-engine shortcut exists anywhere in the module surface.

### What should be done in the future

- Move on to the remaining safe medium-risk cut: narrow the mutable profile-store API so Geppetto core stops advertising profile deletion/default-mutation helpers it no longer uses.

### Code review instructions

- Start with:
  - `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/js/modules/geppetto/module.go`
  - `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/js/modules/geppetto/api_engines.go`
  - `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/sections/`
  - `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/adapters.go`
- Re-run the focused commands shown above.

### Technical details

- After this step, there are no remaining in-repo references to:
  - `GatherFlagsFromProfileRegistry`
  - `RegistrySlugFromString`
  - `ProfileSlugFromString`
  - `RuntimeKeyFromString`
  - `RegistrySlugToString`
  - `ProfileSlugToString`
  - `RuntimeKeyToString`
  - `gp.engines.fromProfile`

## Step 3: Narrow the mutable profile-store API and stop advertising dead mutation helpers

The next safe cut was in the persistence layer. After GP-41, Geppetto’s registry/service layer is read-only, but the store interface still exposed `DeleteProfile` and `SetDefaultProfile`. Those methods no longer had live production callers outside the store implementations and tests, so they were a good candidate for removal.

This is a useful GP-45 pattern: if a capability no longer exists at the public service layer and no internal runtime path uses it, keeping it in the persistence abstraction just preserves the old architecture in miniature.

### Prompt Context

**User prompt (verbatim):** (same as Step 2)

**Assistant interpretation:** Continue landing the concrete GP-45 cleanup slices, not just the easiest dead-symbol deletions.

**Inferred user intent:** Make the persistence layer reflect the simplified read-only registry model rather than leaving dead mutation affordances behind.

**Commit (code):** `1fd2013` — `narrow mutable profile store api`

### What I did

- Removed `DeleteProfile` and `SetDefaultProfile` from:
  - `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/store.go`
  - `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/memory_store.go`
  - `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/sqlite_store.go`
  - `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/file_store_yaml.go`
- Reworked tests so default-profile changes happen through explicit registry upserts instead of hidden helper methods:
  - `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/memory_store_test.go`
  - `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/sqlite_store_test.go`
  - `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/file_store_yaml_test.go`
- Removed closed-store assertions for APIs that no longer exist.
- Validated with:

```bash
cd geppetto && go test ./pkg/profiles -count=1
cd geppetto && ./.bin/golangci-lint run ./pkg/profiles
```

### Why

- The service layer no longer exposes per-profile mutation or default-profile switching.
- Keeping those mutation helpers alive in the store interface suggested an architecture Geppetto no longer wants to support.
- Explicit registry upserts are enough for tests, fixtures, and seeding.

### What worked

- The only real consumers were the store implementations and tests, so the removal stayed localized.
- Rephrasing the tests around explicit registry upserts made the intended model clearer.
- The full commit hook still passed after the interface shrink.

### What didn't work

- The first focused `go test ./pkg/profiles` run failed because one rewritten test reused an existing `reg` variable with `:=`:

```text
pkg/profiles/sqlite_store_test.go:464:15: no new variables on left side of :=
```

- The fix was a simple reassignment:

```go
reg, ok, err = reloaded.GetRegistry(...)
```

### What I learned

- Dead mutation APIs often survive longest in tests, because tests naturally want shortcuts. Removing them forces the tests to demonstrate the real intended flow instead.
- The store layer is now much closer to the actual architecture: read, upsert registry, upsert profile, delete registry.

### What was tricky to build

The subtle part was deciding whether removing these methods would accidentally eliminate useful test setup power. It did not. The tests can still express all necessary state transitions by cloning/upserting registries and profiles directly. That is a better reflection of the post-GP-41 design than bespoke helper methods for “delete one profile” or “flip the default.”

### What warrants a second pair of eyes

- The remaining `DeleteRegistry` capability should be reviewed later to decide whether it still belongs in Geppetto core or should also move behind an app-owned mutable layer.

### What should be done in the future

- Leave the inference/runtime metadata compatibility path for a separate decision pass. The cross-repo grep shows that part is still live, unlike the dead store mutation helpers.

### Code review instructions

- Start with:
  - `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/store.go`
  - `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/memory_store.go`
  - `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/sqlite_store.go`
  - `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/file_store_yaml.go`
- Validate with the focused package commands shown above.

### Technical details

- The live usage check for the still-pending compatibility paths found:
  - `AddRuntimeAttributionToExtra(...)` is still called by the OpenAI, OpenAI Responses, Claude, and Gemini engines.
  - `MirrorLegacyInferenceKeys(...)` is still wired through `PersistInferenceResult(...)` and `RunInferenceWithResult(...)`.
  - Pinocchio still reads older runtime metadata variants in chat persistence code.

## Step 4: Finish the low-risk sweep and repair the first downstream fallout immediately

The next GP-45 slice was the last low-risk cleanup item: remove stale documentation and JS typings that still taught removed profile-mutation or migration-era configuration knobs. At the same time, the first real downstream fallout appeared: Pinocchio still called the deleted `GatherFlagsFromProfileRegistry` bridge, so `go build ./...` broke there.

I treated that as part of the same cleanup step rather than a separate surprise. Once a Geppetto migration shim is removed, fixing the direct downstream compile break is part of finishing the cut properly, not optional cleanup.

### Prompt Context

**User prompt (verbatim):** "go build ./...
# github.com/go-go-golems/pinocchio/pkg/cmds/helpers
pkg/cmds/helpers/parse-helpers.go:88:22: undefined: geppetto_sections.GatherFlagsFromProfileRegistry

btw"

**Assistant interpretation:** Repair the first downstream compile failure caused by GP-45, then finish the remaining low-risk stale-doc and dead-typing cleanup in Geppetto.

**Inferred user intent:** Keep the cleanup moving without leaving the monorepo broken or the public docs/types teaching options that no longer exist.

**Commit (code):** `c37f952` — `clean gp-45 stale docs and js typings`

### What I did

- Fixed the downstream Pinocchio compile break in:
  - `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/pkg/cmds/helpers/parse-helpers.go`
- Committed the downstream repair as:
  - `pinocchio` `82bf805` — `drop dead profile flag bridge call`
- Removed stale doc/help text and dead JS typings from:
  - `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/doc/playbooks/06-operate-sqlite-profile-registry.md`
  - `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/doc/topics/00-docs-index.md`
  - `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/doc/topics/01-profiles.md`
  - `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/doc/topics/07-tools.md`
  - `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/doc/topics/09-middlewares.md`
  - `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/doc/topics/13-js-api-reference.md`
  - `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/doc/types/geppetto.d.ts`
  - `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/js/modules/geppetto/spec/geppetto.d.ts.tmpl`
  - `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/steps/ai/settings/flags/chat.yaml`
- Removed the dead JS `allowedTools` type/documentation field, which no longer had runtime support after the core `AllowedTools` removal.
- Validated with:

```bash
cd pinocchio && go test ./pkg/cmds/helpers -count=1
cd pinocchio && go build ./...
cd geppetto && rg -n "allowedTools\\?: string\\[\\]|ToolConfig\\.AllowedTools|Tool allowlist|GatherFlagsFromProfileRegistry|fromProfile|PolicySpec|RuntimeKeyFallback|request_overrides|writable registry|mutable registry" pkg/doc pkg/js examples pkg/steps/ai/settings --glob '!**/ttmp/**'
cd geppetto && go test ./pkg/js/modules/geppetto ./pkg/doc/... -count=1
```

### Why

- A hard cut is not finished until the direct downstream consumer break is repaired.
- Leaving stale docs and typings behind would keep removed configuration concepts alive even after the runtime paths were gone.
- The dead `allowedTools` JS type was exactly the kind of migration residue GP-45 is meant to remove.

### What worked

- Pinocchio only had one remaining caller of the removed bridge, so the downstream repair was localized and safe.
- The stale-term grep dropped to a single intentional hit: `tools.createRegistry()` still returns a mutable tool registry object, which is real and unrelated to profile-registry cleanup.
- Focused Geppetto tests passed immediately once the docs/types were updated.

### What didn't work

- The failure itself was the signal that one downstream usage had been missed in the original removal pass:

```text
pkg/cmds/helpers/parse-helpers.go:88:22: undefined: geppetto_sections.GatherFlagsFromProfileRegistry
```

- The Pinocchio commit hook was noisier than expected because it rebuilt frontend assets and reran the full lint pipeline, which made a one-line removal look larger operationally than it was.

### What I learned

- Removing a dead symbol in Geppetto is not enough; I need an immediate downstream grep or build pass in Pinocchio for the first consumer layer.
- JS typings and docs can retain dead product vocabulary long after the runtime code has been simplified, so they need to be treated as first-class cleanup targets.

### What was tricky to build

The tricky part was deciding whether the remaining grep hit for "mutable registry" meant the stale-doc sweep was incomplete. It did not. That hit is the JS tool registry, which is intentionally mutable and not part of the removed profile-registry mutation model. The right move was to distinguish real remaining debt from valid current behavior instead of over-cleaning the docs.

### What warrants a second pair of eyes

- The remaining Phase 2 metadata compatibility path still deserves a deliberate cross-repo removal plan before touching it.
- The JS API reference should be regenerated or re-reviewed if the builder/tool-loop surface changes again, because dead options can linger there quietly.

### What should be done in the future

- Add a targeted downstream build/grep checklist for future Geppetto hard cuts so the first Pinocchio consumer layer is checked immediately.
- Continue GP-45 with the metadata compatibility decision pass rather than more broad doc work.

### Code review instructions

- Start with:
  - `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/pkg/cmds/helpers/parse-helpers.go`
  - `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/doc/topics/13-js-api-reference.md`
  - `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/js/modules/geppetto/spec/geppetto.d.ts.tmpl`
  - `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/doc/playbooks/06-operate-sqlite-profile-registry.md`
  - `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/steps/ai/settings/flags/chat.yaml`
- Validate with the commands shown above.

### Technical details

- The Geppetto commit for this slice is `c37f952` — `clean gp-45 stale docs and js typings`.
- The downstream fallout repair is `82bf805` — `drop dead profile flag bridge call`.
- The only remaining grep hit for the "mutable registry" phrase is the JS tool registry API, which is intentional current behavior and not part of the removed profile-registry mutation model.

## Step 5: Trim dead source-chain scaffolding instead of leaving unused construction state behind

After the stale-doc sweep, the next GP-45 candidate in `pkg/profiles/source_chain.go` was straightforward: `ChainedRegistry` still carried `aggregateStore` and `registryOwners`, and `sourceOwner` still carried `label` and `service`. A grep showed those fields were only written during construction and never read anywhere in Geppetto or downstream repos.

I treated that as two tiny structural cleanup commits rather than one larger refactor. The first commit removed the dead fields themselves. The second commit removed the now-unused `StoreRegistry` constructions and `sourceOwner` scaffolding that only existed to populate those fields.

### Prompt Context

**User prompt (verbatim):** "continue"

**Assistant interpretation:** Keep working through the next GP-45 tasks and land the next reviewable cleanup slices instead of stopping after the stale-doc pass.

**Inferred user intent:** Continue reducing low-value legacy scaffolding in small, well-documented commits until the remaining work is only the genuinely risky compatibility paths.

**Commit (code):** `6f08791` — `remove dead source owner scaffolding`

### What I did

- Verified that the following fields had no live readers:
  - `ChainedRegistry.aggregateStore`
  - `ChainedRegistry.registryOwners`
  - `sourceOwner.label`
  - `sourceOwner.service`
- Removed the first pair in:
  - `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/source_chain.go`
- Committed that field-only slice as:
  - `geppetto` `5ec524f` — `remove dead chained registry fields`
- Then removed the follow-on dead scaffolding in the same file:
  - dropped `sourceOwner.label` and `sourceOwner.service`,
  - removed the no-longer-needed `NewStoreRegistry(...)` construction in YAML and SQLite source openers.
- Committed that follow-on slice as:
  - `geppetto` `6f08791` — `remove dead source owner scaffolding`
- Validated with:

```bash
cd geppetto && rg -n "aggregateStore|registryOwners|service\\s+\\*StoreRegistry|\\.service\\b|label\\s+string|\\.label\\b" pkg/profiles --glob '!**/ttmp/**'
cd geppetto && go test ./pkg/profiles -run TestChainedRegistry -count=1
cd geppetto && go test ./pkg/profiles -count=1
cd geppetto && ./.bin/golangci-lint run ./pkg/profiles
```

### Why

- Carrying construction-only state inside `ChainedRegistry` and `sourceOwner` made the source-loading path look more complex than it really is.
- Once GP-45 proved the fields had no readers, keeping them would only preserve misleading structure for future maintainers.

### What worked

- The grep-based evidence was clean and local.
- `pkg/profiles` focused tests and lint passed immediately after each cleanup.
- The second commit also simplified the source-opening paths by removing now-dead `StoreRegistry` setup.

### What didn't work

- The first attempt to commit `5ec524f` hit an unrelated flaky full-repo pre-commit test:

```text
--- FAIL: TestStartWithJSEngineAndMiddleware (0.00s)
    module_test.go:583: failed to inspect promise "__startHandle.promise": ReferenceError: __startHandle is not defined at <eval>:1:1(0)
```

- I re-ran the specific failing test:

```bash
cd geppetto && go test ./pkg/js/modules/geppetto -run TestStartWithJSEngineAndMiddleware -count=1
```

- That passed immediately, and the retried commit hook then passed end to end. The failure was unrelated to `pkg/profiles/source_chain.go`.

### What I learned

- `source_chain.go` had more dead scaffolding than the first grep suggested. Removing one unused field pair exposed a second layer of unused setup cleanly.
- Small structure-only cleanup commits are still worth making when they reduce false complexity in a core configuration path.

### What was tricky to build

The only tricky part was separating real fallout from unrelated validation noise. The flaky JS-module test appeared during the first commit hook, but the failing symbol and file path were completely outside the changed area. Verifying that test in isolation before retrying the commit kept the cleanup honest without overreacting to unrelated instability.

### What warrants a second pair of eyes

- The remaining `profile.stack.trace` and runtime metadata compatibility path still needs a deliberate product/API decision before deletion.
- The next extension-codec cleanup slice should get the same grep-first treatment, because that part of the codebase is much less obviously dead than `source_chain.go`.

### What should be done in the future

- Continue GP-45 with the extension/middleware-extension usage audit or the Pinocchio auto-discovery decision.
- Keep calling out flaky unrelated tests in the diary when they appear during cleanup work, so later reviewers do not attribute them to the wrong change.

### Code review instructions

- Start with:
  - `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/source_chain.go`
- Validate with the commands shown above.

### Technical details

- The two commits for this slice are:
  - `5ec524f` — `remove dead chained registry fields`
  - `6f08791` — `remove dead source owner scaffolding`
- The YAML and SQLite source openers no longer build throwaway `StoreRegistry` values solely to populate unused `sourceOwner` fields.

## Step 6: Remove the dead service-level extension codec hook without touching the live schema path

The broader extension subsystem is not ready for blind deletion because the schema-listing path is still live in Pinocchio and the JS module. But one part of it was clearly dead: `StoreRegistry.extensionCodecs` and `WithExtensionCodecRegistry(...)`. A repo-wide search showed they were only referenced in `service.go` itself and in one test that existed purely to assert the field got wired.

That made this a good GP-45 slice: remove the dead hook from the service layer while leaving the still-live `ExtensionCodecLister` and schema-listing flow alone until a later decision pass.

### Prompt Context

**User prompt (verbatim):** (same as Step 5)

**Assistant interpretation:** Keep moving through the next GP-45 cleanup candidates, but make a narrower cut where the broader extension API still has live consumers.

**Inferred user intent:** Remove obviously dead hooks first and avoid mixing them with larger, more ambiguous extension-system decisions.

**Commit (code):** `70aa268` — `remove dead store registry codec hook`

### What I did

- Verified that `StoreRegistry.extensionCodecs` and `WithExtensionCodecRegistry(...)` had no live in-repo callers outside their own test.
- Left the schema-listing side alone because `ExtensionCodecLister` is still used by:
  - `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/pkg/webchat/http/profile_api.go`
  - `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/js/modules/geppetto/api_schemas.go`
- Removed the dead service hook from:
  - `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/service.go`
- Removed the now-dead option test from:
  - `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/extensions_test.go`
- Validated with:

```bash
cd geppetto && rg -n "WithExtensionCodecRegistry\\(|extensionCodecs\\b" pkg/profiles --glob '!**/ttmp/**'
cd geppetto && go test ./pkg/profiles -count=1
cd geppetto && ./.bin/golangci-lint run ./pkg/profiles
```

### Why

- The service-layer codec hook no longer influenced profile resolution or validation.
- Keeping it suggested that `StoreRegistry` still participated in extension decoding or schema wiring when it no longer does.
- This cut was safer than trying to remove the whole extension-registry API in one go.

### What worked

- The search evidence was clean and local.
- Focused `pkg/profiles` validation passed immediately.
- The full pre-commit hook also passed, so this slice did not inherit the earlier flaky JS test issue.

### What didn't work

- The first patch attempt for the ticket docs afterward failed because I accidentally used the wrong ticket path while updating `index.md`. That was a patching mistake, not a code issue.

### What I learned

- The extension subsystem is split cleanly enough that dead service hooks can be removed independently of live schema-discovery code.
- The next extension-related GP-45 step should be framed around actual surviving responsibilities, not the assumption that all extension machinery is dead together.

### What was tricky to build

The tricky part was resisting the temptation to delete more than the evidence supported. `NormalizeProfileExtensions(...)` and the codec `Lookup(...)` path still look suspicious, but the schema-listing side is demonstrably live. The right cut here was the dead service-level wiring only.

### What warrants a second pair of eyes

- The remaining extension-codec API should get a dedicated grep-first review before any further removal.
- `ErrReadOnlyStore` also looks increasingly suspicious now that registry services are read-only by construction, but it needs its own downstream usage check before deletion.

### What should be done in the future

- Decide whether `NormalizeProfileExtensions(...)` and codec `Lookup(...)` are still worth keeping now that the service hook is gone.
- Continue GP-45 with either that narrower extension follow-up or the metadata compatibility cleanup.

### Code review instructions

- Start with:
  - `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/service.go`
  - `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/extensions_test.go`
- Validate with the commands shown above.

### Technical details

- The service-layer extension codec hook is now gone.
- The live extension schema path still depends on `ExtensionCodecLister`, not on `StoreRegistry`.

## Step 7: Delete the dead NormalizeProfileExtensions helper instead of keeping a private codec-decode path alive

After removing the service-level extension codec hook, the next obvious dead piece was `NormalizeProfileExtensions(...)`. A repo-wide search showed it had no remaining live callers at all. The only references left were the function itself and its dedicated tests. That made it a clean GP-45 hard cut.

I deliberately did not widen this into a full extension-interface redesign. The codec registry and `ExtensionCodecLister` path are still used by schema listing in Pinocchio and the JS module. The dead helper was the right deletion boundary here.

### Prompt Context

**User prompt (verbatim):** "kill it."

**Assistant interpretation:** Remove `NormalizeProfileExtensions(...)` completely rather than leaving it as dead utility code.

**Inferred user intent:** Keep stripping obviously unused profile machinery instead of preserving helpers that no longer participate in any runtime flow.

**Commit (code):** `a68c313` — `remove dead normalize profile extensions helper`

### What I did

- Verified that `NormalizeProfileExtensions(...)` had no live callers outside its own tests.
- Removed the helper from:
  - `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/extensions.go`
- Removed the dedicated tests that only exercised that helper from:
  - `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/extensions_test.go`
- Left the broader codec-registry interface alone for now, because Pinocchio still has a test shape representing a registry without lister support.
- Validated with:

```bash
cd geppetto && go test ./pkg/profiles -count=1
cd geppetto && ./.bin/golangci-lint run ./pkg/profiles
```

### Why

- Dead utility functions create false architectural weight and invite future callers to rebuild abandoned flows.
- This helper no longer participated in profile resolution, validation, or schema serving.

### What worked

- The removal stayed local to `pkg/profiles`.
- Focused tests and lint passed immediately after deletion.

### What didn't work

- N/A

### What I learned

- The extension subsystem still needs a narrower follow-up review, but dead helper functions inside it can be removed independently.

### What was tricky to build

The tricky part was deciding whether to also remove the codec `Lookup(...)` path at the same time. I chose not to, because the current `ExtensionCodecRegistry` type is still used as a shape boundary in live code and tests, even if one of its methods is now suspicious. Removing the dead helper first keeps the cut small and reviewable.

### What warrants a second pair of eyes

- Whether the remaining `Lookup(...)` method should also be removed in the next extension-focused slice.

### What should be done in the future

- Revisit the extension codec interfaces with the narrower question: do we still need a decode-oriented registry at all, or only a lister/schema surface?

### Code review instructions

- Start with:
  - `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/extensions.go`
  - `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/extensions_test.go`
- Validate with the commands shown above.

### Technical details

- This slice removes only the dead normalization helper and its tests; it does not change schema listing behavior.

## Step 8: Collapse the extension registry contract onto the one capability we still actually use

After deleting `NormalizeProfileExtensions(...)`, the remaining awkward bit in the extension subsystem was the interface itself. `ExtensionCodecRegistry` still exposed `Lookup(...)`, but the only live behavior left was schema listing through `ListCodecs()`. That made the contract misleading: the type still looked decode-oriented even though all surviving callers were lister-oriented.

I cleaned that up in two small commits. The Geppetto side removed `Lookup(...)` and simplified the JS schema path. The Pinocchio side followed by removing the now-obsolete "registry without lister" test shape and matching the API helper to the lister-only contract.

### Prompt Context

**User prompt (verbatim):** "go ahead"

**Assistant interpretation:** Continue with the next logical extension cleanup slice after removing the dead normalization helper.

**Inferred user intent:** Keep shrinking the extension surface until it matches the responsibilities that still exist in real runtime code.

**Commit (code):** `a1f2f56` — `simplify extension codec registry to lister`

### What I did

- Removed `Lookup(...)` from `ExtensionCodecRegistry` in:
  - `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/extensions.go`
- Removed the dead `Lookup(...)` implementation from `InMemoryExtensionCodecRegistry` in the same file.
- Simplified Geppetto JS schema listing to call `ListCodecs()` directly in:
  - `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/js/modules/geppetto/api_schemas.go`
- Updated Geppetto docs in:
  - `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/doc/topics/01-profiles.md`
- Landed that as:
  - `geppetto` `a1f2f56` — `simplify extension codec registry to lister`
- Updated Pinocchio’s HTTP schema listing helper and test/docs:
  - `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/pkg/webchat/http/profile_api.go`
  - `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/pkg/webchat/http/profile_api_test.go`
  - `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/pkg/doc/topics/webchat-profile-registry.md`
- Landed that as:
  - `pinocchio` `29e25c7` — `simplify extension schema registry contract`
- Validated with:

```bash
cd geppetto && go test ./pkg/profiles ./pkg/js/modules/geppetto -count=1
cd geppetto && ./.bin/golangci-lint run ./pkg/profiles ./pkg/js/modules/geppetto
cd pinocchio && go test ./pkg/webchat/http -count=1
```

### Why

- The interface should reflect the only remaining supported behavior.
- Keeping `Lookup(...)` after removing the normalize helper would preserve a fake decode-time story that no longer exists.

### What worked

- The simplification stayed tight and mechanical.
- Both repos’ focused validations passed before commit.
- The full commit hooks passed in both repos.

### What didn't work

- N/A

### What I learned

- The extension subsystem is now much easier to explain: schema-discovery codecs are lister-based, not decode-registry based.

### What was tricky to build

The main judgment call was whether to keep a compatibility shape for "registry without lister." That path no longer had product value once the decode-oriented behavior was gone, so removing it was cleaner than preserving a special case only tests still modeled.

### What warrants a second pair of eyes

- The remaining extension subsystem still deserves one more look, especially whether any other decode-oriented helper or naming survives from the older shape.

### What should be done in the future

- Continue GP-45 with the metadata compatibility path, or do one more narrow extension pass if you want to finish that subsystem before switching topics.

### Code review instructions

- Start with:
  - `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/extensions.go`
  - `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/js/modules/geppetto/api_schemas.go`
  - `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/pkg/webchat/http/profile_api.go`
- Validate with the commands shown above.

### Technical details

- The extension registry contract is now lister-only.
- The old "registry without lister" branch is gone from Pinocchio’s schema helper and tests.

## Step 9: Remove legacy scalar inference metadata mirroring and the last fallback reader

Once the extension cleanup was out of the way, I returned to the metadata compatibility path. Earlier I had stopped short because `MirrorLegacyInferenceKeys(...)` still had active readers and writers. The user then made the intended policy explicit: there are no older readers left, so the scalar stop-reason/model/provider/usage projection should be removed outright.

That changed the cleanup decision. Instead of preserving migration glue because it was still wired into the runtime, I treated the runtime wiring itself as the next deletion target and cut both sides together: Geppetto stopped writing the legacy scalar keys, and Temporal Relationships stopped reading them as fallback.

### Prompt Context

**User prompt (verbatim):** "kill it, we have no older reader"

**Assistant interpretation:** Remove the legacy scalar inference metadata projection path instead of merely documenting it, and update the remaining downstream reader that still relied on a fallback.

**Inferred user intent:** Complete the hard cut to canonical `KeyTurnMetaInferenceResult` instead of carrying migration-era duplication indefinitely.

**Commit (code):** `43341a8` — `remove legacy scalar inference metadata`

### What I did

- Removed `MirrorLegacyInferenceKeys(...)` from:
  - `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/inference/engine/run_with_result.go`
- Simplified `RunInferenceWithResult(...)` so it now:
  - prefers engine-returned structured results,
  - otherwise reads canonical `KeyTurnMetaInferenceResult`,
  - otherwise synthesizes a minimal result from turn shape only.
- Simplified `PersistInferenceResult(...)` in:
  - `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/inference/engine/inference_result_metadata.go`
  so it writes only canonical `KeyTurnMetaInferenceResult`.
- Removed the Gemini scalar provider/model writes from:
  - `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/steps/ai/gemini/engine_gemini.go`
- Updated Geppetto tests:
  - `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/inference/engine/run_with_result_test.go`
  - `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/steps/ai/openai_responses/engine_test.go`
- Removed the last fallback reader and test echo-engine writer in Temporal Relationships:
  - `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/temporal-relationships/internal/extractor/gorunner/loop.go`
  - `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/temporal-relationships/internal/extractor/gorunner/run.go`
  - `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/temporal-relationships/internal/extractor/gorunner/run_test.go`
- Updated the Geppetto docs to describe canonical inference metadata only:
  - `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/doc/topics/06-inference-engines.md`
  - `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/doc/topics/08-turns.md`

### Why

- The user explicitly confirmed that no older reader needs the scalar compatibility projection.
- All live provider engines already persist canonical inference metadata.
- The remaining scalar path was adding duplicate write logic, fallback reads, and misleading docs without providing actual product value.

### What worked

- The focused checks validated the intended boundary cleanly:

```bash
cd geppetto && go test ./pkg/inference/engine ./pkg/steps/ai/openai_responses ./pkg/steps/ai/gemini -count=1
cd temporal-relationships && go test ./internal/extractor/gorunner/... -count=1
```

- The grep after the cut showed no remaining runtime writes or readers for the legacy scalar inference keys outside serde tests and generic metadata docs/examples.

### What didn't work

- My first patch attempt assumed the exact tail shape of `run_with_result.go` and failed to apply because I had remembered the legacy usage decoder slightly incorrectly. I had to re-read the full file and re-apply the cut more surgically.

### What I learned

- The live provider engines were already in the right place. The real remaining debt was in helper code and one downstream fallback reader, not in the providers themselves.
- Keeping a minimal synthesis path for custom bare `RunInference` engines is still useful even after removing the legacy scalar metadata path.

### What was tricky to build

- The trickiest design choice was deciding whether to delete `SynthesizeInferenceResult(...)` entirely. I kept it, but reduced it to turn-shape synthesis only, so custom test or toy engines still get a canonical result without preserving old scalar metadata semantics.

### What warrants a second pair of eyes

- Review the updated `run_with_result.go` carefully. The contract is cleaner now, but the fallback semantics changed: engines that do not persist canonical metadata no longer get provider/model/usage synthesized from old scalar keys because those writes are gone.

### What should be done in the future

- Finish the other remaining compatibility cut in `runtimeattrib.AddRuntimeAttributionToExtra(...)`, which still accepts multiple historical input shapes.

### Code review instructions

- Start with:
  - `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/inference/engine/run_with_result.go`
  - `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/inference/engine/inference_result_metadata.go`
  - `/home/manuel/workspaces/2026-03-17/add-opinionated-apis/temporal-relationships/internal/extractor/gorunner/loop.go`
- Verify the focused test commands above.
- Re-run a search for:

```bash
rg -n "MirrorLegacyInferenceKeys|KeyTurnMetaStopReason\\.Get|KeyTurnMetaStopReason\\.Set|KeyTurnMetaProvider\\.Set|KeyTurnMetaModel\\.Set|KeyTurnMetaUsage\\.Set" geppetto temporal-relationships --glob '!**/ttmp/**'
```

### Technical details

- After this step, canonical inference metadata is written once via `KeyTurnMetaInferenceResult` and then projected to generated blocks. The migration-era scalar metadata projection path is gone.

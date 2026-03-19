---
Title: GP-53 Implementation Diary
Ticket: GP-53-GEPPETTO-CLI-BOOTSTRAP
Status: active
Topics:
    - profiles
    - glazed
    - cleanup
DocType: reference
Intent: working-notes
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: ""
LastUpdated: 2026-03-19T13:10:00-04:00
WhatFor: Record the implementation sequence, technical decisions, and verification steps while extracting the generic CLI bootstrap path into Geppetto.
WhenToUse: Read before resuming GP-53 implementation work or reviewing the extraction and wrapper cutover.
---

# GP-53 Implementation Diary

## Step 1: Tighten the ticket into an executable migration plan

The original ticket was directionally correct, but it still grouped the real work into a few large steps. Before moving code, I rewrote the task list so it matches the actual seams in the implementation: first create the Geppetto package and app-config surface, then port the generic resolution layers one by one, then add tests, then migrate Pinocchio onto the new package.

This matters because the extraction has two different review axes. One axis is whether the new Geppetto package is truly generic and free of Pinocchio assumptions. The other is whether Pinocchio still behaves the same after becoming a thin wrapper. Splitting the tasks that way makes the migration reviewable and keeps the diary aligned with the code commits that will follow.

### Prompt Context

**User prompt (verbatim):** "alright, tackle the new ticket then, add tasks, work them off one by one, allow other apps to provide their own env prefix/app name."

**Assistant interpretation:** Start executing GP-53 immediately, refine the ticket task list to reflect the real implementation sequence, then carry the extraction through code and verification with app/environment parameterization as a primary requirement.

**Inferred user intent:** Do the actual Geppetto extraction now, not just discuss it, and make the result reusable by other apps rather than another Pinocchio-shaped abstraction.

### What I did

- Read the existing GP-53 task list and design guide.
- Re-expanded the task list so the implementation phases map directly onto:
  - Geppetto package creation
  - generic contract porting
  - generic config/profile/base/final resolution porting
  - focused Geppetto tests
  - Pinocchio wrapper config and wrapper cutover
- Confirmed that `geppetto/` is currently clean in Git, so the extraction can be implemented without first untangling unrelated Geppetto worktree changes.

### Why

- The original task list was still too coarse for a staged extraction with clean commits.
- The user explicitly asked to add tasks and work them off one by one.
- The migration needs a diary trail because the repository has both old and new bootstrap paths, and the review burden is mostly architectural rather than algorithmic.

### What should be reviewed first

- [tasks.md](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/ttmp/2026/03/19/GP-53-GEPPETTO-CLI-BOOTSTRAP--extract-generic-cli-bootstrap-path-to-geppetto-and-parameterize-app-name/tasks.md)
- [01-generic-geppetto-cli-bootstrap-extraction-and-app-name-parameterization-guide.md](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/ttmp/2026/03/19/GP-53-GEPPETTO-CLI-BOOTSTRAP--extract-generic-cli-bootstrap-path-to-geppetto-and-parameterize-app-name/design-doc/01-generic-geppetto-cli-bootstrap-extraction-and-app-name-parameterization-guide.md)

### Technical note

The main new requirement from the user is not just “move code to Geppetto”. It is “make app name and env prefix caller-configurable so other apps can reuse the path”. That requirement is now a first-class checkpoint in the task list rather than an implementation detail hidden inside a porting task.

## Step 2: Extract the generic bootstrap package into Geppetto and cut Pinocchio over to wrappers

This step implemented the actual architectural move. I created a new Geppetto package at [pkg/cli/bootstrap](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/cli/bootstrap) and ported the generic bootstrap path out of Pinocchio into that package. The exported contract is now app-parameterized rather than Pinocchio-shaped: callers provide app name, env prefix, config mapper, profile-section builder, and baseline-sections builder.

After the generic package compiled and passed focused Geppetto tests, I rewrote [pinocchio/pkg/cmds/profilebootstrap](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/pkg/cmds/profilebootstrap) into a thin wrapper that binds the generic Geppetto package to:

- `AppName: "pinocchio"`
- `EnvPrefix: "PINOCCHIO"`
- the Pinocchio config-file mapper
- the shared Geppetto `profile-settings` section builder
- the shared Geppetto baseline section builder

That left Pinocchio behavior intact while removing the duplicated generic implementation from the application package.

### What I did

- Added the Geppetto package as commit `3dbbb90` (`refactor(cli): add generic bootstrap package`).
- Cut Pinocchio over as commit `c3a2104` (`refactor(profilebootstrap): wrap geppetto cli bootstrap`).
- Added [config.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/cli/bootstrap/config.go) with:
  - `AppBootstrapConfig`
  - config validation
  - `ProfileSettingsSectionSlug`
- Added [profile_selection.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/cli/bootstrap/profile_selection.go) with:
  - `ProfileSettings`
  - `ResolvedCLIProfileSelection`
  - `CLISelectionInput`
  - generic config-file discovery
  - generic profile selection resolution
  - generic parsed-values builder for `config-file` plus `profile-settings`
- Added [engine_settings.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/cli/bootstrap/engine_settings.go) with:
  - `ResolvedCLIEngineSettings`
  - generic base inference settings resolution
  - generic final engine settings resolution
  - generic engine construction helper
- Added [bootstrap_test.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/cli/bootstrap/bootstrap_test.go) to cover:
  - app-name-driven config discovery
  - env-prefix-driven profile selection
  - no implicit `profiles.yaml` fallback
  - profile-without-registries validation
  - base-only mode
  - explicit profile overlay merge behavior
  - from-base parity
- Replaced the implementation in [pinocchio/pkg/cmds/profilebootstrap/profile_selection.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/pkg/cmds/profilebootstrap/profile_selection.go) with a wrapper over the new Geppetto package.
- Replaced the implementation in [pinocchio/pkg/cmds/profilebootstrap/engine_settings.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/pkg/cmds/profilebootstrap/engine_settings.go) with a wrapper over the new Geppetto package.
- Re-ran:
  - `go test ./pkg/cli/bootstrap -count=1` in `geppetto`
  - `go test ./pkg/cmds/profilebootstrap ./pkg/cmds/helpers ./pkg/cmds ./cmd/pinocchio/cmds ./cmd/examples/internal/tuidemo ./cmd/web-chat ./cmd/agents/simple-chat-agent -count=1` in `pinocchio`

### Why

- The generic bootstrap path had already stabilized in Pinocchio. Keeping it there would have made future Geppetto-backed apps either copy it or import Pinocchio for a generic concern.
- The app/environment parameterization requirement is exactly what the older `pkg/sections` helpers do not support cleanly today.
- Moving the implementation first and keeping `pkg/sections` unchanged for now minimizes the review surface. The old middleware/bootstrap helpers can be cleaned up afterward as a separate decision rather than conflating that with the extraction itself.

### What worked

- The config/callback surface was sufficient. No additional app-specific inputs were needed beyond app name, env prefix, config mapper, profile section builder, and base section builder.
- The Pinocchio wrapper became very small once the generic package existed.
- Focused tests in both repositories were enough to validate the extraction without immediately rewriting every Geppetto example.

### What didn't work

- The first Geppetto test compile failed because [sections.NewProfileSettingsSection](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/sections/profile_sections.go) and [sections.CreateGeppettoSections](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/sections/sections.go) are variadic functions and therefore cannot be assigned directly to zero-argument callback fields.
  I fixed that by using small wrapper lambdas in the test config.
- The Pinocchio worktree is still globally dirty from unrelated branch-local work, so the eventual Pinocchio wrapper commit will need the same temporary-index approach used in recent tasks.

### What I learned

- The extracted package boundary is now concrete and viable. The design guide was directionally right; the only meaningful addition was the generic `NewCLISelectionValues(...)` helper, which turned out to belong in the Geppetto package as well.
- The callback-based app config is enough to keep the package generic without introducing reflection or a more complicated registration mechanism.

### What should be reviewed first

- [config.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/cli/bootstrap/config.go)
- [profile_selection.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/cli/bootstrap/profile_selection.go)
- [engine_settings.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/cli/bootstrap/engine_settings.go)
- [bootstrap_test.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/cli/bootstrap/bootstrap_test.go)
- [pinocchio/pkg/cmds/profilebootstrap/profile_selection.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/pkg/cmds/profilebootstrap/profile_selection.go)
- [pinocchio/pkg/cmds/profilebootstrap/engine_settings.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/pinocchio/pkg/cmds/profilebootstrap/engine_settings.go)

### Remaining ticket work

- Decide how far to slim down or deprecate [sections.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/sections/sections.go) and [profile_sections.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/sections/profile_sections.go) now that the generic bootstrap path exists elsewhere.
- Decide which Geppetto examples should adopt the new package versus continuing to use direct parsed-values engine construction.

## Step 3: Close the remaining legacy-surface decisions

With the generic package and the Pinocchio wrapper in place, the two remaining GP-53 tasks were really cleanup decisions rather than implementation work. I closed them by making the split explicit in both code comments and ticket docs.

The decision is:

- keep [CreateGeppettoSections](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/sections/sections.go) and [NewProfileSettingsSection](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/sections/profile_sections.go) as reusable section-construction helpers
- keep [GetCobraCommandGeppettoMiddlewares](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/sections/sections.go) and [GetProfileSettingsMiddleware](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/sections/profile_sections.go) only as legacy middleware wiring for existing examples
- steer all new config/profile/bootstrap work toward [pkg/cli/bootstrap](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/cli/bootstrap)

For examples, the split is:

- examples that intentionally expose full AI flags and directly construct engines should keep `factory.NewEngineFromParsedValues(...)`
- examples that are profile-registry-aware or config-aware are the candidates to adopt the new bootstrap package if they need the richer behavior

### What I did

- Added explicit “legacy helper” guidance comments to:
  - [sections.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/sections/sections.go)
  - [profile_sections.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/sections/profile_sections.go)
- Checked off the final two GP-53 tasks in [tasks.md](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/ttmp/2026/03/19/GP-53-GEPPETTO-CLI-BOOTSTRAP--extract-generic-cli-bootstrap-path-to-geppetto-and-parameterize-app-name/tasks.md)
- Recorded the decisions in [changelog.md](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/ttmp/2026/03/19/GP-53-GEPPETTO-CLI-BOOTSTRAP--extract-generic-cli-bootstrap-path-to-geppetto-and-parameterize-app-name/changelog.md)

### Why

- The old `pkg/sections` surface still exists and is still used by examples, so deleting it immediately would have turned GP-53 into a much larger example-migration ticket.
- The new package needed a clear ownership boundary. Without explicit guidance, future work could still drift back into the legacy middleware helpers.
- The examples do not all solve the same problem. Full-flag teaching examples should stay simple; registry-aware/config-aware examples are the ones that actually benefit from the richer bootstrap package.

### Final state

At the end of this step, GP-53 is effectively complete. The generic package exists, Pinocchio uses it through a thin wrapper, and the old Geppetto bootstrap helpers are explicitly positioned as legacy wiring rather than the preferred future path.

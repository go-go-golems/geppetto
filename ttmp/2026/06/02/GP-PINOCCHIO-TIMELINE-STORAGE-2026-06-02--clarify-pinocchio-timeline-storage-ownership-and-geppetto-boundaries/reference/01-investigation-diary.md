---
title: Investigation diary
doc_type: reference
status: active
intent: long-term
topics:
  - pinocchio
  - geppetto
  - persistence
  - timeline
  - turns
owners:
  - manuel
created: 2026-06-02
updated: 2026-06-02
---

# Diary

## Goal

Record the investigation and design work for the timeline storage ownership boundary between Pinocchio and Geppetto.

## Step 1: Separate timeline storage from turn storage

I investigated Pinocchio's persistence settings, CLI flags, sessionstream hydration store setup, turn-store setup, and interactive chat wiring. The main finding is that Pinocchio already treats timeline and turn persistence as separate stores, and timeline storage is tied to chat UI hydration rather than core Geppetto inference.

I then wrote an intern-oriented guide recommending that timeline storage remain Pinocchio/sessionstream-owned. Geppetto should not gain a `gp.timeline` API in the first storage pass; it should only provide final-turn persistence and event/turn APIs that Pinocchio can consume.

### Prompt Context

**User prompt (verbatim):** (same as Step 1 in `GP-JS-TURNSTORE-2026-06-02` diary)

**Assistant interpretation:** Create two docmgr tickets and write detailed design/implementation guides; this ticket covers the timeline/other-storage concern and clarifies it is probably not a Geppetto concern.

**Inferred user intent:** Prevent storage API scope creep by clearly assigning timeline persistence to Pinocchio while documenting how it composes with Geppetto turn storage.

**Commit (code):** N/A — documentation/design only.

### What I did
- Created docmgr ticket `GP-PINOCCHIO-TIMELINE-STORAGE-2026-06-02`.
- Added design document `design-doc/01-timeline-storage-ownership-and-integration-boundary-guide.md`.
- Read and cited Pinocchio persistence settings, CLI flags, timeline store opener, turn store opener, and interactive chat wiring.
- Proposed an ownership table, diagrams, provider/host boundary, implementation phases, and tests.

### Why
- Timeline storage represents UI/application state, not only Geppetto inference turns.
- The user explicitly suspected timeline was probably not a Geppetto concern, so the design needed to make that boundary concrete.

### What worked
- Pinocchio already has separate helpers for `openCLISessionstreamHydrationStore(...)` and `openCLITurnStore(...)`.
- The CLI already exposes separate `timeline-*` and `turns-*` flags.
- The interactive chat path clearly uses hydration snapshots for visible UI state, which supports keeping timeline ownership in Pinocchio.

### What didn't work
- I initially created an unrelated turn-append ticket while interpreting the second concern, then removed that mistaken workspace and created the correct timeline-storage ticket.
- Command: `rm -rf ttmp/2026/06/02/GP-JS-TURN-APPEND-2026-06-02--design-ergonomic-javascript-turn-append-apis`.

### What I learned
- Pinocchio's `PersistenceSettings` already groups timeline and turn storage values but preserves distinct open paths.
- Resume behavior currently uses turn storage, not timeline storage.
- Timeline persistence should be mediated by host services or a Pinocchio-specific module, not by the generic Geppetto JS API.

### What was tricky to build
- The tricky part was deciding whether provider config should mention timeline at all. Mentioning it in Geppetto config risks implying a JS API; ignoring it makes combined xgoja storage setup less discoverable.
- The guide resolves this by allowing host-mediated timeline config while explicitly not adding `gp.timeline` in Geppetto.

### What warrants a second pair of eyes
- Whether timeline config belongs inside the Geppetto provider block or in a separate Pinocchio/xgoja package config block.
- Whether a future `require("pinocchio/timeline")` module would be more appropriate than any Geppetto namespace.

### What should be done in the future
- Extract shared Pinocchio storage open helpers for CLI and xgoja host reuse.
- Add dependency regression tests preventing Geppetto from importing Pinocchio/sessionstream.
- Decide if and where host-mediated timeline provider config should live.

### Code review instructions
- Start with `pinocchio/pkg/cmds/chat_persistence.go`, `pinocchio/pkg/cmds/cmd.go`, and `pinocchio/pkg/cmds/cmdlayers/helpers.go`.
- Validate that timeline persistence remains Pinocchio-owned and Geppetto exposes no public `gp.timeline` API.

### Technical details
- Timeline opener: `openCLISessionstreamHydrationStore(...)`.
- Turn opener: `openCLITurnStore(...)`.
- Interactive chat runner receives both `HydrationStore` and `TurnStore` through `chatapp.RunnerOptions`.
